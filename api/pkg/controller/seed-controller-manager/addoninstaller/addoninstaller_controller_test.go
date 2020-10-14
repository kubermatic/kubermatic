package addoninstaller

import (
	"context"
	"testing"

	"github.com/go-test/deep"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var addons = kubermaticv1.AddonList{Items: []kubermaticv1.Addon{
	{ObjectMeta: metav1.ObjectMeta{Name: "Foo"}},
	{ObjectMeta: metav1.ObjectMeta{
		Name:        "Bar",
		Labels:      map[string]string{"addons.kubermatic.io/ensure": "true"},
		Annotations: map[string]string{"foo": "bar"},
	}},
}}

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
						IsDefault: true,
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "Bar",
						Namespace:   "cluster-" + name,
						Labels:      map[string]string{"addons.kubermatic.io/ensure": "true"},
						Annotations: map[string]string{"foo": "bar"},
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
						IsDefault: true,
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
					ExtendedHealth: kubermaticv1.ExtendedClusterHealth{

						Apiserver: kubermaticv1.HealthStatusUp,
					},
					NamespaceName: "cluster-" + name,
				},
			},
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {
			objs := []runtime.Object{test.cluster}

			client := ctrlruntimefakeclient.NewFakeClient(objs...)

			reconciler := Reconciler{
				log:              kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				Client:           client,
				kubernetesAddons: addons,
			}

			if _, err := reconciler.reconcile(context.Background(), reconciler.log, test.cluster); err != nil {
				t.Fatalf("Reconciliation failed: %v", err)
			}

			for _, expectedAddon := range test.expectedClusterAddons {
				addonFromClient := &kubermaticv1.Addon{}
				if err := client.Get(context.Background(),
					types.NamespacedName{Namespace: test.cluster.Status.NamespaceName, Name: expectedAddon.Name},
					addonFromClient); err != nil {
					t.Fatalf("Did not find expected addon %q", expectedAddon.Name)
				}
				if diff := deep.Equal(addonFromClient, expectedAddon); diff != nil {
					t.Errorf("created addon is not equal to expected addon, diff: %v", diff)
				}
			}

		})
	}
}

func TestUpdateAddon(t *testing.T) {
	name := "test-cluster"
	tests := []struct {
		name                  string
		existingClusterAddons []*kubermaticv1.Addon
		expectedClusterAddons []*kubermaticv1.Addon
		cluster               *kubermaticv1.Cluster
	}{
		{
			name: "successfully created one addon and updated another",
			existingClusterAddons: []*kubermaticv1.Addon{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kubermatic.k8s.io/v1",
						Kind:       "Addon",
					},
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
						IsDefault: true,
					},
				},
			},
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
						IsDefault: true,
					},
				},
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kubermatic.k8s.io/v1",
						Kind:       "Addon",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:        "Bar",
						Namespace:   "cluster-" + name,
						Labels:      map[string]string{"addons.kubermatic.io/ensure": "true"},
						Annotations: map[string]string{"foo": "bar"},
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
						IsDefault: true,
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
					ExtendedHealth: kubermaticv1.ExtendedClusterHealth{

						Apiserver: kubermaticv1.HealthStatusUp,
					},
					NamespaceName: "cluster-" + name,
				},
			},
		},
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {
			objs := []runtime.Object{test.cluster}
			for _, a := range test.existingClusterAddons {
				objs = append(objs, a)
			}

			client := ctrlruntimefakeclient.NewFakeClient(objs...)

			reconciler := Reconciler{
				log:              kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				Client:           client,
				kubernetesAddons: addons,
			}

			if _, err := reconciler.reconcile(context.Background(), reconciler.log, test.cluster); err != nil {
				t.Fatalf("Reconciliation failed: %v", err)
			}

			for _, expectedAddon := range test.expectedClusterAddons {
				addonFromClient := &kubermaticv1.Addon{}
				if err := client.Get(context.Background(),
					types.NamespacedName{Namespace: test.cluster.Status.NamespaceName, Name: expectedAddon.Name},
					addonFromClient); err != nil {
					t.Fatalf("Did not find expected addon %q", expectedAddon.Name)
				}
				if diff := deep.Equal(addonFromClient, expectedAddon); diff != nil {
					t.Errorf("created addon is not equal to expected addon, diff: %v", diff)
				}
			}

		})
	}
}
