/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package kubermatic

import (
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestSeedConfig() *kubermaticv1.KubermaticConfiguration {
	pprof := ":6600"
	return &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "example.com",
			},
			SeedController: kubermaticv1.KubermaticSeedControllerConfiguration{
				PProfEndpoint: &pprof,
			},
		},
	}
}

func newTestSeed() *kubermaticv1.Seed {
	return &kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "europe",
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.SeedSpec{
			Datacenters: map[string]kubermaticv1.Datacenter{
				"europe-west": {},
			},
		},
	}
}

func TestSeedControllerManagerDeploymentScheduling(t *testing.T) {
	versions := kubermatic.GetFakeVersions()

	scheduling := kubermaticv1.PodSchedulingConfigurations{
		NodeSelector: map[string]string{"role": "seed-controller"},
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "role",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"seed-controller"},
								},
							},
						},
					},
				},
			},
		},
		Tolerations: []corev1.Toleration{
			{
				Key:      "seed-controller",
				Operator: corev1.TolerationOpExists,
				Effect:   corev1.TaintEffectNoSchedule,
			},
		},
		TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
			{
				MaxSkew:           1,
				TopologyKey:       "topology.kubernetes.io/zone",
				WhenUnsatisfiable: corev1.DoNotSchedule,
				LabelSelector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"app": "seed-controller"},
				},
			},
		},
		PriorityClassName: "system-cluster-critical",
	}

	t.Run("with scheduling fields set", func(t *testing.T) {
		cfg := newTestSeedConfig()
		cfg.Spec.SeedController.PodSchedulingConfigurations = scheduling

		creatorGetter := SeedControllerManagerDeploymentReconciler("", versions, cfg, newTestSeed())
		_, creator := creatorGetter()

		deploy := &appsv1.Deployment{}
		result, err := creator(deploy)
		if err != nil {
			t.Fatalf("reconciler failed: %v", err)
		}

		spec := result.Spec.Template.Spec
		if spec.PriorityClassName != scheduling.PriorityClassName {
			t.Errorf("PriorityClassName: expected %q, got %q", scheduling.PriorityClassName, spec.PriorityClassName)
		}
		if len(spec.NodeSelector) != len(scheduling.NodeSelector) {
			t.Errorf("NodeSelector: expected %v, got %v", scheduling.NodeSelector, spec.NodeSelector)
		}
		if spec.Affinity == nil {
			t.Error("Affinity: expected non-nil")
		}
		if len(spec.Tolerations) != len(scheduling.Tolerations) {
			t.Errorf("Tolerations: expected %d, got %d", len(scheduling.Tolerations), len(spec.Tolerations))
		}
		if len(spec.TopologySpreadConstraints) != len(scheduling.TopologySpreadConstraints) {
			t.Errorf("TopologySpreadConstraints: expected %d, got %d", len(scheduling.TopologySpreadConstraints), len(spec.TopologySpreadConstraints))
		}
	})

	t.Run("without scheduling fields", func(t *testing.T) {
		cfg := newTestSeedConfig()

		creatorGetter := SeedControllerManagerDeploymentReconciler("", versions, cfg, newTestSeed())
		_, creator := creatorGetter()

		deploy := &appsv1.Deployment{}
		result, err := creator(deploy)
		if err != nil {
			t.Fatalf("reconciler failed: %v", err)
		}

		spec := result.Spec.Template.Spec
		if spec.PriorityClassName != "" {
			t.Errorf("PriorityClassName: expected empty, got %q", spec.PriorityClassName)
		}
		if len(spec.NodeSelector) != 0 {
			t.Errorf("NodeSelector: expected empty, got %v", spec.NodeSelector)
		}
		if spec.Affinity != nil {
			t.Error("Affinity: expected nil")
		}
		if len(spec.Tolerations) != 0 {
			t.Errorf("Tolerations: expected empty, got %v", spec.Tolerations)
		}
		if len(spec.TopologySpreadConstraints) != 0 {
			t.Errorf("TopologySpreadConstraints: expected empty, got %v", spec.TopologySpreadConstraints)
		}
	})
}
