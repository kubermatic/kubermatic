package vpnsidecar

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	dnatControllerResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("16Mi"),
			corev1.ResourceCPU:    resource.MustParse("5m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
	}
)

type dnatControllerData interface {
	ImageRegistry(string) string
	NodeAccessNetwork() string
	DNATControllerImage() string
}

// DnatControllerContainer returns a sidecar container for running the dnat controller.
func DnatControllerContainer(data dnatControllerData, name, apiserverAddress string) (*corev1.Container, error) {
	procMountType := corev1.DefaultProcMount
	args := []string{
		"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
		"-node-access-network", data.NodeAccessNetwork(),
	}
	if apiserverAddress != "" {
		args = append(args, "-master", apiserverAddress)
	}

	return &corev1.Container{
		Name:    name,
		Image:   data.DNATControllerImage() + ":" + resources.KUBERMATICCOMMIT,
		Command: []string{"/usr/local/bin/kubeletdnat-controller"},
		Args:    args,
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add: []corev1.Capability{"NET_ADMIN"},
			},
			RunAsUser: resources.Int64(0),
			ProcMount: &procMountType,
		},
		Resources: dnatControllerResourceRequirements,
		VolumeMounts: []corev1.VolumeMount{
			{
				MountPath: "/etc/kubernetes/kubeconfig",
				Name:      resources.KubeletDnatControllerKubeconfigSecretName,
				ReadOnly:  true,
			},
		},
	}, nil
}
