//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2025 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package defaultpolicycontroller

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	kubermatictest "k8c.io/kubermatic/v2/pkg/test"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	kubernetesVersion = defaulting.DefaultKubernetesVersioning.Default
	testScheme        = fake.NewScheme()
)

const (
	clusterName       = "test-cluster"
	clusterNamespace  = "cluster-test-cluster"
	defaultDatacenter = "global"
	projectID         = "testproject"
	policyName        = "test-policy"
)

//nolint:gocyclo
func TestReconcile(t *testing.T) {
	log := zap.NewNop().Sugar()

	testCases := []struct {
		name             string
		cluster          *kubermaticv1.Cluster
		policyTemplates  []kubermaticv1.PolicyTemplate
		existingBindings []kubermaticv1.PolicyBinding
		validate         func(cluster *kubermaticv1.Cluster, policyTemplates []kubermaticv1.PolicyTemplate, client ctrlruntimeclient.Client, reconcileErr error) error
	}{
		{
			name:    "scenario 1: no default or enforced policies, no policy bindings created",
			cluster: genCluster(clusterName, defaultDatacenter, true),
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplate(policyName, false, false, kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, policyTemplates []kubermaticv1.PolicyTemplate, client ctrlruntimeclient.Client, reconcileErr error) error {
				conditionType := kubermaticv1.ClusterConditionDefaultPolicyControllerReconcilingSuccess
				if cond := cluster.Status.Conditions[conditionType]; cond.Status != corev1.ConditionTrue {
					return fmt.Errorf("cluster should have %v=%s condition, but does not", conditionType, corev1.ConditionTrue)
				}

				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have produced an error, but returned: %w", reconcileErr)
				}

				bindings := &kubermaticv1.PolicyBindingList{}
				if err := client.List(context.Background(), bindings, ctrlruntimeclient.InNamespace(clusterNamespace)); err != nil {
					return fmt.Errorf("failed to list PolicyBindings: %w", err)
				}

				if len(bindings.Items) != 0 {
					return fmt.Errorf("expected 0 policy bindings, but got %d", len(bindings.Items))
				}

				return nil
			},
		},
		{
			name:    "scenario 2: default policies are installed",
			cluster: genCluster(clusterName, defaultDatacenter, true),
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplate(policyName, true, false, kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, policyTemplates []kubermaticv1.PolicyTemplate, client ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				bindings := &kubermaticv1.PolicyBindingList{}
				if err := client.List(context.Background(), bindings, ctrlruntimeclient.InNamespace(clusterNamespace)); err != nil {
					return fmt.Errorf("failed to list PolicyBindings: %w", err)
				}

				if len(bindings.Items) != len(policyTemplates) {
					return fmt.Errorf("expected %d policy bindings, but got %d", len(policyTemplates), len(bindings.Items))
				}

				for _, template := range policyTemplates {
					found := false
					for _, binding := range bindings.Items {
						if binding.Name == template.Name {
							found = true
							if binding.Spec.PolicyTemplateRef.Name != template.Name {
								return fmt.Errorf("binding %s has incorrect PolicyTemplateRef.Name: expected %s, got %s",
									binding.Name, template.Name, binding.Spec.PolicyTemplateRef.Name)
							}

							if template.Spec.Default && binding.Annotations[kubermaticv1.AnnotationPolicyDefault] != "true" {
								return fmt.Errorf("binding %s missing default annotation", binding.Name)
							}
							if template.Spec.Enforced && binding.Annotations[kubermaticv1.AnnotationPolicyEnforced] != "true" {
								return fmt.Errorf("binding %s missing enforced annotation", binding.Name)
							}
							break
						}
					}
					if !found {
						return fmt.Errorf("policy binding for template %s not found", template.Name)
					}
				}

				return nil
			},
		},
		{
			name:    "scenario 3: enforced policies are installed",
			cluster: genCluster(clusterName, defaultDatacenter, true),
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplate(policyName, false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, policyTemplates []kubermaticv1.PolicyTemplate, client ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				bindings := &kubermaticv1.PolicyBindingList{}
				if err := client.List(context.Background(), bindings, ctrlruntimeclient.InNamespace(clusterNamespace)); err != nil {
					return fmt.Errorf("failed to list PolicyBindings: %w", err)
				}

				if len(bindings.Items) != len(policyTemplates) {
					return fmt.Errorf("expected %d policy bindings, but got %d", len(policyTemplates), len(bindings.Items))
				}

				for _, template := range policyTemplates {
					found := false
					for _, binding := range bindings.Items {
						if binding.Name == template.Name {
							found = true
							// Check annotation
							if binding.Annotations[kubermaticv1.AnnotationPolicyEnforced] != "true" {
								return fmt.Errorf("binding %s missing enforced annotation", binding.Name)
							}
							break
						}
					}
					if !found {
						return fmt.Errorf("policy binding for template %s not found", template.Name)
					}
				}

				return nil
			},
		},
		{
			name:    "scenario 4: both default and enforced policies are installed",
			cluster: genCluster(clusterName, defaultDatacenter, true),
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplate("policy1", true, false, kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil, nil),
				*genPolicyTemplate("policy2", false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil, nil),
				*genPolicyTemplate("policy3", true, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, policyTemplates []kubermaticv1.PolicyTemplate, client ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				bindings := &kubermaticv1.PolicyBindingList{}
				if err := client.List(context.Background(), bindings, ctrlruntimeclient.InNamespace(clusterNamespace)); err != nil {
					return fmt.Errorf("failed to list PolicyBindings: %w", err)
				}

				if len(bindings.Items) != len(policyTemplates) {
					return fmt.Errorf("expected %d policy bindings, but got %d", len(policyTemplates), len(bindings.Items))
				}

				return nil
			},
		},
		{
			name:    "scenario 5: policies with project targeting are correctly filtered",
			cluster: genCluster(clusterName, defaultDatacenter, true),
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplate("policy1", false, true, kubermaticv1.PolicyTemplateVisibilityProject, projectID, nil, nil),
				*genPolicyTemplate("policy2", false, true, kubermaticv1.PolicyTemplateVisibilityProject, "different-project", nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, policyTemplates []kubermaticv1.PolicyTemplate, client ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				// Check that only the matching policy binding was created
				bindings := &kubermaticv1.PolicyBindingList{}
				if err := client.List(context.Background(), bindings, ctrlruntimeclient.InNamespace(clusterNamespace)); err != nil {
					return fmt.Errorf("failed to list PolicyBindings: %w", err)
				}

				if len(bindings.Items) != 1 {
					return fmt.Errorf("expected 1 policy binding, but got %d", len(bindings.Items))
				}

				if bindings.Items[0].Name != "policy1" {
					return fmt.Errorf("expected policy binding for template policy1, but got %s", bindings.Items[0].Name)
				}

				return nil
			},
		},
		{
			name:    "scenario 6: policies with label selector targeting are correctly filtered",
			cluster: genCluster(clusterName, defaultDatacenter, true),
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplate("policy1", false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "",
					&metav1.LabelSelector{
						MatchLabels: map[string]string{
							"environment": "test",
						},
					}, nil),
				*genPolicyTemplate("policy2", false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "",
					&metav1.LabelSelector{
						MatchLabels: map[string]string{
							"environment": "prod",
						},
					}, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, policyTemplates []kubermaticv1.PolicyTemplate, client ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				bindings := &kubermaticv1.PolicyBindingList{}
				if err := client.List(context.Background(), bindings, ctrlruntimeclient.InNamespace(clusterNamespace)); err != nil {
					return fmt.Errorf("failed to list PolicyBindings: %w", err)
				}

				if len(bindings.Items) != 1 {
					return fmt.Errorf("expected 1 policy binding, but got %d", len(bindings.Items))
				}

				if bindings.Items[0].Name != "policy1" {
					return fmt.Errorf("expected policy binding for template policy1, but got %s", bindings.Items[0].Name)
				}

				return nil
			},
		},
		{
			name:    "scenario 7: unhealthy cluster is skipped",
			cluster: genUnhealthyCluster(clusterName, defaultDatacenter, true),
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplate(policyName, false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, policyTemplates []kubermaticv1.PolicyTemplate, client ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("expected no error for unhealthy cluster, but got: %w", reconcileErr)
				}

				bindings := &kubermaticv1.PolicyBindingList{}
				if err := client.List(context.Background(), bindings, ctrlruntimeclient.InNamespace(clusterNamespace)); err != nil {
					return fmt.Errorf("failed to list PolicyBindings: %w", err)
				}

				if len(bindings.Items) != 0 {
					return fmt.Errorf("expected 0 policy bindings for unhealthy cluster, but got %d", len(bindings.Items))
				}

				return nil
			},
		},
		{
			name:    "scenario 8: default policies are ignored if default policy bindings already created",
			cluster: genClusterWithDefaultPolicyBindingsCreated(clusterName, defaultDatacenter, true),
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplate("policy1", true, false, kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil, nil),
				*genPolicyTemplate("policy2", false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, policyTemplates []kubermaticv1.PolicyTemplate, client ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				bindings := &kubermaticv1.PolicyBindingList{}
				if err := client.List(context.Background(), bindings, ctrlruntimeclient.InNamespace(clusterNamespace)); err != nil {
					return fmt.Errorf("failed to list PolicyBindings: %w", err)
				}

				if len(bindings.Items) != 1 {
					return fmt.Errorf("expected 1 policy binding (only enforced), but got %d", len(bindings.Items))
				}

				if bindings.Items[0].Name != "policy2" {
					return fmt.Errorf("expected policy binding for template policy2, but got %s", bindings.Items[0].Name)
				}

				return nil
			},
		},
		{
			name:    "scenario 9: existing policy bindings are updated",
			cluster: genCluster(clusterName, defaultDatacenter, true),
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplate(policyName, true, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil, nil),
			},
			existingBindings: []kubermaticv1.PolicyBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      policyName,
						Namespace: clusterNamespace,
						Annotations: map[string]string{
							kubermaticv1.AnnotationPolicyDefault: "true",
						},
					},
					Spec: kubermaticv1.PolicyBindingSpec{
						PolicyTemplateRef: corev1.ObjectReference{
							Name: policyName,
						},
					},
				},
			},
			validate: func(cluster *kubermaticv1.Cluster, policyTemplates []kubermaticv1.PolicyTemplate, client ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				// Check that binding was updated with enforced annotation
				binding := &kubermaticv1.PolicyBinding{}
				if err := client.Get(context.Background(), types.NamespacedName{Name: policyName, Namespace: clusterNamespace}, binding); err != nil {
					return fmt.Errorf("failed to get PolicyBinding: %w", err)
				}

				if binding.Annotations[kubermaticv1.AnnotationPolicyEnforced] != "true" {
					return fmt.Errorf("binding %s should have enforced annotation, but doesn't", binding.Name)
				}

				return nil
			},
		},
		{
			name:    "scenario 10: enforced policy binding is immediately recreated when deleted",
			cluster: genCluster(clusterName, defaultDatacenter, true),
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplate(policyName, false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil, nil),
			},
			validate: func(cluster *kubermaticv1.Cluster, policyTemplates []kubermaticv1.PolicyTemplate, client ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				bindings := &kubermaticv1.PolicyBindingList{}
				if err := client.List(context.Background(), bindings, ctrlruntimeclient.InNamespace(clusterNamespace)); err != nil {
					return fmt.Errorf("failed to list PolicyBindings: %w", err)
				}

				if len(bindings.Items) != 1 {
					return fmt.Errorf("expected 1 policy binding, but got %d", len(bindings.Items))
				}

				originalBinding := bindings.Items[0]
				if originalBinding.Name != policyName {
					return fmt.Errorf("expected policy binding named %s, but got %s", policyName, originalBinding.Name)
				}

				if originalBinding.Annotations[kubermaticv1.AnnotationPolicyEnforced] != "true" {
					return fmt.Errorf("binding %s should have enforced annotation", originalBinding.Name)
				}

				return nil
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			objects := getSeedObjects(test.cluster, test.policyTemplates, test.existingBindings)

			// Create a seed client with our test objects
			seedClient := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(objects...).
				Build()

			ctx := context.Background()

			r := &Reconciler{
				Client:       seedClient,
				recorder:     &events.FakeRecorder{},
				log:          log,
				versions:     kubermatic.GetFakeVersions(),
				configGetter: kubermatictest.NewConfigGetter(nil),
				seedGetter: func() (*kubermaticv1.Seed, error) {
					return &kubermaticv1.Seed{
						Spec: kubermaticv1.SeedSpec{
							Datacenters: map[string]kubermaticv1.Datacenter{
								defaultDatacenter: {
									Spec: kubermaticv1.DatacenterSpec{
										Hetzner: &kubermaticv1.DatacenterSpecHetzner{
											Datacenter: "hel1",
											Network:    "default",
										},
									},
								},
							},
						},
					}, nil
				},
			}

			nName := types.NamespacedName{Name: test.cluster.Name}

			_, reconcileErr := r.Reconcile(ctx, reconcile.Request{NamespacedName: nName})

			newCluster := &kubermaticv1.Cluster{}
			if err := r.Get(ctx, nName, newCluster); err != nil {
				if validateErr := test.validate(test.cluster, test.policyTemplates, seedClient, err); validateErr != nil {
					t.Fatalf("Test failed: cluster could not be found and validation failed: %v", validateErr)
				}
				return
			}

			if err := test.validate(newCluster, test.policyTemplates, seedClient, reconcileErr); err != nil {
				t.Fatalf("Test failed: %v", err)
			}
		})
	}
}

