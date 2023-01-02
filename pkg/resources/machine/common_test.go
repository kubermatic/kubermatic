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
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	anexia "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/anexia"
	anexiatypes "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/anexia/types"
	vsphere "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vsphere/types"
	providerconfigtypes "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	"k8s.io/utils/pointer"
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
				Datastore:     providerconfigtypes.ConfigVarString{Value: "default-datastore"},
				AllowInsecure: providerconfigtypes.ConfigVarBool{Value: pointer.Bool(false)},
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
				Datastore:     providerconfigtypes.ConfigVarString{Value: "my-datastore"},
				AllowInsecure: providerconfigtypes.ConfigVarBool{Value: pointer.Bool(false)},
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
				Datastore:     providerconfigtypes.ConfigVarString{Value: "my-datastore-cluster"},
				AllowInsecure: providerconfigtypes.ConfigVarBool{Value: pointer.Bool(false)},
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
			if !reflect.DeepEqual(gotRawConf, tt.wantRawConf) {
				t.Errorf("getVSphereProviderSpec() = %+v, want %+v", gotRawConf, tt.wantRawConf)
			}
		})
	}
}

func TestGetAnexiaProviderSpec(t *testing.T) {
	const (
		vlanID     = "vlan-identifier"
		templateID = "template-identifier"
		locationID = "location-identifier"
	)

	tests := []struct {
		name           string
		anexiaNodeSpec apiv1.AnexiaNodeSpec
		wantRawConf    *anexiatypes.RawConfig
		wantErr        error
	}{
		{
			name: "Anexia node spec with DiskSize attribute",
			anexiaNodeSpec: apiv1.AnexiaNodeSpec{
				VlanID:     vlanID,
				TemplateID: templateID,
				CPUs:       4,
				Memory:     4096,
				DiskSize:   pointer.Int64(80),
			},
			wantRawConf: &anexiatypes.RawConfig{
				VlanID:     providerconfigtypes.ConfigVarString{Value: vlanID},
				TemplateID: providerconfigtypes.ConfigVarString{Value: templateID},
				LocationID: providerconfigtypes.ConfigVarString{Value: locationID},
				CPUs:       4,
				Memory:     4096,
				DiskSize:   80,
				Disks:      nil,
			},
			wantErr: nil,
		},
		{
			name: "Anexia node spec with Disks attribute",
			anexiaNodeSpec: apiv1.AnexiaNodeSpec{
				VlanID:     vlanID,
				TemplateID: templateID,
				CPUs:       4,
				Memory:     4096,
				Disks: []apiv1.AnexiaDiskConfig{
					{
						Size:            80,
						PerformanceType: pointer.String("ENT2"),
					},
				},
			},
			wantRawConf: &anexiatypes.RawConfig{
				VlanID:     providerconfigtypes.ConfigVarString{Value: vlanID},
				TemplateID: providerconfigtypes.ConfigVarString{Value: templateID},
				LocationID: providerconfigtypes.ConfigVarString{Value: locationID},
				CPUs:       4,
				Memory:     4096,
				DiskSize:   0,
				Disks: []anexiatypes.RawDisk{
					{
						Size:            80,
						PerformanceType: providerconfigtypes.ConfigVarString{Value: "ENT2"},
					},
				},
			},
			wantErr: nil,
		},
		{
			name: "Anexia node spec with both DiskSize and Disks attributes",
			anexiaNodeSpec: apiv1.AnexiaNodeSpec{
				VlanID:     vlanID,
				TemplateID: templateID,
				CPUs:       4,
				Memory:     4096,
				DiskSize:   pointer.Int64(80),
				Disks: []apiv1.AnexiaDiskConfig{
					{
						Size:            80,
						PerformanceType: pointer.String("ENT2"),
					},
				},
			},
			wantRawConf: nil,
			wantErr:     anexia.ErrConfigDiskSizeAndDisks,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dc := kubermaticv1.Datacenter{
				Spec: kubermaticv1.DatacenterSpec{
					Anexia: &kubermaticv1.DatacenterSpecAnexia{
						LocationID: locationID,
					},
				},
			}

			nodeSpec := apiv1.NodeSpec{
				Cloud: apiv1.NodeCloudSpec{
					Anexia: &test.anexiaNodeSpec,
				},
			}

			result, err := getAnexiaProviderSpec(nodeSpec, &dc)

			if test.wantErr != nil {
				assert.Nil(t, result, "expected an error, not a result")
				assert.ErrorIs(t, err, test.wantErr, "expected an error, not a result")
			} else {
				assert.NotNil(t, result)

				resultRawConfig := anexiatypes.RawConfig{}
				err := json.Unmarshal(result.Raw, &resultRawConfig)
				assert.Nil(t, err)

				assert.Equal(t, resultRawConfig.VlanID.Value, vlanID, "VLAN should be set correctly")
				assert.Equal(t, resultRawConfig.TemplateID.Value, templateID, "Template should be set correctly")
				assert.Equal(t, resultRawConfig.LocationID.Value, locationID, "Location should be set correctly")

				assert.EqualValues(t, resultRawConfig.CPUs, test.anexiaNodeSpec.CPUs, "CPUs should be set correctly")
				assert.EqualValues(t, resultRawConfig.Memory, test.anexiaNodeSpec.Memory, "Memory should be set correctly")

				if test.anexiaNodeSpec.DiskSize != nil {
					assert.EqualValues(t, resultRawConfig.DiskSize, *test.anexiaNodeSpec.DiskSize, "DiskSize should be set correctly")
					assert.Nil(t, resultRawConfig.Disks, "Disks attribute should be nil")
				} else {
					assert.EqualValues(t, resultRawConfig.DiskSize, 0, "DiskSize should be set to 0")
					assert.Len(t, resultRawConfig.Disks, len(test.anexiaNodeSpec.Disks), "Disks attribute should have correct length")

					for i, dc := range test.anexiaNodeSpec.Disks {
						assert.EqualValues(t, resultRawConfig.Disks[i].Size, dc.Size, "Disk entry should have correct size")

						if dc.PerformanceType != nil {
							assert.EqualValues(t, resultRawConfig.Disks[i].PerformanceType.Value, *dc.PerformanceType, "Disk entry should have correct performance type")
						} else {
							assert.Empty(t, resultRawConfig.Disks[i].PerformanceType.Value, "Disk entry should have no performance type")
						}
					}
				}
			}
		})
	}
}
