/*
Copyright 2019 The Machine Controller Authors.

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
	"fmt"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/types"
)

const (
	ethCardType = "vmxnet3"
)

// Based on https://github.com/kubernetes-sigs/cluster-api-provider-vsphere/blob/master/pkg/cloud/vsphere/services/govmomi/vcenter/clone.go#L158
func GetNetworkSpecs(ctx context.Context, session *Session, devices object.VirtualDeviceList, network string) ([]types.BaseVirtualDeviceConfigSpec, error) {
	var deviceSpecs []types.BaseVirtualDeviceConfigSpec

	// Remove any existing NICs.
	for _, dev := range devices.SelectByType((*types.VirtualEthernetCard)(nil)) {
		deviceSpecs = append(deviceSpecs, &types.VirtualDeviceConfigSpec{
			Device:    dev,
			Operation: types.VirtualDeviceConfigSpecOperationRemove,
		})
	}

	// Add new NICs based on the machine config.
	ref, err := session.Finder.Network(ctx, network)
	if err != nil {
		return nil, fmt.Errorf("failed to find network %q: %v", network, err)
	}
	backing, err := ref.EthernetCardBackingInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create new ethernet card backing info for network %q: %v", network, err)
	}
	dev, err := object.EthernetCardTypes().CreateEthernetCard(ethCardType, backing)
	if err != nil {
		return nil, fmt.Errorf("failed to create new ethernet card %q for network %q: %v", ethCardType, network, ctx)
	}

	// Get the actual NIC object. This is safe to assert without a check
	// because "object.EthernetCardTypes().CreateEthernetCard" returns a
	// "types.BaseVirtualEthernetCard" as a "types.BaseVirtualDevice".
	nic := dev.(types.BaseVirtualEthernetCard).GetVirtualEthernetCard()

	// Assign a temporary device key to ensure that a unique one will be
	// generated when the device is created.
	nic.Key = devices.NewKey()

	deviceSpecs = append(deviceSpecs, &types.VirtualDeviceConfigSpec{
		Device:    dev,
		Operation: types.VirtualDeviceConfigSpecOperationAdd,
	})

	return deviceSpecs, nil
}
