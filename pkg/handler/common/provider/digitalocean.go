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
	"regexp"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	doprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/digitalocean"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

var reStandard = regexp.MustCompile("(^s|S)")
var reOptimized = regexp.MustCompile("(^c|C)")

func DigitaloceanSizeWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, settingsProvider provider.SettingsProvider, projectID, clusterID string) (interface{}, error) {

	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}
	if cluster.Spec.Cloud.Digitalocean == nil {
		return nil, errors.NewNotFound("cloud spec for ", clusterID)
	}

	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	accessToken, err := doprovider.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
	if err != nil {
		return nil, err
	}

	settings, err := settingsProvider.GetGlobalSettings()
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return DigitaloceanSize(ctx, settings.Spec.MachineDeploymentVMResourceQuota, accessToken)

}

func DigitaloceanSize(ctx context.Context, quota kubermaticv1.MachineDeploymentVMResourceQuota, token string) (apiv1.DigitaloceanSizeList, error) {
	static := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := godo.NewClient(oauth2.NewClient(context.Background(), static))

	listOptions := &godo.ListOptions{
		Page:    1,
		PerPage: 1000,
	}

	sizes, _, err := client.Sizes.List(ctx, listOptions)
	if err != nil {
		return apiv1.DigitaloceanSizeList{}, fmt.Errorf("failed to list sizes: %v", err)
	}

	sizeList := apiv1.DigitaloceanSizeList{}
	// currently there are 3 types of sizes: 1) starting with s, 2) starting with c and 3) the old ones
	// type 3 isn't listed in the pricing anymore and only will be available for legacy issues until July 1st, 2018
	// therefore we might not want to log all cases that aren't starting with s or c
	for k := range sizes {
		s := apiv1.DigitaloceanSize{
			Slug:         sizes[k].Slug,
			Available:    sizes[k].Available,
			Transfer:     sizes[k].Transfer,
			PriceMonthly: sizes[k].PriceMonthly,
			PriceHourly:  sizes[k].PriceHourly,
			Memory:       sizes[k].Memory,
			VCPUs:        sizes[k].Vcpus,
			Disk:         sizes[k].Disk,
			Regions:      sizes[k].Regions,
		}
		switch {
		case reStandard.MatchString(sizes[k].Slug):
			sizeList.Standard = append(sizeList.Standard, s)
		case reOptimized.MatchString(sizes[k].Slug):
			sizeList.Optimized = append(sizeList.Optimized, s)
		}
	}

	return filterDigitalOceanByQuota(sizeList, quota), nil
}

func filterDigitalOceanByQuota(instances apiv1.DigitaloceanSizeList, quota kubermaticv1.MachineDeploymentVMResourceQuota) apiv1.DigitaloceanSizeList {
	filteredRecords := apiv1.DigitaloceanSizeList{}

	for _, r := range instances.Optimized {
		keep := true

		if !handlercommon.FilterCPU(r.VCPUs, quota.MinCPU, quota.MaxCPU) {
			keep = false
		}
		if !handlercommon.FilterMemory(r.Memory/1024, quota.MinRAM, quota.MaxRAM) {
			keep = false
		}

		if keep {
			filteredRecords.Optimized = append(filteredRecords.Optimized, r)
		}
	}
	for _, r := range instances.Standard {
		keep := true

		if !handlercommon.FilterCPU(r.VCPUs, quota.MinCPU, quota.MaxCPU) {
			keep = false
		}
		if !handlercommon.FilterMemory(r.Memory/1024, quota.MinRAM, quota.MaxRAM) {
			keep = false
		}

		if keep {
			filteredRecords.Standard = append(filteredRecords.Standard, r)
		}
	}

	return filteredRecords
}
