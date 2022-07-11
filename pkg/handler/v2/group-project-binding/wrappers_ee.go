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

package groupprojectbinding

import (
	"context"
	"net/http"

	groupprojectbinding "k8c.io/kubermatic/v2/pkg/ee/group-project-binding/handler"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func listGroupProjectBindings(
	ctx context.Context, req interface{},
	userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	bindingProvider provider.GroupProjectBindingProvider,
) (interface{}, error) {
	return groupprojectbinding.ListGroupProjectBindings(ctx, req, userInfoGetter, projectProvider, privilegedProjectProvider, bindingProvider)
}

func DecodeGetGroupProjectBindingReq(c context.Context, r *http.Request) (interface{}, error) {
	return groupprojectbinding.DecodeGetGroupProjectBindingReq(c, r)
}

func getGroupProjectBinding(
	ctx context.Context, req interface{},
	userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	bindingProvider provider.GroupProjectBindingProvider,
) (interface{}, error) {
	return groupprojectbinding.GetGroupProjectBinding(ctx, req, userInfoGetter, projectProvider, privilegedProjectProvider, bindingProvider)
}

func DecodeCreateGroupProjectBindingReq(c context.Context, r *http.Request) (interface{}, error) {
	return groupprojectbinding.DecodeCreateGroupProjectBindingReq(c, r)
}

func createGroupProjectBinding(
	ctx context.Context, req interface{},
	userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	bindingProvider provider.GroupProjectBindingProvider,
) error {
	return groupprojectbinding.CreateGroupProjectBinding(ctx, req, userInfoGetter, projectProvider, privilegedProjectProvider, bindingProvider)
}

func DecodeDeleteGroupProjectBindingReq(c context.Context, r *http.Request) (interface{}, error) {
	return groupprojectbinding.DecodeDeleteGroupProjectBindingReq(c, r)
}

func deleteGroupProjectBinding(
	ctx context.Context, req interface{},
	userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	bindingProvider provider.GroupProjectBindingProvider,
) error {
	return groupprojectbinding.DeleteGroupProjectBinding(ctx, req, userInfoGetter, projectProvider, privilegedProjectProvider, bindingProvider)
}

func DecodePatchGroupProjectBindingReq(c context.Context, r *http.Request) (interface{}, error) {
	return groupprojectbinding.DecodePatchGroupProjectBindingReq(c, r)
}

func patchGroupProjectBinding(
	ctx context.Context, req interface{},
	userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	bindingProvider provider.GroupProjectBindingProvider,
) error {
	return groupprojectbinding.PatchGroupProjectBinding(ctx, req, userInfoGetter, projectProvider, privilegedProjectProvider, bindingProvider)
}