// TestSelectorTargeting tests selector scenarios.
func TestSelectorTargeting(t *testing.T) {
	log := zap.NewNop().Sugar()

	testCases := []struct {
		name             string
		clusters         []*kubermaticv1.Cluster
		projects         []*kubermaticv1.Project
		policyTemplates  []kubermaticv1.PolicyTemplate
		expectedBindings map[string][]string
	}{
		{
			name: "global visibility with project selector only",
			clusters: []*kubermaticv1.Cluster{
				genClusterWithProject("cluster1", "project1", map[string]string{"env": "test"}),
				genClusterWithProject("cluster2", "project2", map[string]string{"env": "prod"}),
			},
			projects: []*kubermaticv1.Project{
				genProject("project1", map[string]string{"team": "backend"}),
				genProject("project2", map[string]string{"team": "frontend"}),
			},
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplate("policy-backend", false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil,
					&metav1.LabelSelector{MatchLabels: map[string]string{"team": "backend"}}),
				*genPolicyTemplate("policy-frontend", false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil,
					&metav1.LabelSelector{MatchLabels: map[string]string{"team": "frontend"}}),
			},
			expectedBindings: map[string][]string{
				"cluster1": {"policy-backend"},
				"cluster2": {"policy-frontend"},
			},
		},
		{
			name: "global visibility with cluster selector only",
			clusters: []*kubermaticv1.Cluster{
				genClusterWithProject("cluster1", "project1", map[string]string{"env": "test", "tier": "dev"}),
				genClusterWithProject("cluster2", "project1", map[string]string{"env": "prod", "tier": "production"}),
				genClusterWithProject("cluster3", "project1", map[string]string{"env": "staging", "tier": "dev"}),
			},
			projects: []*kubermaticv1.Project{
				genProject("project1", map[string]string{"team": "backend"}),
			},
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplate("policy-dev", false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "",
					&metav1.LabelSelector{MatchLabels: map[string]string{"tier": "dev"}}, nil),
				*genPolicyTemplate("policy-prod", false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "",
					&metav1.LabelSelector{MatchLabels: map[string]string{"env": "prod"}}, nil),
			},
			expectedBindings: map[string][]string{
				"cluster1": {"policy-dev"},
				"cluster2": {"policy-prod"},
				"cluster3": {"policy-dev"},
			},
		},
		{
			name: "global visibility with both project and cluster selectors (AND logic)",
			clusters: []*kubermaticv1.Cluster{
				genClusterWithProject("cluster1", "project1", map[string]string{"env": "test", "critical": "true"}),
				genClusterWithProject("cluster2", "project1", map[string]string{"env": "prod", "critical": "false"}),
				genClusterWithProject("cluster3", "project2", map[string]string{"env": "test", "critical": "true"}),
			},
			projects: []*kubermaticv1.Project{
				genProject("project1", map[string]string{"team": "backend", "priority": "high"}),
				genProject("project2", map[string]string{"team": "frontend", "priority": "low"}),
			},
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplate("policy-critical-backend", false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "",
					&metav1.LabelSelector{MatchLabels: map[string]string{"critical": "true"}},
					&metav1.LabelSelector{MatchLabels: map[string]string{"team": "backend"}}),
			},
			expectedBindings: map[string][]string{
				"cluster1": {"policy-critical-backend"},
				"cluster2": {},
				"cluster3": {},
			},
		},
		{
			name: "project visibility with cluster selector",
			clusters: []*kubermaticv1.Cluster{
				genClusterWithProject("cluster1", "project1", map[string]string{"env": "test"}),
				genClusterWithProject("cluster2", "project1", map[string]string{"env": "prod"}),
				genClusterWithProject("cluster3", "project2", map[string]string{"env": "test"}),
			},
			projects: []*kubermaticv1.Project{
				genProject("project1", map[string]string{"team": "backend"}),
				genProject("project2", map[string]string{"team": "backend"}),
			},
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplate("policy-test", false, true, kubermaticv1.PolicyTemplateVisibilityProject, "project1",
					&metav1.LabelSelector{MatchLabels: map[string]string{"env": "test"}}, nil),
			},
			expectedBindings: map[string][]string{
				"cluster1": {"policy-test"},
				"cluster2": {},
				"cluster3": {},
			},
		},
		{
			name: "empty selectors should match all",
			clusters: []*kubermaticv1.Cluster{
				genClusterWithProject("cluster1", "project1", map[string]string{"env": "test"}),
				genClusterWithProject("cluster2", "project2", map[string]string{"env": "prod"}),
			},
			projects: []*kubermaticv1.Project{
				genProject("project1", map[string]string{"team": "backend"}),
				genProject("project2", map[string]string{"team": "frontend"}),
			},
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplate("policy-all", false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "",
					&metav1.LabelSelector{}, &metav1.LabelSelector{}),
			},
			expectedBindings: map[string][]string{
				"cluster1": {"policy-all"},
				"cluster2": {"policy-all"},
			},
		},
		{
			name: "match expressions in selectors",
			clusters: []*kubermaticv1.Cluster{
				genClusterWithProject("cluster1", "project1", map[string]string{"env": "test", "version": "1.25"}),
				genClusterWithProject("cluster2", "project1", map[string]string{"env": "prod", "version": "1.24"}),
				genClusterWithProject("cluster3", "project1", map[string]string{"env": "dev", "version": "1.26"}),
			},
			projects: []*kubermaticv1.Project{
				genProject("project1", map[string]string{"team": "backend"}),
			},
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplateWithExpressions("policy-not-prod", false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "",
					&metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: "env", Operator: metav1.LabelSelectorOpNotIn, Values: []string{"prod"}},
						},
					}, nil),
				*genPolicyTemplateWithExpressions("policy-newer-versions", false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "",
					&metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: "version", Operator: metav1.LabelSelectorOpIn, Values: []string{"1.25", "1.26"}},
						},
					}, nil),
			},
			expectedBindings: map[string][]string{
				"cluster1": {"policy-not-prod", "policy-newer-versions"},
				"cluster2": {},
				"cluster3": {"policy-not-prod", "policy-newer-versions"},
			},
		},
		{
			name: "project selector with match expressions",
			clusters: []*kubermaticv1.Cluster{
				genClusterWithProject("cluster1", "project1", map[string]string{"env": "test"}),
				genClusterWithProject("cluster2", "project2", map[string]string{"env": "test"}),
				genClusterWithProject("cluster3", "project3", map[string]string{"env": "test"}),
			},
			projects: []*kubermaticv1.Project{
				genProject("project1", map[string]string{"tier": "premium", "status": "active"}),
				genProject("project2", map[string]string{"tier": "basic", "status": "active"}),
				genProject("project3", map[string]string{"tier": "premium", "status": "inactive"}),
			},
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplateWithExpressions("policy-active-projects", false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "",
					nil,
					&metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: "status", Operator: metav1.LabelSelectorOpIn, Values: []string{"active"}},
						},
					}),
			},
			expectedBindings: map[string][]string{
				"cluster1": {"policy-active-projects"},
				"cluster2": {"policy-active-projects"},
				"cluster3": {},
			},
		},
		{
			name: "no matching selectors",
			clusters: []*kubermaticv1.Cluster{
				genClusterWithProject("cluster1", "project1", map[string]string{"env": "test"}),
				genClusterWithProject("cluster2", "project1", map[string]string{"env": "prod"}),
			},
			projects: []*kubermaticv1.Project{
				genProject("project1", map[string]string{"team": "backend"}),
			},
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplate("policy-staging", false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "",
					&metav1.LabelSelector{MatchLabels: map[string]string{"env": "staging"}}, nil),
				*genPolicyTemplate("policy-frontend", false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil,
					&metav1.LabelSelector{MatchLabels: map[string]string{"team": "frontend"}}),
			},
			expectedBindings: map[string][]string{
				"cluster1": {},
				"cluster2": {},
			},
		},
		{
			name: "complex combined selectors with match labels and expressions",
			clusters: []*kubermaticv1.Cluster{
				genClusterWithProject("cluster1", "project1", map[string]string{"env": "test", "region": "us-east", "critical": "true"}),
				genClusterWithProject("cluster2", "project1", map[string]string{"env": "prod", "region": "us-west", "critical": "true"}),
				genClusterWithProject("cluster3", "project2", map[string]string{"env": "prod", "region": "us-east", "critical": "false"}),
			},
			projects: []*kubermaticv1.Project{
				genProject("project1", map[string]string{"team": "backend", "budget": "unlimited"}),
				genProject("project2", map[string]string{"team": "frontend", "budget": "limited"}),
			},
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplateWithExpressions("policy-complex", false, true, kubermaticv1.PolicyTemplateVisibilityGlobal, "",
					&metav1.LabelSelector{
						MatchLabels: map[string]string{"critical": "true"},
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: "region", Operator: metav1.LabelSelectorOpIn, Values: []string{"us-east", "us-west"}},
						},
					},
					&metav1.LabelSelector{
						MatchLabels: map[string]string{"team": "backend"},
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: "budget", Operator: metav1.LabelSelectorOpNotIn, Values: []string{"limited"}},
						},
					}),
			},
			expectedBindings: map[string][]string{
				"cluster1": {"policy-complex"},
				"cluster2": {"policy-complex"},
				"cluster3": {},
			},
		},
		{
			name: "project visibility with project ID and empty cluster selector",
			clusters: []*kubermaticv1.Cluster{
				genClusterWithProject("cluster1", "project1", map[string]string{"env": "test"}),
				genClusterWithProject("cluster2", "project1", map[string]string{"env": "prod"}),
				genClusterWithProject("cluster3", "project2", map[string]string{"env": "test"}),
			},
			projects: []*kubermaticv1.Project{
				genProject("project1", map[string]string{"team": "backend"}),
				genProject("project2", map[string]string{"team": "backend"}),
			},
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplate("policy-project1", false, true, kubermaticv1.PolicyTemplateVisibilityProject, "project1",
					&metav1.LabelSelector{}, nil),
			},
			expectedBindings: map[string][]string{
				"cluster1": {"policy-project1"},
				"cluster2": {"policy-project1"},
				"cluster3": {},
			},
		},
		{
			name: "project visibility with target but nil cluster selector",
			clusters: []*kubermaticv1.Cluster{
				genClusterWithProject("cluster1", "project1", map[string]string{"env": "test"}),
				genClusterWithProject("cluster2", "project1", map[string]string{"env": "prod"}),
				genClusterWithProject("cluster3", "project2", map[string]string{"env": "test"}),
			},
			projects: []*kubermaticv1.Project{
				genProject("project1", map[string]string{"team": "backend"}),
				genProject("project2", map[string]string{"team": "backend"}),
			},
			policyTemplates: []kubermaticv1.PolicyTemplate{
				genPolicyTemplateWithNilSelectors("policy-project1-nil-selector", false, true,
					kubermaticv1.PolicyTemplateVisibilityProject, "project1", true, false, false),
			},
			expectedBindings: map[string][]string{
				"cluster1": {"policy-project1-nil-selector"},
				"cluster2": {"policy-project1-nil-selector"},
				"cluster3": {},
			},
		},
		{
			name: "global visibility with target but nil selectors",
			clusters: []*kubermaticv1.Cluster{
				genClusterWithProject("cluster1", "project1", map[string]string{"env": "test"}),
				genClusterWithProject("cluster2", "project2", map[string]string{"env": "prod"}),
				genClusterWithProject("cluster3", "project1", map[string]string{"env": "staging"}),
			},
			projects: []*kubermaticv1.Project{
				genProject("project1", map[string]string{"team": "backend"}),
				genProject("project2", map[string]string{"team": "frontend"}),
			},
			policyTemplates: []kubermaticv1.PolicyTemplate{
				genPolicyTemplateWithNilSelectors("policy-global-nil-selectors", false, true,
					kubermaticv1.PolicyTemplateVisibilityGlobal, "", true, false, false),
			},
			expectedBindings: map[string][]string{
				"cluster1": {"policy-global-nil-selectors"},
				"cluster2": {"policy-global-nil-selectors"},
				"cluster3": {"policy-global-nil-selectors"},
			},
		},
		{
			name: "invalid visibility should not match any clusters",
			clusters: []*kubermaticv1.Cluster{
				genClusterWithProject("cluster1", "project1", map[string]string{"env": "test"}),
				genClusterWithProject("cluster2", "project2", map[string]string{"env": "prod"}),
			},
			projects: []*kubermaticv1.Project{
				genProject("project1", map[string]string{"team": "backend"}),
				genProject("project2", map[string]string{"team": "frontend"}),
			},
			policyTemplates: []kubermaticv1.PolicyTemplate{
				*genPolicyTemplate("policy-invalid", false, true, "InvalidVisibility", "",
					&metav1.LabelSelector{MatchLabels: map[string]string{"env": "test"}}, nil),
			},
			expectedBindings: map[string][]string{
				"cluster1": {},
				"cluster2": {},
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			var objects []ctrlruntimeclient.Object

			// Add projects
			for _, project := range test.projects {
				objects = append(objects, project)
			}

			// Add clusters and their namespaces
			for _, cluster := range test.clusters {
				objects = append(objects, cluster)
				objects = append(objects, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: cluster.Status.NamespaceName,
					},
				})
			}

			// Add policy templates
			for _, template := range test.policyTemplates {
				templateCopy := template
				objects = append(objects, &templateCopy)
			}

			// Create a seed client with our test objects
			seedClient := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(objects...).
				Build()

			ctx := context.Background()

			r := &Reconciler{
				Client:       seedClient,
				recorder:     &events.FakeRecorder{},
				log:          log,
				versions:     kubermatic.GetFakeVersions(),
				configGetter: kubermatictest.NewConfigGetter(nil),
				seedGetter: func() (*kubermaticv1.Seed, error) {
					return &kubermaticv1.Seed{
						Spec: kubermaticv1.SeedSpec{
							Datacenters: map[string]kubermaticv1.Datacenter{
								defaultDatacenter: {
									Spec: kubermaticv1.DatacenterSpec{
										Hetzner: &kubermaticv1.DatacenterSpecHetzner{
											Datacenter: "hel1",
											Network:    "default",
										},
									},
								},
							},
						},
					}, nil
				},
			}

			// Reconcile each cluster
			for _, cluster := range test.clusters {
				nName := types.NamespacedName{Name: cluster.Name}
				_, reconcileErr := r.Reconcile(ctx, reconcile.Request{NamespacedName: nName})
				if reconcileErr != nil {
					t.Fatalf("reconciling cluster %s should not have caused an error, but did: %v", cluster.Name, reconcileErr)
				}
			}

			// Validate expected bindings for each cluster
			for _, cluster := range test.clusters {
				expectedPolicies := test.expectedBindings[cluster.Name]

				bindings := &kubermaticv1.PolicyBindingList{}
				if err := seedClient.List(ctx, bindings, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName)); err != nil {
					t.Fatalf("failed to list PolicyBindings for cluster %s: %v", cluster.Name, err)
				}

				if len(bindings.Items) != len(expectedPolicies) {
					t.Errorf("cluster %s: expected %d policy bindings, but got %d", cluster.Name, len(expectedPolicies), len(bindings.Items))
					continue
				}

				// Check that all expected policies are present
				actualPolicies := make(map[string]bool)
				for _, binding := range bindings.Items {
					actualPolicies[binding.Name] = true
				}

				for _, expectedPolicy := range expectedPolicies {
					if !actualPolicies[expectedPolicy] {
						t.Errorf("cluster %s: expected policy binding %s not found", cluster.Name, expectedPolicy)
					}
				}

				// Check that no unexpected policies are present
				for actualPolicy := range actualPolicies {
					found := false
					for _, expectedPolicy := range expectedPolicies {
						if actualPolicy == expectedPolicy {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("cluster %s: unexpected policy binding %s found", cluster.Name, actualPolicy)
					}
				}
			}
		})
	}
}

