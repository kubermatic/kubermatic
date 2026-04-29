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
	"k8s.io/utils/ptr"
)

func pprofPtr(s string) *string {
	return &s
}

func newTestConfig() *kubermaticv1.KubermaticConfiguration {
	return &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "example.com",
			},
			API: kubermaticv1.KubermaticAPIConfiguration{
				PProfEndpoint: pprofPtr(":6600"),
			},
			UI: kubermaticv1.KubermaticUIConfiguration{
				Replicas: ptr.To[int32](1),
			},
			MasterController: kubermaticv1.KubermaticMasterControllerConfiguration{
				PProfEndpoint: pprofPtr(":6600"),
			},
			Webhook: kubermaticv1.KubermaticWebhookConfiguration{
				PProfEndpoint: pprofPtr(":6600"),
			},
			SeedController: kubermaticv1.KubermaticSeedControllerConfiguration{
				PProfEndpoint: pprofPtr(":6600"),
			},
		},
	}
}

func testSchedulingFields() kubermaticv1.PodSchedulingConfigurations {
	return kubermaticv1.PodSchedulingConfigurations{
		NodeSelector: map[string]string{"role": "test"},
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "role",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"test"},
								},
							},
						},
					},
				},
			},
		},
		Tolerations: []corev1.Toleration{
			{
				Key:      "test",
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
					MatchLabels: map[string]string{"app": "test"},
				},
			},
		},
		PriorityClassName: "system-cluster-critical",
	}
}

func assertSchedulingFields(t *testing.T, spec corev1.PodSpec, expected kubermaticv1.PodSchedulingConfigurations) {
	t.Helper()

	if spec.PriorityClassName != expected.PriorityClassName {
		t.Errorf("PriorityClassName: expected %q, got %q", expected.PriorityClassName, spec.PriorityClassName)
	}

	if len(spec.NodeSelector) != len(expected.NodeSelector) {
		t.Errorf("NodeSelector: expected %v, got %v", expected.NodeSelector, spec.NodeSelector)
	} else {
		for k, v := range expected.NodeSelector {
			if spec.NodeSelector[k] != v {
				t.Errorf("NodeSelector[%q]: expected %q, got %q", k, v, spec.NodeSelector[k])
			}
		}
	}

	if spec.Affinity == nil && expected.Affinity != nil {
		t.Error("Affinity: expected non-nil, got nil")
	}

	if len(spec.Tolerations) != len(expected.Tolerations) {
		t.Errorf("Tolerations: expected %d items, got %d", len(expected.Tolerations), len(spec.Tolerations))
	}

	if len(spec.TopologySpreadConstraints) != len(expected.TopologySpreadConstraints) {
		t.Errorf("TopologySpreadConstraints: expected %d items, got %d", len(expected.TopologySpreadConstraints), len(spec.TopologySpreadConstraints))
	}
}

func assertNoSchedulingFields(t *testing.T, spec corev1.PodSpec) {
	t.Helper()

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
}

func TestAPIDeploymentScheduling(t *testing.T) {
	versions := kubermatic.GetFakeVersions()

	t.Run("with scheduling fields set", func(t *testing.T) {
		cfg := newTestConfig()
		scheduling := testSchedulingFields()
		cfg.Spec.API.PodSchedulingConfigurations = scheduling

		creatorGetter := APIDeploymentReconciler(cfg, "", versions)
		_, creator := creatorGetter()

		deploy := &appsv1.Deployment{}
		result, err := creator(deploy)
		if err != nil {
			t.Fatalf("reconciler failed: %v", err)
		}

		assertSchedulingFields(t, result.Spec.Template.Spec, scheduling)
	})

	t.Run("without scheduling fields", func(t *testing.T) {
		cfg := newTestConfig()

		creatorGetter := APIDeploymentReconciler(cfg, "", versions)
		_, creator := creatorGetter()

		deploy := &appsv1.Deployment{}
		result, err := creator(deploy)
		if err != nil {
			t.Fatalf("reconciler failed: %v", err)
		}

		assertNoSchedulingFields(t, result.Spec.Template.Spec)
	})
}

func TestUIDeploymentScheduling(t *testing.T) {
	versions := kubermatic.GetFakeVersions()

	t.Run("with scheduling fields set", func(t *testing.T) {
		cfg := newTestConfig()
		scheduling := testSchedulingFields()
		cfg.Spec.UI.PodSchedulingConfigurations = scheduling

		creatorGetter := UIDeploymentReconciler(cfg, versions)
		_, creator := creatorGetter()

		deploy := &appsv1.Deployment{}
		result, err := creator(deploy)
		if err != nil {
			t.Fatalf("reconciler failed: %v", err)
		}

		assertSchedulingFields(t, result.Spec.Template.Spec, scheduling)
	})

	t.Run("without scheduling fields", func(t *testing.T) {
		cfg := newTestConfig()

		creatorGetter := UIDeploymentReconciler(cfg, versions)
		_, creator := creatorGetter()

		deploy := &appsv1.Deployment{}
		result, err := creator(deploy)
		if err != nil {
			t.Fatalf("reconciler failed: %v", err)
		}

		assertNoSchedulingFields(t, result.Spec.Template.Spec)
	})
}

func TestMasterControllerManagerDeploymentScheduling(t *testing.T) {
	versions := kubermatic.GetFakeVersions()

	t.Run("with scheduling fields set", func(t *testing.T) {
		cfg := newTestConfig()
		scheduling := testSchedulingFields()
		cfg.Spec.MasterController.PodSchedulingConfigurations = scheduling

		creatorGetter := MasterControllerManagerDeploymentReconciler(cfg, "", versions, nil)
		_, creator := creatorGetter()

		deploy := &appsv1.Deployment{}
		result, err := creator(deploy)
		if err != nil {
			t.Fatalf("reconciler failed: %v", err)
		}

		assertSchedulingFields(t, result.Spec.Template.Spec, scheduling)
	})

	t.Run("without scheduling fields", func(t *testing.T) {
		cfg := newTestConfig()

		creatorGetter := MasterControllerManagerDeploymentReconciler(cfg, "", versions, nil)
		_, creator := creatorGetter()

		deploy := &appsv1.Deployment{}
		result, err := creator(deploy)
		if err != nil {
			t.Fatalf("reconciler failed: %v", err)
		}

		assertNoSchedulingFields(t, result.Spec.Template.Spec)
	})
}
