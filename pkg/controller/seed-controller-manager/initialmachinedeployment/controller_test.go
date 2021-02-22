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

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	v1 "k8c.io/kubermatic/v2/pkg/api/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	kubernetesVersion = "v1.19.0"
	datacenterName    = "testdc"
	projectID         = "testproject"
)

func init() {
	if err := clusterv1alpha1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
		panic(fmt.Sprintf("failed to add clusterv1alpha1 to scheme: %v", err))
	}
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

func genCluster(annotation string) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testcluster",
			Annotations: map[string]string{
				v1.InitialMachineDeploymentRequestAnnotation: annotation,
			},
			Labels: map[string]string{
				kubermaticv1.ProjectIDLabelKey: projectID,
			},
		},
		Spec: kubermaticv1.ClusterSpec{
			Version: *semver.NewSemverOrDie(kubernetesVersion),
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: datacenterName,
			},
		},
		Status: kubermaticv1.ClusterStatus{
			ExtendedHealth: healthy(),
		},
	}
}

func TestReconcile(t *testing.T) {
	log := zap.NewNop().Sugar()

	testCases := []struct {
		name     string
		cluster  *kubermaticv1.Cluster
		validate func(cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error
	}{
		{
			name:    "no annotation exists, nothing should happen",
			cluster: genCluster(""),
			validate: func(cluster *kubermaticv1.Cluster, _ ctrlruntimeclient.Client, reconcileErr error) error {
				// cluster should now have its special condition
				name := kubermaticv1.ClusterConditionMachineDeploymentControllerReconcilingSuccess

				if _, cond := helper.GetClusterCondition(cluster, name); cond == nil {
					return fmt.Errorf("cluster should have %v condition, but does not", name)
				}

				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have produced an error, but returned: %v", reconcileErr)
				}

				return nil
			},
		},

		{
			name: "vanilla case, create a MachineDeployment from the annotation",
			cluster: func() *kubermaticv1.Cluster {
				nd := v1.NodeDeployment{
					ObjectMeta: v1.ObjectMeta{
						Name: "test",
					},
					Spec: v1.NodeDeploymentSpec{
						Replicas: 1,
						Template: v1.NodeSpec{
							Versions: v1.NodeVersionInfo{
								Kubelet: kubernetesVersion,
							},
							OperatingSystem: v1.OperatingSystemSpec{
								Ubuntu: &v1.UbuntuSpec{},
							},
							Cloud: v1.NodeCloudSpec{
								Hetzner: &v1.HetznerNodeSpec{
									Type:    "big",
									Network: "test",
								},
							},
						},
					},
				}

				data, err := json.Marshal(nd)
				if err != nil {
					panic(fmt.Sprintf("cannot marshal initial machine deployment: %v", err))
				}

				return genCluster(string(data))
			}(),
			validate: func(cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %v", reconcileErr)
				}

				if ann, ok := cluster.Annotations[v1.InitialMachineDeploymentRequestAnnotation]; ok {
					return fmt.Errorf("annotation should be have been removed, but found %q on the cluster", ann)
				}

				machineDeployments := clusterv1alpha1.MachineDeploymentList{}
				if err := userClusterClient.List(context.Background(), &machineDeployments); err != nil {
					return fmt.Errorf("failed to list MachineDeployments in user cluster: %v", err)
				}

				if len(machineDeployments.Items) == 0 {
					return errors.New("did not find a MachineDeployment in the user cluster after the reconciler finished")
				}

				return nil
			},
		},

		{
			name:    "invalid annotations should cause errors and then be removed",
			cluster: genCluster("I am not valid JSON!"),
			validate: func(cluster *kubermaticv1.Cluster, _ ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr == nil {
					return errors.New("reconciling a bad annotation should have produced an error, but got nil")
				}

				if ann, ok := cluster.Annotations[v1.InitialMachineDeploymentRequestAnnotation]; ok {
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
			seedClient := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(test.cluster, project).
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
				log:      log,
				versions: kubermatic.NewFakeVersions(),

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
			if err := r.Client.Get(ctx, nName, newCluster); err != nil {
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
