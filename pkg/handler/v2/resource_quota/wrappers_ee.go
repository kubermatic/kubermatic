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

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	resourcequotas "k8c.io/kubermatic/v2/pkg/ee/resource-quotas"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func getResourceQuotaForProject(ctx context.Context, request interface{}, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter,
	quotaProvider provider.ResourceQuotaProvider) (*apiv1.ResourceQuota, error) {
	return resourcequotas.GetResourceQuotaForProject(ctx, request, projectProvider, privilegedProjectProvider, userInfoGetter, quotaProvider)
}
