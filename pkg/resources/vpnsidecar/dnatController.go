/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vpnsidecar

import (
	"k8c.io/kubermatic/v2/pkg/resources"

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
	DNATControllerTag() string
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
		Image:   data.DNATControllerImage() + ":" + data.DNATControllerTag(),
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
