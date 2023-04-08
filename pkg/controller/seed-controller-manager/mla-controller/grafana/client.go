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

package grafana

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	grafanasdk "github.com/kubermatic/grafanasdk"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CredentialSecretUserKey     = "admin-user"
	CredentialSecretPasswordKey = "admin-password"

	// This is Grafana's ID for its default organization.
	// Has to match whatever Grafana uses.
	DefaultOrgID = 1
)

type Client interface {
	CreateOrg(ctx context.Context, org grafanasdk.Org) (grafanasdk.StatusMessage, error)
	GetOrgByOrgName(ctx context.Context, orgName string) (grafanasdk.Org, error)
	WithOrgIDHeader(orgID uint) Client
	SetOrgIDHeader(orgID uint)

	CreateOAuthUser(ctx context.Context, email string) (*grafanasdk.User, error)
	LookupUser(ctx context.Context, loginOrEmail string) (grafanasdk.User, error)
	GetOrgUsers(ctx context.Context, orgID uint) ([]grafanasdk.OrgUser, error)
	AddOrgUser(ctx context.Context, userRole grafanasdk.UserRole, orgID uint) (grafanasdk.StatusMessage, error)
	UpdateOrgUser(ctx context.Context, userRole grafanasdk.UserRole, orgID uint, userID uint) (grafanasdk.StatusMessage, error)
	DeleteOrgUser(ctx context.Context, orgID uint, userID uint) (grafanasdk.StatusMessage, error)
	DeleteGlobalUser(ctx context.Context, userID uint) (grafanasdk.StatusMessage, error)

	CreateDatasource(ctx context.Context, ds grafanasdk.Datasource) (grafanasdk.StatusMessage, error)
	GetDatasourceByName(ctx context.Context, name string) (grafanasdk.Datasource, error)
	GetDatasourceByUID(ctx context.Context, uid string) (grafanasdk.Datasource, error)
	UpdateDatasource(ctx context.Context, ds grafanasdk.Datasource) (grafanasdk.StatusMessage, error)
	DeleteDatasourceByUID(ctx context.Context, uid string) (grafanasdk.StatusMessage, error)

	SetDashboard(ctx context.Context, board grafanasdk.Board, params grafanasdk.SetDashboardParams) (grafanasdk.StatusMessage, error)
	DeleteDashboardByUID(ctx context.Context, uid string) (grafanasdk.StatusMessage, error)
}

// clientWrapper is here because there exists no function in the SDK
// to create OAuth users, but we want this functionality bundled in our single
// grafanaClient interface.
// Also, this wrapper is necessary to do handle the implicit type assertion
// in the WithOrgIDHeader() function.
type clientWrapper struct {
	*grafanasdk.Client

	baseURL    string
	httpClient *http.Client
	authHeader string
}

func (w *clientWrapper) WithOrgIDHeader(orgID uint) Client {
	return &clientWrapper{
		Client:     w.Client.WithOrgIDHeader(orgID),
		baseURL:    w.baseURL,
		httpClient: w.httpClient,
		authHeader: w.authHeader,
	}
}

// ensureGrafanaOAuthUser ensures a Grafana user exists for the given email
// address. Note that creation for OAuth users happens implicitly by
// GET-ing the endpoint for the actual (current) user.
func (r *clientWrapper) CreateOAuthUser(ctx context.Context, email string) (*grafanasdk.User, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, r.baseURL+"/api/user", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(r.authHeader, email)

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	grafanaUser := &grafanasdk.User{}
	defer resp.Body.Close()
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(grafanaUser); err != nil {
		return nil, fmt.Errorf("unable to decode response: %w", err)
	}

	if grafanaUser.ID == 0 {
		return nil, errors.New("unable to create OAuth user")
	}

	return grafanaUser, nil
}

func NewClient(httpClient *http.Client, url, username, password string) (Client, error) {
	basicAuth := fmt.Sprintf("%s:%s", username, password)

	client, err := grafanasdk.NewClient(url, basicAuth, httpClient)
	if err != nil {
		return nil, err
	}

	return &clientWrapper{
		Client:     client,
		baseURL:    url,
		httpClient: httpClient,
		authHeader: "X-Forwarded-Email",
	}, nil
}

type ClientProvider func(ctx context.Context) (Client, error)

func NewClientProvider(client ctrlruntimeclient.Client, httpClient *http.Client, secretName string, grafanaURL string, enabled bool) (ClientProvider, error) {
	split := strings.Split(secretName, "/")
	if n := len(split); n != 2 {
		return nil, fmt.Errorf("splitting value of %q didn't yield two but %d results", secretName, n)
	}

	return func(ctx context.Context) (Client, error) {
		secret := corev1.Secret{}
		if err := client.Get(ctx, types.NamespacedName{Name: split[1], Namespace: split[0]}, &secret); err != nil {
			// TODO: Shouldn't this be the first check in this function?
			if !enabled {
				return nil, nil
			}

			return nil, fmt.Errorf("failed to get Grafana Secret: %w", err)
		}

		adminName, ok := secret.Data[CredentialSecretUserKey]
		if !ok {
			return nil, fmt.Errorf("Grafana Secret %q does not contain %s key", secretName, CredentialSecretUserKey)
		}

		adminPass, ok := secret.Data[CredentialSecretPasswordKey]
		if !ok {
			return nil, fmt.Errorf("Grafana Secret %q does not contain %s key", secretName, CredentialSecretPasswordKey)
		}

		return NewClient(httpClient, grafanaURL, string(adminName), string(adminPass))
	}, nil
}

func IsNotFoundErr(err error) bool {
	return errors.Is(err, grafanasdk.ErrNotFound{})
}
