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

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticcontext "github.com/kubermatic/kubermatic/api/pkg/util/context"
)

// UserInfoGetter is a function to retrieve a UserInfo
type UserInfoGetter = func(ctx context.Context, projectID string) (*UserInfo, error)

func UserInfoGetterFactory(userProjectMapper ProjectMemberMapper) (UserInfoGetter, error) {

	return func(ctx context.Context, projectID string) (*UserInfo, error) {
		user, ok := ctx.Value(kubermaticcontext.UserCRContextKey).(*kubermaticapiv1.User)
		if !ok {
			// This happens if middleware.UserSaver is not enabled.
			return nil, fmt.Errorf("unable to get authenticated user object")
		}

		var group string
		if projectID != "" {
			var err error
			group, err = userProjectMapper.MapUserToGroup(user.Spec.Email, projectID)
			if err != nil {
				return nil, err
			}
		}

		return &UserInfo{Email: user.Spec.Email, Group: group, IsAdmin: user.Spec.IsAdmin}, nil
	}, nil
}
