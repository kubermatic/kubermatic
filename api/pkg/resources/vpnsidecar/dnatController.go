package vpnsidecar

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("32Mi"),
			corev1.ResourceCPU:    resource.MustParse("5m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("64Mi"),
			corev1.ResourceCPU:    resource.MustParse("5m"),
		},
	}
)

// DnatControllerContainer returns a sidecar container for running the dnat controller.
func DnatControllerContainer(data *resources.TemplateData, name string) (*corev1.Container, error) {
	kcDir := "/etc/kubernetes/dnat-controller-kubeconfig"
	return &corev1.Container{
		Name:            name,
		Image:           data.ImageRegistry(resources.RegistryQuay) + "/kubermatic/vpnsidecar-dnat-controller:v0.2.0-rc4",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/usr/local/bin/kubeletdnat-controller"},
		Args: []string{
			"--kubeconfig", fmt.Sprintf("%s/%s", kcDir, resources.KubeletDnatControllerKubeconfigSecretName),
			"--node-access-network", data.NodeAccessNetwork,
			"-v", "6",
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
				MountPath: kcDir,
				Name:      resources.KubeletDnatControllerKubeconfigSecretName,
				ReadOnly:  true,
			},
		},
	}, nil
}
