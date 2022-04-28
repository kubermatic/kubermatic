//go:build dualstack

/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package dualstack

import (
	"context"
	"flag"
	httptransport "github.com/go-openapi/runtime/client"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client/project"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/models"
	"net/http"
	"testing"
)

var (
	bearerToken string
)

func init() {
	flag.StringVar(&bearerToken, "bearerToken", "", "bearerToken for kubermatic API")
}

//
//func azureCloudSpec() *models.AzureCloudSpec {
//	return &models.AzureCloudSpec{
//		ClientID:        az.ClientID(),
//		ClientSecret:    az.ClientSecret(),
//		SubscriptionID:  az.SubscriptionID(),
//		TenantID:        az.TenantID(),
//		LoadBalancerSKU: "standard",
//	}
//}

func makeCluster(os string, provider string) {}

func TestSetupDualstackCluster(t *testing.T) {
	clusterSpec := &models.CreateClusterSpec{}

	apiClient := client.NewHTTPClientWithConfig(nil, &client.TransportConfig{
		Host: "dev.kubermatic.io",
		// BasePath: "/api/v2",
		Schemes: client.DefaultSchemes,
	})

	_, err := apiClient.Project.CreateClusterV2(&project.CreateClusterV2Params{
		Body:       clusterSpec,
		ProjectID:  "sszxpzjcnm",
		Context:    context.Background(),
		HTTPClient: http.DefaultClient,
	}, httptransport.BearerToken(bearerToken))

	if err != nil {
		t.Fatalf(": %v", err)
	}

}
