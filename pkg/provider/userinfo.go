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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticcontext "k8c.io/kubermatic/v2/pkg/util/context"

	"k8s.io/apimachinery/pkg/util/sets"
)

// UserInfoGetter is a function to retrieve a UserInfo.
type UserInfoGetter = func(ctx context.Context, projectID string) (*UserInfo, error)

func UserInfoGetterFactory(userProjectMapper ProjectMemberMapper) (UserInfoGetter, error) {
	return func(ctx context.Context, projectID string) (*UserInfo, error) {
		user, ok := ctx.Value(kubermaticcontext.UserCRContextKey).(*kubermaticv1.User)
		if !ok {
			// This happens if middleware.UserSaver is not enabled.
			return nil, fmt.Errorf("unable to get authenticated user object")
		}

		groups := sets.NewString()
		roles := sets.NewString()

		if projectID != "" {
			var err error
			groups, err = userProjectMapper.MapUserToGroups(ctx, user, projectID)
			if err != nil {
				return nil, err
			}

			roles, err = userProjectMapper.MapUserToRoles(ctx, user, projectID)
			if err != nil {
				return nil, err
			}
		} else {
			for _, group := range user.Spec.Groups {
				groupName := group
				if projectID != "" {
					groupName += "-" + projectID
				}
				groups.Insert(groupName)
			}
		}

		return &UserInfo{Email: user.Spec.Email, Groups: groups.List(), IsAdmin: user.Spec.IsAdmin, Roles: roles}, nil
	}, nil
}
