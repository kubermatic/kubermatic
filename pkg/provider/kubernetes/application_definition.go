package kubernetes

import (
	"context"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
)

type ApplicationDefinitionProvider struct {
	masterClientToImpersonate ImpersonationClient
}

var _ provider.ApplicationDefinitionProvider = &ApplicationDefinitionProvider{}

func NewApplicationDefinitionProvider(masterClientToImpersonate ImpersonationClient) *ApplicationDefinitionProvider {
	return &ApplicationDefinitionProvider{
		masterClientToImpersonate: masterClientToImpersonate,
	}
}

func (p *ApplicationDefinitionProvider) List(ctx context.Context, userInfo *provider.UserInfo) (*appskubermaticv1.ApplicationDefinitionList, error) {
	impersonatedMasterClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.masterClientToImpersonate)
	if err != nil {
		return nil, err
	}

	appDefList := &appskubermaticv1.ApplicationDefinitionList{}
	if err := impersonatedMasterClient.List(ctx, appDefList); err != nil {
		return nil, err
	}
	return appDefList, nil
}
