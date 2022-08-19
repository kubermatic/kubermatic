/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package admin

import (
	"context"
	"fmt"

	"github.com/go-kit/kit/endpoint"

	"k8c.io/kubermatic/v2/pkg/provider"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateOrUpdateMeteringCredentials creates or updates metering tool SecretReq.
func CreateOrUpdateMeteringCredentials(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%q doesn't have admin rights", userInfo.Email))
		}

		if err := createOrUpdateMeteringCredentials(ctx, req, seedsGetter, seedClientGetter); err != nil {
			return nil, fmt.Errorf("failed to create/update metering credentials: %w", err)
		}

		return nil, nil
	}
}

// CreateOrUpdateMeteringConfigurations configures kkp metering tool.
func CreateOrUpdateMeteringConfigurations(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, masterClient ctrlruntimeclient.Client) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%q doesn't have admin rights", userInfo.Email))
		}

		if err := createOrUpdateMeteringConfigurations(ctx, req, seedsGetter, masterClient); err != nil {
			return nil, fmt.Errorf("failed to create/update metering SecretReq: %w", err)
		}

		return nil, nil
	}
}

// GetMeteringReportConfigurationEndpoint list report configurations for kkp metering tool.
func GetMeteringReportConfigurationEndpoint(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%q doesn't have admin rights", userInfo.Email))
		}

		resp, err := getMeteringReportConfiguration(seedsGetter, req)
		if err != nil {
			return nil, fmt.Errorf("failed to get metering report configuration: %w", err)
		}

		return resp, nil
	}
}

// ListMeteringReportConfigurationsEndpoint list report configurations for kkp metering tool.
func ListMeteringReportConfigurationsEndpoint(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%q doesn't have admin rights", userInfo.Email))
		}

		resp, err := listMeteringReportConfigurations(seedsGetter)
		if err != nil {
			return nil, fmt.Errorf("failed to list metering report configurations: %w", err)
		}

		return resp, nil
	}
}

// CreateMeteringReportConfigurationEndpoint creates report configuration entry for kkp metering tool.
func CreateMeteringReportConfigurationEndpoint(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, masterClient ctrlruntimeclient.Client) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%q doesn't have admin rights", userInfo.Email))
		}

		if err := createMeteringReportConfiguration(ctx, req, seedsGetter, masterClient); err != nil {
			return nil, fmt.Errorf("failed to create metering report configuration: %w", err)
		}

		return nil, nil
	}
}

// UpdateMeteringReportConfigurationEndpoint updates existing report configuration entry for kkp metering tool.
func UpdateMeteringReportConfigurationEndpoint(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, masterClient ctrlruntimeclient.Client) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%q doesn't have admin rights", userInfo.Email))
		}

		if err := updateMeteringReportConfiguration(ctx, req, seedsGetter, masterClient); err != nil {
			return nil, fmt.Errorf("failed to update metering report configuration: %w", err)
		}

		return nil, nil
	}
}

// DeleteMeteringReportConfigurationEndpoint deletes report configuration entry for kkp metering tool.
func DeleteMeteringReportConfigurationEndpoint(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, masterClient ctrlruntimeclient.Client) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%q doesn't have admin rights", userInfo.Email))
		}

		if err := deleteMeteringReportConfiguration(ctx, req, seedsGetter, masterClient); err != nil {
			return nil, fmt.Errorf("failed to delete metering report configuration: %w", err)
		}

		return nil, nil
	}
}

// ListMeteringReportsEndpoint lists available reports.
func ListMeteringReportsEndpoint(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%q doesn't have admin rights", userInfo.Email))
		}

		exports, err := listMeteringReports(ctx, req, seedsGetter, seedClientGetter)
		if err != nil {
			return nil, err
		}

		return exports, nil
	}
}

// GetMeteringReportEndpoint get a presigned url to download specific report.
func GetMeteringReportEndpoint(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%q doesn't have admin rights", userInfo.Email))
		}

		report, err := getMeteringReport(ctx, req, seedsGetter, seedClientGetter)
		if err != nil {
			return nil, err
		}

		return report, nil
	}
}

// DeleteMeteringReportEndpoint removes a specific report.
func DeleteMeteringReportEndpoint(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, apierrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("%q doesn't have admin rights", userInfo.Email))
		}

		err = deleteMeteringReport(ctx, req, seedsGetter, seedClientGetter)
		if err != nil {
			return nil, err
		}

		return nil, nil
	}
}
