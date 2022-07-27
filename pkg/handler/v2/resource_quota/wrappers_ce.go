//go:build !ee

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
	"k8c.io/kubermatic/v2/pkg/provider"
)

func getResourceQuotaForProject(_ context.Context, _ interface{}, _ provider.ProjectProvider,
	_ provider.PrivilegedProjectProvider, _ provider.UserInfoGetter, _ provider.ResourceQuotaProvider) (*apiv2.ResourceQuota, error) {
	return nil, nil
}
func getResourceQuota(_ context.Context, _ interface{}, _ provider.ResourceQuotaProvider, _ provider.PrivilegedProjectProvider) (*apiv2.ResourceQuota, error) {
	return nil, nil
}

func listResourceQuotas(_ context.Context, _ interface{}, _ provider.ResourceQuotaProvider, _ provider.ProjectProvider) ([]apiv2.ResourceQuota, error) {
	return nil, nil
}

func createResourceQuota(_ context.Context, _ interface{}, _ provider.ResourceQuotaProvider) error {
	return nil
}

func patchResourceQuota(_ context.Context, _ interface{}, _ provider.ResourceQuotaProvider) error {
	return nil
}

func deleteResourceQuota(_ context.Context, _ interface{}, _ provider.ResourceQuotaProvider) error {
	return nil
}

func DecodeResourceQuotasReq(_ context.Context, _ *http.Request) (interface{}, error) {
	return nil, nil
}

func DecodeListResourceQuotasReq(_ context.Context, _ *http.Request) (interface{}, error) {
	return nil, nil
}

func DecodeCreateResourceQuotasReq(_ context.Context, _ *http.Request) (interface{}, error) {
	return nil, nil
}

func DecodePatchResourceQuotasReq(_ context.Context, _ *http.Request) (interface{}, error) {
	return nil, nil
}
