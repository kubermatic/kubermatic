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

package vsphere

import (
	"context"
	"strings"
	"testing"

	"github.com/vmware/govmomi/simulator"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/machine-controller/sdk/providerconfig"
)

const (
	fakeStoragePolicy = "fake_storage_policy"
)

func TestGetCredentialsForCluster(t *testing.T) {
	tcs := []struct {
		name              string
		cloudspec         kubermaticv1.CloudSpec
		secretKeySelector provider.SecretKeySelectorValueFunc
		dc                *kubermaticv1.DatacenterSpecVSphere
		expectedUser      string
		expectedPassword  string
		expectedError     string
	}{
		{
			name:             "User from cluster",
			cloudspec:        testVsphereCloudSpec("user", "pass", "", "", true),
			expectedUser:     "user",
			expectedPassword: "pass",
		},
		{
			name:      "User from secret",
			cloudspec: testVsphereCloudSpec("", "", "", "", true),
			secretKeySelector: testSecretKeySelectorValueFuncFactory(map[string]string{
				resources.VsphereUsername: "user",
				resources.VspherePassword: "pass",
			}),
			expectedUser:     "user",
			expectedPassword: "pass",
		},
		{
			name:             "InfraManagementUser from clusters InfraManagementUser field",
			cloudspec:        testVsphereCloudSpec("a", "b", "infraManagementUser", "infraManagementUserPass", true),
			expectedUser:     "infraManagementUser",
			expectedPassword: "infraManagementUserPass",
		},
		{
			name:             "InfraManagementUser from clusters user field",
			cloudspec:        testVsphereCloudSpec("user", "pass", "", "", false),
			expectedUser:     "user",
			expectedPassword: "pass",
		},
		{
			name:      "InfraManagementUser from secrets InfraManagementUser field",
			cloudspec: testVsphereCloudSpec("", "", "", "", true),
			secretKeySelector: testSecretKeySelectorValueFuncFactory(map[string]string{
				resources.VsphereInfraManagementUserUsername: "user",
				resources.VsphereInfraManagementUserPassword: "pass",
			}),
			expectedUser:     "user",
			expectedPassword: "pass",
		},
		{
			name:      "InfraManagementUser from secrets User field",
			cloudspec: testVsphereCloudSpec("", "", "", "", true),
			secretKeySelector: testSecretKeySelectorValueFuncFactory(map[string]string{
				resources.VsphereUsername: "user",
				resources.VspherePassword: "pass",
			}),
			expectedUser:     "user",
			expectedPassword: "pass",
		},
		{
			name:      "InfraManagementUser from DC takes precedence",
			cloudspec: testVsphereCloudSpec("", "", "", "", true),
			secretKeySelector: testSecretKeySelectorValueFuncFactory(map[string]string{
				resources.VsphereUsername: "user",
				resources.VspherePassword: "pass",
			}),
			dc: &kubermaticv1.DatacenterSpecVSphere{
				InfraManagementUser: &kubermaticv1.VSphereCredentials{
					Username: "dc-user",
					Password: "dc-pass",
				},
			},
			expectedUser:     "dc-user",
			expectedPassword: "dc-pass",
		},
		{
			name:      "InfraManagementUser from DC takes precedence over InfraManagementUser from cluster",
			cloudspec: testVsphereCloudSpec("", "", "cluster-infra-user", "cluster-infra-pass", true),
			dc: &kubermaticv1.DatacenterSpecVSphere{
				InfraManagementUser: &kubermaticv1.VSphereCredentials{
					Username: "dc-user",
					Password: "dc-pass",
				},
			},
			expectedUser:     "dc-user",
			expectedPassword: "dc-pass",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("panic in test %q: %v", tc.name, r)
				}
			}()
			user, password, err := getCredentialsForCluster(tc.cloudspec, tc.secretKeySelector, tc.dc)
			if (tc.expectedError == "" && err != nil) || (tc.expectedError != "" && err == nil) {
				t.Fatalf("Expected error %q, got error %v", tc.expectedError, err)
			}
			if user != tc.expectedUser {
				t.Errorf("expected user %q, got user %q", tc.expectedUser, user)
			}
			if password != tc.expectedPassword {
				t.Errorf("expected password %q, got password %q", tc.expectedPassword, password)
			}
		})
	}
}

func testSecretKeySelectorValueFuncFactory(values map[string]string) provider.SecretKeySelectorValueFunc {
	return func(_ *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
		if val, ok := values[key]; ok {
			return val, nil
		}
		return "", nil
	}
}

func testVsphereCloudSpec(user, password, infraManagementUser, infraManagementUserPass string, credRefSet bool) kubermaticv1.CloudSpec {
	var credRef *providerconfig.GlobalSecretKeySelector
	if credRefSet {
		credRef = &providerconfig.GlobalSecretKeySelector{}
	}
	return kubermaticv1.CloudSpec{
		VSphere: &kubermaticv1.VSphereCloudSpec{
			CredentialsReference: credRef,
			Username:             user,
			Password:             password,
			InfraManagementUser: kubermaticv1.VSphereCredentials{
				Username: infraManagementUser,
				Password: infraManagementUserPass,
			},
		},
	}
}

