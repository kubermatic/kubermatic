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

	"k8c.io/kubermatic/v2/pkg/handler/v1/metering"
	"k8c.io/kubermatic/v2/pkg/provider"
	k8cerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

// CreateOrUpdateMeteringCredentials creates or updates metrering tool MeteringSecretReq.
func CreateOrUpdateMeteringCredentials(seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {

		request, ok := req.(metering.MeteringSecretReq)
		if !ok {
			return "", k8cerrors.NewBadRequest("invalid request")
		}

		if err := request.Validate(); err != nil {
			return "", err
		}

		if err := createOrUpdateMeteringCredentials(ctx, request, seedsGetter, seedClientGetter); err != nil {
			return nil, fmt.Errorf("failed to create/update metering MeteringSecretReq: %v", err)
		}

		return nil, nil
	}
}

// CreateOrUpdateMeteringConfigurations configures kkp metering tool.
func CreateOrUpdateMeteringConfigurations(seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {

		request, ok := req.(metering.MeteringConfigurationReq)
		if !ok {
			return "", k8cerrors.NewBadRequest("invalid request")
		}

		if err := request.Validate(); err != nil {
			return "", err
		}

		if err := createOrUpdateMeteringConfigurations(ctx, request, seedsGetter, seedClientGetter); err != nil {
			return nil, fmt.Errorf("failed to create/update metering MeteringSecretReq: %v", err)
		}

		return nil, nil
	}
}

func ListMeteringReportsEndpoint(seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {

		request, ok := req.(metering.ListMeteringReportReq)
		if !ok {
			return "", k8cerrors.NewBadRequest("invalid request")
		}

		exports, err := listMeteringReports(ctx, request, seedsGetter, seedClientGetter)
		if err != nil {
			return nil, err
		}

		return exports, nil
	}
}

func GetMeteringReportEndpoint(seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {

		request, ok := req.(metering.GetMeteringReportReq)
		if !ok {
			return "", k8cerrors.NewBadRequest("invalid request")
		}

		report, err := getMeteringReport(ctx, request, seedsGetter, seedClientGetter)
		if err != nil {
			return nil, err
		}

		return report, nil
	}
}
