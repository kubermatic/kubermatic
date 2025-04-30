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

package common

import (
	"context"
	"fmt"
	"testing"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/google/go-cmp/cmp"

	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestComparableVersionSuffix(t *testing.T) {
	testcases := []struct {
		greater string
		smaller string
	}{
		// { X > Y }
		{"v1.10", "v1.9"},
		{"v1.0.1", "v1.0.0"},
		{"v1.0.10", "v1.0.9"},
		{"v1.0.0", "v1.0.0-beta.1"},
		{"v1.0.0-alpha.1", "v1.0.0-alpha.0"},
		{"v1.0.0-beta.0", "v1.0.0-alpha.2"},
		{"v1.0.0-1-gabcdef", "v1.0.0"},
		{"v1.0.0-10-gabcdef", "v1.0.0-9-gabcdef"},
		{"v1.0.1", "v1.0.0-9-gabcdef"},
		{"v1.0.1-beta.1-9-gabcdef", "v1.0.0-beta.1"},
		{"v1.0.1-beta.1-10-gabcdef", "v1.0.0-beta.1-9-gabcdef"},
		{"v1.0.0-beta.2", "v1.0.0-beta.1-7-gabcdef"},
	}

	for _, testcase := range testcases {
		t.Run(fmt.Sprintf("%s > %s", testcase.greater, testcase.smaller), func(t *testing.T) {
			smaller, err := semverlib.NewVersion(comparableVersionSuffix(testcase.smaller))
			if err != nil {
				t.Fatalf("Failed to parse smaller value %q: %v", testcase.smaller, err)
			}

			greater, err := semverlib.NewVersion(comparableVersionSuffix(testcase.greater))
			if err != nil {
				t.Fatalf("Failed to parse greater value %q: %v", testcase.greater, err)
			}

			if !greater.GreaterThan(smaller) {
				t.Fatalf("Comparing %q > %q after patching (%q > %q) should have yielded true.", testcase.greater, testcase.smaller, greater.String(), smaller.String())
			}
		})
	}
}

func TestReconcilingCRDs(t *testing.T) {
	const crdName = "objects.kubermatic.k8c.io"

	theUpdate := &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: crdName,
		},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Scope: apiextensionsv1.ClusterScoped,
		},
	}

	testcases := []struct {
		name     string
		versions kubermaticversion.Versions
		existing *apiextensionsv1.CustomResourceDefinition
		expected *apiextensionsv1.CustomResourceDefinition
	}{
		{
			name:     "CRD doesn't exist yet, should be created",
			versions: kubermaticversion.Versions{KubermaticContainerTag: "irrelevant-too", GitVersion: "does-not-matter-in-this-case"},
			existing: nil,
			expected: &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: crdName,
					Annotations: map[string]string{
						resources.VersionLabel: "does-not-matter-in-this-case",
					},
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Scope: apiextensionsv1.ClusterScoped,
				},
			},
		},
		{
			name:     "upgrade existing v1 CRD to v2",
			versions: kubermaticversion.Versions{KubermaticContainerTag: "v2.0.0", GitVersion: "v2.0.0"},
			existing: &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: crdName,
					Annotations: map[string]string{
						resources.VersionLabel: "v1.9.0",
					},
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Scope: apiextensionsv1.NamespaceScoped,
				},
			},
			expected: &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: crdName,
					Annotations: map[string]string{
						resources.VersionLabel: "v2.0.0",
					},
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Scope: apiextensionsv1.ClusterScoped,
				},
			},
		},
		{
			name:     "do not downgrade a v3 CRD to v2",
			versions: kubermaticversion.Versions{KubermaticContainerTag: "v2.0.0", GitVersion: "v2.0.0"},
			existing: &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: crdName,
					Annotations: map[string]string{
						resources.VersionLabel: "v3.1.0",
					},
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Scope: apiextensionsv1.NamespaceScoped,
				},
			},
			expected: &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: crdName,
					Annotations: map[string]string{
						resources.VersionLabel: "v3.1.0",
					},
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Scope: apiextensionsv1.NamespaceScoped,
				},
			},
		},
		{
			name:     "upgrade from non-tagged version to a newer tagged version",
			versions: kubermaticversion.Versions{KubermaticContainerTag: "v2.26.0-beta.2", GitVersion: "v2.26.0-beta.2"},
			existing: &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: crdName,
					Annotations: map[string]string{
						resources.VersionLabel: "v2.26.0-beta.1-13-g108b0b653",
					},
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Scope: apiextensionsv1.NamespaceScoped,
				},
			},
			expected: &apiextensionsv1.CustomResourceDefinition{
				ObjectMeta: metav1.ObjectMeta{
					Name: crdName,
					Annotations: map[string]string{
						resources.VersionLabel: "v2.26.0-beta.2",
					},
				},
				Spec: apiextensionsv1.CustomResourceDefinitionSpec{
					Scope: apiextensionsv1.ClusterScoped,
				},
			},
		},
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))

	logger := log.NewDefault().Sugar()

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			clientBuilder := fakectrlruntimeclient.NewClientBuilder().WithScheme(scheme)
			if tc.existing != nil {
				clientBuilder.WithObjects(tc.existing)
			}

			client := clientBuilder.Build()

			factories := []reconciling.NamedCustomResourceDefinitionReconcilerFactory{
				CRDReconciler(theUpdate, logger, tc.versions),
			}

			// reconcile the CRD
			err := reconciling.ReconcileCustomResourceDefinitions(ctx, factories, "", client)
			if err != nil {
				t.Fatalf("Failed to reconcile CRD: %v", err)
			}

			// get the new state in-cluster
			newState := &apiextensionsv1.CustomResourceDefinition{}
			if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(theUpdate), newState); err != nil {
				t.Fatalf("Failed to get new CRD state from cluster: %v", err)
			}

			// assert that the desired changes (if any) were made
			if !cmp.Equal(newState.Annotations, tc.expected.Annotations) {
				t.Errorf("Expected annotations = %v, but got %v.", tc.expected.Annotations, newState.Annotations)
			}

			// makes comparisons easier in this testcase
			newState.Spec.Conversion = nil

			if !cmp.Equal(newState.Spec, tc.expected.Spec) {
				t.Errorf("Expected spec = %+v, but got %+v.", tc.expected.Spec, newState.Spec)
			}
		})
	}
}
