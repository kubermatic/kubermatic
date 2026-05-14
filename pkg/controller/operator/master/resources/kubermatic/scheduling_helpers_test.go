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

	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func pprofPtr(s string) *string { return &s }

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

func fullScheduling() kubermaticv1.PodSchedulingConfigurations {
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

func ptrScheduling(s kubermaticv1.PodSchedulingConfigurations) *kubermaticv1.PodSchedulingConfigurations {
	return &s
}

func applySchedulingToDeploy(d *appsv1.Deployment, s kubermaticv1.PodSchedulingConfigurations) {
	common.ApplyPodScheduling(&d.Spec.Template.Spec, s)
}

func assertPodScheduling(t *testing.T, spec corev1.PodSpec, want *kubermaticv1.PodSchedulingConfigurations) {
	t.Helper()

	if want == nil {
		want = &kubermaticv1.PodSchedulingConfigurations{}
	}

	assert.Equal(t, want.PriorityClassName, spec.PriorityClassName, "PriorityClassName")
	assert.Equal(t, want.NodeSelector, spec.NodeSelector, "NodeSelector")
	assert.Equal(t, want.Affinity, spec.Affinity, "Affinity")
	assert.Equal(t, want.Tolerations, spec.Tolerations, "Tolerations")
	assert.Equal(t, want.TopologySpreadConstraints, spec.TopologySpreadConstraints, "TopologySpreadConstraints")
}

type schedulingTestCase struct {
	name        string
	preExisting *kubermaticv1.PodSchedulingConfigurations
	input       *kubermaticv1.PodSchedulingConfigurations
	want        *kubermaticv1.PodSchedulingConfigurations
}

// runSchedulingTest applies tt.input to cfg via setSched, runs the reconciler returned by newCreator,
// and asserts the resulting PodSpec matches tt.want.
func runSchedulingTest(
	t *testing.T,
	tt schedulingTestCase,
	setSched func(cfg *kubermaticv1.KubermaticConfiguration, s kubermaticv1.PodSchedulingConfigurations),
	newCreator func(cfg *kubermaticv1.KubermaticConfiguration) func(*appsv1.Deployment) (*appsv1.Deployment, error),
) {
	t.Helper()

	cfg := newTestConfig()
	if tt.input != nil {
		setSched(cfg, *tt.input)
	}

	deploy := &appsv1.Deployment{}
	if tt.preExisting != nil {
		applySchedulingToDeploy(deploy, *tt.preExisting)
	}

	result, err := newCreator(cfg)(deploy)
	if err != nil {
		t.Fatalf("reconciler failed: %v", err)
	}

	assertPodScheduling(t, result.Spec.Template.Spec, tt.want)
}
