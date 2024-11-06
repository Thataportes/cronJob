package e2e

import (
	"fmt"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"tutorial.kubebuilder.io/project/test/utils"
)

const (
	namespace            = "project-system"
	prometheusNamespace  = "prometheus-operator"
	certManagerNamespace = "cert-manager"
)

var _ = Describe("controller", Ordered, func() {
	BeforeAll(func() {
		By("installing Prometheus operator")
		cmd := exec.Command("kubectl", "create", "-f", "https://github.com/prometheus-operator/prometheus-operator/releases/download/v0.72.0/bundle.yaml")
		output, err := utils.Run(cmd)
		fmt.Fprintf(GinkgoWriter, "Prometheus install output: %s\n", output)
		Expect(err).NotTo(HaveOccurred(), "Failed to install Prometheus operator")

		// Verifique a existência do namespace logo após a instalação
		By("verifying Prometheus namespace is created")
		Eventually(func() error {
			cmd := exec.Command("kubectl", "get", "namespace", prometheusNamespace)
			_, err := utils.Run(cmd)
			return err
		}, 1*time.Minute, 5*time.Second).Should(Succeed(), fmt.Sprintf("Namespace %s was not created", prometheusNamespace))

		// Aguardar o operador Prometheus estar pronto
		By("waiting for Prometheus operator to be ready")
		cmd = exec.Command("kubectl", "wait", "--for=condition=Available", "deployment", "-l", "app.kubernetes.io/name=prometheus-operator", "--timeout=10m", "-n", prometheusNamespace)
		_, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "Prometheus operator did not become ready in time")

		By("installing cert-manager")
		cmd = exec.Command("kubectl", "apply", "-f", "https://github.com/jetstack/cert-manager/releases/download/v1.14.4/cert-manager.yaml")
		output, err = utils.Run(cmd)
		fmt.Fprintf(GinkgoWriter, "Cert-manager install output: %s\n", output)
		Expect(err).NotTo(HaveOccurred(), "Failed to install cert-manager")

		By("verifying cert-manager namespace exists")
		Eventually(func() error {
			cmd := exec.Command("kubectl", "get", "namespace", certManagerNamespace)
			_, err := utils.Run(cmd)
			return err
		}, 2*time.Minute, 5*time.Second).Should(Succeed(), fmt.Sprintf("Namespace %s does not exist", certManagerNamespace))

		By("waiting for cert-manager to be ready")
		cmd = exec.Command("kubectl", "wait", "--for=condition=Available", "deployment", "-l", "app.kubernetes.io/name=cert-manager", "--timeout=10m", "-n", certManagerNamespace)
		_, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred(), "cert-manager did not become ready in time")

		By("checking if manager namespace already exists")
		cmd = exec.Command("kubectl", "get", "ns", namespace)
		_, err = utils.Run(cmd)
		if err != nil {
			fmt.Fprintf(GinkgoWriter, "Namespace %s does not exist, creating...\n", namespace)
			cmd = exec.Command("kubectl", "create", "ns", namespace)
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())
		} else {
			fmt.Fprintf(GinkgoWriter, "Namespace %s already exists\n", namespace)
		}
	})

	AfterAll(func() {
		By("uninstalling the Prometheus manager bundle")
		cmd := exec.Command("kubectl", "delete", "-f", "https://github.com/prometheus-operator/prometheus-operator/releases/download/v0.72.0/bundle.yaml")
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to uninstall Prometheus operator")

		By("uninstalling the cert-manager bundle")
		cmd = exec.Command("kubectl", "delete", "-f", "https://github.com/jetstack/cert-manager/releases/download/v1.14.4/cert-manager.yaml")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to uninstall cert-manager")

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace, "--wait=true")
		_, err = utils.Run(cmd)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())
		fmt.Fprintf(GinkgoWriter, "Namespace %s deleted\n", namespace)
	})

	Context("Operator", func() {
		It("should run successfully", func() {
			var controllerPodName string
			var err error

			var projectimage = "example.com/project:v0.0.1"

			By("building the manager(Operator) image")
			cmd := exec.Command("make", "docker-build", fmt.Sprintf("IMG=%s", projectimage))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("loading the manager(Operator) image on Kind")
			err = utils.LoadImageToKindClusterWithName(projectimage, "cronjob-cluster-kind")
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("installing CRDs")
			cmd = exec.Command("make", "install")
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("deploying the controller-manager")
			cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectimage))
			_, err = utils.Run(cmd)
			ExpectWithOffset(1, err).NotTo(HaveOccurred())

			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func() error {
				cmd = exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)
				podOutput, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				podNames := utils.GetNonEmptyLines(string(podOutput))
				if len(podNames) != 1 {
					return fmt.Errorf("expect 1 controller pod running, but got %d", len(podNames))
				}
				controllerPodName = podNames[0]
				ExpectWithOffset(2, controllerPodName).Should(ContainSubstring("controller-manager"))

				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				status, err := utils.Run(cmd)
				ExpectWithOffset(2, err).NotTo(HaveOccurred())
				if string(status) != "Running" {
					return fmt.Errorf("controller pod in %s status", status)
				}
				return nil
			}
			EventuallyWithOffset(1, verifyControllerUp, 2*time.Minute, time.Second).Should(Succeed())
		})
	})
})
