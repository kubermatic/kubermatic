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
	"strconv"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

var (
	defaultResourceRequirements = map[string]*corev1.ResourceRequirements{
		resources.EnvoyAgentDaemonSetName: {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("64Mi"),
				corev1.ResourceCPU:    resource.MustParse("100m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("512Mi"),
				corev1.ResourceCPU:    resource.MustParse("1"),
			},
		},
	}
)

const (
	envoyImageName = "docker.io/envoyproxy/envoy"
)

// DaemonSetReconciler returns the function to create and update the Envoy DaemonSet.
func DaemonSetReconciler(cluster *kubermaticv1.Cluster, agentIP net.IP, versions kubermatic.Versions, configHash string, imageRewriter registry.ImageRewriter) reconciling.NamedDaemonSetReconcilerFactory {
	return func() (string, reconciling.DaemonSetReconciler) {
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

				// Used to force the restart of the envoy-agent to re-read its configuration
				// from the configMap when it changes. Necessary to support switching to/from Konnectivity.
				Annotations: map[string]string{
					"checksum/config":      configHash,
					"prometheus.io/scrape": "true",
					"prometheus.io/port":   strconv.Itoa(int(StatsPort)),
					"prometheus.io/path":   "/stats/prometheus",
				},
			}

			initContainers, err := getInitContainers(agentIP, versions, imageRewriter)
			if err != nil {
				return nil, err
			}

			containers, err := getContainers(versions, imageRewriter, agentIP)
			if err != nil {
				return nil, err
			}

			ds.Spec.Template.Spec = corev1.PodSpec{
				InitContainers: initContainers,
				Containers:     containers,
				// TODO(youssefazrak) needed?
				PriorityClassName:             "system-cluster-critical",
				DNSPolicy:                     corev1.DNSClusterFirst,
				HostNetwork:                   true,
				Volumes:                       getVolumes(),
				RestartPolicy:                 corev1.RestartPolicyAlways,
				TerminationGracePeriodSeconds: ptr.To[int64](30),
				SecurityContext: &corev1.PodSecurityContext{
					SeccompProfile: &corev1.SeccompProfile{
						Type: corev1.SeccompProfileTypeRuntimeDefault,
					},
				},
				SchedulerName: corev1.DefaultSchedulerName,
			}
			ds.Spec.Template.Spec.Tolerations = []corev1.Toleration{
				{
					Effect:   corev1.TaintEffectNoSchedule,
					Operator: corev1.TolerationOpExists,
				},
				{
					Effect:   corev1.TaintEffectNoExecute,
					Operator: corev1.TolerationOpExists,
				},
			}

			var overrides map[string]*corev1.ResourceRequirements
			if cluster != nil {
				overrides = resources.GetOverrides(cluster.Spec.ComponentsOverride)
			}

			if err := resources.SetResourceRequirements(ds.Spec.Template.Spec.Containers, defaultResourceRequirements, overrides, ds.Annotations); err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			return ds, nil
		}
	}
}

func getInitContainers(ip net.IP, versions kubermatic.Versions, imageRewriter registry.ImageRewriter) ([]corev1.Container, error) {
	image := registry.Must(imageRewriter(fmt.Sprintf("%s/%s:%s", resources.RegistryQuay, resources.EnvoyAgentDeviceSetupImage, versions.KubermaticContainerTag)))
	// TODO: we are creating and configuring the a dummy interface
	// using init containers. This approach is good enough for the tech preview
	// but it is definitely not production ready. This should be replaced with
	// a binary that properly handles error conditions and reconciles the
	// interface in a loop.
	return []corev1.Container{
		{
			Name:  resources.EnvoyAgentCreateInterfaceInitContainerName,
			Image: image,
			Args: []string{
				"-mode",
				"init",
				"-if",
				"envoyagent",
				"-addr",
				ip.String(),
			},
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
	}, nil
}

func getContainers(versions kubermatic.Versions, imageRewriter registry.ImageRewriter, ip net.IP) ([]corev1.Container, error) {
	image := registry.Must(imageRewriter(fmt.Sprintf("%s/%s:%s", resources.RegistryQuay, resources.EnvoyAgentDeviceSetupImage, versions.KubermaticContainerTag)))
	return []corev1.Container{
		{
			Name:            resources.EnvoyAgentDaemonSetName,
			Image:           registry.Must(imageRewriter(fmt.Sprintf("%s:%s", envoyImageName, nodeportproxy.EnvoyVersion))),
			ImagePullPolicy: corev1.PullIfNotPresent,

			// This amount of logs will be kept for the Tech Preview of
			// the new expose strategy
			Args: []string{"--config-path", "etc/envoy/envoy.yaml", "--use-dynamic-base-id"},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "config-volume",
					MountPath: "/etc/envoy/envoy.yaml",
					SubPath:   resources.EnvoyAgentConfigFileName,
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
		{
			Name:            resources.EnvoyAgentAssignAddressContainerName,
			Image:           image,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Args: []string{
				"-mode", "probe",
				"-if", "envoyagent",
				"-addr", ip.String(),
			},
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
	}, nil
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "config-volume",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					DefaultMode: ptr.To[int32](corev1.ConfigMapVolumeSourceDefaultMode),
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.EnvoyAgentConfigMapName,
					},
				},
			},
		},
	}
}
