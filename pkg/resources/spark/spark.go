package spark

import (
	"fmt"

	dcv1alpha1 "github.com/dominodatalab/distributed-compute-operator/api/v1alpha1"
	"github.com/dominodatalab/distributed-compute-operator/pkg/resources"
)

// Component is used to drive Kubernetes object generation for different spark types.
type Component string

const (
	// ComponentNone indicates a generic spark resource.
	ComponentNone Component = "none"
	// ComponentMaster indicates a spark master resource.
	ComponentMaster Component = "master"
	// ComponentWorker indicates a spark worker resource.
	ComponentWorker Component = "worker"
	// ApplicationName defines the static name used to generate spark object metadata.
	ApplicationName = "spark"
)

// InstanceObjectName returns the name that will be used to create most owned cluster resources.
func InstanceObjectName(instance string, comp Component) string {
	if comp == ComponentNone {
		return fmt.Sprintf("%s-%s", instance, ApplicationName)
	}

	return fmt.Sprintf("%s-%s-%s", instance, ApplicationName, comp)
}

// HeadServiceName returns the name of the service that points to the spark head pod.
func HeadServiceName(name string) string {
	return InstanceObjectName(name, ComponentMaster)
}

func HeadlessServiceName(name string) string {
	return InstanceObjectName(name, ComponentWorker)
}

// MetadataLabels returns standard metadata for spark resources.
func MetadataLabels(sc *dcv1alpha1.SparkCluster) map[string]string {
	return resources.MetadataLabels(ApplicationName, sc.Name, sc.Spec.Image.Tag)
}

// MetadataLabelsWithComponent returns standard component metadata for spark resources.
func MetadataLabelsWithComponent(sc *dcv1alpha1.SparkCluster, comp Component) map[string]string {
	return resources.MetadataLabelsWithComponent(ApplicationName, sc.Name, sc.Spec.Image.Tag, string(comp))
}

// SelectorLabels returns a resource selector clause for spark resources.
func SelectorLabels(sc *dcv1alpha1.SparkCluster) map[string]string {
	return resources.SelectorLabels(ApplicationName, sc.Name)
}

// SelectorLabelsWithComponent returns a resource component selector clause for spark resources.
func SelectorLabelsWithComponent(sc *dcv1alpha1.SparkCluster, comp Component) map[string]string {
	return resources.SelectorLabelsWithComponent(ApplicationName, sc.Name, string(comp))
}
