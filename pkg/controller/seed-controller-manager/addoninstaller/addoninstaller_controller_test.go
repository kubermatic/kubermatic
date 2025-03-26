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
	"fmt"
	"strconv"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

var addons = kubermaticv1.AddonList{Items: []kubermaticv1.Addon{
	{ObjectMeta: metav1.ObjectMeta{Name: "Foo"}},
	{ObjectMeta: metav1.ObjectMeta{
		Name:        "Bar",
		Labels:      map[string]string{"addons.kubermatic.io/ensure": "true"},
		Annotations: map[string]string{"foo": "bar"},
	}},
}}

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
						Name:            "Foo",
						Namespace:       kubernetes.NamespaceName(name),
						ResourceVersion: "1",
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
						Name:            "Bar",
						Namespace:       kubernetes.NamespaceName(name),
						ResourceVersion: "1",
						Labels:          map[string]string{"addons.kubermatic.io/ensure": "true"},
						Annotations:     map[string]string{"foo": "bar"},
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
				Spec: kubermaticv1.ClusterSpec{},
				Status: kubermaticv1.ClusterStatus{
					ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
						Apiserver: kubermaticv1.HealthStatusUp,
					},
					NamespaceName: kubernetes.NamespaceName(name),
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			client := fake.
				NewClientBuilder().
				WithObjects(test.cluster).
				WithIndex(&kubermaticv1.Addon{}, addonDefaultKey, func(rawObj ctrlruntimeclient.Object) []string {
					a := rawObj.(*kubermaticv1.Addon)
					return []string{strconv.FormatBool(a.Spec.IsDefault)}
				}).
				Build()

			config := createKubermaticConfiguration(addons)
			configGetter, err := kubernetes.StaticKubermaticConfigurationGetterFactory(config)
			if err != nil {
				t.Fatalf("Failed to create Config Getter: %v", err)
			}

			reconciler := Reconciler{
				log:          kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				Client:       client,
				configGetter: configGetter,
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
				if d := diff.ObjectDiff(expectedAddon, addonFromClient); d != "" {
					t.Errorf("created addon is not equal to expected addon:\n%v", d)
				}
			}
		})
	}
}

func createKubermaticConfiguration(addons kubermaticv1.AddonList) *kubermaticv1.KubermaticConfiguration {
	encoded, err := yaml.Marshal(addons)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal addon list: %v", err))
	}

	return &kubermaticv1.KubermaticConfiguration{
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
				Addons: kubermaticv1.KubermaticAddonsConfiguration{
					DefaultManifests: string(encoded),
				},
			},
		},
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
					ObjectMeta: metav1.ObjectMeta{
						Name:            "Bar",
						Namespace:       kubernetes.NamespaceName(name),
						ResourceVersion: "1",
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
						Name:            "Foo",
						Namespace:       kubernetes.NamespaceName(name),
						ResourceVersion: "1",
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
						Name:            "Bar",
						Namespace:       kubernetes.NamespaceName(name),
						Labels:          map[string]string{"addons.kubermatic.io/ensure": "true"},
						Annotations:     map[string]string{"foo": "bar"},
						ResourceVersion: "2",
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
				Spec: kubermaticv1.ClusterSpec{},
				Status: kubermaticv1.ClusterStatus{
					ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
						Apiserver: kubermaticv1.HealthStatusUp,
					},
					NamespaceName: kubernetes.NamespaceName(name),
				},
			},
		},
		{
			name: "successfully created two addons and deleted one",
			existingClusterAddons: []*kubermaticv1.Addon{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:            "to-be-deleted",
						Namespace:       kubernetes.NamespaceName(name),
						ResourceVersion: "1",
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
					ObjectMeta: metav1.ObjectMeta{
						Name:            "Foo",
						Namespace:       kubernetes.NamespaceName(name),
						ResourceVersion: "1",
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
						Name:            "Bar",
						Namespace:       kubernetes.NamespaceName(name),
						Labels:          map[string]string{"addons.kubermatic.io/ensure": "true"},
						Annotations:     map[string]string{"foo": "bar"},
						ResourceVersion: "1",
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
				Spec: kubermaticv1.ClusterSpec{},
				Status: kubermaticv1.ClusterStatus{
					ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
						Apiserver: kubermaticv1.HealthStatusUp,
					},
					NamespaceName: kubernetes.NamespaceName(name),
				},
			},
		},
		// TODO Add test to ensure that user added addons are not
		// deleted when the following is merged:
		// https://github.com/kubernetes-sigs/controller-runtime/pull/800
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			objs := []ctrlruntimeclient.Object{test.cluster}
			for _, a := range test.existingClusterAddons {
				objs = append(objs, a)
			}

			client := fake.NewClientBuilder().
				WithObjects(objs...).
				WithIndex(&kubermaticv1.Addon{}, addonDefaultKey, func(rawObj ctrlruntimeclient.Object) []string {
					a := rawObj.(*kubermaticv1.Addon)
					return []string{strconv.FormatBool(a.Spec.IsDefault)}
				}).
				Build()

			config := createKubermaticConfiguration(addons)
			configGetter, err := kubernetes.StaticKubermaticConfigurationGetterFactory(config)
			if err != nil {
				t.Fatalf("Failed to create Config Getter: %v", err)
			}

			reconciler := Reconciler{
				log:          kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
				Client:       client,
				configGetter: configGetter,
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
				if d := diff.ObjectDiff(expectedAddon, addonFromClient); d != "" {
					t.Errorf("created addon is not equal to expected addon:\n%v", d)
				}
			}
		})
	}
}
