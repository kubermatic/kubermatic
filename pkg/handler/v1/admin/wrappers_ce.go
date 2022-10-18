//go:build !ee

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
	"net/http"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func createOrUpdateMeteringCredentials(_ context.Context, _ interface{}, _ provider.SeedsGetter, _ provider.SeedClientGetter) error {
	return nil
}

func createOrUpdateMeteringConfigurations(_ context.Context, _ interface{}, _ provider.SeedsGetter, masterClient ctrlruntimeclient.Client) error {
	return nil
}

func getMeteringReportConfiguration(_ provider.SeedsGetter, _ interface{}) (*kubermaticv1.MeteringReportConfiguration, error) {
	return nil, nil
}

func listMeteringReportConfigurations(_ provider.SeedsGetter) ([]apiv1.MeteringReportConfiguration, error) {
	return nil, nil
}

func createMeteringReportConfiguration(_ context.Context, _ interface{}, _ provider.SeedsGetter, _ ctrlruntimeclient.Client) (*apiv1.MeteringReportConfiguration, error) {
	return nil, nil
}

func updateMeteringReportConfiguration(_ context.Context, _ interface{}, _ provider.SeedsGetter, _ ctrlruntimeclient.Client) (*apiv1.MeteringReportConfiguration, error) {
	return nil, nil
}

func deleteMeteringReportConfiguration(_ context.Context, _ interface{}, _ provider.SeedsGetter, _ ctrlruntimeclient.Client) error {
	return nil
}

func listMeteringReports(_ context.Context, _ interface{}, _ provider.SeedsGetter, _ provider.SeedClientGetter) ([]apiv1.MeteringReport, error) {
	return nil, nil
}

func getMeteringReport(_ context.Context, _ interface{}, _ provider.SeedsGetter, _ provider.SeedClientGetter) (string, error) {
	return "", nil
}

func deleteMeteringReport(_ context.Context, _ interface{}, _ provider.SeedsGetter, _ provider.SeedClientGetter) error {
	return nil
}

func DecodeGetMeteringReportConfigurationReq(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func DecodeCreateMeteringReportConfigurationReq(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func DecodeUpdateMeteringReportConfigurationReq(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func DecodeDeleteMeteringReportConfigurationReq(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func DecodeMeteringSecretReq(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func DecodeListMeteringReportReq(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func DecodeGetMeteringReportReq(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func DecodeMeteringConfigurationsReq(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func DecodeDeleteMeteringReportReq(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}
