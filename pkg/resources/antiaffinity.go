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
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// HostnameAntiAffinity returns a simple Affinity rule to prevent* scheduling of same kind pods on the same node.
// *if scheduling is not possible with this rule, it will be ignored.
func HostnameAntiAffinity(app string, antiAffinityType kubermaticv1.AntiAffinityType) *corev1.Affinity {
	return antiAffinity(AppLabelKey, app, antiAffinityType, TopologyKeyHostname)
}

// HostnameAntiAffinityForLabel returns a simple Affinity rule to prevent* scheduling of same kind pods on the same node.
// *if scheduling is not possible with this rule, it will be ignored.
func HostnameAntiAffinityForLabel(labelKey, labelValue string, antiAffinityType kubermaticv1.AntiAffinityType) *corev1.Affinity {
	return antiAffinity(labelKey, labelValue, antiAffinityType, TopologyKeyHostname)
}

// FailureDomainZoneAntiAffinity ensures that same-kind pods are spread across different availability zones.
func FailureDomainZoneAntiAffinity(app string, antiAffinityType kubermaticv1.AntiAffinityType) *corev1.Affinity {
	return antiAffinity(AppLabelKey, app, antiAffinityType, TopologyKeyZone)
}

// FailureDomainZoneAntiAffinityForLabel ensures that same-kind pods are spread across different availability zones.
func FailureDomainZoneAntiAffinityForLabel(labelKey, labelValue string, antiAffinityType kubermaticv1.AntiAffinityType) *corev1.Affinity {
	return antiAffinity(labelKey, labelValue, antiAffinityType, TopologyKeyZone)
}

func MergeAffinities(a *corev1.Affinity, b *corev1.Affinity) *corev1.Affinity {
	if a == nil && b == nil {
		return a
	}
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	a.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(
		a.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
		b.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution...,
	)
	a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(
		a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
		b.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution...,
	)
	return a
}

func antiAffinity(labelKey, labelValue string, antiAffinity kubermaticv1.AntiAffinityType, topologyKey string) *corev1.Affinity {
	if antiAffinity == kubermaticv1.AntiAffinityTypeRequired {
		return &corev1.Affinity{
			PodAntiAffinity: &corev1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: podAffinityTerm(labelKey, labelValue, topologyKey),
			},
		}
	}

	return &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: weightedPodAffinityTerm(labelKey, labelValue, topologyKey),
		},
	}
}

func weightedPodAffinityTerm(labelKey, labelValue string, topologyKey string) []corev1.WeightedPodAffinityTerm {
	return []corev1.WeightedPodAffinityTerm{
		// Avoid that we schedule multiple same-kind pods within the same namespace on a single node.
		{
			Weight: 100,
			PodAffinityTerm: corev1.PodAffinityTerm{
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						labelKey: labelValue,
					},
				},
				TopologyKey: topologyKey,
			},
		},
	}
}

func podAffinityTerm(labelKey, labelValue string, topologyKey string) []corev1.PodAffinityTerm {
	return []corev1.PodAffinityTerm{
		// Avoid that we schedule multiple same-kind pods within the same namespace on a single node.
		{
			LabelSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					labelKey: labelValue,
				},
			},
			TopologyKey: topologyKey,
		},
	}
}
