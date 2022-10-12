/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package common

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/version"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// OIDCConfiguration is a struct that holds
// OIDC provider configuration data, read from command line arguments.
type OIDCConfiguration struct {
	// URL holds OIDC Issuer URL address
	URL string
	// ClientID holds OIDC ClientID
	ClientID string
	// ClientSecret holds OIDC ClientSecret
	ClientSecret string
	// CookieHashKey is required, used to authenticate the cookie value using HMAC
	// It is recommended to use a key with 32 or 64 bytes.
	CookieHashKey string
	// CookieSecureMode if true then cookie received only with HTTPS otherwise with HTTP.
	CookieSecureMode bool
	// OfflineAccessAsScope if true then "offline_access" scope will be used
	// otherwise 'access_type=offline" query param will be passed
	OfflineAccessAsScope bool
}

// UpdateManager specifies a set of methods to handle cluster versions & updates.
type UpdateManager interface {
	GetVersions() ([]*version.Version, error)
	GetVersionsForProvider(kubermaticv1.ProviderType, ...kubermaticv1.ConditionType) ([]*version.Version, error)
	GetDefault() (*version.Version, error)
	GetPossibleUpdates(from string, provider kubermaticv1.ProviderType, condition ...kubermaticv1.ConditionType) ([]*version.Version, error)
}

type SupportManager interface {
	GetIncompatibilities() []*version.ProviderIncompatibility
}

// ServerMetrics defines metrics used by the API.
type ServerMetrics struct {
	HTTPRequestsTotal          *prometheus.CounterVec
	HTTPRequestsDuration       *prometheus.HistogramVec
	InitNodeDeploymentFailures *prometheus.CounterVec
}

type CredentialsData struct {
	Ctx               context.Context
	KubermaticCluster *kubermaticv1.Cluster
	Client            ctrlruntimeclient.Client
}

func (d CredentialsData) Cluster() *kubermaticv1.Cluster {
	return d.KubermaticCluster
}

func (d CredentialsData) GetGlobalSecretKeySelectorValue(configVar *providerconfig.GlobalSecretKeySelector, key string) (string, error) {
	return provider.SecretKeySelectorValueFuncFactory(d.Ctx, d.Client)(configVar, key)
}

func GetProject(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID string, options *provider.ProjectGetOptions) (*kubermaticv1.Project, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}

	// check first if project exist
	adminProject, err := privilegedProjectProvider.GetUnsecured(ctx, projectID, options)
	if err != nil {
		return nil, err
	}

	if adminUserInfo.IsAdmin {
		// get any project for admin
		return adminProject, nil
	}

	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}
	return projectProvider.Get(ctx, userInfo, projectID, options)
}
