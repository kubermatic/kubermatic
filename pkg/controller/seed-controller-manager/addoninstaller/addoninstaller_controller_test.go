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

package addoninstaller

import (
	"context"
	"testing"

	"github.com/go-test/deep"

	"k8c.io/kubermatic/v2/pkg/crd/client/clientset/versioned/scheme"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
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
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kubermatic.k8s.io/v1",
						Kind:       "Addon",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            "Foo",
						Namespace:       "cluster-" + name,
						ResourceVersion: "1",
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
						Name:            "Bar",
						Namespace:       "cluster-" + name,
						ResourceVersion: "1",
						Labels:          map[string]string{"addons.kubermatic.io/ensure": "true"},
						Annotations:     map[string]string{"foo": "bar"},
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
			client := ctrlruntimefakeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(test.cluster).
				Build()

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
						Name:            "Bar",
						Namespace:       "cluster-" + name,
						ResourceVersion: "1",
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
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kubermatic.k8s.io/v1",
						Kind:       "Addon",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            "Foo",
						Namespace:       "cluster-" + name,
						ResourceVersion: "1",
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
						Name:            "Bar",
						Namespace:       "cluster-" + name,
						Labels:          map[string]string{"addons.kubermatic.io/ensure": "true"},
						Annotations:     map[string]string{"foo": "bar"},
						ResourceVersion: "2",
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
		{
			name: "successfully created two addons and deleted one",
			existingClusterAddons: []*kubermaticv1.Addon{
				{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kubermatic.k8s.io/v1",
						Kind:       "Addon",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            "to-be-deleted",
						Namespace:       "cluster-" + name,
						ResourceVersion: "1",
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
						Name: "ToBeDeleted",
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
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kubermatic.k8s.io/v1",
						Kind:       "Addon",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:            "Foo",
						Namespace:       "cluster-" + name,
						ResourceVersion: "1",
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
						Name:            "Bar",
						Namespace:       "cluster-" + name,
						Labels:          map[string]string{"addons.kubermatic.io/ensure": "true"},
						Annotations:     map[string]string{"foo": "bar"},
						ResourceVersion: "1",
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
		//TODO(irozzo) Add test to ensure that user added addons are not
		//deleted when the following is merged:
		// https://github.com/kubernetes-sigs/controller-runtime/pull/800
	}

	for _, test := range tests {

		t.Run(test.name, func(t *testing.T) {
			objs := []ctrlruntimeclient.Object{test.cluster}
			for _, a := range test.existingClusterAddons {
				objs = append(objs, a)
			}

			client := ctrlruntimefakeclient.NewClientBuilder().WithObjects(objs...).Build()

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
