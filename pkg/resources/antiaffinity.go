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

package resources

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HostnameAntiAffinity returns a simple Affinity rule to prevent* scheduling of same kind pods on the same node.
// It contains 2 AntiAffinity terms:
// High priority: We don't schedule multiple pods of this app & cluster on a single node
// Low priority: We don't schedule multiple pods of this app on a single node - regardless of the cluster.
// This prevents that we schedule all API server pods on a single node
// *if scheduling is not possible with this rule, it will be ignored.
func HostnameAntiAffinity(app, clusterName string) *corev1.Affinity {
	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: hostnameAntiAffinity(app, clusterName),
		},
	}
}

func hostnameAntiAffinity(app, clusterName string) []corev1.WeightedPodAffinityTerm {
	return []corev1.WeightedPodAffinityTerm{
		// Avoid that we schedule multiple same-kind pods of a cluster on a single node
		{
			Weight: 100,
			PodAffinityTerm: corev1.PodAffinityTerm{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						AppLabelKey:     app,
						ClusterLabelKey: clusterName,
					},
				},
				TopologyKey: TopologyKeyHostname,
			},
		},
		// Avoid that we schedule multiple same-kind pods on a single node
		{
			Weight: 10,
			PodAffinityTerm: corev1.PodAffinityTerm{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						AppLabelKey: app,
					},
				},
				TopologyKey: TopologyKeyHostname,
			},
		},
	}
}

// FailureDomainZoneAntiAffinity ensures that same-kind pods are spread across different availability zones.
func FailureDomainZoneAntiAffinity(app, clusterName string) corev1.WeightedPodAffinityTerm {
	return corev1.WeightedPodAffinityTerm{
		Weight: 100,
		PodAffinityTerm: corev1.PodAffinityTerm{
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					AppLabelKey: app,
				},
			},
			TopologyKey: TopologyKeyFailureDomainZone,
		},
	}
}
