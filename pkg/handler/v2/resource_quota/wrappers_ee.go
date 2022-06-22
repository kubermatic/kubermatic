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

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	resourcequotas "k8c.io/kubermatic/v2/pkg/ee/resource-quotas"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func getResourceQuotaForProject(ctx context.Context, request interface{}, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter,
	quotaProvider provider.ResourceQuotaProvider) (*apiv1.ResourceQuota, error) {
	return resourcequotas.GetResourceQuotaForProject(ctx, request, projectProvider, privilegedProjectProvider, userInfoGetter, quotaProvider)
}

func getResourceQuota(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider) (*apiv1.ResourceQuota, error) {
	return resourcequotas.GetResourceQuota(ctx, request, provider)
}

func listResourceQuotas(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider) ([]apiv1.ResourceQuota, error) {
	return resourcequotas.ListResourceQuotas(ctx, request, provider)
}

func createResourceQuota(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider) error {
	return resourcequotas.CreateResourceQuota(ctx, request, provider)
}

func updateResourceQuota(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider) error {
	return resourcequotas.UpdateResourceQuota(ctx, request, provider)
}

func deleteResourceQuota(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider) error {
	return resourcequotas.DeleteResourceQuota(ctx, request, provider)
}

func DecodeResourceQuotasReq(_ context.Context, r *http.Request) (interface{}, error) {
	return resourcequotas.DecodeResourceQuotaReq(r)
}

func DecodeListResourceQuotasReq(_ context.Context, r *http.Request) (interface{}, error) {
	return resourcequotas.DecodeListResourceQuotaReq(r)
}

func DecodeCreateResourceQuotasReq(_ context.Context, r *http.Request) (interface{}, error) {
	return resourcequotas.DecodeCreateResourceQuotaReq(r)
}

func DecodeUpdateResourceQuotasReq(_ context.Context, r *http.Request) (interface{}, error) {
	return resourcequotas.DecodeUpdateResourceQuotaReq(r)
}
