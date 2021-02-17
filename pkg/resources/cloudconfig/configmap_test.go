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
	"testing"

	"github.com/go-test/deep"
	"gopkg.in/gcfg.v1"
	"k8s.io/utils/pointer"

	openstack "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/openstack/types"
	vsphere "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vsphere/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"
)

func TestVSphereCloudConfig(t *testing.T) {
	testCases := []struct {
		name       string
		cluster    *kubermaticv1.Cluster
		dc         *kubermaticv1.Datacenter
		wantConfig *vsphere.CloudConfig
	}{
		{
			name: "port gets defaulted to 443",
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
			wantConfig: &vsphere.CloudConfig{
				Global: vsphere.GlobalOpts{
					VCenterPort: "443",
					VCenterIP:   "vsphere.com",
				},
				Disk: vsphere.DiskOpts{
					SCSIControllerType: "pvscsi",
				},
				Workspace: vsphere.WorkspaceOpts{
					VCenterIP: "vsphere.com",
				},
			},
		},
		{
			name: "port from url gets used",
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
			wantConfig: &vsphere.CloudConfig{
				Global: vsphere.GlobalOpts{
					VCenterPort: "9443",
					VCenterIP:   "vsphere.com",
				},
				Disk: vsphere.DiskOpts{
					SCSIControllerType: "pvscsi",
				},
				Workspace: vsphere.WorkspaceOpts{
					VCenterIP: "vsphere.com",
				},
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
			wantConfig: &vsphere.CloudConfig{
				Global: vsphere.GlobalOpts{
					VCenterPort:      "9443",
					VCenterIP:        "vsphere.com",
					DefaultDatastore: "super-cool-datastore",
				},
				Disk: vsphere.DiskOpts{
					SCSIControllerType: "pvscsi",
				},
				Workspace: vsphere.WorkspaceOpts{
					VCenterIP:        "vsphere.com",
					DefaultDatastore: "super-cool-datastore",
				},
			},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			cloudConfig, err := CloudConfig(tc.cluster, tc.dc, resources.Credentials{})
			if err != nil {
				t.Fatalf("Error trying to get cloud-config: %v", err)
			}
			t.Logf("config: %v", cloudConfig)
			actual := vsphere.CloudConfig{}
			unmarshalINICloudConfig(t, &actual, cloudConfig)

			if diff := deep.Equal(&actual, tc.wantConfig); len(diff) > 0 {
				t.Errorf("cloud-config differs from the expected one: %s", diff)
			}
		})
	}
}

func TestOpenStackCloudConfig(t *testing.T) {
	testCases := []struct {
		name       string
		cluster    *kubermaticv1.Cluster
		dc         *kubermaticv1.Datacenter
		wantConfig *openstack.CloudConfig
	}{
		{
			name: "use-octavia enabled at cluster level",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Version: *semver.NewSemverOrDie("v1.1.1"),
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							UseOctavia: pointer.BoolPtr(true),
						},
					},
				},
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					Openstack: &kubermaticv1.DatacenterSpecOpenstack{},
				},
			},
			wantConfig: &openstack.CloudConfig{
				LoadBalancer: openstack.LoadBalancerOpts{
					LBVersion:  "v2",
					LBMethod:   "ROUND_ROBIN",
					UseOctavia: pointer.BoolPtr(true),
				},
				BlockStorage: openstack.BlockStorageOpts{
					BSVersion: "auto",
				},
			},
		},
		{
			name: "use-octavia enabled at seed level",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Version: *semver.NewSemverOrDie("v1.1.1"),
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
				},
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					Openstack: &kubermaticv1.DatacenterSpecOpenstack{
						UseOctavia: pointer.BoolPtr(false),
					},
				},
			},
			wantConfig: &openstack.CloudConfig{
				LoadBalancer: openstack.LoadBalancerOpts{
					LBVersion:  "v2",
					LBMethod:   "ROUND_ROBIN",
					UseOctavia: pointer.BoolPtr(false),
				},
				BlockStorage: openstack.BlockStorageOpts{
					BSVersion: "auto",
				},
			},
		},
		{
			name: "use-octavia not set",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Version: *semver.NewSemverOrDie("v1.1.1"),
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							UseOctavia: pointer.BoolPtr(false),
						},
					},
				},
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					Openstack: &kubermaticv1.DatacenterSpecOpenstack{
						UseOctavia: pointer.BoolPtr(true),
					},
				},
			},
			wantConfig: &openstack.CloudConfig{
				LoadBalancer: openstack.LoadBalancerOpts{
					LBVersion:  "v2",
					LBMethod:   "ROUND_ROBIN",
					UseOctavia: pointer.BoolPtr(false),
				},
				BlockStorage: openstack.BlockStorageOpts{
					BSVersion: "auto",
				},
			},
		},
		{
			name: "use-octavia not set",
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Version: *semver.NewSemverOrDie("v1.1.1"),
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
				},
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					Openstack: &kubermaticv1.DatacenterSpecOpenstack{},
				},
			},
			wantConfig: &openstack.CloudConfig{
				LoadBalancer: openstack.LoadBalancerOpts{
					LBVersion: "v2",
					LBMethod:  "ROUND_ROBIN",
				},
				BlockStorage: openstack.BlockStorageOpts{
					BSVersion: "auto",
				},
			},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			cloudConfig, err := CloudConfig(tc.cluster, tc.dc, resources.Credentials{})
			if err != nil {
				t.Fatalf("Error trying to get cloud-config: %v", err)
			}
			t.Logf("config: %v", cloudConfig)
			actual := openstack.CloudConfig{}
			unmarshalINICloudConfig(t, &actual, cloudConfig)

			if diff := deep.Equal(&actual, tc.wantConfig); len(diff) > 0 {
				t.Errorf("cloud-config differs from the expected one: %s", diff)
			}
		})
	}
}

func unmarshalINICloudConfig(t *testing.T, config interface{}, rawConfig string) {
	if err := gcfg.ReadStringInto(config, rawConfig); err != nil {
		t.Fatalf("error occurred while marshaling config: %v", err)
	}
}
