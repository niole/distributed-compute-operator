package dask

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dcv1alpha1 "github.com/dominodatalab/distributed-compute-operator/api/v1alpha1"
	"github.com/dominodatalab/distributed-compute-operator/pkg/cluster/metadata"
	"github.com/dominodatalab/distributed-compute-operator/pkg/controller/components"
	"github.com/dominodatalab/distributed-compute-operator/pkg/util"
)

type statefulSetDS struct {
	tc   typeConfig
	dc   *dcv1alpha1.DaskCluster
	comp metadata.Component
}

func SchedulerStatefulSet(obj client.Object) components.StatefulSetDataSource {
	dc := obj.(*dcv1alpha1.DaskCluster)
	tc := &schedulerConfig{dc: dc}

	return &statefulSetDS{tc, dc, ComponentScheduler}
}

func WorkerStatefulSet(obj client.Object) components.StatefulSetDataSource {
	dc := obj.(*dcv1alpha1.DaskCluster)
	tc := &workerConfig{dc: dc}

	return &statefulSetDS{tc, dc, ComponentWorker}
}

func (s *statefulSetDS) GetStatefulSet() (*appsv1.StatefulSet, error) {
	imageDef, err := util.ParseImageDefinition(s.image())
	if err != nil {
		return nil, fmt.Errorf("cannot parse image: %w", err)
	}

	sts := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.name(),
			Namespace: s.namespace(),
			Labels:    s.labels(),
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas:    s.replicas(),
			ServiceName: s.serviceName(),
			Selector: &metav1.LabelSelector{
				MatchLabels: s.matchLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      s.labels(),
					Annotations: s.podAnnotations(),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: s.serviceAccountName(),
					NodeSelector:       s.nodeSelector(),
					Affinity:           s.affinity(),
					Tolerations:        s.tolerations(),
					InitContainers:     s.initContainers(),
					ImagePullSecrets:   s.imagePullSecrets(),
					SecurityContext:    s.securityContext(),
					Volumes:            s.volumes(),
					Containers: []corev1.Container{
						{
							Name:            s.applicationName(),
							Command:         s.command(),
							Args:            s.commandArgs(),
							Image:           imageDef,
							ImagePullPolicy: s.image().PullPolicy,
							Ports:           s.ports(),
							Env:             s.env(),
							VolumeMounts:    s.volumeMounts(),
							Resources:       s.resources(),
							LivenessProbe:   s.probe(),
							ReadinessProbe:  s.probe(),
						},
					},
				},
			},
			PodManagementPolicy: appsv1.ParallelPodManagement,
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
		},
	}

	return sts, nil
}

func (s *statefulSetDS) applicationName() string {
	return ApplicationName
}

func (s *statefulSetDS) name() string {
	return meta.InstanceName(s.dc, s.comp)
}

func (s *statefulSetDS) namespace() string {
	return s.dc.Namespace
}

func (s *statefulSetDS) labels() map[string]string {
	labels := meta.StandardLabelsWithComponent(s.dc, s.comp)
	return util.MergeStringMaps(s.tc.podConfig().Labels, labels)
}

func (s *statefulSetDS) matchLabels() map[string]string {
	return meta.MatchLabelsWithComponent(s.dc, s.comp)
}

func (s *statefulSetDS) serviceName() string {
	return meta.InstanceName(s.dc, s.comp)
}

func (s *statefulSetDS) serviceAccountName() string {
	return meta.InstanceName(s.dc, metadata.ComponentNone)
}

func (s *statefulSetDS) image() *dcv1alpha1.OCIImageDefinition {
	return s.dc.Spec.Image
}

func (s *statefulSetDS) imagePullSecrets() []corev1.LocalObjectReference {
	return s.dc.Spec.ImagePullSecrets
}

func (s *statefulSetDS) securityContext() *corev1.PodSecurityContext {
	return s.dc.Spec.PodSecurityContext
}

func (s *statefulSetDS) env() []corev1.EnvVar {
	envvars := s.dc.Spec.EnvVars
	envvars = append(envvars, s.tc.containerEnv()...)

	return envvars
}

func (s *statefulSetDS) replicas() *int32 {
	return pointer.Int32Ptr(s.tc.replicas())
}

func (s *statefulSetDS) command() []string {
	return s.tc.command()
}

func (s *statefulSetDS) commandArgs() []string {
	return s.tc.commandArgs()
}

func (s *statefulSetDS) ports() []corev1.ContainerPort {
	return s.tc.containerPorts()
}

