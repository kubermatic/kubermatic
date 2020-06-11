package cloudconfig

import (
	"fmt"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/pkg/resources"
	vsphere "github.com/kubermatic/machine-controller/pkg/cloudprovider/provider/vsphere/types"
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
