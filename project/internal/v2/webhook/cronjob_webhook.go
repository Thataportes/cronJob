package webhook

import (
	"context"
	"fmt"

	"github.com/robfig/cron"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	validationutils "k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	batchv1 "tutorial.kubebuilder.io/project/api/v1"
)

var cronjoblog = logf.Log.WithName("cronjob-resource")

// SetupCronJobWebhookWithManager registers the webhook for CronJob in the manager.
func SetupCronJobWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&batchv1.CronJob{}).
		WithValidator(&CronJobCustomValidator{}).
		WithDefaulter(&CronJobCustomDefaulter{
			DefaultConcurrencyPolicy:          batchv1.AllowConcurrent,
			DefaultSuspend:                    false,
			DefaultSuccessfulJobsHistoryLimit: 3,
			DefaultFailedJobsHistoryLimit:     1,
		}).
		Complete()
}

// +kubebuilder:webhook:path=/mutate-batch-tutorial-kubebuilder-io-v1-cronjob,mutating=true,failurePolicy=fail,sideEffects=None,groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=create;update,versions=v1,name=mcronjob-v1.kb.io,admissionReviewVersions=v1

// CronJobCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind CronJob when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type CronJobCustomDefaulter struct {

	// Default values for various CronJob fields
	DefaultConcurrencyPolicy          batchv1.ConcurrencyPolicy
	DefaultSuspend                    bool
	DefaultSuccessfulJobsHistoryLimit int32
	DefaultFailedJobsHistoryLimit     int32
}

var _ webhook.CustomDefaulter = &CronJobCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind CronJob.
func (d *CronJobCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	cronjob, ok := obj.(*batchv1.CronJob)

	if !ok {
		return fmt.Errorf("expected an CronJob object but got %T", obj)
	}
	cronjoblog.Info("Defaulting for CronJob", "name", cronjob.GetName())

	// Set default values
	d.applyDefaults(cronjob)
	return nil
}

// applyDefaults applies default values to CronJob fields.
func (d *CronJobCustomDefaulter) applyDefaults(cronJob *batchv1.CronJob) {
	if cronJob.Spec.ConcurrencyPolicy == "" {
		cronJob.Spec.ConcurrencyPolicy = d.DefaultConcurrencyPolicy
	}
	if cronJob.Spec.Suspend == nil {
		cronJob.Spec.Suspend = new(bool)
		*cronJob.Spec.Suspend = d.DefaultSuspend
	}
	if cronJob.Spec.SuccessfulJobsHistoryLimit == nil {
		cronJob.Spec.SuccessfulJobsHistoryLimit = new(int32)
		*cronJob.Spec.SuccessfulJobsHistoryLimit = d.DefaultSuccessfulJobsHistoryLimit
	}
	if cronJob.Spec.FailedJobsHistoryLimit == nil {
		cronJob.Spec.FailedJobsHistoryLimit = new(int32)
		*cronJob.Spec.FailedJobsHistoryLimit = d.DefaultFailedJobsHistoryLimit
	}
}

// +kubebuilder:webhook:path=/validate-batch-tutorial-kubebuilder-io-v1-cronjob,mutating=false,failurePolicy=fail,sideEffects=None,groups=batch.tutorial.kubebuilder.io,resources=cronjobs,verbs=create;update,versions=v1,name=vcronjob-v1.kb.io,admissionReviewVersions=v1

// CronJobCustomValidator struct is responsible for validating the CronJob resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type CronJobCustomValidator struct {
	//TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &CronJobCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type CronJob.
func (v *CronJobCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cronjob, ok := obj.(*batchv1.CronJob)
	if !ok {
		return nil, fmt.Errorf("expected a CronJob object but got %T", obj)
	}
	cronjoblog.Info("Validation for CronJob upon creation", "name", cronjob.GetName())

	return nil, validateCronJob(cronjob)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type CronJob.
func (v *CronJobCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	cronjob, ok := newObj.(*batchv1.CronJob)
	if !ok {
		return nil, fmt.Errorf("expected a CronJob object for the newObj but got %T", newObj)
	}
	cronjoblog.Info("Validation for CronJob upon update", "name", cronjob.GetName())

	return nil, validateCronJob(cronjob)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type CronJob.
func (v *CronJobCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	cronjob, ok := obj.(*batchv1.CronJob)
	if !ok {
		return nil, fmt.Errorf("expected a CronJob object but got %T", obj)
	}
	cronjoblog.Info("Validation for CronJob upon deletion", "name", cronjob.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}

// validateCronJob validates the fields of a CronJob object.
func validateCronJob(cronjob *batchv1.CronJob) error {
	var allErrs field.ErrorList
	if err := validateCronJobName(cronjob); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := validateCronJobSpec(cronjob); err != nil {
		allErrs = append(allErrs, err)
	}
	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: "batch.tutorial.kubebuilder.io", Kind: "CronJob"},
		cronjob.Name, allErrs)
}
func validateCronJobSpec(cronjob *batchv1.CronJob) *field.Error {
	// The field helpers from the kubernetes API machinery help us return nicely
	// structured validation errors.
	return validateScheduleFormat(
		cronjob.Spec.Schedule,
		field.NewPath("spec").Child("schedule"))
}
func validateScheduleFormat(schedule string, fldPath *field.Path) *field.Error {
	if _, err := cron.ParseStandard(schedule); err != nil {
		return field.Invalid(fldPath, schedule, err.Error())
	}
	return nil
}
func validateCronJobName(cronjob *batchv1.CronJob) *field.Error {
	if len(cronjob.ObjectMeta.Name) > validationutils.DNS1035LabelMaxLength-11 {
		// The job name length is 63 characters like all Kubernetes objects
		// (which must fit in a DNS subdomain). The cronjob controller appends
		// a 11-character suffix to the cronjob (`-$TIMESTAMP`) when creating
		// a job. The job name length limit is 63 characters. Therefore cronjob
		// names must have length <= 63-11=52. If we don't validate this here,
		// then job creation will fail later.
		return field.Invalid(field.NewPath("metadata").Child("name"), cronjob.ObjectMeta.Name, "must be no more than 52 characters")
	}
	return nil
}