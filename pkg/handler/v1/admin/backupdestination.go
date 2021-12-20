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
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	"k8c.io/kubermatic/v2/pkg/provider"
	k8cerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeleteBackupDestinationEndpoint deletes a backup destination from a seed
func DeleteBackupDestinationEndpoint(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, masterClient client.Client) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(backupDestinationReq)
		if !ok {
			return nil, k8cerrors.NewBadRequest("invalid request")
		}
		seed, err := getSeed(ctx, req.seedReq, userInfoGetter, seedsGetter)
		if err != nil {
			return nil, err
		}

		if seed.Spec.EtcdBackupRestore == nil {
			return nil, k8cerrors.New(http.StatusNotFound, fmt.Sprintf("backup destination %q does not exist for seed %q", req.BackupDestination, req.Name))
		}

		_, ok = seed.Spec.EtcdBackupRestore.Destinations[req.BackupDestination]
		if !ok {
			return nil, k8cerrors.New(http.StatusNotFound, fmt.Sprintf("backup destination %q does not exist for seed %q", req.BackupDestination, req.Name))
		}

		delete(seed.Spec.EtcdBackupRestore.Destinations, req.BackupDestination)

		if err := masterClient.Update(ctx, seed); err != nil {
			return nil, fmt.Errorf("failed to update seed: %v", err)
		}

		return nil, nil
	}
}

// backupDestinationReq defines HTTP request for backupDestination
// swagger:parameters deleteBackupDestination
type backupDestinationReq struct {
	seedReq
	// in: path
	// required: true
	BackupDestination string `json:"backup_destination"`
}

func DecodeBackupDestinationReq(c context.Context, r *http.Request) (interface{}, error) {
	var req backupDestinationReq

	s, err := DecodeSeedReq(c, r)
	if err != nil {
		return nil, err
	}
	req.seedReq = s.(seedReq)

	bdest := mux.Vars(r)["backup_destination"]
	if bdest == "" {
		return nil, fmt.Errorf("'backup_destination' parameter is required but was not provided")
	}
	req.BackupDestination = bdest

	return req, nil
}
