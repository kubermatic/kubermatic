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

package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
)

var NewAzureClientSet = func(subscriptionID, clientID, clientSecret, tenantID string) (AzureClientSet, error) {
	cred, err := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	if err != nil {
		return nil, err
	}

	sizesClient, err := armcompute.NewVirtualMachineSizesClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	return &azureClientSetImpl{
		vmSizeClient: sizesClient,
	}, nil
}

type azureClientSetImpl struct {
	vmSizeClient *armcompute.VirtualMachineSizesClient
}

type AzureClientSet interface {
	ListVMSize(ctx context.Context, location string) ([]armcompute.VirtualMachineSize, error)
}

func (s *azureClientSetImpl) ListVMSize(ctx context.Context, location string) ([]armcompute.VirtualMachineSize, error) {
	pager := s.vmSizeClient.NewListPager(location, nil)

	result := []armcompute.VirtualMachineSize{}
	for pager.More() {
		nextResult, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list sizes: %w", err)
		}

		for i := range nextResult.Value {
			result = append(result, *nextResult.Value[i])
		}
	}

	return result, nil
}

func GetAzureVMSize(ctx context.Context, subscriptionID, clientID, clientSecret, tenantID, location, vmName string) (*apiv1.AzureSize, error) {
	sizesClient, err := NewAzureClientSet(subscriptionID, clientID, clientSecret, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorizer for size client: %w", err)
	}

	// get all available VM size types for given location
	listVMSize, err := sizesClient.ListVMSize(ctx, location)
	if err != nil {
		return nil, fmt.Errorf("failed to list sizes: %w", err)
	}

	for _, vm := range listVMSize {
		if strings.EqualFold(*vm.Name, vmName) {
			return &apiv1.AzureSize{
				NumberOfCores:        *vm.NumberOfCores,
				ResourceDiskSizeInMB: *vm.ResourceDiskSizeInMB,
				MemoryInMB:           *vm.MemoryInMB,
			}, nil
		}
	}

	return nil, fmt.Errorf("could not find Azure VM Size named %q", vmName)
}
