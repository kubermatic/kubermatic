package vpnsidecar

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("10Mi"),
			corev1.ResourceCPU:    resource.MustParse("5m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("20Mi"),
			corev1.ResourceCPU:    resource.MustParse("20m"),
		},
	}
)

// DnatControllerContainer returns a sidecar container for running the dnat controller.
func DnatControllerContainer(data *resources.TemplateData, name string) (*corev1.Container, error) {
	return &corev1.Container{
		Name:            name,
		Image:           data.ImageRegistry(resources.RegistryQuay) + "/kubermatic/vpnsidecar-dnat-controller:v0.2.0",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/usr/local/bin/kubeletdnat-controller"},
		Args: []string{
			"--kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
			"--node-access-network", data.NodeAccessNetwork,
			"-v", "4",
			"-logtostderr",
		},
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"NET_ADMIN"},
			},
		},
		TerminationMessagePath:   corev1.TerminationMessagePathDefault,
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		Resources:                defaultResourceRequirements,
		VolumeMounts: []corev1.VolumeMount{
			{
				MountPath: "/etc/kubernetes/kubeconfig",
				Name:      resources.KubeletDnatControllerKubeconfigSecretName,
				ReadOnly:  true,
			},
		},
	}, nil
}