func TestProviderValidateCloudSpec(t *testing.T) {
	tests := []struct {
		name    string
		dc      *kubermaticv1.DatacenterSpecVSphere
		spec    kubermaticv1.CloudSpec
		wantErr bool
	}{
		{
			name: "No datastore at Datacenter level nor datastore or datastore cluster at cluster level",
			dc:   &kubermaticv1.DatacenterSpecVSphere{},
			spec: kubermaticv1.CloudSpec{
				VSphere: &kubermaticv1.VSphereCloudSpec{},
			},
			wantErr: true,
		},
		{
			name: "No datastore at Datacenter level but datastore at cluster level",
			dc:   &kubermaticv1.DatacenterSpecVSphere{DefaultStoragePolicy: fakeStoragePolicy},
			spec: kubermaticv1.CloudSpec{
				VSphere: &kubermaticv1.VSphereCloudSpec{
					Datastore:     "LocalDS_0",
					StoragePolicy: fakeStoragePolicy,
				},
			},
		},
		{
			name: "No datastore at Datacenter level but datastore cluster at cluster level",
			dc:   &kubermaticv1.DatacenterSpecVSphere{DefaultStoragePolicy: fakeStoragePolicy},
			spec: kubermaticv1.CloudSpec{
				VSphere: &kubermaticv1.VSphereCloudSpec{
					DatastoreCluster: "DC0_POD0",
				},
			},
		},
		{
			name: "Both datastore and datastore cluster at cluster level",
			dc:   &kubermaticv1.DatacenterSpecVSphere{},
			spec: kubermaticv1.CloudSpec{
				VSphere: &kubermaticv1.VSphereCloudSpec{
					Datastore:        "LocalDS_0",
					DatastoreCluster: "DC0_POD0",
				},
			},
			wantErr: true,
		},
		{
			name: "Default datastore at datacenter level",
			dc: &kubermaticv1.DatacenterSpecVSphere{
				DefaultDatastore:     "LocalDS_0",
				DefaultStoragePolicy: fakeStoragePolicy,
			},
			spec: kubermaticv1.CloudSpec{
				VSphere: &kubermaticv1.VSphereCloudSpec{},
			},
		},
		{
			name: "Non existing default datastore at datacenter level",
			dc: &kubermaticv1.DatacenterSpecVSphere{
				DefaultDatastore: "whao",
			},
			spec: kubermaticv1.CloudSpec{
				VSphere: &kubermaticv1.VSphereCloudSpec{},
			},
			wantErr: true,
		},
		{
			name: "Default datastore at datacenter overridden at cluster level",
			dc: &kubermaticv1.DatacenterSpecVSphere{
				DefaultDatastore:     "LocalDS_0",
				DefaultStoragePolicy: fakeStoragePolicy,
			},
			spec: kubermaticv1.CloudSpec{
				VSphere: &kubermaticv1.VSphereCloudSpec{
					Datastore: "LocalDS_0",
				},
			},
		},
		{
			name: "Default datastore at datacenter level overridden at cluster level by non existing datastore",
			dc: &kubermaticv1.DatacenterSpecVSphere{
				DefaultDatastore: "LocalDS_0",
			},
			spec: kubermaticv1.CloudSpec{
				VSphere: &kubermaticv1.VSphereCloudSpec{
					Datastore: "i-do-not-exist",
				},
			},
			wantErr: true,
		},
		{
			name: "Default datastore at datacenter and datastore cluster at cluster level",
			dc: &kubermaticv1.DatacenterSpecVSphere{
				DefaultDatastore:     "LocalDS_0",
				DefaultStoragePolicy: fakeStoragePolicy,
			},
			spec: kubermaticv1.CloudSpec{
				VSphere: &kubermaticv1.VSphereCloudSpec{
					DatastoreCluster: "DC0_POD0",
				},
			},
		},
		{
			name: "Inaccessible default datastore at datacenter level and datastore cluster at cluster level",
			dc: &kubermaticv1.DatacenterSpecVSphere{
				DefaultDatastore: "i-am-inaccessible",
			},
			spec: kubermaticv1.CloudSpec{
				VSphere: &kubermaticv1.VSphereCloudSpec{
					DatastoreCluster: "DC0_POD0",
				},
			},
		},
		{
			name: "Default datastore at datacenter level overridden at cluster level by non existing Datastore",
			dc: &kubermaticv1.DatacenterSpecVSphere{
				DefaultDatastore: "LocalDS_0",
			},
			spec: kubermaticv1.CloudSpec{
				VSphere: &kubermaticv1.VSphereCloudSpec{
					DatastoreCluster: "i-do-not--exist",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := vSphereSimulator{t: t}
			sim.setUp()
			defer sim.tearDown()
			sim.fillClientInfo(tt.dc)
			v := &VSphere{
				dc: tt.dc,
			}
			if err := v.ValidateCloudSpec(context.Background(), tt.spec); (err != nil) != tt.wantErr {
				t.Errorf("VSphere.ValidateCloudSpec() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// The following resources are made available:
// * Datastore named: LocalDS_0
// * Datastore cluster named: DC0_POD0.
type vSphereSimulator struct {
	t      *testing.T
	model  *simulator.Model
	server *simulator.Server
}

func (v *vSphereSimulator) setUp() {
	v.model = simulator.VPX()
	// Pod == StoragePod == DatastoreCluster
	v.model.Pod++
	v.model.Cluster++

	err := v.model.Create()
	if err != nil {
		v.t.Fatal(err)
	}

	v.server = v.model.Service.NewServer()
}

func (v *vSphereSimulator) tearDown() {
	v.model.Remove()
	v.server.Close()
}

func (v *vSphereSimulator) fillClientInfo(dc *kubermaticv1.DatacenterSpecVSphere) {
	dc.Endpoint = strings.TrimSuffix(v.server.URL.String(), "/sdk")
	dc.InfraManagementUser = &kubermaticv1.VSphereCredentials{
		Username: simulator.DefaultLogin.Username(),
	}
	dc.InfraManagementUser.Password, _ = simulator.DefaultLogin.Password()
	dc.Datacenter = "DC0"
}
