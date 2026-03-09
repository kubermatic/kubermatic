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

package initialmachinedeployment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/machine"
	"k8c.io/kubermatic/v2/pkg/machine/operatingsystem"
	"k8c.io/kubermatic/v2/pkg/machine/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/events"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	kubernetesVersion = defaulting.DefaultKubernetesVersioning.Default
	testScheme        = fake.NewScheme()
)

const (
	datacenterName = "testdc"
	projectID      = "testproject"
)

func init() {
	utilruntime.Must(clusterv1alpha1.AddToScheme(testScheme))
}

func healthy() kubermaticv1.ExtendedClusterHealth {
	return kubermaticv1.ExtendedClusterHealth{
		Apiserver:                    kubermaticv1.HealthStatusUp,
		ApplicationController:        kubermaticv1.HealthStatusUp,
		Scheduler:                    kubermaticv1.HealthStatusUp,
		Controller:                   kubermaticv1.HealthStatusUp,
		MachineController:            kubermaticv1.HealthStatusUp,
		Etcd:                         kubermaticv1.HealthStatusUp,
		OpenVPN:                      kubermaticv1.HealthStatusUp,
		CloudProviderInfrastructure:  kubermaticv1.HealthStatusUp,
		UserClusterControllerManager: kubermaticv1.HealthStatusUp,
	}
}

func genCluster(annotation string) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testcluster",
			Annotations: map[string]string{
				kubermaticv1.InitialMachineDeploymentRequestAnnotation: annotation,
			},
			Labels: map[string]string{
				kubermaticv1.ProjectIDLabelKey: projectID,
			},
		},
		Spec: kubermaticv1.ClusterSpec{
			Version: *kubernetesVersion,
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: datacenterName,
				ProviderName:   string(kubermaticv1.HetznerCloudProvider),
				Hetzner:        &kubermaticv1.HetznerCloudSpec{},
			},
		},
		Status: kubermaticv1.ClusterStatus{
			ExtendedHealth: healthy(),
			NamespaceName:  "cluster-testcluster",
		},
	}
}

