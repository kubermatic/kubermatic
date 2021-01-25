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
	"reflect"
	"testing"

	"github.com/go-test/deep"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
)

func TestListConstraintTemplates(t *testing.T) {
	testCases := []struct {
		name            string
		existingObjects []ctrlruntimeclient.Object
		expectedCTList  []*kubermaticv1.ConstraintTemplate
	}{
		{
			name:            "test: list constraint templates",
			existingObjects: []ctrlruntimeclient.Object{genConstraintTemplate("ct1"), genConstraintTemplate("ct2")},
			expectedCTList:  []*kubermaticv1.ConstraintTemplate{genConstraintTemplate("ct1"), genConstraintTemplate("ct2")},
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
			provider, err := kubernetes.NewConstraintTemplateProvider(fakeImpersonationClient, client)
			if err != nil {
				t.Fatal(err)
			}

			ctList, err := provider.List()
			if err != nil {
				t.Fatal(err)
			}
			if len(tc.expectedCTList) != len(ctList.Items) {
				t.Fatalf("expected to get %d cts, but got %d", len(tc.expectedCTList), len(ctList.Items))
			}
			for _, returnedCT := range ctList.Items {
				ctFound := false
				for _, expectedCT := range tc.expectedCTList {
					if dif := deep.Equal(returnedCT, *expectedCT); dif == nil {
						ctFound = true
						break
					}
				}
				if !ctFound {
					t.Fatalf("returned ct was not found on the list of expected ones, ct = %#v", returnedCT)
				}
			}
		})
	}
}

func TestGetConstraintTemplates(t *testing.T) {
	testCases := []struct {
		name            string
		existingObjects []ctrlruntimeclient.Object
		expectedCT      *kubermaticv1.ConstraintTemplate
	}{
		{
			name:            "test: get constraint template",
			existingObjects: []ctrlruntimeclient.Object{genConstraintTemplate("ct1")},
			expectedCT:      genConstraintTemplate("ct1"),
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
			provider, err := kubernetes.NewConstraintTemplateProvider(fakeImpersonationClient, client)
			if err != nil {
				t.Fatal(err)
			}

			ct, err := provider.Get("ct1")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(ct, tc.expectedCT) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(ct, tc.expectedCT))
			}
		})
	}
}

func TestCreateConstraintTemplates(t *testing.T) {
	testCases := []struct {
		name       string
		ctToCreate *kubermaticv1.ConstraintTemplate
		expectedCT *kubermaticv1.ConstraintTemplate
	}{
		{
			name:       "test: create constraint template",
			ctToCreate: genConstraintTemplate("ct1"),
			expectedCT: genConstraintTemplate("ct1"),
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme)
			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return client, nil
			}
			provider, err := kubernetes.NewConstraintTemplateProvider(fakeImpersonationClient, client)
			if err != nil {
				t.Fatal(err)
			}

			ct, err := provider.Create(tc.ctToCreate)
			if err != nil {
				t.Fatal(err)
			}

			// set the RV because it gets set when created
			tc.expectedCT.ResourceVersion = "1"
			if !reflect.DeepEqual(ct, tc.expectedCT) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(ct, tc.expectedCT))
			}
		})
	}
}

func TestUpdateConstraintTemplates(t *testing.T) {
	testCases := []struct {
		name            string
		ctToUpdate      *kubermaticv1.ConstraintTemplate
		existingObjects []ctrlruntimeclient.Object
		expectedCT      *kubermaticv1.ConstraintTemplate
	}{
		{
			name: "test: update constraint template",
			ctToUpdate: func() *kubermaticv1.ConstraintTemplate {
				defaultCT := genConstraintTemplate("ct1")
				defaultCT.Spec.CRD.Spec.Names.ShortNames = []string{"lc", "lcon"}
				return defaultCT
			}(),
			existingObjects: []ctrlruntimeclient.Object{genConstraintTemplate("ct1")},
			expectedCT: func() *kubermaticv1.ConstraintTemplate {
				defaultCT := genConstraintTemplate("ct1")
				defaultCT.Spec.CRD.Spec.Names.ShortNames = []string{"lc", "lcon"}
				defaultCT.ResourceVersion = "1"
				return defaultCT
			}(),
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
			provider, err := kubernetes.NewConstraintTemplateProvider(fakeImpersonationClient, client)
			if err != nil {
				t.Fatal(err)
			}

			ct, err := provider.Update(tc.ctToUpdate)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(ct, tc.expectedCT) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(ct, tc.expectedCT))
			}
		})
	}
}

func TestDeleteConstraintTemplates(t *testing.T) {
	testCases := []struct {
		name            string
		existingObjects []ctrlruntimeclient.Object
		CTtoDelete      *kubermaticv1.ConstraintTemplate
	}{
		{
			name:            "test: delete constraint template",
			existingObjects: []ctrlruntimeclient.Object{genConstraintTemplate("ct1")},
			CTtoDelete:      genConstraintTemplate("ct1"),
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
			provider, err := kubernetes.NewConstraintTemplateProvider(fakeImpersonationClient, client)
			if err != nil {
				t.Fatal(err)
			}

			err = provider.Delete(tc.CTtoDelete)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
