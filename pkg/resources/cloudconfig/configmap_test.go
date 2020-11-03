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

package cloudconfig

import (
	"fmt"
	"testing"

	vsphere "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vsphere/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
)

func TestGetVsphereCloudConfig(t *testing.T) {
	testCases := []struct {
		name    string
		cluster *kubermaticv1.Cluster
		dc      *kubermaticv1.Datacenter
		verify  func(*vsphere.CloudConfig) error
	}{
		{
			name: "Vsphere port gets defaulted to 443",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						VSphere: &kubermaticv1.VSphereCloudSpec{},
					},
				},
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					VSphere: &kubermaticv1.DatacenterSpecVSphere{
						Endpoint: "https://vsphere.com",
					},
				},
			},
			verify: func(cc *vsphere.CloudConfig) error {
				if cc.Global.VCenterPort != "443" {
					return fmt.Errorf("Expected port to be 443, was %q", cc.Global.VCenterPort)
				}
				return nil
			},
		},
		{
			name: "Vsphere port from url gets used",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						VSphere: &kubermaticv1.VSphereCloudSpec{},
					},
				},
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					VSphere: &kubermaticv1.DatacenterSpecVSphere{
						Endpoint: "https://vsphere.com:9443",
					},
				},
			},
			verify: func(cc *vsphere.CloudConfig) error {
				if cc.Global.VCenterPort != "9443" {
					return fmt.Errorf("Expected port to be 9443, was %q", cc.Global.VCenterPort)
				}
				return nil
			},
		},
		{
			name: "Datastore overridden at cluster level",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						VSphere: &kubermaticv1.VSphereCloudSpec{
							Datastore: "super-cool-datastore",
						},
					},
				},
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					VSphere: &kubermaticv1.DatacenterSpecVSphere{
						Endpoint:         "https://vsphere.com:9443",
						DefaultDatastore: "less-cool-datastore",
					},
				},
			},
			verify: func(cc *vsphere.CloudConfig) error {
				if cc.Workspace.DefaultDatastore != "super-cool-datastore" {
					return fmt.Errorf("Expected default-datastore to be %q, was %q", "super-cool-datastore",
						cc.Workspace.DefaultDatastore)
				}
				return nil
			},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			cloudConfig, err := getVsphereCloudConfig(tc.cluster, tc.dc, resources.Credentials{})
			if err != nil {
				t.Fatalf("Error trying to get cloudconfig: %v", err)
			}
			if err := tc.verify(cloudConfig); err != nil {
				t.Error(err)
			}
		})
	}
}
