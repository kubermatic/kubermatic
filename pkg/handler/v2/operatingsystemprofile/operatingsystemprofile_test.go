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

package operatingsystemprofile_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	"k8c.io/kubermatic/v2/pkg/handler/test"
	"k8c.io/kubermatic/v2/pkg/handler/test/hack"
	osmv1alpha1 "k8c.io/operating-system-manager/pkg/crd/osm/v1alpha1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestListOperatingSystemProfiles(t *testing.T) {
	testCases := []struct {
		name                  string
		existingObjects       []ctrlruntimeclient.Object
		apiUser               *apiv1.User
		expectedResult        []*apiv2.OperatingSystemProfile
		expectedHTTPStatus    int
		expectedErrorResponse []byte
	}{
		{
			name: "base case",
			existingObjects: []ctrlruntimeclient.Object{
				&osmv1alpha1.OperatingSystemProfile{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-osp-1",
						Namespace: "kubermatic",
					},
					Spec: osmv1alpha1.OperatingSystemProfileSpec{
						ProvisioningConfig: osmv1alpha1.OSPConfig{
							Units: nil,
						},
					},
				},
				&osmv1alpha1.OperatingSystemProfile{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-osp-2",
						Namespace: "kubermatic",
					},
					Spec: osmv1alpha1.OperatingSystemProfileSpec{
						ProvisioningConfig: osmv1alpha1.OSPConfig{
							Units: nil,
						},
					},
				},
			},
			apiUser:            test.GenDefaultAdminAPIUser(),
			expectedHTTPStatus: http.StatusOK,
			expectedResult: []*apiv2.OperatingSystemProfile{
				{
					Name:      "test-osp-1",
					Namespace: "kubermatic",
				},
				{
					Name:      "test-osp-2",
					Namespace: "kubermatic",
				},
				{
					Name: "osp-amzn2",
				},
				{
					Name: "osp-centos",
				},
				{
					Name: "osp-flatcar",
				},
				{
					Name: "osp-rhel",
				},
				{
					Name: "osp-rockylinux",
				},
				{
					Name: "osp-sles",
				},
				{
					Name: "osp-ubuntu",
				},
			},
		},
		{
			name: "non-admin",
			existingObjects: []ctrlruntimeclient.Object{
				&osmv1alpha1.OperatingSystemProfile{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-osp-1",
						Namespace: "kubermatic",
					},
					Spec: osmv1alpha1.OperatingSystemProfileSpec{
						ProvisioningConfig: osmv1alpha1.OSPConfig{
							Units: nil,
						},
					},
				},
				&osmv1alpha1.OperatingSystemProfile{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-osp-2",
						Namespace: "kubermatic",
					},
					Spec: osmv1alpha1.OperatingSystemProfileSpec{
						ProvisioningConfig: osmv1alpha1.OSPConfig{
							Units: nil,
						},
					},
				},
			},
			apiUser:               test.GenDefaultAPIUser(),
			expectedHTTPStatus:    http.StatusForbidden,
			expectedErrorResponse: []byte("{\"error\":{\"code\":403,\"message\":\"bob@acme.com doesn't have admin rights\"}}\n"),
		},
		{
			name:               "default OSPs",
			existingObjects:    []ctrlruntimeclient.Object{},
			apiUser:            test.GenDefaultAdminAPIUser(),
			expectedHTTPStatus: http.StatusOK,
			expectedResult: []*apiv2.OperatingSystemProfile{
				{
					Name: "osp-amzn2",
				},
				{
					Name: "osp-centos",
				},
				{
					Name: "osp-flatcar",
				},
				{
					Name: "osp-rhel",
				},
				{
					Name: "osp-rockylinux",
				},
				{
					Name: "osp-sles",
				},
				{
					Name: "osp-ubuntu",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.existingObjects = append(tc.existingObjects, test.APIUserToKubermaticUser(*tc.apiUser), test.GenTestSeed())

			req := httptest.NewRequest("GET", "/api/v2/seeds/us-central1/operatingsystemprofiles", strings.NewReader(""))
			res := httptest.NewRecorder()
			ep, err := test.CreateTestEndpoint(*tc.apiUser, nil, tc.existingObjects, nil, hack.NewTestRouting)
			assert.NoError(t, err)

			ep.ServeHTTP(res, req)

			assert.Equal(t, tc.expectedHTTPStatus, res.Code)

			if res.Code == http.StatusOK {
				var osps []*apiv2.OperatingSystemProfile
				err = json.Unmarshal(res.Body.Bytes(), &osps)
				assert.NoError(t, err)

				assert.Equal(t, tc.expectedResult, osps)
			} else {
				assert.Equal(t, tc.expectedErrorResponse, res.Body.Bytes())
			}
		})
	}
}
