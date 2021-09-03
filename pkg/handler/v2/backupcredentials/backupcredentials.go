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

package backupcredentials

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

const (
	accessKeyID                = "ACCESS_KEY_ID"
	secretAccessKey            = "SECRET_ACCESS_KEY"
	s3credentialsName          = "s3-credentials"
	backupCredentialsNamespace = "kube-system"
)

func CreateOrUpdateEndpoint(userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createOrUpdateBackupCredentialsReq)

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !userInfo.IsAdmin {
			return userInfo, errors.New(http.StatusForbidden, "Only admins are allowed to create backup credentials")
		}

		backupCredentialsProvider, ok := ctx.Value(middleware.BackupCredentialsProviderContextKey).(provider.BackupCredentialsProvider)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "can't find backup credentials provider")
		}

		bc := convertAPIToInternalBackupCredentials(&req.Body.BackupCredentials)

		// Update if already exists
		_, err = backupCredentialsProvider.GetUnsecured()
		if err != nil {
			_, err = backupCredentialsProvider.CreateUnsecured(bc)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
		} else {
			_, err = backupCredentialsProvider.UpdateUnsecured(bc)
			if err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
		}

		return nil, nil
	}
}

// createOrUpdateBackupCredentialsReq represents a request for creating or updating backup credentials
// swagger:parameters createOrUpdateBackupCredentials
type createOrUpdateBackupCredentialsReq struct {
	// in: path
	// required: true
	SeedName string `json:"seed_name"`
	// in: body
	Body bcBody
}

func (r createOrUpdateBackupCredentialsReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		SeedName: r.SeedName,
	}
}

type bcBody struct {
	// BackupCredentials credentials for backups
	BackupCredentials apiv2.BackupCredentials `json:"backup_credentials"`
}

func DecodeBackupCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createOrUpdateBackupCredentialsReq
	seedName := mux.Vars(r)["seed_name"]
	if seedName == "" {
		return "", fmt.Errorf("'seed_name' parameter is required but was not provided")
	}
	req.SeedName = seedName

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}
	return req, nil
}

func convertAPIToInternalBackupCredentials(bc *apiv2.BackupCredentials) *v1.Secret {
	return &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s3credentialsName,
			Namespace: backupCredentialsNamespace,
		},
		StringData: map[string]string{
			accessKeyID:     bc.S3BackupCredentials.AccessKeyID,
			secretAccessKey: bc.S3BackupCredentials.SecretAccessKey,
		},
	}
}
