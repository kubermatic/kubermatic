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
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
)

func TestMerge(t *testing.T) {
	a := &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution:  podAffinityTerm(AppLabelKey, "app", "zone"),
			PreferredDuringSchedulingIgnoredDuringExecution: weightedPodAffinityTerm(AppLabelKey, "app", "zone"),
		},
	}
	b := &corev1.Affinity{
		PodAntiAffinity: &corev1.PodAntiAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution:  podAffinityTerm(AppLabelKey, "app", "zone"),
			PreferredDuringSchedulingIgnoredDuringExecution: weightedPodAffinityTerm(AppLabelKey, "app", "zone"),
		},
	}
	c := MergeAffinities(a, b)

	if len(c.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution) != 2 {
		t.Errorf("Merge failed, expected length of PreferredDuringSchedulingIgnoredDuringExecution to be 2, got %d", len(a.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution))
	}
	if len(c.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution) != 2 {
		t.Errorf("Merge failed, expected length of RequiredDuringSchedulingIgnoredDuringExecution to be 2, got %d", len(a.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution))
	}
}

func TestHostnameAntiAffinityForLabel(t *testing.T) {
	t.Parallel()

	affinity := HostnameAntiAffinityForLabel("app.kubernetes.io/name", "nodeport-proxy-envoy", kubermaticv1.AntiAffinityTypePreferred)
	terms := affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution
	if len(terms) != 1 {
		t.Fatalf("expected 1 preferred anti-affinity term, got %d", len(terms))
	}

	got := terms[0].PodAffinityTerm.LabelSelector.MatchLabels
	if got["app.kubernetes.io/name"] != "nodeport-proxy-envoy" {
		t.Fatalf("unexpected label selector: got %#v", got)
	}
	if _, exists := got[AppLabelKey]; exists {
		t.Fatalf("expected custom label key to be used, got %#v", got)
	}
}
