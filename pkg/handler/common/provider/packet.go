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
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/packethost/packngo"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
)

// Used to decode response object.
type plansRoot struct {
	Plans []packngo.Plan `json:"plans"`
}

func DescribePacketSize(apiKey, projectID, instanceType string) (packngo.Plan, error) {
	plan := packngo.Plan{}

	if len(apiKey) == 0 {
		return plan, fmt.Errorf("missing required parameter: apiKey")
	}

	if len(projectID) == 0 {
		return plan, fmt.Errorf("missing required parameter: projectID")
	}

	packetclient := packngo.NewClientWithAuth("kubermatic", apiKey, nil)
	req, err := packetclient.NewRequest(http.MethodGet, "/projects/"+projectID+"/plans", nil)
	if err != nil {
		return plan, err
	}
	root := new(plansRoot)

	_, err = packetclient.Do(req, root)
	if err != nil {
		return plan, err
	}

	plans := root.Plans
	for _, currentPlan := range plans {
		if currentPlan.Slug == instanceType {
			return currentPlan, nil
		}
	}
	return plan, fmt.Errorf("packet instanceType:%s not found", instanceType)
}

func filterPacketByQuota(instances apiv1.PacketSizeList, quota kubermaticv1.MachineDeploymentVMResourceQuota) apiv1.PacketSizeList {
	filteredRecords := apiv1.PacketSizeList{}

	// Range over the records and apply all the filters to each record.
	// If the record passes all the filters, add it to the final slice.
	for _, r := range instances {
		keep := true

		memoryGB := strings.TrimSuffix(r.Memory, "GB")
		memory, err := strconv.Atoi(memoryGB)
		if err == nil {
			if !handlercommon.FilterCPU(r.CPUs[0].Count, quota.MinCPU, quota.MaxCPU) {
				keep = false
			}

			if !handlercommon.FilterMemory(memory, quota.MinRAM, quota.MaxRAM) {
				keep = false
			}

			if keep {
				filteredRecords = append(filteredRecords, r)
			}
		}
	}

	return filteredRecords
}

func toPacketSize(plan packngo.Plan) apiv1.PacketSize {
	drives := make([]apiv1.PacketDrive, 0)
	for _, drive := range plan.Specs.Drives {
		drives = append(drives, apiv1.PacketDrive{
			Count: drive.Count,
			Size:  drive.Size,
			Type:  drive.Type,
		})
	}

	memory := "N/A"
	if plan.Specs.Memory != nil {
		memory = plan.Specs.Memory.Total
	}

	cpus := make([]apiv1.PacketCPU, 0)
	for _, cpu := range plan.Specs.Cpus {
		cpus = append(cpus, apiv1.PacketCPU{
			Count: cpu.Count,
			Type:  cpu.Type,
		})
	}

	return apiv1.PacketSize{
		Name:   plan.Name,
		CPUs:   cpus,
		Memory: memory,
		Drives: drives,
	}
}
