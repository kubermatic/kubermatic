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

package initialmachinedeploymentcontroller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v3/pkg/cluster/client"
	"k8c.io/kubermatic/v3/pkg/defaulting"
	"k8c.io/kubermatic/v3/pkg/machine"
	"k8c.io/kubermatic/v3/pkg/machine/operatingsystem"
	"k8c.io/kubermatic/v3/pkg/machine/provider"
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/kubermatic/v3/pkg/test"
	"k8c.io/kubermatic/v3/pkg/util/edition"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	kubernetesVersion = defaulting.DefaultKubernetesVersioning.Default
)

const (
	datacenterName = "testdc"
	projectID      = "testproject"
)

func init() {
	if err := clusterv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		panic(fmt.Sprintf("failed to add clusterv1alpha1 to scheme: %v", err))
	}
}

func healthy() kubermaticv1.ExtendedClusterHealth {
	up := kubermaticv1.HealthStatusUp

	return kubermaticv1.ExtendedClusterHealth{
		Apiserver:                    kubermaticv1.HealthStatusUp,
		ApplicationController:        kubermaticv1.HealthStatusUp,
		Scheduler:                    kubermaticv1.HealthStatusUp,
		Controller:                   kubermaticv1.HealthStatusUp,
		MachineController:            kubermaticv1.HealthStatusUp,
		Etcd:                         kubermaticv1.HealthStatusUp,
		OpenVPN:                      &up,
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
		},
		Spec: kubermaticv1.ClusterSpec{
			Version: *kubernetesVersion,
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: datacenterName,
				ProviderName:   kubermaticv1.CloudProviderHetzner,
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
		WithOperatingSystemSpec(operatingsystem.NewUbuntuSpecBuilder(kubermaticv1.CloudProviderHetzner).Build()).
		WithCloudProviderSpec(provider.NewHetznerConfig().WithServerType("cx21").Build()).
		BuildProviderSpec()
	if err != nil {
		t.Fatalf("Failed to create provider spec: %v", err)
	}

	dummyMD := clusterv1alpha1.MachineDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
		Spec: clusterv1alpha1.MachineDeploymentSpec{
			Replicas: pointer.Int32(1),
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

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			webhook := &appsv1.Deployment{}
			webhook.Name = resources.MachineControllerWebhookDeploymentName
			webhook.Namespace = tc.cluster.Status.NamespaceName
			webhook.Spec.Replicas = pointer.Int32(1)

			if tc.mcHealthy {
				webhook.Status.Replicas = *webhook.Spec.Replicas
				webhook.Status.AvailableReplicas = *webhook.Spec.Replicas
				webhook.Status.ReadyReplicas = *webhook.Spec.Replicas
				webhook.Status.UpdatedReplicas = *webhook.Spec.Replicas
			}

			seedClient := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.cluster, webhook).
				Build()

			userClusterObjects := []ctrlruntimeclient.Object{}
			userClusterClient := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(userClusterObjects...).
				Build()

			ctx := context.Background()
			r := &Reconciler{
				Client:   seedClient,
				recorder: &record.FakeRecorder{},
				log:      logger,
				versions: kubermatic.NewFakeVersions(edition.CommunityEdition),

				userClusterConnectionProvider: newFakeClientProvider(userClusterClient),

				// this dummy datacenterGetter returns the same dummy hetzner DC for all tests
				datacenterGetter: test.NewDatacenterGetter(&kubermaticv1.Datacenter{
					Spec: kubermaticv1.DatacenterSpec{
						Provider: kubermaticv1.DatacenterProviderSpec{
							Hetzner: &kubermaticv1.DatacenterSpecHetzner{
								Datacenter: "hel1",
								Network:    "default",
							},
						},
					},
				}),
			}

			nName := types.NamespacedName{Name: tc.cluster.Name}

			// let the magic happen
			_, reconcileErr := r.Reconcile(ctx, reconcile.Request{NamespacedName: nName})

			// fetch potentially updated cluster object
			newCluster := &kubermaticv1.Cluster{}
			if err := r.Client.Get(ctx, nName, newCluster); err != nil {
				t.Fatalf("Cluster object in seed cluster could not be found anymore: %v", err)
			}

			// validate the result
			if err := tc.validate(newCluster, userClusterClient, reconcileErr); err != nil {
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