func TestPolicyBindingDeletionMapping(t *testing.T) {
	log := zap.NewNop().Sugar()

	testCases := []struct {
		name                 string
		policyBinding        *kubermaticv1.PolicyBinding
		cluster              *kubermaticv1.Cluster
		policyTemplate       *kubermaticv1.PolicyTemplate
		expectReconciliation bool
		description          string
	}{
		{
			name: "enforced policy binding deletion should trigger reconciliation",
			policyBinding: &kubermaticv1.PolicyBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-enforced-policy",
					Namespace: "cluster-test-cluster",
				},
				Spec: kubermaticv1.PolicyBindingSpec{
					PolicyTemplateRef: corev1.ObjectReference{
						Name: "test-enforced-policy",
					},
				},
			},
			cluster: genCluster("test-cluster", defaultDatacenter, true),
			policyTemplate: genPolicyTemplate("test-enforced-policy", false, true,
				kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil, nil),
			expectReconciliation: true,
			description:          "Should trigger reconciliation for enforced policy",
		},
		{
			name: "non-enforced policy binding deletion should not trigger reconciliation",
			policyBinding: &kubermaticv1.PolicyBinding{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-default-policy",
					Namespace: "cluster-test-cluster",
				},
				Spec: kubermaticv1.PolicyBindingSpec{
					PolicyTemplateRef: corev1.ObjectReference{
						Name: "test-default-policy",
					},
				},
			},
			cluster: genCluster("test-cluster", defaultDatacenter, true),
			policyTemplate: genPolicyTemplate("test-default-policy", true, false,
				kubermaticv1.PolicyTemplateVisibilityGlobal, "", nil, nil),
			expectReconciliation: false,
			description:          "Should not trigger reconciliation for non-enforced policy",
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			objects := []ctrlruntimeclient.Object{
				test.cluster,
				test.policyTemplate,
			}

			// Create a client with our test objects
			client := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(objects...).
				Build()

			ctx := context.Background()

			r := &Reconciler{
				Client:   client,
				recorder: &events.FakeRecorder{},
				log:      log,
				versions: kubermatic.GetFakeVersions(),
			}

			var requests []reconcile.Request
			clusterNamespace := test.policyBinding.Namespace
			if strings.HasPrefix(clusterNamespace, "cluster-") {
				clusterName := strings.TrimPrefix(clusterNamespace, "cluster-")

				// Verify the cluster exists
				cluster := &kubermaticv1.Cluster{}
				if err := r.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err == nil {
					// Check if the referenced PolicyTemplate is enforced
					if test.policyBinding.Spec.PolicyTemplateRef.Name != "" {
						policyTemplate := &kubermaticv1.PolicyTemplate{}
						if err := r.Get(ctx, types.NamespacedName{Name: test.policyBinding.Spec.PolicyTemplateRef.Name}, policyTemplate); err == nil {
							if policyTemplate.Spec.Enforced && r.isClusterTargeted(ctx, cluster, policyTemplate) {
								requests = append(requests, reconcile.Request{
									NamespacedName: types.NamespacedName{
										Name: clusterName,
									},
								})
							}
						}
					}
				}
			}

			if test.expectReconciliation {
				if len(requests) != 1 {
					t.Errorf("%s: expected 1 reconcile request, got %d", test.description, len(requests))
					return
				}
				expectedClusterName := strings.TrimPrefix(test.policyBinding.Namespace, "cluster-")
				if test.policyBinding.Namespace == "cluster-non-existent" {
					expectedClusterName = "non-existent"
				}
				if requests[0].Name != expectedClusterName {
					t.Errorf("%s: expected cluster name %s, got %s", test.description, expectedClusterName, requests[0].Name)
				}
			} else if len(requests) > 0 {
				t.Errorf("%s: expected 0 reconcile requests, got %d", test.description, len(requests))
			}
		})
	}
}

