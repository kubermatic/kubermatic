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

func ListResourceQuotasEndpoint(userInfoGetter provider.UserInfoGetter, provider provider.ResourceQuotaProvider) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%s doesn't have admin rights", userInfo.Email))
		}

		resp, err := listResourceQuotas(ctx, req, provider)
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

func UpdateResourceQuotaEndpoint(userInfoGetter provider.UserInfoGetter, provider provider.ResourceQuotaProvider) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%s doesn't have admin rights", userInfo.Email))
		}

		err = updateResourceQuota(ctx, req, provider)
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
