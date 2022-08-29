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

	"gopkg.in/gcfg.v1"

	openstack "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/openstack/types"
	vsphere "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vsphere/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	k8csemverv1 "k8c.io/kubermatic/v2/pkg/semver/v1"
	"k8c.io/kubermatic/v2/pkg/test/diff"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
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
				VirtualCenter: map[string]*vsphere.VirtualCenterConfig{
					"vsphere.com": {
						VCenterPort: "443",
					},
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
				VirtualCenter: map[string]*vsphere.VirtualCenterConfig{
					"vsphere.com": {
						VCenterPort: "9443",
					},
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
				VirtualCenter: map[string]*vsphere.VirtualCenterConfig{
					"vsphere.com": {
						VCenterPort: "9443",
					},
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
			actual := vsphere.CloudConfig{}
			unmarshalINICloudConfig(t, &actual, cloudConfig)

			if !diff.SemanticallyEqual(tc.wantConfig, &actual) {
				t.Fatalf("cloud-config differs from the expected one:\n%v", diff.ObjectDiff(tc.wantConfig, &actual))
			}
		})
	}
}

func TestVSphereCloudConfigClusterID(t *testing.T) {
	testCases := []struct {
		name              string
		cluster           *kubermaticv1.Cluster
		dc                *kubermaticv1.Datacenter
		expectedClusterID string
	}{
		{
			name: "vsphereCSIClusterID feature flag disabled",
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
						Cluster:  "cl-1",
					},
				},
			},
			expectedClusterID: "cl-1",
		},
		{
			name: "vsphereCSIClusterID feature flag enabled",
			cluster: &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster",
				},
				Spec: kubermaticv1.ClusterSpec{

					Cloud: kubermaticv1.CloudSpec{
						VSphere: &kubermaticv1.VSphereCloudSpec{},
					},
					Features: map[string]bool{
						kubermaticv1.ClusterFeatureVsphereCSIClusterID: true,
					},
				},
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					VSphere: &kubermaticv1.DatacenterSpecVSphere{
						Endpoint: "https://vsphere.com",
						Cluster:  "cl-1",
					},
				},
			},
			expectedClusterID: "test-cluster",
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			cloudConfig, err := getVsphereCloudConfig(tc.cluster, tc.dc, resources.Credentials{})
			if err != nil {
				t.Fatalf("Error trying to get cloud-config: %v", err)
			}
			if cloudConfig.Global.ClusterID != tc.expectedClusterID {
				t.Errorf("expected cluster-id %q, but got: %q", tc.expectedClusterID, cloudConfig.Global.ClusterID)
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
					Version: *k8csemverv1.NewSemverOrDie("v1.1.1"),
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							UseOctavia: pointer.BoolPtr(true),
						},
					},
				},
				Status: kubermaticv1.ClusterStatus{
					Versions: kubermaticv1.ClusterVersionsStatus{
						ControlPlane: *k8csemverv1.NewSemverOrDie("v1.1.1"),
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
					Version: *k8csemverv1.NewSemverOrDie("v1.1.1"),
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
				},
				Status: kubermaticv1.ClusterStatus{
					Versions: kubermaticv1.ClusterVersionsStatus{
						ControlPlane: *k8csemverv1.NewSemverOrDie("v1.1.1"),
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
					Version: *k8csemverv1.NewSemverOrDie("v1.1.1"),
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{
							UseOctavia: pointer.BoolPtr(false),
						},
					},
				},
				Status: kubermaticv1.ClusterStatus{
					Versions: kubermaticv1.ClusterVersionsStatus{
						ControlPlane: *k8csemverv1.NewSemverOrDie("v1.1.1"),
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
					Version: *k8csemverv1.NewSemverOrDie("v1.1.1"),
					Cloud: kubermaticv1.CloudSpec{
						Openstack: &kubermaticv1.OpenstackCloudSpec{},
					},
				},
				Status: kubermaticv1.ClusterStatus{
					Versions: kubermaticv1.ClusterVersionsStatus{
						ControlPlane: *k8csemverv1.NewSemverOrDie("v1.1.1"),
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
			actual := openstack.CloudConfig{}
			unmarshalINICloudConfig(t, &actual, cloudConfig)

			if !diff.SemanticallyEqual(tc.wantConfig, &actual) {
				t.Fatalf("cloud-config differs from the expected one:\n%v", diff.ObjectDiff(tc.wantConfig, &actual))
			}
		})
	}
}

func unmarshalINICloudConfig(t *testing.T, config interface{}, rawConfig string) {
	if err := gcfg.ReadStringInto(config, rawConfig); err != nil {
		t.Fatalf("error occurred while marshaling config: %v", err)
	}
}