func TestReconcile(t *testing.T) {
	logger := zap.NewNop().Sugar()

	providerSpec, err := machine.NewBuilder().
		WithOperatingSystemSpec(operatingsystem.NewUbuntuSpecBuilder(kubermaticv1.HetznerCloudProvider).Build()).
		WithCloudProviderSpec(provider.NewHetznerConfig().WithServerType("cx22").Build()).
		BuildProviderSpec()
	if err != nil {
		t.Fatalf("Failed to create provider spec: %v", err)
	}

	dummyMD := clusterv1alpha1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: clusterv1alpha1.MachineDeploymentSpec{
			Replicas: ptr.To[int32](1),
			Template: clusterv1alpha1.MachineTemplateSpec{
				Spec: clusterv1alpha1.MachineSpec{
					Versions: clusterv1alpha1.MachineVersionInfo{
						Kubelet: kubernetesVersion.String(),
					},
					ProviderSpec: *providerSpec,
				},
			},
		},
	}

	mdAnnotation, err := json.Marshal(dummyMD)
	if err != nil {
		t.Fatalf("Cannot marshal initial machine deployment: %v", err)
	}

	testCases := []struct {
		name      string
		mcHealthy bool
		cluster   *kubermaticv1.Cluster
		validate  func(cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error
	}{
		{
			name:      "no annotation exists, nothing should happen",
			mcHealthy: true,
			cluster:   genCluster(""),
			validate: func(cluster *kubermaticv1.Cluster, _ ctrlruntimeclient.Client, reconcileErr error) error {
				// cluster should now have its special condition
				name := kubermaticv1.ClusterConditionMachineDeploymentControllerReconcilingSuccess

				if cond := cluster.Status.Conditions[name]; cond.Status != corev1.ConditionTrue {
					return fmt.Errorf("cluster should have %v=%s condition, but does not", name, corev1.ConditionTrue)
				}

				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have produced an error, but returned: %w", reconcileErr)
				}

				return nil
			},
		},

		{
			name:      "MC webhook is not healthy, nothing should happen",
			mcHealthy: false,
			cluster:   genCluster(string(mdAnnotation)),
			validate: func(cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				// cluster should now have its special condition
				name := kubermaticv1.ClusterConditionMachineDeploymentControllerReconcilingSuccess

				if cond := cluster.Status.Conditions[name]; cond.Status != corev1.ConditionTrue {
					return fmt.Errorf("cluster should have %v=%s condition, but does not", name, corev1.ConditionTrue)
				}

				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have produced an error, but returned: %w", reconcileErr)
				}

				return nil
			},
		},

		{
			name:      "vanilla case, create a MachineDeployment from the annotation",
			mcHealthy: true,
			cluster:   genCluster(string(mdAnnotation)),
			validate: func(cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				if ann, ok := cluster.Annotations[kubermaticv1.InitialMachineDeploymentRequestAnnotation]; ok {
					return fmt.Errorf("annotation should be have been removed, but found %q on the cluster", ann)
				}

				machineDeployments := clusterv1alpha1.MachineDeploymentList{}
				if err := userClusterClient.List(context.Background(), &machineDeployments); err != nil {
					return fmt.Errorf("failed to list MachineDeployments in user cluster: %w", err)
				}

				if len(machineDeployments.Items) == 0 {
					return errors.New("did not find a MachineDeployment in the user cluster after the reconciler finished")
				}

				return nil
			},
		},

		{
			name:      "invalid annotations should cause errors and then be removed",
			mcHealthy: true,
			cluster:   genCluster("I am not valid JSON!"),
			validate: func(cluster *kubermaticv1.Cluster, _ ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr == nil {
					return errors.New("reconciling a bad annotation should have produced an error, but got nil")
				}

				if ann, ok := cluster.Annotations[kubermaticv1.InitialMachineDeploymentRequestAnnotation]; ok {
					return fmt.Errorf("bad annotation should be have been removed, but found %q on the cluster", ann)
				}

				return nil
			},
		},
	}

	project := &kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: projectID,
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			webhook := &appsv1.Deployment{}
			webhook.Name = resources.MachineControllerWebhookDeploymentName
			webhook.Namespace = test.cluster.Status.NamespaceName
			webhook.Spec.Replicas = ptr.To[int32](1)

			if test.mcHealthy {
				webhook.Status.Replicas = *webhook.Spec.Replicas
				webhook.Status.AvailableReplicas = *webhook.Spec.Replicas
				webhook.Status.ReadyReplicas = *webhook.Spec.Replicas
				webhook.Status.UpdatedReplicas = *webhook.Spec.Replicas
			}

			seedClient := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(test.cluster, project, webhook).
				Build()

			userClusterObjects := []ctrlruntimeclient.Object{}
			userClusterClient := fake.
				NewClientBuilder().
				WithScheme(testScheme).
				WithObjects(userClusterObjects...).
				Build()

			ctx := context.Background()
			r := &Reconciler{
				Client:   seedClient,
				recorder: &events.FakeRecorder{},
				log:      logger,
				versions: kubermatic.GetFakeVersions(),

				userClusterConnectionProvider: newFakeClientProvider(userClusterClient),

				// this dummy seedGetter returns the same dummy hetzner DC for all tests
				seedGetter: func() (*kubermaticv1.Seed, error) {
					return &kubermaticv1.Seed{
						Spec: kubermaticv1.SeedSpec{
							Datacenters: map[string]kubermaticv1.Datacenter{
								datacenterName: {
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

			// let the magic happen
			_, reconcileErr := r.Reconcile(ctx, reconcile.Request{NamespacedName: nName})

			// fetch potentially updated cluster object
			newCluster := &kubermaticv1.Cluster{}
			if err := r.Get(ctx, nName, newCluster); err != nil {
				t.Fatalf("Cluster object in seed cluster could not be found anymore: %v", err)
			}

			// validate the result
			if err := test.validate(newCluster, userClusterClient, reconcileErr); err != nil {
				t.Fatalf("Test failed: %v", err)
			}
		})
	}
}

type fakeClientProvider struct {
	client ctrlruntimeclient.Client
}

func newFakeClientProvider(client ctrlruntimeclient.Client) *fakeClientProvider {
	return &fakeClientProvider{
		client: client,
	}
}

func (f *fakeClientProvider) GetClient(ctx context.Context, c *kubermaticv1.Cluster, options ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error) {
	return f.client, nil
}
