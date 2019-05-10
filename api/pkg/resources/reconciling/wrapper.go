package reconciling

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// OwnerRefWrapper is responsible for wrapping a ObjectCreator function, solely to set the OwnerReference to the cluster object
func OwnerRefWrapper(ref metav1.OwnerReference) ObjectModifier {
	return func(create ObjectCreator) ObjectCreator {
		return func(existing runtime.Object) (runtime.Object, error) {
			obj, err := create(existing)
			if err != nil {
				return obj, err
			}

			obj.(metav1.Object).SetOwnerReferences([]metav1.OwnerReference{ref})
			return obj, nil
		}
	}
}

// DefaultContainer defaults all Container attributes to the same values as they would get from the Kubernetes API
func DefaultContainer(c *corev1.Container, procMountType *corev1.ProcMountType) {
	if c.ImagePullPolicy == "" {
		c.ImagePullPolicy = corev1.PullIfNotPresent
	}
	if c.TerminationMessagePath == "" {
		c.TerminationMessagePath = corev1.TerminationMessagePathDefault
	}
	if c.TerminationMessagePolicy == "" {
		c.TerminationMessagePolicy = corev1.TerminationMessageReadFile
	}

	// This attribut was added in 1.12
	if c.SecurityContext != nil {
		c.SecurityContext.ProcMount = procMountType
	}
}

// DefaultDeployment defaults all Deployment attributes to the same values as they would et from the Kubernetes API
func DefaultDeployment(creator DeploymentCreator) DeploymentCreator {
	return func(d *appsv1.Deployment) (*appsv1.Deployment, error) {

		// Find out the procMountType before running the creator
		initContaineProcMountType := map[string]*corev1.ProcMountType{}
		containerProcMountType := map[string]*corev1.ProcMountType{}
		for _, container := range d.Spec.Template.Spec.InitContainers {
			if container.SecurityContext != nil {
				initContaineProcMountType[container.Name] = container.SecurityContext.ProcMount
			}
		}
		for _, container := range d.Spec.Template.Spec.Containers {
			if container.SecurityContext != nil {
				containerProcMountType[container.Name] = container.SecurityContext.ProcMount
			}
		}

		d, err := creator(d)
		if err != nil {
			return nil, err
		}
		for idx, container := range d.Spec.Template.Spec.InitContainers {
			DefaultContainer(&d.Spec.Template.Spec.InitContainers[idx], initContaineProcMountType[container.Name])
		}
		for idx, container := range d.Spec.Template.Spec.Containers {
			DefaultContainer(&d.Spec.Template.Spec.Containers[idx], containerProcMountType[container.Name])
		}
		return d, nil
	}
}
