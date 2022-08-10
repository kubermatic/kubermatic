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
	"fmt"
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
						OSName: osmv1alpha1.OperatingSystemUbuntu,
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
						OSName: osmv1alpha1.OperatingSystemUbuntu,
						ProvisioningConfig: osmv1alpha1.OSPConfig{
							Units: nil,
						},
					},
				},
			},
			apiUser:            test.GenDefaultAPIUser(),
			expectedHTTPStatus: http.StatusOK,
			expectedResult: []*apiv2.OperatingSystemProfile{
				{
					Name:            "test-osp-1",
					OperatingSystem: "ubuntu",
				},
				{
					Name:            "test-osp-2",
					OperatingSystem: "ubuntu",
				},
				{
					Name:                    "osp-amzn2",
					OperatingSystem:         "amzn2",
					SupportedCloudProviders: []string{"aws"},
				},
				{
					Name:                    "osp-centos",
					OperatingSystem:         "centos",
					SupportedCloudProviders: []string{"alibaba", "aws", "azure", "digitalocean", "equinixmetal", "hetzner", "kubevirt", "nutanix", "openstack", "vsphere"},
				},
				{
					Name:                    "osp-flatcar",
					OperatingSystem:         "flatcar",
					SupportedCloudProviders: []string{"aws", "azure", "equinixmetal", "kubevirt", "openstack", "vsphere"},
				},
				{
					Name:                    "osp-rhel",
					OperatingSystem:         "rhel",
					SupportedCloudProviders: []string{"aws", "azure", "equinixmetal", "kubevirt", "openstack", "vsphere"},
				},
				{
					Name:                    "osp-rockylinux",
					OperatingSystem:         "rockylinux",
					SupportedCloudProviders: []string{"aws", "azure", "digitalocean", "equinixmetal", "hetzner", "kubevirt", "openstack", "vsphere"},
				},
				{
					Name:                    "osp-sles",
					OperatingSystem:         "sles",
					SupportedCloudProviders: []string{"aws"},
				},
				{
					Name:                    "osp-ubuntu",
					OperatingSystem:         "ubuntu",
					SupportedCloudProviders: []string{"alibaba", "aws", "azure", "digitalocean", "equinixmetal", "gce", "hetzner", "kubevirt", "nutanix", "openstack", "vmware-cloud-director", "vsphere"},
				},
			},
		},
		{
			name:               "default OSPs",
			existingObjects:    []ctrlruntimeclient.Object{},
			apiUser:            test.GenDefaultAPIUser(),
			expectedHTTPStatus: http.StatusOK,
			expectedResult: []*apiv2.OperatingSystemProfile{
				{
					Name:                    "osp-amzn2",
					OperatingSystem:         "amzn2",
					SupportedCloudProviders: []string{"aws"},
				},
				{
					Name:                    "osp-centos",
					OperatingSystem:         "centos",
					SupportedCloudProviders: []string{"alibaba", "aws", "azure", "digitalocean", "equinixmetal", "hetzner", "kubevirt", "nutanix", "openstack", "vsphere"},
				},
				{
					Name:                    "osp-flatcar",
					OperatingSystem:         "flatcar",
					SupportedCloudProviders: []string{"aws", "azure", "equinixmetal", "kubevirt", "openstack", "vsphere"},
				},
				{
					Name:                    "osp-rhel",
					OperatingSystem:         "rhel",
					SupportedCloudProviders: []string{"aws", "azure", "equinixmetal", "kubevirt", "openstack", "vsphere"},
				},
				{
					Name:                    "osp-rockylinux",
					OperatingSystem:         "rockylinux",
					SupportedCloudProviders: []string{"aws", "azure", "digitalocean", "equinixmetal", "hetzner", "kubevirt", "openstack", "vsphere"},
				},
				{
					Name:                    "osp-sles",
					OperatingSystem:         "sles",
					SupportedCloudProviders: []string{"aws"},
				},
				{
					Name:                    "osp-ubuntu",
					OperatingSystem:         "ubuntu",
					SupportedCloudProviders: []string{"alibaba", "aws", "azure", "digitalocean", "equinixmetal", "gce", "hetzner", "kubevirt", "nutanix", "openstack", "vmware-cloud-director", "vsphere"},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tc.existingObjects = append(tc.existingObjects, test.APIUserToKubermaticUser(*tc.apiUser), test.GenTestSeed())

			req := httptest.NewRequest(http.MethodGet, "/api/v2/seeds/us-central1/operatingsystemprofiles", strings.NewReader(""))
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

func TestListOperatingSystemProfilesEndpointForCluster(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name                  string
		ProjectID             string
		ClusterID             string
		existingObjects       []ctrlruntimeclient.Object
		apiUser               *apiv1.User
		expectedResult        []*apiv2.OperatingSystemProfile
		expectedHTTPStatus    int
		expectedErrorResponse []byte
	}{
		{
			name:      "should list OSPs",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			existingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
				genOperatingSystemProfile("test-osp-1"),
				genOperatingSystemProfile("test-osp-2"),
			),
			apiUser:            test.GenDefaultAPIUser(),
			expectedHTTPStatus: http.StatusOK,
			expectedResult: []*apiv2.OperatingSystemProfile{
				{
					Name:                    "test-osp-1",
					OperatingSystem:         "ubuntu",
					SupportedCloudProviders: []string{"aws"},
				},
				{
					Name:                    "test-osp-2",
					OperatingSystem:         "ubuntu",
					SupportedCloudProviders: []string{"aws"},
				},
			},
		},
		{
			name:      "should return an empty response",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			existingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			apiUser:            test.GenDefaultAPIUser(),
			expectedHTTPStatus: http.StatusOK,
			expectedResult:     []*apiv2.OperatingSystemProfile{},
		},
		{
			name:      "John cannot list Bob's OSP",
			ProjectID: test.GenDefaultProject().Name,
			ClusterID: test.GenDefaultCluster().Name,
			existingObjects: test.GenDefaultKubermaticObjects(
				test.GenTestSeed(),
				test.GenDefaultCluster(),
			),
			apiUser:            test.GenAPIUser("John", "john@acme.com"),
			expectedHTTPStatus: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			requestURL := fmt.Sprintf("/api/v2/projects/%s/clusters/%s/operatingsystemprofiles", tc.ProjectID, tc.ClusterID)
			req := httptest.NewRequest(http.MethodGet, requestURL, strings.NewReader(""))
			res := httptest.NewRecorder()

			ep, err := test.CreateTestEndpoint(*tc.apiUser, nil, tc.existingObjects, nil, hack.NewTestRouting)
			assert.NoError(t, err)

			ep.ServeHTTP(res, req)

			if res.Code != tc.expectedHTTPStatus {
				t.Errorf("Expected HTTP status code %d, got %d: %s", tc.expectedHTTPStatus, res.Code, res.Body.String())
				return
			}

			if res.Code == http.StatusForbidden {
				return
			}

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

func genOperatingSystemProfile(name string) *osmv1alpha1.OperatingSystemProfile {
	return &osmv1alpha1.OperatingSystemProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: test.GenDefaultCluster().Status.NamespaceName,
		},
		Spec: osmv1alpha1.OperatingSystemProfileSpec{
			OSName: osmv1alpha1.OperatingSystemUbuntu,
			SupportedCloudProviders: []osmv1alpha1.CloudProviderSpec{
				{
					Name: "aws",
				},
			},
			ProvisioningConfig: osmv1alpha1.OSPConfig{
				Units: nil,
			},
		},
	}
}