func (s *statefulSetDS) podAnnotations() map[string]string {
	return s.tc.podConfig().Annotations
}

func (s *statefulSetDS) nodeSelector() map[string]string {
	return s.tc.podConfig().NodeSelector
}

func (s *statefulSetDS) affinity() *corev1.Affinity {
	return s.tc.podConfig().Affinity
}

func (s *statefulSetDS) tolerations() []corev1.Toleration {
	return s.tc.podConfig().Tolerations
}

func (s *statefulSetDS) initContainers() []corev1.Container {
	return s.tc.podConfig().InitContainers
}

func (s *statefulSetDS) volumes() []corev1.Volume {
	return s.tc.podConfig().Volumes
}

func (s *statefulSetDS) volumeMounts() []corev1.VolumeMount {
	return s.tc.podConfig().VolumeMounts
}

func (s *statefulSetDS) resources() corev1.ResourceRequirements {
	return s.tc.podConfig().Resources
}

func (s *statefulSetDS) probe() *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/health",
				Port: intstr.FromString("dashboard"),
			},
		},
	}
}

type typeConfig interface {
	podConfig() dcv1alpha1.WorkloadConfig
	replicas() int32
	command() []string
	commandArgs() []string
	containerEnv() []corev1.EnvVar
	containerPorts() []corev1.ContainerPort
}

type schedulerConfig struct {
	dc *dcv1alpha1.DaskCluster
}

func (c *schedulerConfig) podConfig() dcv1alpha1.WorkloadConfig {
	return c.dc.Spec.Scheduler
}

func (c *schedulerConfig) replicas() int32 {
	return 1
}

func (c *schedulerConfig) command() []string {
	return []string{"dask-scheduler"}
}

func (c *schedulerConfig) commandArgs() []string {
	return []string{
		fmt.Sprintf("--port=%d", c.dc.Spec.SchedulerPort),
		fmt.Sprintf("--dashboard-address=:%d", c.dc.Spec.DashboardPort),
	}
}

func (c *schedulerConfig) containerEnv() []corev1.EnvVar {
	return nil
}

func (c *schedulerConfig) containerPorts() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{
			Name:          "serve",
			ContainerPort: c.dc.Spec.SchedulerPort,
		},
		{
			Name:          "dashboard",
			ContainerPort: c.dc.Spec.DashboardPort,
		},
	}
}

type workerConfig struct {
	dc *dcv1alpha1.DaskCluster
}

func (c *workerConfig) podConfig() dcv1alpha1.WorkloadConfig {
	return c.dc.Spec.Worker.WorkloadConfig
}

func (c *workerConfig) replicas() int32 {
	return c.dc.Spec.Worker.Replicas
}

func (c *workerConfig) command() []string {
	return []string{"dask-worker"}
}

func (c *workerConfig) commandArgs() []string {
	return []string{
		"--name=$(MY_POD_NAME)",
		// NOTE: it looks like the dask worker can infer its threads/memory from resource.limits
		// "--nthreads=$(MY_CPU_LIMIT)",
		// "--memory=$(MY_MEM_LIMIT)",
		fmt.Sprintf("--worker-port=%d", c.dc.Spec.WorkerPort),
		fmt.Sprintf("--nanny-port=%d", c.dc.Spec.NannyPort),
		fmt.Sprintf("--dashboard-address=:%d", c.dc.Spec.DashboardPort),
		fmt.Sprintf("%s:%d", meta.InstanceName(c.dc, ComponentScheduler), c.dc.Spec.SchedulerPort),
	}
}

func (c *workerConfig) containerEnv() []corev1.EnvVar {
	return []corev1.EnvVar{
		{
			Name: "MY_POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
		//{
		//	Name: "MY_CPU_LIMIT",
		//	ValueFrom: &corev1.EnvVarSource{
		//		ResourceFieldRef: &corev1.ResourceFieldSelector{
		//			Resource: "limits.cpu",
		//		},
		//	},
		//},
		//{
		//	Name: "MY_MEM_LIMIT",
		//	ValueFrom: &corev1.EnvVarSource{
		//		ResourceFieldRef: &corev1.ResourceFieldSelector{
		//			Resource: "limits.memory",
		//		},
		//	},
		//},
	}
}

func (c *workerConfig) containerPorts() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{
			Name:          "worker",
			ContainerPort: c.dc.Spec.WorkerPort,
		},
		{
			Name:          "nanny",
			ContainerPort: c.dc.Spec.NannyPort,
		},
		{
			Name:          "dashboard",
			ContainerPort: c.dc.Spec.DashboardPort,
		},
	}
}
