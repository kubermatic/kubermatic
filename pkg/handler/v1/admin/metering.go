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
	"strconv"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	"k8c.io/kubermatic/v2/pkg/provider"
	k8cerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

// swagger:parameters listMeteringReports
type ListMeteringReportReq struct {
	// required: false
	// in: query
	StartAfter string `json:"start_after"`
	// required: false
	// in: query
	MaxKeys int `json:"max_keys"`
}

// swagger:parameters getMeteringReport
type GetMeteringReportReq struct {
	// in: path
	// required: true
	ReportName string `json:"report_name"`
}

func ListMeteringReportsEndpoint(seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) endpoint.Endpoint {
	return func(ctx context.Context, req interface{}) (interface{}, error) {

		request, ok := req.(ListMeteringReportReq)
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

		request, ok := req.(GetMeteringReportReq)
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

func DecodeListMeteringReportReq(ctx context.Context, r *http.Request) (interface{}, error) {
	var req ListMeteringReportReq

	maxKeys := r.URL.Query().Get("max_keys")

	if maxKeys == "" {
		req.MaxKeys = 1000
	} else {
		mK, err := strconv.Atoi(maxKeys)
		if err != nil {
			return nil, k8cerrors.NewBadRequest("invalid value for `may_keys`")
		}
		req.MaxKeys = mK
	}

	req.StartAfter = r.URL.Query().Get("start_after")

	return req, nil
}

func DecodeGetMeteringReportReq(ctx context.Context, r *http.Request) (interface{}, error) {
	var req GetMeteringReportReq
	req.ReportName = mux.Vars(r)["report_name"]

	return req, nil
}
