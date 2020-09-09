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

	"github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
)

func TestListConstraintTemplates(t *testing.T) {
	testCases := []struct {
		name           string
		existingCTs    []v1beta1.ConstraintTemplate
		expectedCTList []v1beta1.ConstraintTemplate
	}{
		{
			name:           "test: list constraint templates",
			existingCTs:    []v1beta1.ConstraintTemplate{genConstraintTemplate("ct1")},
			expectedCTList: []v1beta1.ConstraintTemplate{genConstraintTemplate("ct1")},
		},
	}

	if err := v1beta1.AddToSchemes.AddToScheme(scheme.Scheme); err != nil {
		t.Fatal(err)
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			kubermaticObjects := []runtime.Object{}
			for _, ct := range tc.existingCTs {
				kubermaticObjects = append(kubermaticObjects, &ct)
			}

			client := fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, kubermaticObjects...)
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
			if !reflect.DeepEqual(ctList.Items, tc.expectedCTList) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(ctList.Items, tc.expectedCTList))
			}
		})
	}
}
