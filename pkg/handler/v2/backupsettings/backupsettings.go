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

package backupsettings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	"k8c.io/kubermatic/v2/pkg/util/errors"

	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
)

func UpdateEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(updateBackupSettingsReq)

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return nil, errors.New(http.StatusForbidden, fmt.Sprintf("Only admins are allowed to update backup settings"))
		}

		// get if exists
		ebc, err := convertAPIToInternalEtcdBackupConfig(req.Body.Name, &req.Body.Spec, c)
		if err != nil {
			return nil, err
		}

		ebc, err = createEtcdBackupConfig(ctx, userInfoGetter, req.ProjectID, ebc)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertInternalToAPIEtcdBackupConfig(ebc), nil
	}
}

// updateBackupSettingsReq represents a request for updating the backup settings
// swagger:parameters updateBackupSettings
type updateBackupSettingsReq struct {
	// in: body
	BackupSettings apiv2.BackupSettings
}

func DecodeUpdateBackupSettingsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req updateBackupSettingsReq

	if err := json.NewDecoder(r.Body).Decode(&req.BackupSettings); err != nil {
		return nil, err
	}
	return req, nil
}
