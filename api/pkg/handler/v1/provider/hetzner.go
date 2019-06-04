package provider

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
)

// HetznerCredentialEndpoint returns custom credential list name for Hetzner provider
func HetznerCredentialEndpoint(credentialManager common.CredentialManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		credentials := apiv1.CredentialList{}
		names := make([]string, 0)
		if credentialManager.GetCredentials().Hetzner != nil {
			for _, do := range credentialManager.GetCredentials().Hetzner {
				names = append(names, do.Name)
			}
		}
		credentials.Names = names
		return credentials, nil
	}
}
