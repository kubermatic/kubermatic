/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package initialapplicationinstallationcontroller

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	appskubermaticv1 "k8c.io/api/v3/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	apiv1 "k8c.io/kubermatic/v3/pkg/api/v1"
	clusterclient "k8c.io/kubermatic/v3/pkg/cluster/client"
	"k8c.io/kubermatic/v3/pkg/defaulting"
	"k8c.io/kubermatic/v3/pkg/util/edition"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	kubernetesVersion = defaulting.DefaultKubernetesVersioning.Default
)

const (
	datacenterName  = "testdc"
	projectID       = "testproject"
	applicationName = "katana"
)

func init() {
	utilruntime.Must(appskubermaticv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(clusterv1alpha1.AddToScheme(scheme.Scheme))
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
				kubermaticv1.InitialApplicationInstallationsRequestAnnotation: annotation,
			},
		},
		Spec: kubermaticv1.ClusterSpec{
			Version: *kubernetesVersion,
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
				name := kubermaticv1.ClusterConditionApplicationInstallationControllerReconcilingSuccess

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
			name: "create a single ApplicationInstallation from the annotation",
			cluster: func() *kubermaticv1.Cluster {
				app := generateApplication(applicationName)
				applications := []apiv1.Application{app}

				data, err := json.Marshal(applications)
				if err != nil {
					panic(fmt.Sprintf("cannot marshal initial application installations: %v", err))
				}

				return genCluster(string(data))
			}(),
			validate: func(cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				if ann, ok := cluster.Annotations[kubermaticv1.InitialApplicationInstallationsRequestAnnotation]; ok {
					return fmt.Errorf("annotation should be have been removed, but found %q on the cluster", ann)
				}

				apps := appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), &apps); err != nil {
					return fmt.Errorf("failed to list ApplicationInstallations in user cluster: %w", err)
				}

				if len(apps.Items) != 1 {
					return errors.New("did not find an ApplicationInstallation in the user cluster after the reconciler finished")
				}

				return nil
			},
		},
		{
			name: "create multiple ApplicationInstallation from the annotation",
			cluster: func() *kubermaticv1.Cluster {
				app := generateApplication(applicationName)
				app2 := generateApplication("kold")
				applications := []apiv1.Application{app, app2}

				data, err := json.Marshal(applications)
				if err != nil {
					panic(fmt.Sprintf("cannot marshal initial application installations: %v", err))
				}
				return genCluster(string(data))
			}(),
			validate: func(cluster *kubermaticv1.Cluster, userClusterClient ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr != nil {
					return fmt.Errorf("reconciling should not have caused an error, but did: %w", reconcileErr)
				}

				if ann, ok := cluster.Annotations[kubermaticv1.InitialApplicationInstallationsRequestAnnotation]; ok {
					return fmt.Errorf("annotation should be have been removed, but found %q on the cluster", ann)
				}

				apps := appskubermaticv1.ApplicationInstallationList{}
				if err := userClusterClient.List(context.Background(), &apps); err != nil {
					return fmt.Errorf("failed to list ApplicationInstallations in user cluster: %w", err)
				}

				if len(apps.Items) != 2 {
					return errors.New("did not find the expected ApplicationInstallations in the user cluster after the reconciler finished")
				}

				return nil
			},
		},
		{
			name:    "invalid annotation should result in error and be removed",
			cluster: genCluster("I am not valid JSON!"),
			validate: func(cluster *kubermaticv1.Cluster, _ ctrlruntimeclient.Client, reconcileErr error) error {
				if reconcileErr == nil {
					return errors.New("reconciling a bad annotation should have produced an error, but got nil")
				}

				if ann, ok := cluster.Annotations[kubermaticv1.InitialApplicationInstallationsRequestAnnotation]; ok {
					return fmt.Errorf("bad annotation should be have been removed, but found %q on the cluster", ann)
				}

				return nil
			},
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			seedClient := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(test.cluster).
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
				versions: kubermatic.NewFakeVersions(edition.CommunityEdition),

				userClusterConnectionProvider: newFakeClientProvider(userClusterClient),
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

func generateApplication(name string) apiv1.Application {
	var values json.RawMessage
	err := json.Unmarshal([]byte(`{
		"key": "value",
		"key2": "value2"
	}`), &values)

	if err != nil {
		panic(fmt.Sprintf("can not unmarshall values: %v", err))
	}

	return apiv1.Application{
		ObjectMeta: apiv1.ObjectMeta{
			Name: name,
		},
		Spec: apiv1.ApplicationSpec{
			Namespace: apiv1.NamespaceSpec{
				Name:        fmt.Sprintf("app-%s", name),
				Create:      true,
				Labels:      map[string]string{"key": "value"},
				Annotations: map[string]string{"key": "value"},
			},
			ApplicationRef: apiv1.ApplicationRef{
				Name:    name,
				Version: "1.0.0",
			},
			Values: values,
		},
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