func getSeedObjects(cluster *kubermaticv1.Cluster, policyTemplates []kubermaticv1.PolicyTemplate, existingBindings []kubermaticv1.PolicyBinding) []ctrlruntimeclient.Object {
	objects := []ctrlruntimeclient.Object{cluster}

	for _, template := range policyTemplates {
		templateCopy := template
		objects = append(objects, &templateCopy)
	}

	for _, binding := range existingBindings {
		bindingCopy := binding
		objects = append(objects, &bindingCopy)
	}

	// Add cluster namespace
	objects = append(objects, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterNamespace,
		},
	})

	return objects
}

func healthy() kubermaticv1.ExtendedClusterHealth {
	return kubermaticv1.ExtendedClusterHealth{
		Apiserver:                    kubermaticv1.HealthStatusUp,
		Scheduler:                    kubermaticv1.HealthStatusUp,
		Controller:                   kubermaticv1.HealthStatusUp,
		MachineController:            kubermaticv1.HealthStatusUp,
		Etcd:                         kubermaticv1.HealthStatusUp,
		CloudProviderInfrastructure:  kubermaticv1.HealthStatusUp,
		UserClusterControllerManager: kubermaticv1.HealthStatusUp,
	}
}

func unhealthy() kubermaticv1.ExtendedClusterHealth {
	health := healthy()
	health.Apiserver = kubermaticv1.HealthStatusDown
	return health
}

