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

package machine

import (
	"encoding/json"
	"testing"

	apiv1 "github.com/kubermatic/kubermatic/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	vsphere "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vsphere/types"
	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
)

func TestGetVSphereProviderSpec(t *testing.T) {
	tests := []struct {
		name        string
		cluster     *kubermaticv1.Cluster
		nodeSpec    apiv1.NodeSpec
		dc          *kubermaticv1.Datacenter
		wantRawConf vsphere.RawConfig
		wantErr     bool
	}{
		{
			name: "Datastore",
			nodeSpec: apiv1.NodeSpec{
				Cloud: apiv1.NodeCloudSpec{
					VSphere: &apiv1.VSphereNodeSpec{},
				},
			},
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
						DefaultDatastore: "default-datastore",
					},
				},
			},
			wantRawConf: vsphere.RawConfig{
				Datastore: providerconfigtypes.ConfigVarString{Value: "default-datastore"},
			},
		},
		{
			name: "Default datastore",
			nodeSpec: apiv1.NodeSpec{
				Cloud: apiv1.NodeCloudSpec{
					VSphere: &apiv1.VSphereNodeSpec{},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						VSphere: &kubermaticv1.VSphereCloudSpec{
							Datastore: "my-datastore",
						},
					},
				},
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					VSphere: &kubermaticv1.DatacenterSpecVSphere{
						DefaultDatastore: "default-datastore",
					},
				},
			},
			wantRawConf: vsphere.RawConfig{
				Datastore: providerconfigtypes.ConfigVarString{Value: "my-datastore"},
			},
		},
		{
			name: "Datastore cluster",
			nodeSpec: apiv1.NodeSpec{
				Cloud: apiv1.NodeCloudSpec{
					VSphere: &apiv1.VSphereNodeSpec{},
				},
			},
			cluster: &kubermaticv1.Cluster{
				Spec: kubermaticv1.ClusterSpec{
					Cloud: kubermaticv1.CloudSpec{
						VSphere: &kubermaticv1.VSphereCloudSpec{
							Datastore: "my-datastore-cluster",
						},
					},
				},
			},
			dc: &kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					VSphere: &kubermaticv1.DatacenterSpecVSphere{
						DefaultDatastore: "default-datastore",
					},
				},
			},
			wantRawConf: vsphere.RawConfig{
				Datastore: providerconfigtypes.ConfigVarString{Value: "my-datastore-cluster"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getVSphereProviderSpec(tt.cluster, tt.nodeSpec, tt.dc)
			if (err != nil) != tt.wantErr {
				t.Errorf("getVSphereProviderSpec() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotRawConf := vsphere.RawConfig{}
			err = json.Unmarshal(got.Raw, &gotRawConf)
			if err != nil {
				t.Fatalf("error occurred whil unmarshaling raw config: %v", err)
			}
			if gotRawConf != tt.wantRawConf {
				t.Errorf("getVSphereProviderSpec() = %+v, want %+v", gotRawConf, tt.wantRawConf)
			}
		})
	}
}
