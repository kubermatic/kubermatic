package provider

import (
	"context"
	"fmt"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"
	kubermaticcontext "github.com/kubermatic/kubermatic/pkg/util/context"
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