func genCluster(name, datacenter string, withNamespace bool) *kubermaticv1.Cluster {
	namespace := ""
	if withNamespace {
		namespace = "cluster-" + name
	}

	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				kubermaticv1.ProjectIDLabelKey: projectID,
				"environment":                  "test",
			},
		},
		Spec: kubermaticv1.ClusterSpec{
			Version: *kubernetesVersion,
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: datacenter,
			},
		},
		Status: kubermaticv1.ClusterStatus{
			ExtendedHealth: healthy(),
			NamespaceName:  namespace,
		},
	}
}

func genUnhealthyCluster(name, datacenter string, withNamespace bool) *kubermaticv1.Cluster {
	cluster := genCluster(name, datacenter, withNamespace)
	cluster.Status.ExtendedHealth = unhealthy()
	return cluster
}

func genClusterWithDefaultPolicyBindingsCreated(name, datacenter string, withNamespace bool) *kubermaticv1.Cluster {
	cluster := genCluster(name, datacenter, withNamespace)
	cluster.Status.Conditions = map[kubermaticv1.ClusterConditionType]kubermaticv1.ClusterCondition{
		kubermaticv1.ClusterConditionDefaultPolicyBindingsControllerCreatedSuccessfully: {
			Status: corev1.ConditionTrue,
		},
	}
	return cluster
}

