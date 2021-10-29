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
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testClusterName         = "test-constraints"
	testNamespace           = "cluster-test-constraints"
	testKubermaticNamespace = "kubermatic"
	testKind                = "reqlabel"
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
				test.GenConstraint("ct1", testNamespace, testKind),
				test.GenConstraint("ct2", testNamespace, testKind),
				test.GenConstraint("ct3", "other-ns", testKind),
			},
			cluster:             genCluster(testClusterName, "kubernetes", "my-first-project-ID", "test-constraints", "john@acme.com"),
			expectedConstraints: []*kubermaticv1.Constraint{test.GenConstraint("ct1", testNamespace, testKind), test.GenConstraint("ct2", testNamespace, testKind)},
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
				test.GenConstraint("ct1", testNamespace, testKind),
				test.GenConstraint("ct2", testNamespace, testKind),
				test.GenConstraint("ct3", "other-ns", testKind),
			},
			cluster:            genCluster(testClusterName, "kubernetes", "my-first-project-ID", "test-constraints", "john@acme.com"),
			expectedConstraint: test.GenConstraint("ct1", testNamespace, testKind),
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
				test.GenConstraint("ct1", testNamespace, testKind),
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
			constraint: test.GenConstraint("ct1", testNamespace, testKind),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := fakectrlruntimeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
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
				test.GenConstraint("ct1", testNamespace, testKind),
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

func TestCreateDefaultConstraint(t *testing.T) {

	testCases := []struct {
		name       string
		ctToCreate *kubermaticv1.Constraint
		expectedCT *kubermaticv1.Constraint
	}{
		{
			name:       "scenario 1: create constraint",
			ctToCreate: test.GenConstraint("ct", testKubermaticNamespace, testKind),
			expectedCT: test.GenConstraint("ct", testKubermaticNamespace, testKind),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			client := fakectrlruntimeclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return client, nil
			}
			defaultConstraintProvider, err := kubernetes.NewDefaultConstraintProvider(fakeImpersonationClient, client, testKubermaticNamespace)
			if err != nil {
				t.Fatal(err)
			}

			constraint, err := defaultConstraintProvider.Create(tc.ctToCreate)
			if err != nil {
				t.Fatal(err)
			}

			// set the RV because it gets set when created
			tc.expectedCT.ResourceVersion = "1"
			if !reflect.DeepEqual(constraint, tc.expectedCT) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(constraint, tc.expectedCT))
			}
		})
	}
}

func TestListDefaultConstraints(t *testing.T) {

	testCases := []struct {
		name                string
		existingObjects     []ctrlruntimeclient.Object
		expectedConstraints []*kubermaticv1.Constraint
	}{
		{
			name: "scenario 1: list constraints",
			existingObjects: []ctrlruntimeclient.Object{
				test.GenConstraint("ct1", testKubermaticNamespace, testKind),
				test.GenConstraint("ct2", testKubermaticNamespace, testKind),
				test.GenConstraint("ct3", "other-ns", testKind),
			},
			expectedConstraints: []*kubermaticv1.Constraint{test.GenConstraint("ct1", testKubermaticNamespace, testKind), test.GenConstraint("ct2", testKubermaticNamespace, testKind)},
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
			defaultConstraintProvider, err := kubernetes.NewDefaultConstraintProvider(fakeImpersonationClient, client, testKubermaticNamespace)
			if err != nil {
				t.Fatal(err)
			}

			constraintList, err := defaultConstraintProvider.List()
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
					t.Fatalf("returned default constraint was not found on the list of expected ones, ct = %#v", returnedConstraint)
				}
			}
		})
	}
}

func TestGetDefaultConstraint(t *testing.T) {

	testCases := []struct {
		name               string
		existingObjects    []ctrlruntimeclient.Object
		expectedConstraint *kubermaticv1.Constraint
	}{
		{
			name: "scenario 1: get constraint",
			existingObjects: []ctrlruntimeclient.Object{
				test.GenConstraint("ct1", testKubermaticNamespace, testKind),
				test.GenConstraint("ct2", testKubermaticNamespace, testKind),
				test.GenConstraint("ct3", "other-ns", testKind),
			},
			expectedConstraint: test.GenConstraint("ct1", testKubermaticNamespace, testKind),
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
			defaultConstraintProvider, err := kubernetes.NewDefaultConstraintProvider(fakeImpersonationClient, client, testKubermaticNamespace)
			if err != nil {
				t.Fatal(err)
			}

			constraint, err := defaultConstraintProvider.Get(tc.expectedConstraint.Name)
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

func TestDeleteDefaultConstraint(t *testing.T) {
	testCases := []struct {
		name            string
		existingObjects []ctrlruntimeclient.Object
		CTtoDelete      *kubermaticv1.Constraint
	}{
		{
			name:            "test: delete default constraint",
			existingObjects: []ctrlruntimeclient.Object{test.GenConstraint("ct", testKubermaticNamespace, testKind)},
			CTtoDelete:      test.GenConstraint("ct", testKubermaticNamespace, testKind),
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
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
			provider, err := kubernetes.NewDefaultConstraintProvider(fakeImpersonationClient, client, testKubermaticNamespace)
			if err != nil {
				t.Fatal(err)
			}

			err = provider.Delete(tc.CTtoDelete.Name)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUpdateDefaultConstraint(t *testing.T) {
	testCases := []struct {
		name             string
		updateConstraint func(*kubermaticv1.Constraint)
		existingObjects  []ctrlruntimeclient.Object
		expectedCT       *kubermaticv1.Constraint
	}{
		{
			name: "scenario 1: update default constraint",
			updateConstraint: func(ct *kubermaticv1.Constraint) {
				ct.Spec.Match.Kinds = append(ct.Spec.Match.Kinds, kubermaticv1.Kind{Kinds: []string{"pod"}, APIGroups: []string{"v1"}})
				ct.Spec.Match.Scope = "*"
			},
			existingObjects: []ctrlruntimeclient.Object{
				test.GenConstraint("ct1", testKubermaticNamespace, testKind),
			},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			client := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.existingObjects...).
				Build()

			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return client, nil
			}
			provider, err := kubernetes.NewDefaultConstraintProvider(fakeImpersonationClient, client, testKubermaticNamespace)
			if err != nil {
				t.Fatal(err)
			}

			// fetch default constraint to get the ResourceVersion
			constraint := &kubermaticv1.Constraint{}
			if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(tc.existingObjects[0]), constraint); err != nil {
				t.Fatal(err)
			}

			updatedCT := constraint.DeepCopy()
			tc.updateConstraint(updatedCT)

			constraint, err = provider.Update(updatedCT)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(constraint, updatedCT) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(constraint, updatedCT))
			}
		})
	}
}
