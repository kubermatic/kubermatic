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
	"fmt"

	"github.com/go-kit/kit/endpoint"

	"k8c.io/kubermatic/v2/pkg/provider"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetForProjectEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	userInfoGetter provider.UserInfoGetter, quotaProvider provider.ResourceQuotaProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return getResourceQuotaForProject(ctx, request, projectProvider, privilegedProjectProvider, userInfoGetter, quotaProvider)
	}
}

func GetResourceQuotaEndpoint(userInfoGetter provider.UserInfoGetter, provider provider.ResourceQuotaProvider, projectProvider provider.PrivilegedProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%s doesn't have admin rights", userInfo.Email))
		}

		resp, err := getResourceQuota(ctx, req, provider, projectProvider)
		if err != nil {
			return nil, err
		}

		return resp, nil
	}
}

func ListResourceQuotasEndpoint(userInfoGetter provider.UserInfoGetter, provider provider.ResourceQuotaProvider, projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%s doesn't have admin rights", userInfo.Email))
		}

		resp, err := listResourceQuotas(ctx, req, provider, projectProvider)
		if err != nil {
			return nil, err
		}

		return resp, nil
	}
}

func CreateResourceQuotaEndpoint(userInfoGetter provider.UserInfoGetter, provider provider.ResourceQuotaProvider) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%s doesn't have admin rights", userInfo.Email))
		}

		err = createResourceQuota(ctx, req, provider)
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
}

func PatchResourceQuotaEndpoint(userInfoGetter provider.UserInfoGetter, provider provider.ResourceQuotaProvider) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%s doesn't have admin rights", userInfo.Email))
		}

		err = patchResourceQuota(ctx, req, provider)
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
}

func DeleteResourceQuotaEndpoint(userInfoGetter provider.UserInfoGetter, provider provider.ResourceQuotaProvider) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%s doesn't have admin rights", userInfo.Email))
		}

		err = deleteResourceQuota(ctx, req, provider)
		if err != nil {
			return nil, err
		}
		return nil, nil
	}
}