func genPolicyTemplate(name string, defaultPolicy, enforced bool, visibility string, projectID string, clusterSelector, projectSelector *metav1.LabelSelector) *kubermaticv1.PolicyTemplate {
	template := &kubermaticv1.PolicyTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.PolicyTemplateSpec{
			Default:    defaultPolicy,
			Enforced:   enforced,
			Visibility: visibility,
			ProjectID:  projectID,
		},
	}

	if clusterSelector != nil || projectSelector != nil {
		template.Spec.Target = &kubermaticv1.PolicyTemplateTarget{
			ClusterSelector: clusterSelector,
			ProjectSelector: projectSelector,
		}
	}

	return template
}

func genClusterWithProject(name, projectID string, labels map[string]string) *kubermaticv1.Cluster {
	clusterLabels := map[string]string{
		kubermaticv1.ProjectIDLabelKey: projectID,
		"environment":                  "test",
	}
	// Merge provided labels
	for k, v := range labels {
		clusterLabels[k] = v
	}

	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: clusterLabels,
		},
		Spec: kubermaticv1.ClusterSpec{
			Version: *kubernetesVersion,
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: "global",
			},
		},
		Status: kubermaticv1.ClusterStatus{
			ExtendedHealth: healthy(),
			NamespaceName:  "cluster-" + name,
		},
	}
}

