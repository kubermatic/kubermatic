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
	"net/http"
	"strconv"
	"strings"

	"github.com/packethost/packngo"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/packet"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

// Used to decode response object
type plansRoot struct {
	Plans []packngo.Plan `json:"plans"`
}

func PacketSizesWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, settingsProvider provider.SettingsProvider, projectID, clusterID string) (interface{}, error) {

	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.Packet == nil {
		return nil, errors.NewNotFound("cloud spec for ", clusterID)
	}

	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "clusterprovider is not a kubernetesprovider.Clusterprovider")
	}
	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	apiKey, projectID, err := packet.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}

	settings, err := settingsProvider.GetGlobalSettings()
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return PacketSizes(apiKey, projectID, settings.Spec.MachineDeploymentVMResourceQuota)

}

func PacketSizes(apiKey, projectID string, quota kubermaticv1.MachineDeploymentVMResourceQuota) (apiv1.PacketSizeList, error) {
	sizes := apiv1.PacketSizeList{}
	root := new(plansRoot)

	if len(apiKey) == 0 {
		return sizes, fmt.Errorf("missing required parameter: apiKey")
	}

	if len(projectID) == 0 {
		return sizes, fmt.Errorf("missing required parameter: projectID")
	}

	client := packngo.NewClientWithAuth("kubermatic", apiKey, nil)
	req, err := client.NewRequest("GET", "/projects/"+projectID+"/plans", nil)
	if err != nil {
		return sizes, err
	}

	_, err = client.Do(req, root)
	if err != nil {
		return sizes, err
	}

	plans := root.Plans
	for _, plan := range plans {
		sizes = append(sizes, toPacketSize(plan))
	}

	return filterPacketByQuota(sizes, quota), nil
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
