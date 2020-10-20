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

package api

import (
	"context"
	"fmt"
	"os"

	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/api/utils/dex"
)

const (
	LoginEnvironmentVariable         = "KUBERMATIC_OIDC_LOGIN"
	PasswordEnvironmentVariable      = "KUBERMATIC_OIDC_PASSWORD"
	DexValuesFileEnvironmentVariable = "KUBERMATIC_DEX_VALUES_FILE"
)

// OIDCCredentials takes the login name and password from environment variables and
// returns them.
func OIDCCredentials() (string, string, error) {
	login := os.Getenv(LoginEnvironmentVariable)
	if len(login) == 0 {
		return "", "", fmt.Errorf("no OIDC username specified ($%s is unset)", LoginEnvironmentVariable)
	}

	password := os.Getenv(PasswordEnvironmentVariable)
	if len(password) == 0 {
		return "", "", fmt.Errorf("no OIDC password specified ($%s is unset)", PasswordEnvironmentVariable)
	}

	return login, password, nil
}

// OIDCAdminCredentials takes the admin login name and password from environment variables and
// returns them.
func OIDCAdminCredentials() (string, string, error) {
	password := os.Getenv(PasswordEnvironmentVariable)
	if len(password) == 0 {
		return "", "", fmt.Errorf("no OIDC password specified ($%s is unset)", PasswordEnvironmentVariable)
	}

	return "roxy2@loodse.com", password, nil
}

// these variables are runtime caches to not have to login to Dex
// over and over again
var (
	masterToken      = ""
	adminMasterToken = ""
)

func retrieveMasterToken() (string, error) {
	login, password, err := OIDCCredentials()
	if err != nil {
		return "", err
	}

	return retrieveToken(&masterToken, login, password)
}

func retrieveAdminMasterToken() (string, error) {
	login, password, err := OIDCAdminCredentials()
	if err != nil {
		return "", err
	}

	return retrieveToken(&adminMasterToken, login, password)
}

func retrieveToken(token *string, login string, password string) (string, error) {
	// re-use the previous token
	if token != nil && *token != "" {
		return *token, nil
	}

	valuesFile := os.Getenv(DexValuesFileEnvironmentVariable)
	if len(valuesFile) == 0 {
		return "", fmt.Errorf("no Helm values.yaml specified via $%s env variable", DexValuesFileEnvironmentVariable)
	}

	logger := log.New(false, log.FormatJSON).Sugar()

	client, err := dex.NewClientFromHelmValues(valuesFile, "kubermatic", logger)
	if err != nil {
		return "", fmt.Errorf("failed to create OIDC client: %v", err)
	}

	newToken, err := client.Login(context.Background(), login, password)
	if err != nil {
		return "", err
	}

	// update runtime cache
	*token = newToken

	return newToken, nil
}
