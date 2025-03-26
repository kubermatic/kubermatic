/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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
	"strings"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/diff"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCloudConfigToString(t *testing.T) {
	tests := []struct {
		name     string
		config   *CloudConfig
		expected string
	}{
		{
			name: "simple-config",
			config: &CloudConfig{
				Global: GlobalOpts{
					User:         "admin",
					Password:     "password",
					InsecureFlag: true,
				},
				Workspace: WorkspaceOpts{
					VCenterIP:        "https://127.0.0.1:8443",
					ResourcePoolPath: "/some-resource-pool",
					DefaultDatastore: "Datastore",
					Folder:           "some-folder",
					Datacenter:       "Datacenter",
				},
				Disk: DiskOpts{
					SCSIControllerType: "pvscsi",
				},
			},
			expected: strings.TrimSpace(`
[Global]
user = "admin"
password = "password"
port = ""
insecure-flag = true
working-dir = ""
datacenter = ""
datastore = ""
server = ""
cluster-id = ""

[Disk]
scsicontrollertype = "pvscsi"

[Workspace]
server = "https://127.0.0.1:8443"
datacenter = "Datacenter"
folder = "some-folder"
default-datastore = "Datastore"
resourcepool-path = "/some-resource-pool"
`),
		},
		{
			name: "2-virtual-centers",
			config: &CloudConfig{
				Global: GlobalOpts{
					User:         "admin",
					Password:     "password",
					InsecureFlag: true,
				},
				Workspace: WorkspaceOpts{
					VCenterIP:        "https://127.0.0.1:8443",
					ResourcePoolPath: "/some-resource-pool",
					DefaultDatastore: "Datastore",
					Folder:           "some-folder",
					Datacenter:       "Datacenter",
				},
				Disk: DiskOpts{
					SCSIControllerType: "pvscsi",
				},
				VirtualCenter: map[string]VirtualCenterConfig{
					"vc1": {
						User:        "1-some-user",
						Password:    "1-some-password",
						VCenterPort: "443",
						Datacenters: "1-foo",
					},
					"vc2": {
						User:        "2-some-user",
						Password:    `foo"bar`,
						VCenterPort: "443",
						Datacenters: "2-foo",
					},
				},
			},
			expected: strings.TrimSpace(`
[Global]
user = "admin"
password = "password"
port = ""
insecure-flag = true
working-dir = ""
datacenter = ""
datastore = ""
server = ""
cluster-id = ""

[Disk]
scsicontrollertype = "pvscsi"

[Workspace]
server = "https://127.0.0.1:8443"
datacenter = "Datacenter"
folder = "some-folder"
default-datastore = "Datastore"
resourcepool-path = "/some-resource-pool"

[VirtualCenter "vc1"]
user = "1-some-user"
password = "1-some-password"
port = "443"
datacenters = "1-foo"

[VirtualCenter "vc2"]
user = "2-some-user"
password = "foo\"bar"
port = "443"
datacenters = "2-foo"
`),
		},
		{
			name: "3-dual-stack",
			config: &CloudConfig{
				Global: GlobalOpts{
					User:         "admin",
					Password:     "password",
					InsecureFlag: true,
					IPFamily:     "ipv4,ipv6",
				},
				Workspace: WorkspaceOpts{
					VCenterIP:        "https://127.0.0.1:8443",
					ResourcePoolPath: "/some-resource-pool",
					DefaultDatastore: "Datastore",
					Folder:           "some-folder",
					Datacenter:       "Datacenter",
				},
				Disk: DiskOpts{
					SCSIControllerType: "pvscsi",
				},
				VirtualCenter: map[string]VirtualCenterConfig{
					"vc1": {
						User:        "1-some-user",
						Password:    "1-some-password",
						VCenterPort: "443",
						Datacenters: "1-foo",
						IPFamily:    "ipv4,ipv6",
					},
				},
			},
			expected: strings.TrimSpace(`
[Global]
user = "admin"
password = "password"
port = ""
insecure-flag = true
working-dir = ""
datacenter = ""
datastore = ""
server = ""
cluster-id = ""
ip-family = "ipv4,ipv6"

[Disk]
scsicontrollertype = "pvscsi"

[Workspace]
server = "https://127.0.0.1:8443"
datacenter = "Datacenter"
folder = "some-folder"
default-datastore = "Datastore"
resourcepool-path = "/some-resource-pool"

[VirtualCenter "vc1"]
user = "1-some-user"
password = "1-some-password"
port = "443"
datacenters = "1-foo"
ip-family = "ipv4,ipv6"
`),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, err := test.config.String()
			if err != nil {
				t.Fatalf("failed to convert to string: %v", err)
			}
			s = strings.TrimSpace(s)
			if changes := diff.StringDiff(test.expected, s); changes != "" {
				t.Fatalf("output is not as expected:\n\n%s", changes)
			}
		})
	}
}

func TestCloudConfig(t *testing.T) {
	testCases := []struct {
		name       string
		cluster    *kubermaticv1.Cluster
		dc         *kubermaticv1.Datacenter
		wantConfig *CloudConfig
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
			wantConfig: &CloudConfig{
				Global: GlobalOpts{
					VCenterPort: "443",
					VCenterIP:   "vsphere.com",
				},
				Disk: DiskOpts{
					SCSIControllerType: "pvscsi",
				},
				Workspace: WorkspaceOpts{
					VCenterIP: "vsphere.com",
				},
				VirtualCenter: map[string]VirtualCenterConfig{
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
			wantConfig: &CloudConfig{
				Global: GlobalOpts{
					VCenterPort: "9443",
					VCenterIP:   "vsphere.com",
				},
				Disk: DiskOpts{
					SCSIControllerType: "pvscsi",
				},
				Workspace: WorkspaceOpts{
					VCenterIP: "vsphere.com",
				},
				VirtualCenter: map[string]VirtualCenterConfig{
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
			wantConfig: &CloudConfig{
				Global: GlobalOpts{
					VCenterPort:      "9443",
					VCenterIP:        "vsphere.com",
					DefaultDatastore: "super-cool-datastore",
				},
				Disk: DiskOpts{
					SCSIControllerType: "pvscsi",
				},
				Workspace: WorkspaceOpts{
					VCenterIP:        "vsphere.com",
					DefaultDatastore: "super-cool-datastore",
				},
				VirtualCenter: map[string]VirtualCenterConfig{
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
			cc, err := ForCluster(tc.cluster, tc.dc, resources.Credentials{})
			if err != nil {
				t.Fatalf("Error trying to get cloud-config: %v", err)
			}

			if !diff.SemanticallyEqual(tc.wantConfig, cc) {
				t.Fatalf("cloud-config differs from the expected one:\n%v", diff.ObjectDiff(tc.wantConfig, cc))
			}
		})
	}
}

func TestClusterID(t *testing.T) {
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
			cloudConfig, err := ForCluster(tc.cluster, tc.dc, resources.Credentials{})
			if err != nil {
				t.Fatalf("Error trying to get cloud-config: %v", err)
			}
			if cloudConfig.Global.ClusterID != tc.expectedClusterID {
				t.Errorf("expected cluster-id %q, but got: %q", tc.expectedClusterID, cloudConfig.Global.ClusterID)
			}
		})
	}
}
