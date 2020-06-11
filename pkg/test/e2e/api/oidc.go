package api

import (
	"context"
	"fmt"
	"os"

	"github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/dex"
)

// OIDCCredentials takes the login name and password from environment variables and
// returns them.
func OIDCCredentials() (string, string) {
	return os.Getenv("KUBERMATIC_OIDC_LOGIN"), os.Getenv("KUBERMATIC_OIDC_PASSWORD")
}

// OIDCAdminCredentials takes the admin login name and password from environment variables and
// returns them.
func OIDCAdminCredentials() (string, string) {
	return "roxy2@loodse.com", os.Getenv("KUBERMATIC_OIDC_PASSWORD")
}

var masterToken = ""

// this is just a helper to make the tests more readable
func retrieveMasterToken() (string, error) {
	// re-use the previous token
	if masterToken != "" {
		return masterToken, nil
	}

	valuesFile := os.Getenv("KUBERMATIC_DEX_VALUES_FILE")
	if len(valuesFile) == 0 {
		return "", fmt.Errorf("no Helm values.yaml specified via KUBERMATIC_DEX_VALUES_FILE env variable")
	}

	logger := log.New(true, log.FormatJSON).Sugar()

	client, err := dex.NewClientFromHelmValues(valuesFile, "kubermatic", logger)
	if err != nil {
		return "", fmt.Errorf("failed to create OIDC client: %v", err)
	}

	login, password := OIDCCredentials()

	masterToken, err = client.Login(context.Background(), login, password)
	if err != nil {
		return "", err
	}

	return masterToken, nil
}

var adminMasterToken = ""

// this is just a helper to make the tests more readable
func retrieveAdminMasterToken() (string, error) {
	// re-use the previous token
	if adminMasterToken != "" {
		return adminMasterToken, nil
	}

	valuesFile := os.Getenv("KUBERMATIC_DEX_VALUES_FILE")
	if len(valuesFile) == 0 {
		return "", fmt.Errorf("no Helm values.yaml specified via KUBERMATIC_DEX_VALUES_FILE env variable")
	}

	logger := log.New(true, log.FormatJSON).Sugar()

	client, err := dex.NewClientFromHelmValues(valuesFile, "kubermatic", logger)
	if err != nil {
		return "", fmt.Errorf("failed to create OIDC client: %v", err)
	}

	login, password := OIDCAdminCredentials()

	adminMasterToken, err = client.Login(context.Background(), login, password)
	if err != nil {
		return "", err
	}

	return adminMasterToken, nil
}
