package user

import (
	"context"
	"fmt"
	"github.com/go-kit/kit/endpoint"
	v1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"
	"net/http"
)

func ListEndpoint(userInfoGetter provider.UserInfoGetter, userProvider provider.UserProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		if !userInfo.IsAdmin {
			return nil, errors.New(http.StatusForbidden,
				fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", userInfo.Email))
		}

		list, err := userProvider.List()
		if err != nil {
			return nil, err
		}

		result := make([]v1.User, 0)
		for _, crdUser := range list {
			apiUser := v1.ConvertInternalUserToExternal(&crdUser, false)
			result = append(result, *apiUser)
		}

		return result, nil
	}
}
