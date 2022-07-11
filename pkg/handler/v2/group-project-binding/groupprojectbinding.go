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

package groupprojectbinding

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	"k8c.io/kubermatic/v2/pkg/provider"
)

func ListGroupProjectBindingsEndpoint(
	userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	bindingProvider provider.GroupProjectBindingProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		return listGroupProjectBindings(
			ctx,
			req,
			userInfoGetter,
			projectProvider,
			privilegedProjectProvider,
			bindingProvider,
		)
	}
}

func GetGroupProjectBindingEndpoint(
	userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	bindingProvider provider.GroupProjectBindingProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		return getGroupProjectBinding(
			ctx,
			req,
			userInfoGetter,
			projectProvider,
			privilegedProjectProvider,
			bindingProvider,
		)
	}
}

func CreateGroupProjectBindingEndpoint(
	userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	bindingProvider provider.GroupProjectBindingProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		if err := createGroupProjectBinding(
			ctx,
			req,
			userInfoGetter,
			projectProvider,
			privilegedProjectProvider,
			bindingProvider,
		); err != nil {
			return nil, err
		}
		return nil, nil
	}
}

func DeleteGroupProjectBindingEndpoint(
	userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	bindingProvider provider.GroupProjectBindingProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		if err := deleteGroupProjectBinding(
			ctx,
			req,
			userInfoGetter,
			projectProvider,
			privilegedProjectProvider,
			bindingProvider,
		); err != nil {
			return nil, err
		}
		return nil, nil
	}
}
