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

package kubernetes_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/go-test/deep"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testClusterName = "test-constraints"
	testNamespace   = "cluster-test-constraints"
)

func TestListConstraints(t *testing.T) {

	testCases := []struct {
		name                string
		existingObjects     []ctrlruntimeclient.Object
		cluster             *kubermaticv1.Cluster
		expectedConstraints []*kubermaticv1.Constraint
	}{
		{
			name: "scenario 1: list constraints",
			existingObjects: []ctrlruntimeclient.Object{
				genConstraint("ct1", testNamespace),
				genConstraint("ct2", testNamespace),
				genConstraint("ct3", "other-ns"),
			},
			cluster:             genCluster(testClusterName, "kubernetes", "my-first-project-ID", "test-constraints", "john@acme.com"),
			expectedConstraints: []*kubermaticv1.Constraint{genConstraint("ct1", testNamespace), genConstraint("ct2", testNamespace)},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.existingObjects...).
				Build()

			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return client, nil
			}
			constraintProvider, err := kubernetes.NewConstraintProvider(fakeImpersonationClient, client)
			if err != nil {
				t.Fatal(err)
			}

			constraintList, err := constraintProvider.List(tc.cluster)
			if err != nil {
				t.Fatal(err)
			}
			if len(tc.expectedConstraints) != len(constraintList.Items) {
				t.Fatalf("expected to get %d constraints, but got %d", len(tc.expectedConstraints), len(constraintList.Items))
			}

			for _, returnedConstraint := range constraintList.Items {
				returnedConstraint.ResourceVersion = ""
				cFound := false
				for _, expectedCT := range tc.expectedConstraints {
					expectedCT.ResourceVersion = ""
					if dif := deep.Equal(returnedConstraint, *expectedCT); dif == nil {
						cFound = true
						break
					}
				}
				if !cFound {
					t.Fatalf("returned constraint was not found on the list of expected ones, ct = %#v", returnedConstraint)
				}
			}
		})
	}
}

func TestGetConstraint(t *testing.T) {

	testCases := []struct {
		name               string
		existingObjects    []ctrlruntimeclient.Object
		cluster            *kubermaticv1.Cluster
		expectedConstraint *kubermaticv1.Constraint
	}{
		{
			name: "scenario 1: get constraint",
			existingObjects: []ctrlruntimeclient.Object{
				genConstraint("ct1", testNamespace),
				genConstraint("ct2", testNamespace),
				genConstraint("ct3", "other-ns"),
			},
			cluster:            genCluster(testClusterName, "kubernetes", "my-first-project-ID", "test-constraints", "john@acme.com"),
			expectedConstraint: genConstraint("ct1", testNamespace),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.existingObjects...).
				Build()

			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return client, nil
			}
			constraintProvider, err := kubernetes.NewConstraintProvider(fakeImpersonationClient, client)
			if err != nil {
				t.Fatal(err)
			}

			constraint, err := constraintProvider.Get(tc.cluster, tc.expectedConstraint.Name)
			if err != nil {
				t.Fatal(err)
			}

			tc.expectedConstraint.ResourceVersion = constraint.ResourceVersion

			if !reflect.DeepEqual(constraint, tc.expectedConstraint) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(constraint, tc.expectedConstraint))
			}
		})
	}
}

func TestDeleteConstraint(t *testing.T) {

	testCases := []struct {
		name            string
		existingObjects []ctrlruntimeclient.Object
		userInfo        *provider.UserInfo
		cluster         *kubermaticv1.Cluster
		constraintName  string
	}{
		{
			name: "scenario 1: delete constraint",
			existingObjects: []ctrlruntimeclient.Object{
				genConstraint("ct1", testNamespace),
			},
			userInfo:       &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			cluster:        genCluster(testClusterName, "kubernetes", "my-first-project-ID", "test-constraints", "john@acme.com"),
			constraintName: "ct1",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.existingObjects...).
				Build()

			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return client, nil
			}
			constraintProvider, err := kubernetes.NewConstraintProvider(fakeImpersonationClient, client)
			if err != nil {
				t.Fatal(err)
			}

			err = constraintProvider.Delete(tc.cluster, tc.userInfo, tc.constraintName)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestCreateConstraint(t *testing.T) {

	testCases := []struct {
		name       string
		cluster    *kubermaticv1.Cluster
		userInfo   *provider.UserInfo
		constraint *kubermaticv1.Constraint
	}{
		{
			name:       "scenario 1: create constraint",
			cluster:    genCluster(testClusterName, "kubernetes", "my-first-project-ID", "test-constraints", "john@acme.com"),
			userInfo:   &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			constraint: genConstraint("ct1", testNamespace),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme)
			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return client, nil
			}
			constraintProvider, err := kubernetes.NewConstraintProvider(fakeImpersonationClient, client)
			if err != nil {
				t.Fatal(err)
			}

			_, err = constraintProvider.Create(tc.userInfo, tc.constraint)
			if err != nil {
				t.Fatal(err)
			}

			constraint, err := constraintProvider.Get(tc.cluster, tc.constraint.Name)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(constraint, tc.constraint) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(constraint, tc.constraint))
			}
		})
	}
}

func TestUpdateConstraint(t *testing.T) {

	testCases := []struct {
		name             string
		updateConstraint func(*kubermaticv1.Constraint)
		existingObjects  []ctrlruntimeclient.Object
		cluster          *kubermaticv1.Cluster
		userInfo         *provider.UserInfo
	}{
		{
			name: "scenario 1: update constraint",
			updateConstraint: func(ct *kubermaticv1.Constraint) {
				ct.Spec.Match.Kinds = append(ct.Spec.Match.Kinds, kubermaticv1.Kind{Kinds: []string{"pod"}, APIGroups: []string{"v1"}})
				ct.Spec.Match.Scope = "*"
			},
			existingObjects: []ctrlruntimeclient.Object{
				genConstraint("ct1", testNamespace),
			},
			cluster:  genCluster(testClusterName, "kubernetes", "my-first-project-ID", "test-constraints", "john@acme.com"),
			userInfo: &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			client := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.existingObjects...).
				Build()

			// fetch constraint to get the ResourceVersion
			constraint := &kubermaticv1.Constraint{}
			if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(tc.existingObjects[0]), constraint); err != nil {
				t.Fatal(err)
			}

			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return client, nil
			}
			constraintProvider, err := kubernetes.NewConstraintProvider(fakeImpersonationClient, client)
			if err != nil {
				t.Fatal(err)
			}

			updatedConstraint := constraint.DeepCopy()
			tc.updateConstraint(updatedConstraint)

			constraint, err = constraintProvider.Update(tc.userInfo, updatedConstraint)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(constraint, updatedConstraint) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(constraint, updatedConstraint))
			}
		})
	}
}

func genConstraint(name, namespace string) *kubermaticv1.Constraint {
	ct := &kubermaticv1.Constraint{}
	ct.Kind = kubermaticv1.ConstraintKind
	ct.APIVersion = kubermaticv1.SchemeGroupVersion.String()
	ct.Name = name
	ct.Namespace = namespace
	ct.Spec = kubermaticv1.ConstraintSpec{
		ConstraintType: "requiredlabels",
		Match: kubermaticv1.Match{
			Kinds: []kubermaticv1.Kind{
				{Kinds: []string{"namespace"}, APIGroups: []string{""}},
			},
		},
		Parameters: kubermaticv1.Parameters{
			RawJSON: `{"labels":[ "gatekeeper", "opa"]}`,
		},
	}

	return ct
}
