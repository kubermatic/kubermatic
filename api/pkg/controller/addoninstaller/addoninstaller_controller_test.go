package addoninstaller

import (
	"context"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var addons = []string{"Foo", "Bar"}

func truePtr() *bool {
	b := true
	return &b
}

func TestCreateAddon(t *testing.T) {
	name := "test-cluster"
	tests := []struct {
		name                  string
		expectedClusterAddons []*kubermaticv1.Addon
		cluster               *kubermaticv1.Cluster
	}{
		{
			name: "successfully created",
			expectedClusterAddons: []*kubermaticv1.Addon{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "Foo",
						Namespace: "cluster-" + name,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "kubermatic.k8s.io/v1",
								Kind:               "Cluster",
								Name:               name,
								Controller:         truePtr(),
								BlockOwnerDeletion: truePtr(),
							},
						},
					},
					Spec: kubermaticv1.AddonSpec{
						Name: "Foo",
						Cluster: corev1.ObjectReference{
							Kind: "Cluster",
							Name: name,
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "Bar",
						Namespace: "cluster-" + name,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         "kubermatic.k8s.io/v1",
								Kind:               "Cluster",
								Name:               name,
								Controller:         truePtr(),
								BlockOwnerDeletion: truePtr(),
							},
						},
					},
					Spec: kubermaticv1.AddonSpec{
						Name: "Bar",
						Cluster: corev1.ObjectReference{
							Kind: "Cluster",
							Name: name,
						},
					},
				},
			},
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: name,
				},
				Spec:    kubermaticv1.ClusterSpec{},
				Address: kubermaticv1.ClusterAddress{},
				Status: kubermaticv1.ClusterStatus{
					Health: kubermaticv1.ClusterHealth{
						ClusterHealthStatus: kubermaticv1.ClusterHealthStatus{
							Apiserver: true,
						},
					},
					NamespaceName: "cluster-" + name,
				},
			},
		},
	}

	for _, test := range tests {
		if err := kubermaticv1.SchemeBuilder.AddToScheme(scheme.Scheme); err != nil {
			t.Fatalf("failed to add kubermaticv1 scheme to scheme.Scheme: %v", err)
		}

		t.Run(test.name, func(t *testing.T) {
			objs := []runtime.Object{test.cluster}

			client := ctrlruntimefakeclient.NewFakeClient(objs...)

			reconciler := Reconciler{
				Client:           client,
				kubernetesAddons: addons,
			}

			if _, err := reconciler.reconcile(context.Background(), test.cluster); err != nil {
				t.Fatalf("Reconciliation failed: %v", err)
			}

			for _, expectedAddon := range test.expectedClusterAddons {
				addonFromClient := &kubermaticv1.Addon{}
				if err := client.Get(context.Background(),
					types.NamespacedName{Namespace: test.cluster.Status.NamespaceName, Name: expectedAddon.Name},
					addonFromClient); err != nil {
					t.Fatalf("Did not find expected addon %q", expectedAddon.Name)
				}
				if equal := equality.Semantic.DeepEqual(expectedAddon, addonFromClient); !equal {
					t.Fatalf("created addon is not equal to expected addon\n%+v\n---\n%+v", expectedAddon, addonFromClient)
				}
			}

		})
	}
}
