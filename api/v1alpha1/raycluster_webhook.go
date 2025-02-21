package v1alpha1

import (
	"fmt"

	securityv1beta1 "istio.io/api/security/v1beta1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	rayMinValidPort int32 = 1024
	rayMaxValidPort int32 = 65535
)

var (
	rayDefaultRedisPort           int32 = 6379
	rayDefaultClientServerPort    int32 = 10001
	rayDefaultObjectManagerPort   int32 = 2384
	rayDefaultNodeManagerPort     int32 = 2385
	rayDefaultGCSServerPort       int32 = 2386
	rayDefaultDashboardPort       int32 = 8265
	rayDefaultRedisShardPorts           = []int32{6380, 6381}
	rayDefaultEnableDashboard           = pointer.BoolPtr(true)
	rayDefaultEnableNetworkPolicy       = pointer.BoolPtr(true)
	rayDefaultWorkerReplicas            = pointer.Int32Ptr(1)
	rayDefaultNetworkPolicyLabels       = map[string]string{
		"ray-client": "true",
	}

	rayDefaultImage = &OCIImageDefinition{
		Repository: "rayproject/ray",
		Tag:        "1.3.0-cpu",
	}
)

// logger is for webhook logging.
var logger = logf.Log.WithName("webhooks").WithName("RayCluster")

// SetupWebhookWithManager creates and registers this webhook with the manager.
func (r *RayCluster) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//+kubebuilder:webhook:path=/mutate-distributed-compute-dominodatalab-com-v1alpha1-raycluster,mutating=true,failurePolicy=fail,sideEffects=None,groups=distributed-compute.dominodatalab.com,resources=rayclusters,verbs=create;update,versions=v1alpha1,name=mraycluster.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &RayCluster{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *RayCluster) Default() {
	log := logger.WithValues("raycluster", client.ObjectKeyFromObject(r))
	log.Info("applying defaults")

	if r.Spec.Port == 0 {
		log.Info("setting default port", "value", rayDefaultRedisPort)
		r.Spec.Port = rayDefaultRedisPort
	}
	if r.Spec.RedisShardPorts == nil {
		log.Info("setting default redis shard ports", "value", rayDefaultRedisShardPorts)
		r.Spec.RedisShardPorts = rayDefaultRedisShardPorts
	}
	if r.Spec.ClientServerPort == 0 {
		log.Info("setting default client server port", "value", rayDefaultClientServerPort)
		r.Spec.ClientServerPort = rayDefaultClientServerPort
	}
	if r.Spec.ObjectManagerPort == 0 {
		log.Info("setting default object manager port", "value", rayDefaultObjectManagerPort)
		r.Spec.ObjectManagerPort = rayDefaultObjectManagerPort
	}
	if r.Spec.GCSServerPort == 0 {
		log.Info("setting default gcs server port", "value", rayDefaultGCSServerPort)
		r.Spec.GCSServerPort = rayDefaultGCSServerPort
	}
	if r.Spec.NodeManagerPort == 0 {
		log.Info("setting default node manager port", "value", rayDefaultNodeManagerPort)
		r.Spec.NodeManagerPort = rayDefaultNodeManagerPort
	}
	if r.Spec.DashboardPort == 0 {
		log.Info("setting default dashboard port", "value", rayDefaultDashboardPort)
		r.Spec.DashboardPort = rayDefaultDashboardPort
	}
	if r.Spec.EnableDashboard == nil {
		log.Info("setting enable dashboard flag", "value", *rayDefaultEnableDashboard)
		r.Spec.EnableDashboard = rayDefaultEnableDashboard
	}
	if r.Spec.NetworkPolicy.Enabled == nil {
		log.Info("setting enable network policy flag", "value", *rayDefaultEnableNetworkPolicy)
		r.Spec.NetworkPolicy.Enabled = rayDefaultEnableNetworkPolicy
	}
	if r.Spec.NetworkPolicy.ClientServerLabels == nil {
		log.Info("setting default network policy client server labels", "value", rayDefaultNetworkPolicyLabels)
		r.Spec.NetworkPolicy.ClientServerLabels = rayDefaultNetworkPolicyLabels
	}
	if r.Spec.NetworkPolicy.DashboardLabels == nil {
		log.Info("setting default network policy dashboard labels", "value", rayDefaultNetworkPolicyLabels)
		r.Spec.NetworkPolicy.DashboardLabels = rayDefaultNetworkPolicyLabels
	}
	if r.Spec.Worker.Replicas == nil {
		log.Info("setting default worker replicas", "value", *rayDefaultWorkerReplicas)
		r.Spec.Worker.Replicas = rayDefaultWorkerReplicas
	}
	if r.Spec.Image == nil {
		log.Info("setting default image", "value", *rayDefaultImage)
		r.Spec.Image = rayDefaultImage
	}
}

