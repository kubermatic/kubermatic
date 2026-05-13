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
	"github.com/stretchr/testify/require"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

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

// schedulingFieldSetter configures scheduling fields on the given config for a specific component.
type schedulingFieldSetter func(cfg *kubermaticv1.KubermaticConfiguration, s kubermaticv1.PodSchedulingConfigurations)

type reconcilerResult struct {
	name    string
	creator func(d *appsv1.Deployment) (*appsv1.Deployment, error)
}

func TestMasterComponentScheduling(t *testing.T) {
	versions := kubermatic.GetFakeVersions()

	tests := []struct {
		name        string
		setSched    schedulingFieldSetter
		newReconc   func(cfg *kubermaticv1.KubermaticConfiguration) reconcilerResult
		preExisting *kubermaticv1.PodSchedulingConfigurations
		input       *kubermaticv1.PodSchedulingConfigurations
		want        *kubermaticv1.PodSchedulingConfigurations
	}{
		{
			name: "api with all scheduling fields set",
			setSched: func(cfg *kubermaticv1.KubermaticConfiguration, s kubermaticv1.PodSchedulingConfigurations) {
				cfg.Spec.API.PodSchedulingConfigurations = s
			},
			newReconc: func(cfg *kubermaticv1.KubermaticConfiguration) reconcilerResult {
				n, c := APIDeploymentReconciler(cfg, "", versions)()
				return reconcilerResult{n, c}
			},
			input: ptrScheduling(fullScheduling()),
			want:  ptrScheduling(fullScheduling()),
		},
		{
			name: "api with no scheduling fields",
			setSched: func(cfg *kubermaticv1.KubermaticConfiguration, s kubermaticv1.PodSchedulingConfigurations) {
				cfg.Spec.API.PodSchedulingConfigurations = s
			},
			newReconc: func(cfg *kubermaticv1.KubermaticConfiguration) reconcilerResult {
				n, c := APIDeploymentReconciler(cfg, "", versions)()
				return reconcilerResult{n, c}
			},
			want: nil,
		},
		{
			name: "api with only nodeSelector",
			setSched: func(cfg *kubermaticv1.KubermaticConfiguration, s kubermaticv1.PodSchedulingConfigurations) {
				cfg.Spec.API.PodSchedulingConfigurations = s
			},
			newReconc: func(cfg *kubermaticv1.KubermaticConfiguration) reconcilerResult {
				n, c := APIDeploymentReconciler(cfg, "", versions)()
				return reconcilerResult{n, c}
			},
			input: &kubermaticv1.PodSchedulingConfigurations{
				NodeSelector: map[string]string{"role": "test"},
			},
			want: &kubermaticv1.PodSchedulingConfigurations{
				NodeSelector: map[string]string{"role": "test"},
			},
		},
		{
			name: "api overwrites pre-existing scheduling fields",
			setSched: func(cfg *kubermaticv1.KubermaticConfiguration, s kubermaticv1.PodSchedulingConfigurations) {
				cfg.Spec.API.PodSchedulingConfigurations = s
			},
			newReconc: func(cfg *kubermaticv1.KubermaticConfiguration) reconcilerResult {
				n, c := APIDeploymentReconciler(cfg, "", versions)()
				return reconcilerResult{n, c}
			},
			preExisting: &kubermaticv1.PodSchedulingConfigurations{
				NodeSelector:      map[string]string{"old": "value"},
				PriorityClassName: "old-priority",
			},
			input: ptrScheduling(fullScheduling()),
			want:  ptrScheduling(fullScheduling()),
		},
		{
			name: "ui with all scheduling fields set",
			setSched: func(cfg *kubermaticv1.KubermaticConfiguration, s kubermaticv1.PodSchedulingConfigurations) {
				cfg.Spec.UI.PodSchedulingConfigurations = s
			},
			newReconc: func(cfg *kubermaticv1.KubermaticConfiguration) reconcilerResult {
				n, c := UIDeploymentReconciler(cfg, versions)()
				return reconcilerResult{n, c}
			},
			input: ptrScheduling(fullScheduling()),
			want:  ptrScheduling(fullScheduling()),
		},
		{
			name: "ui with no scheduling fields",
			setSched: func(cfg *kubermaticv1.KubermaticConfiguration, s kubermaticv1.PodSchedulingConfigurations) {
				cfg.Spec.UI.PodSchedulingConfigurations = s
			},
			newReconc: func(cfg *kubermaticv1.KubermaticConfiguration) reconcilerResult {
				n, c := UIDeploymentReconciler(cfg, versions)()
				return reconcilerResult{n, c}
			},
			want: nil,
		},
		{
			name: "ui with only tolerations",
			setSched: func(cfg *kubermaticv1.KubermaticConfiguration, s kubermaticv1.PodSchedulingConfigurations) {
				cfg.Spec.UI.PodSchedulingConfigurations = s
			},
			newReconc: func(cfg *kubermaticv1.KubermaticConfiguration) reconcilerResult {
				n, c := UIDeploymentReconciler(cfg, versions)()
				return reconcilerResult{n, c}
			},
			input: &kubermaticv1.PodSchedulingConfigurations{
				Tolerations: []corev1.Toleration{
					{Key: "test", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
				},
			},
			want: &kubermaticv1.PodSchedulingConfigurations{
				Tolerations: []corev1.Toleration{
					{Key: "test", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule},
				},
			},
		},
		{
			name: "master-controller-manager with all scheduling fields set",
			setSched: func(cfg *kubermaticv1.KubermaticConfiguration, s kubermaticv1.PodSchedulingConfigurations) {
				cfg.Spec.MasterController.PodSchedulingConfigurations = s
			},
			newReconc: func(cfg *kubermaticv1.KubermaticConfiguration) reconcilerResult {
				n, c := MasterControllerManagerDeploymentReconciler(cfg, "", versions, nil)()
				return reconcilerResult{n, c}
			},
			input: ptrScheduling(fullScheduling()),
			want:  ptrScheduling(fullScheduling()),
		},
		{
			name: "master-controller-manager with no scheduling fields",
			setSched: func(cfg *kubermaticv1.KubermaticConfiguration, s kubermaticv1.PodSchedulingConfigurations) {
				cfg.Spec.MasterController.PodSchedulingConfigurations = s
			},
			newReconc: func(cfg *kubermaticv1.KubermaticConfiguration) reconcilerResult {
				n, c := MasterControllerManagerDeploymentReconciler(cfg, "", versions, nil)()
				return reconcilerResult{n, c}
			},
			want: nil,
		},
		{
			name: "master-controller-manager with only priorityClassName",
			setSched: func(cfg *kubermaticv1.KubermaticConfiguration, s kubermaticv1.PodSchedulingConfigurations) {
				cfg.Spec.MasterController.PodSchedulingConfigurations = s
			},
			newReconc: func(cfg *kubermaticv1.KubermaticConfiguration) reconcilerResult {
				n, c := MasterControllerManagerDeploymentReconciler(cfg, "", versions, nil)()
				return reconcilerResult{n, c}
			},
			input: &kubermaticv1.PodSchedulingConfigurations{
				PriorityClassName: "system-cluster-critical",
			},
			want: &kubermaticv1.PodSchedulingConfigurations{
				PriorityClassName: "system-cluster-critical",
			},
		},
		{
			name: "master-controller-manager clears pre-existing fields when config is empty",
			setSched: func(cfg *kubermaticv1.KubermaticConfiguration, s kubermaticv1.PodSchedulingConfigurations) {
				cfg.Spec.MasterController.PodSchedulingConfigurations = s
			},
			newReconc: func(cfg *kubermaticv1.KubermaticConfiguration) reconcilerResult {
				n, c := MasterControllerManagerDeploymentReconciler(cfg, "", versions, nil)()
				return reconcilerResult{n, c}
			},
			preExisting: ptrScheduling(fullScheduling()),
			want:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := newTestConfig()
			if tt.input != nil {
				tt.setSched(cfg, *tt.input)
			}

			res := tt.newReconc(cfg)
			deploy := &appsv1.Deployment{}
			if tt.preExisting != nil {
				applySchedulingToDeploy(deploy, *tt.preExisting)
			}

			result, err := res.creator(deploy)
			require.NoError(t, err)

			assertPodScheduling(t, result.Spec.Template.Spec, tt.want)
		})
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
