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
	"testing"

	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/pkg/provider"
	"github.com/kubermatic/kubermatic/pkg/provider/kubernetes"

	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/kubernetes/scheme"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateServiceAccount(t *testing.T) {
	// test data
	testcases := []struct {
		name                      string
		existingKubermaticObjects []runtime.Object
		project                   *kubermaticv1.Project
		userInfo                  *provider.UserInfo
		saName                    string
		saGroup                   string
		expectedSA                *kubermaticv1.User
		expectedSAName            string
	}{
		{
			name:     "scenario 1, create service account `test` for editors group",
			userInfo: &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			project:  genDefaultProject(),
			saName:   "test",
			saGroup:  "editors-my-first-project-ID",
			existingKubermaticObjects: []runtime.Object{
				createAuthenitactedUser(),
				genDefaultProject(),
			},
			expectedSA: func() *kubermaticv1.User {
				sa := createSANoPrefix("test", "my-first-project-ID", "editors", "1")
				sa.ResourceVersion = "1"
				return sa
			}(),
			expectedSAName: "1",
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			fakeClient := fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, tc.existingKubermaticObjects...)
			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return fakeClient, nil
			}
			// act
			target := kubernetes.NewServiceAccountProvider(fakeImpersonationClient, fakeClient, "localhost")

			sa, err := target.Create(tc.userInfo, tc.project, tc.saName, tc.saGroup)

			// validate
			if err != nil {
				t.Fatal(err)
			}

			//remove autogenerated fields
			sa.Name = tc.expectedSAName
			sa.Spec.Email = ""
			sa.Spec.ID = ""

			if !equality.Semantic.DeepEqual(sa, tc.expectedSA) {
				t.Fatalf("%v", diff.ObjectGoPrintSideBySide(tc.expectedSA, sa))
			}
		})
	}
}

func TestList(t *testing.T) {
	// test data
	testcases := []struct {
		name                      string
		existingKubermaticObjects []runtime.Object
		project                   *kubermaticv1.Project
		saName                    string
		userInfo                  *provider.UserInfo
		expectedSA                []*kubermaticv1.User
	}{
		{
			name:     "scenario 1, get existing service accounts",
			userInfo: &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			project:  genDefaultProject(),
			saName:   "test-1",
			existingKubermaticObjects: []runtime.Object{
				createAuthenitactedUser(),
				createSA("test-1", "my-first-project-ID", "editors", "1"),
				createSA("test-2", "abcd", "viewers", "2"),
				createSA("test-1", "dcba", "viewers", "3"),
			},
			expectedSA: []*kubermaticv1.User{
				createSANoPrefix("test-1", "my-first-project-ID", "editors", "1"),
			},
		},
		{
			name:     "scenario 2, service accounts not found for the project",
			userInfo: &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			project:  genDefaultProject(),
			saName:   "test",
			existingKubermaticObjects: []runtime.Object{
				createAuthenitactedUser(),
				createSA("test", "bbbb", "editors", "1"),
				createSA("fake", "abcd", "editors", "2"),
			},
			expectedSA: []*kubermaticv1.User{},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, tc.existingKubermaticObjects...)
			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return fakeClient, nil
			}
			// act
			target := kubernetes.NewServiceAccountProvider(fakeImpersonationClient, fakeClient, "localhost")

			saList, err := target.List(tc.userInfo, tc.project, &provider.ServiceAccountListOptions{ServiceAccountName: tc.saName})
			// validate
			if err != nil {
				t.Fatal(err)
			}
			if !equality.Semantic.DeepEqual(saList, tc.expectedSA) {
				t.Fatalf("%v", diff.ObjectGoPrintSideBySide(tc.expectedSA, saList))
			}
		})
	}
}