//+kubebuilder:webhook:path=/validate-distributed-compute-dominodatalab-com-v1alpha1-raycluster,mutating=false,failurePolicy=fail,sideEffects=None,groups=distributed-compute.dominodatalab.com,resources=rayclusters,verbs=create;update,versions=v1alpha1,name=vraycluster.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &RayCluster{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type.
func (r *RayCluster) ValidateCreate() error {
	logger.WithValues("raycluster", client.ObjectKeyFromObject(r)).Info("validating create")

	return r.validateRayCluster()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type.
func (r *RayCluster) ValidateUpdate(old runtime.Object) error {
	logger.WithValues("raycluster", client.ObjectKeyFromObject(r)).Info("validating update")

	return r.validateRayCluster()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type.
// Not used, just here for interface compliance.
func (r *RayCluster) ValidateDelete() error {
	return nil
}

func (r *RayCluster) validateRayCluster() error {
	var allErrs field.ErrorList

	if err := r.validateMutualTLSMode(); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := r.validateWorkerReplicas(); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := r.validateWorkerResourceRequestsCPU(); err != nil {
		allErrs = append(allErrs, err)
	}
	if err := r.validateObjectStoreMemoryBytes(); err != nil {
		allErrs = append(allErrs, err)
	}
	if errs := r.validatePorts(); errs != nil {
		allErrs = append(allErrs, errs...)
	}
	if errs := r.validateAutoscaler(); errs != nil {
		allErrs = append(allErrs, errs...)
	}
	if errs := r.validateImage(); errs != nil {
		allErrs = append(allErrs, errs...)
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(
		schema.GroupKind{Group: GroupVersion.Group, Kind: "RayCluster"},
		r.Name,
		allErrs,
	)
}

func (r *RayCluster) validateMutualTLSMode() *field.Error {
	if r.Spec.MutualTLSMode == "" {
		return nil
	}
	if _, ok := securityv1beta1.PeerAuthentication_MutualTLS_Mode_value[r.Spec.MutualTLSMode]; ok {
		return nil
	}

	var validModes []string
	for s := range securityv1beta1.PeerAuthentication_MutualTLS_Mode_value {
		validModes = append(validModes, s)
	}

	return field.Invalid(
		field.NewPath("spec").Child("istioMutualTLSMode"),
		r.Spec.MutualTLSMode,
		fmt.Sprintf("mode must be one of the following: %v", validModes),
	)
}

func (r *RayCluster) validateImage() field.ErrorList {
	var errs field.ErrorList
	fldPath := field.NewPath("spec").Child("image")

	if r.Spec.Image.Repository == "" {
		errs = append(errs, field.Required(fldPath.Child("repository"), "cannot be blank"))
	}
	if r.Spec.Image.Tag == "" {
		errs = append(errs, field.Required(fldPath.Child("tag"), "cannot be blank"))
	}

	return errs
}

func (r *RayCluster) validateWorkerReplicas() *field.Error {
	replicas := r.Spec.Worker.Replicas
	if replicas == nil || *replicas >= 0 {
		return nil
	}

	return field.Invalid(
		field.NewPath("spec").Child("worker").Child("replicas"),
		replicas,
		"should be greater than or equal to 0",
	)
}

func (r *RayCluster) validateWorkerResourceRequestsCPU() *field.Error {
	if r.Spec.Autoscaling == nil {
		return nil
	}
	if _, ok := r.Spec.Worker.Resources.Requests[v1.ResourceCPU]; ok {
		return nil
	}

	return field.Required(
		field.NewPath("spec").Child("worker").Child("resources").Child("requests").Child("cpu"),
		"is mandatory when autoscaling is enabled",
	)
}

func (r *RayCluster) validateObjectStoreMemoryBytes() *field.Error {
	memBytes := r.Spec.ObjectStoreMemoryBytes

	if memBytes == nil || *memBytes >= 78643200 {
		return nil
	}

	return field.Invalid(
		field.NewPath("spec").Child("objectStoreMemoryBytes"),
		memBytes,
		"should be greater than or equal to 78643200",
	)
}

func (r *RayCluster) validatePorts() field.ErrorList {
	var errs field.ErrorList

	if err := r.validatePort(r.Spec.Port, field.NewPath("spec").Child("port")); err != nil {
		errs = append(errs, err)
	}

	for idx, port := range r.Spec.RedisShardPorts {
		name := fmt.Sprintf("redisShardPorts[%d]", idx)
		if err := r.validatePort(port, field.NewPath("spec").Child(name)); err != nil {
			errs = append(errs, err)
		}
	}

	for idx, port := range r.Spec.WorkerPorts {
		name := fmt.Sprintf("workerPorts[%d]", idx)
		if err := r.validatePort(port, field.NewPath("spec").Child(name)); err != nil {
			errs = append(errs, err)
		}
	}

	if err := r.validatePort(r.Spec.ClientServerPort, field.NewPath("spec").Child("clientServerPort")); err != nil {
		errs = append(errs, err)
	}
	if err := r.validatePort(r.Spec.ObjectManagerPort, field.NewPath("spec").Child("objectManagerPort")); err != nil {
		errs = append(errs, err)
	}
	if err := r.validatePort(r.Spec.NodeManagerPort, field.NewPath("spec").Child("nodeManagerPort")); err != nil {
		errs = append(errs, err)
	}
	if err := r.validatePort(r.Spec.GCSServerPort, field.NewPath("spec").Child("gcsServerPort")); err != nil {
		errs = append(errs, err)
	}
	if err := r.validatePort(r.Spec.DashboardPort, field.NewPath("spec").Child("dashboardPort")); err != nil {
		errs = append(errs, err)
	}

	// TODO: add validation to prevent port values overlap

	return errs
}

func (r *RayCluster) validatePort(port int32, fldPath *field.Path) *field.Error {
	if port < rayMinValidPort {
		return field.Invalid(fldPath, port, fmt.Sprintf("must be greater than or equal to %d", rayMinValidPort))
	}
	if port > rayMaxValidPort {
		return field.Invalid(fldPath, port, fmt.Sprintf("must be less than or equal to %d", rayMaxValidPort))
	}

	return nil
}

// nolint:dupl
func (r *RayCluster) validateAutoscaler() field.ErrorList {
	var errs field.ErrorList

	as := r.Spec.Autoscaling
	if as == nil {
		return nil
	}

	fldPath := field.NewPath("spec").Child("autoscaling")

	if as.MinReplicas != nil {
		if *as.MinReplicas < 1 {
			errs = append(errs, field.Invalid(
				fldPath.Child("minReplicas"),
				as.MinReplicas,
				"must be greater than or equal to 1",
			))
		}

		if *as.MinReplicas > as.MaxReplicas {
			errs = append(errs, field.Invalid(
				fldPath.Child("maxReplicas"),
				as.MaxReplicas,
				"cannot be less than spec.autoscaling.minReplicas",
			))
		}
	}

	if as.MaxReplicas < 1 {
		errs = append(errs, field.Invalid(
			fldPath.Child("maxReplicas"),
			as.MaxReplicas,
			"must be greater than or equal to 1",
		))
	}

	if as.AverageCPUUtilization != nil && *as.AverageCPUUtilization <= 0 {
		errs = append(errs, field.Invalid(
			fldPath.Child("averageUtilization"),
			as.AverageCPUUtilization,
			"must be greater than 0",
		))
	}

	if as.ScaleDownStabilizationWindowSeconds != nil && *as.ScaleDownStabilizationWindowSeconds < 0 {
		errs = append(errs, field.Invalid(
			fldPath.Child("scaleDownStabilizationWindowSeconds"),
			as.ScaleDownStabilizationWindowSeconds,
			"must be greater than or equal to 0",
		))
	}

	return errs
}
