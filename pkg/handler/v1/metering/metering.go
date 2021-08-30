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

package metering

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"k8s.io/apimachinery/pkg/api/resource"

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

// swagger:parameters createOrUpdateMeteringConfigurations
type ConfigurationReq struct {
	// in: body
	Enabled bool `json:"enabled"`
	// in: body
	StorageClassName string `json:"storageClassName"`
	// in: body
	StorageSize string `json:"storageSize"`
}

func (m ConfigurationReq) Validate() error {
	if m.Enabled {
		if m.StorageClassName == "" || m.StorageSize == "" {
			return errors.New("storageClassName or storageSize cannot be empty when the metering tool is enabled")
		}

		if _, err := resource.ParseQuantity(m.StorageSize); err != nil {
			return fmt.Errorf("inapproperiate storageClass size: %v", err)
		}
	}

	return nil
}

func DecodeMeteringConfigurationsReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req ConfigurationReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

// SecretReq contains the s3 secrets to access s3 bucket.
// swagger:parameters createOrUpdateMeteringCredentials
type SecretReq struct {
	// in: body
	// required: true
	BucketName string `json:"bucketName"`
	// in: body
	// required: true
	AccessKey string `json:"accessKey"`
	// in: body
	// required: true
	SecretKey string `json:"secretKey"`
	// in: body
	// required: true
	Endpoint string `json:"endpoint"`
}

func (c SecretReq) Validate() error {
	if c.Endpoint == "" || c.AccessKey == "" || c.SecretKey == "" || c.BucketName == "" {
		return fmt.Errorf("accessKey, secretKey, bucketName or endpoint cannot be empty")
	}

	return nil
}

func DecodeMeteringSecretReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req SecretReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}
