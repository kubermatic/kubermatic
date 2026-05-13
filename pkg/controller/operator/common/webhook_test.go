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

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func newTestWebhookConfig() *kubermaticv1.KubermaticConfiguration {
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
			Webhook: kubermaticv1.KubermaticWebhookConfiguration{
				PProfEndpoint: &pprof,
			},
		},
	}
}

func webhookFullScheduling() kubermaticv1.PodSchedulingConfigurations {
	return kubermaticv1.PodSchedulingConfigurations{
		NodeSelector: map[string]string{"role": "webhook"},
		Affinity: &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{
						{
							MatchExpressions: []corev1.NodeSelectorRequirement{
								{
									Key:      "role",
									Operator: corev1.NodeSelectorOpIn,
									Values:   []string{"webhook"},
								},
							},
						},
					},
				},
			},
		},
		Tolerations: []corev1.Toleration{
			{
				Key:      "webhook",
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
					MatchLabels: map[string]string{"app": "webhook"},
				},
			},
		},
		PriorityClassName: "system-cluster-critical",
	}
}

func TestWebhookDeploymentScheduling(t *testing.T) {
	versions := kubermatic.GetFakeVersions()

	tests := []struct {
		name        string
		preExisting *kubermaticv1.PodSchedulingConfigurations
		input       *kubermaticv1.PodSchedulingConfigurations
		want        *kubermaticv1.PodSchedulingConfigurations
	}{
		{
			name:  "all scheduling fields set",
			input: webhookSchedulingPtr(webhookFullScheduling()),
			want:  webhookSchedulingPtr(webhookFullScheduling()),
		},
		{
			name: "no scheduling fields",
			want: nil,
		},
		{
			name: "only affinity set",
			input: &kubermaticv1.PodSchedulingConfigurations{
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{Key: "role", Operator: corev1.NodeSelectorOpIn, Values: []string{"webhook"}},
									},
								},
							},
						},
					},
				},
			},
			want: &kubermaticv1.PodSchedulingConfigurations{
				Affinity: &corev1.Affinity{
					NodeAffinity: &corev1.NodeAffinity{
						RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
							NodeSelectorTerms: []corev1.NodeSelectorTerm{
								{
									MatchExpressions: []corev1.NodeSelectorRequirement{
										{Key: "role", Operator: corev1.NodeSelectorOpIn, Values: []string{"webhook"}},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "overwrites pre-existing scheduling fields",
			preExisting: &kubermaticv1.PodSchedulingConfigurations{
				NodeSelector:      map[string]string{"old": "value"},
				PriorityClassName: "old-priority",
			},
			input: webhookSchedulingPtr(webhookFullScheduling()),
			want:  webhookSchedulingPtr(webhookFullScheduling()),
		},
		{
			name:        "clears pre-existing fields when config is empty",
			preExisting: webhookSchedulingPtr(webhookFullScheduling()),
			want:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newTestWebhookConfig()
			if tt.input != nil {
				cfg.Spec.Webhook.PodSchedulingConfigurations = *tt.input
			}

			_, creator := WebhookDeploymentReconciler(cfg, versions, nil, false)()

			deploy := &appsv1.Deployment{}
			if tt.preExisting != nil {
				webhookApplyScheduling(deploy, *tt.preExisting)
			}

			result, err := creator(deploy)
			require.NoError(t, err)

			webhookAssertScheduling(t, result.Spec.Template.Spec, tt.want)
		})
	}
}

func webhookSchedulingPtr(s kubermaticv1.PodSchedulingConfigurations) *kubermaticv1.PodSchedulingConfigurations {
	return &s
}

func webhookApplyScheduling(d *appsv1.Deployment, s kubermaticv1.PodSchedulingConfigurations) {
	ApplyPodScheduling(&d.Spec.Template.Spec, s)
}

func webhookAssertScheduling(t *testing.T, spec corev1.PodSpec, want *kubermaticv1.PodSchedulingConfigurations) {
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
