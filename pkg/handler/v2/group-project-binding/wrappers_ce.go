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

package groupprojectbinding

import (
	"context"
	"net/http"

	"k8c.io/kubermatic/v2/pkg/provider"
)

func listGroupProjectBindings(
	_ context.Context,
	_ interface{},
	_ provider.UserInfoGetter,
	_ provider.ProjectProvider,
	_ provider.PrivilegedProjectProvider,
	_ provider.GroupProjectBindingProvider,
) (interface{}, error) {
	return nil, nil
}

func DecodeGetGroupProjectBindingReq(_ context.Context, _ *http.Request) (interface{}, error) {
	return nil, nil
}

func getGroupProjectBinding(
	_ context.Context,
	_ interface{},
	_ provider.UserInfoGetter,
	_ provider.ProjectProvider,
	_ provider.PrivilegedProjectProvider,
	_ provider.GroupProjectBindingProvider,
) (interface{}, error) {
	return nil, nil
}

func DecodeCreateGroupProjectBindingReq(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func createGroupProjectBinding(
	_ context.Context,
	_ interface{},
	_ provider.UserInfoGetter,
	_ provider.ProjectProvider,
	_ provider.PrivilegedProjectProvider,
	_ provider.GroupProjectBindingProvider,
) (interface{}, error) {
	return nil, nil
}

func DecodeDeleteGroupProjectBindingReq(_ context.Context, _ *http.Request) (interface{}, error) {
	return nil, nil
}

func deleteGroupProjectBinding(
	_ context.Context,
	_ interface{},
	_ provider.UserInfoGetter,
	_ provider.ProjectProvider,
	_ provider.PrivilegedProjectProvider,
	_ provider.GroupProjectBindingProvider,
) error {
	return nil
}

func DecodePatchGroupProjectBindingReq(_ context.Context, _ *http.Request) (interface{}, error) {
	return nil, nil
}

func patchGroupProjectBinding(
	_ context.Context,
	_ interface{},
	_ provider.UserInfoGetter,
	_ provider.ProjectProvider,
	_ provider.PrivilegedProjectProvider,
	_ provider.GroupProjectBindingProvider,
) (interface{}, error) {
	return nil, nil
}