func TestGet(t *testing.T) {
	// test data
	testcases := []struct {
		name                      string
		existingKubermaticObjects []runtime.Object
		project                   *kubermaticv1.Project
		saName                    string
		userInfo                  *provider.UserInfo
		expectedSA                *kubermaticv1.User
	}{
		{
			name:     "scenario 1, get existing service account",
			userInfo: &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			project:  genDefaultProject(),
			saName:   "1",
			existingKubermaticObjects: []runtime.Object{
				createAuthenitactedUser(),
				createSA("test-1", "my-first-project-ID", "editors", "1"),
				createSA("test-2", "abcd", "viewers", "2"),
				createSA("test-1", "dcba", "viewers", "3"),
			},
			expectedSA: func() *kubermaticv1.User {
				sa := createSANoPrefix("test-1", "my-first-project-ID", "editors", "1")
				sa.Kind = "User"
				sa.APIVersion = "kubermatic.k8s.io/v1"
				return sa
			}(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, tc.existingKubermaticObjects...)
			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return fakeClient, nil
			}
			// act
			target := kubernetes.NewServiceAccountProvider(fakeImpersonationClient, fakeClient, "localhost")

			sa, err := target.Get(tc.userInfo, tc.saName, nil)
			// validate
			if err != nil {
				t.Fatal(err)
			}
			if !equality.Semantic.DeepEqual(sa, tc.expectedSA) {
				t.Fatalf("%v", diff.ObjectGoPrintSideBySide(tc.expectedSA, sa))
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	// test data
	testcases := []struct {
		name                      string
		existingKubermaticObjects []runtime.Object
		project                   *kubermaticv1.Project
		saName                    string
		newName                   string
		userInfo                  *provider.UserInfo
		expectedSA                *kubermaticv1.User
	}{
		{
			name:     "scenario 1, change name for service account",
			userInfo: &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			project:  genDefaultProject(),
			saName:   "1",
			existingKubermaticObjects: []runtime.Object{
				createAuthenitactedUser(),
				createSA("test-1", "my-first-project-ID", "viewers", "1"),
				createSA("test-2", "abcd", "viewers", "2"),
				createSA("test-1", "dcba", "viewers", "3"),
			},
			newName: "new-name",
			expectedSA: func() *kubermaticv1.User {
				sa := createSANoPrefix("new-name", "my-first-project-ID", "viewers", "1")
				sa.Kind = "User"
				sa.APIVersion = "kubermatic.k8s.io/v1"
				sa.ResourceVersion = "1"
				return sa
			}(),
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			fakeClient := fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, tc.existingKubermaticObjects...)
			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return fakeClient, nil
			}
			// act
			target := kubernetes.NewServiceAccountProvider(fakeImpersonationClient, fakeClient, "localhost")

			sa, err := target.Get(tc.userInfo, tc.saName, nil)
			if err != nil {
				t.Fatal(err)
			}

			sa.Spec.Name = tc.newName

			expectedSA, err := target.Update(tc.userInfo, sa)
			if err != nil {
				t.Fatal(err)
			}
			if !equality.Semantic.DeepEqual(expectedSA, tc.expectedSA) {
				t.Fatalf("%v", diff.ObjectGoPrintSideBySide(tc.expectedSA, expectedSA))
			}
		})
	}
}

func TestDelete(t *testing.T) {
	// test data
	testcases := []struct {
		name                      string
		existingKubermaticObjects []runtime.Object
		saName                    string
		userInfo                  *provider.UserInfo
	}{
		{
			name:     "scenario 1, delete service account",
			userInfo: &provider.UserInfo{Email: "john@acme.com", Group: "owners-abcd"},
			saName:   "1",
			existingKubermaticObjects: []runtime.Object{
				createAuthenitactedUser(),
				createSA("test-1", "my-first-project-ID", "viewers", "1"),
				createSA("test-2", "abcd", "viewers", "2"),
				createSA("test-1", "dcba", "viewers", "3"),
			},
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			fakeClient := fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, tc.existingKubermaticObjects...)
			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return fakeClient, nil
			}
			// act
			target := kubernetes.NewServiceAccountProvider(fakeImpersonationClient, fakeClient, "localhost")

			err := target.Delete(tc.userInfo, tc.saName)
			if err != nil {
				t.Fatal(err)
			}

			_, err = target.Get(tc.userInfo, tc.saName, nil)
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}
