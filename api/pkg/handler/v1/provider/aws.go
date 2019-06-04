package provider

import (
	"context"

	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
)

// AWSCredentialEndpoint returns custom credential list name for AWS provider
func AWSCredentialEndpoint(credentialManager common.CredentialManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		credentials := apiv1.CredentialList{}
		names := make([]string, 0)
		if credentialManager.GetCredentials().AWS != nil {
			for _, do := range credentialManager.GetCredentials().AWS {
				names = append(names, do.Name)
			}
		}
		credentials.Names = names
		return credentials, nil
	}
}
