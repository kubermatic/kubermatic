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

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gcp"
)

func GetGCPInstanceSize(ctx context.Context, machineType, sa, zone string) (*apiv1.GCPMachineSize, error) {
	computeService, project, err := gcp.ConnectToComputeService(ctx, sa)
	if err != nil {
		return nil, err
	}

	req := computeService.MachineTypes.Get(project, zone, machineType)
	m, err := req.Do()
	if err != nil {
		return nil, fmt.Errorf("error getting GCP Instance Size: %w", err)
	}

	return &apiv1.GCPMachineSize{
		Memory: m.MemoryMb,
		VCPUs:  m.GuestCpus,
	}, nil
}
