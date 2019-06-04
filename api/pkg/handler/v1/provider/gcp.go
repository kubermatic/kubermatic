package provider

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
)

// GCPCredentialEndpoint returns custom credential list name for GCP provider
func GCPCredentialEndpoint(credentialManager common.CredentialManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		credentials := apiv1.CredentialList{}
		names := make([]string, 0)
		if credentialManager.GetCredentials().GCP != nil {
			for _, do := range credentialManager.GetCredentials().GCP {
				names = append(names, do.Name)
			}
		}
		credentials.Names = names
		return credentials, nil
	}
}
