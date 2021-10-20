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

package envoyagent

import (
	"fmt"
	"net"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilpointer "k8s.io/utils/pointer"
)

var (
	defaultResourceRequirements = map[string]*corev1.ResourceRequirements{
		resources.EnvoyAgentDaemonSetName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("32Mi"),
				corev1.ResourceCPU:    resource.MustParse("50m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("64Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
		},
	}
)

const (
	envoyImageName = "envoyproxy/envoy"
)

// DaemonSetCreator returns the function to create and update the Envoy DaemonSet
func DaemonSetCreator(agentIP net.IP, versions kubermatic.Versions, registryWithOverwrite registry.WithOverwriteFunc) reconciling.NamedDaemonSetCreatorGetter {
	return func() (string, reconciling.DaemonSetCreator) {
		return resources.EnvoyAgentDaemonSetName, func(ds *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
			ds.Name = resources.EnvoyAgentDaemonSetName
			ds.Namespace = metav1.NamespaceSystem
			ds.Labels = resources.BaseAppLabels(resources.EnvoyAgentDaemonSetName, nil)

			ds.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(resources.EnvoyAgentDaemonSetName,
					map[string]string{"app.kubernetes.io/name": "envoy-agent"}),
			}

			// has to be the same as the selector
			ds.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: resources.BaseAppLabels(resources.EnvoyAgentDaemonSetName,
					map[string]string{"app.kubernetes.io/name": "envoy-agent"}),
			}

			ds.Spec.Template.Spec = corev1.PodSpec{
				InitContainers: getInitContainers(agentIP, versions, registryWithOverwrite(resources.RegistryQuay)),
				Containers:     getContainers(versions, registryWithOverwrite(resources.RegistryDocker)),
				// TODO(youssefazrak) needed?
				PriorityClassName:             "system-cluster-critical",
				DNSPolicy:                     corev1.DNSClusterFirst,
				HostNetwork:                   true,
				Volumes:                       getVolumes(),
				RestartPolicy:                 corev1.RestartPolicyAlways,
				TerminationGracePeriodSeconds: utilpointer.Int64Ptr(30),
				SecurityContext:               &corev1.PodSecurityContext{},
				SchedulerName:                 corev1.DefaultSchedulerName,
			}
			if err := resources.SetResourceRequirements(ds.Spec.Template.Spec.Containers, defaultResourceRequirements, nil, ds.Annotations); err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %v", err)
			}

			return ds, nil
		}
	}
}

func getInitContainers(ip net.IP, versions kubermatic.Versions, registry string) []corev1.Container {
	// TODO: we are creating and configuring the a dummy interface
	// using init containers. This approach is good enough for the tech preview
	// but it is definitely not production ready. This should be replaced with
	// a binary that properly handles error conditions and reconciles the
	// interface in a loop.
	return []corev1.Container{
		{
			Name:    resources.EnvoyAgentCreateInterfaceInitContainerName,
			Image:   fmt.Sprintf("%s/%s:%s", registry, resources.EnvoyAgentDeviceSetupImage, versions.Kubermatic),
			Command: []string{"sh", "-c", "ip link add envoyagent type dummy || true"},
			SecurityContext: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{
						"NET_ADMIN",
					},
					Drop: []corev1.Capability{
						"all",
					},
				},
			},
		},
		{
			Name:    resources.EnvoyAgentAssignAddressInitContainerName,
			Image:   fmt.Sprintf("%s/%s:%s", registry, resources.EnvoyAgentDeviceSetupImage, versions.Kubermatic),
			Command: []string{"sh", "-c", fmt.Sprintf("ip addr add %s/32 dev envoyagent scope host || true", ip.String())},
			SecurityContext: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{
						"NET_ADMIN",
					},
					Drop: []corev1.Capability{
						"all",
					},
				},
			},
		},
	}
}

func getContainers(versions kubermatic.Versions, registry string) []corev1.Container {
	return []corev1.Container{
		{
			Name:            resources.EnvoyAgentDaemonSetName,
			Image:           fmt.Sprintf("%s/%s:%s", registry, envoyImageName, versions.Envoy),
			ImagePullPolicy: corev1.PullIfNotPresent,

			// This amount of logs will be kept for the Tech Preview of
			// the new expose strategy
			Args: []string{"--config-path", "etc/envoy/envoy.yaml", "--component-log-level", "upstream:trace,connection:trace,http:trace,router:trace,filter:trace"},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "config-volume",
					MountPath: "/etc/envoy/envoy.yaml",
					SubPath:   "envoy.yaml",
				},
			},
			SecurityContext: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{
						"CHOWN",
						"SETGID",
						"SETUID",
						"NET_BIND_SERVICE",
					},
					Drop: []corev1.Capability{
						"all",
					},
				},
			},
		},
	}
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "config-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: utilpointer.Int32Ptr(corev1.ConfigMapVolumeSourceDefaultMode),
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.EnvoyAgentConfigMapName,
					},
				},
			},
		},
	}
}
