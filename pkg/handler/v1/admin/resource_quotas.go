package admin

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"

	"k8c.io/kubermatic/v2/pkg/provider"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GetResourceQuotaEndpoint(userInfoGetter provider.UserInfoGetter, provider provider.ResourceQuotaProvider) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%s doesn't have admin rights", userInfo.Email))
		}

		resp, err := getResourceQuota(ctx, req, provider)
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
}
