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
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/ee/metering"
	meteringApi "k8c.io/kubermatic/v2/pkg/handler/v1/metering"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func createOrUpdateMeteringCredentials(ctx context.Context, request meteringApi.SecretReq, seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) error {
	return metering.CreateOrUpdateCredentials(ctx, request, seedsGetter, seedClientGetter)
}

func createOrUpdateMeteringConfigurations(ctx context.Context, request meteringApi.ConfigurationReq, masterClient client.Client) error {
	return metering.CreateOrUpdateConfigurations(ctx, request, masterClient)
}

func listMeteringReports(ctx context.Context, request meteringApi.ListMeteringReportReq, seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) ([]v1.MeteringReport, error) {
	return metering.ListReports(ctx, request, seedsGetter, seedClientGetter)
}

func getMeteringReport(ctx context.Context, request meteringApi.GetMeteringReportReq, seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) (string, error) {
	return metering.GetReport(ctx, request, seedsGetter, seedClientGetter)
}
