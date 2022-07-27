//go:build ee

/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package resourcequota

import (
	"context"
	"net/http"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	resourcequota "k8c.io/kubermatic/v2/pkg/ee/resource-quota"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func getResourceQuotaForProject(ctx context.Context, request interface{}, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter,
	quotaProvider provider.ResourceQuotaProvider) (*apiv2.ResourceQuota, error) {
	return resourcequota.GetResourceQuotaForProject(ctx, request, projectProvider, privilegedProjectProvider, userInfoGetter, quotaProvider)
}

func getResourceQuota(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider, projectProvider provider.PrivilegedProjectProvider) (*apiv2.ResourceQuota, error) {
	return resourcequota.GetResourceQuota(ctx, request, provider, projectProvider)
}

func listResourceQuotas(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider, projectProvider provider.ProjectProvider) ([]*apiv2.ResourceQuota, error) {
	return resourcequota.ListResourceQuotas(ctx, request, provider, projectProvider)
}

func createResourceQuota(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider) error {
	return resourcequota.CreateResourceQuota(ctx, request, provider)
}

func patchResourceQuota(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider) error {
	return resourcequota.PatchResourceQuota(ctx, request, provider)
}

func deleteResourceQuota(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider) error {
	return resourcequota.DeleteResourceQuota(ctx, request, provider)
}

func DecodeResourceQuotasReq(_ context.Context, r *http.Request) (interface{}, error) {
	return resourcequota.DecodeResourceQuotaReq(r)
}

func DecodeListResourceQuotasReq(_ context.Context, r *http.Request) (interface{}, error) {
	return resourcequota.DecodeListResourceQuotaReq(r)
}

func DecodeCreateResourceQuotasReq(_ context.Context, r *http.Request) (interface{}, error) {
	return resourcequota.DecodeCreateResourceQuotaReq(r)
}

func DecodePatchResourceQuotasReq(_ context.Context, r *http.Request) (interface{}, error) {
	return resourcequota.DecodePatchResourceQuotaReq(r)
}