func genProject(name string, labels map[string]string) *kubermaticv1.Project {
	return &kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Spec: kubermaticv1.ProjectSpec{
			Name: name,
		},
	}
}

func genPolicyTemplateWithExpressions(name string, defaultPolicy, enforced bool, visibility string, projectID string, clusterSelector, projectSelector *metav1.LabelSelector) *kubermaticv1.PolicyTemplate {
	template := &kubermaticv1.PolicyTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.PolicyTemplateSpec{
			Default:    defaultPolicy,
			Enforced:   enforced,
			Visibility: visibility,
			ProjectID:  projectID,
		},
	}

	if clusterSelector != nil || projectSelector != nil {
		template.Spec.Target = &kubermaticv1.PolicyTemplateTarget{
			ClusterSelector: clusterSelector,
			ProjectSelector: projectSelector,
		}
	}

	return template
}

// genPolicyTemplateWithNilSelectors creates a PolicyTemplate with Target containing nil selectors.
// This is used to test the case where target: {} in YAML creates Target with nil selectors.
func genPolicyTemplateWithNilSelectors(name string, defaultPolicy, enforced bool, visibility string, projectID string, createTarget, setClusterSelector, setProjectSelector bool) kubermaticv1.PolicyTemplate {
	template := kubermaticv1.PolicyTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: kubermaticv1.PolicyTemplateSpec{
			Default:    defaultPolicy,
			Enforced:   enforced,
			Visibility: visibility,
			ProjectID:  projectID,
		},
	}

	if createTarget {
		template.Spec.Target = &kubermaticv1.PolicyTemplateTarget{}
		if setClusterSelector {
			template.Spec.Target.ClusterSelector = &metav1.LabelSelector{}
		}
		if setProjectSelector {
			template.Spec.Target.ProjectSelector = &metav1.LabelSelector{}
		}
	}

	return template
}
