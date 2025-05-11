//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0")
                     Copyright © 2025 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
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
	"k8s.io/client-go/tools/record"
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
				recorder:     &record.FakeRecorder{},
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
		OpenVPN:                      kubermaticv1.HealthStatusUp,
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
