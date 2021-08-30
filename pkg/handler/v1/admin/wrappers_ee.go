// +build ee

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
	"k8c.io/kubermatic/v2/pkg/ee/metering"
	"k8c.io/kubermatic/v2/pkg/provider"
	"net/http"
)

func createOrUpdateMeteringCredentials(ctx context.Context, request interface{}, seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) error {
	return metering.CreateOrUpdateCredentials(ctx, request, seedsGetter, seedClientGetter)
}

func DecodeMeteringReq(_ context.Context, r *http.Request) (interface{}, error) {
	return metering.DecodeMeteringToolCredentials(r)
}

func createOrUpdateMeteringConfigurations(ctx context.Context, request interface{}, seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) error {
	return metering.CreateOrUpdateConfigurations(ctx, request, seedsGetter, seedClientGetter)
}

func DecodeMeteringConfigurationsReq(_ context.Context, r *http.Request) (interface{}, error) {
	return metering.DecodeMeteringToolConfigurations(r)
}
