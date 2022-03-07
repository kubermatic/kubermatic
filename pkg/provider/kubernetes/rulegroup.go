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

package kubernetes

import (
	"context"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// RuleGroupProvider struct that holds required components in order to manage RuleGroup objects.
type RuleGroupProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	// whenever a connection to Seed API server is required
	createSeedImpersonatedClient ImpersonationClient

	// privilegedClient is used for admins to interact with RuleGroup objects.
	privilegedClient ctrlruntimeclient.Client
}

var _ provider.RuleGroupProvider = &RuleGroupProvider{}
var _ provider.PrivilegedRuleGroupProvider = &RuleGroupProvider{}

// NewRuleGroupProvider returns a ruleGroup provider.
func NewRuleGroupProvider(createSeedImpersonatedClient ImpersonationClient, privilegedClient ctrlruntimeclient.Client) *RuleGroupProvider {
	return &RuleGroupProvider{
		createSeedImpersonatedClient: createSeedImpersonatedClient,
		privilegedClient:             privilegedClient,
	}
}

func RuleGroupProviderFactory(mapper meta.RESTMapper, seedKubeconfigGetter provider.SeedKubeconfigGetter) provider.RuleGroupProviderGetter {
	return func(seed *kubermaticv1.Seed) (provider.RuleGroupProvider, error) {
		cfg, err := seedKubeconfigGetter(seed)
		if err != nil {
			return nil, err
		}
		defaultImpersonationClientForSeed := NewImpersonationClient(cfg, mapper)
		privilegedClient, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Mapper: mapper})
		if err != nil {
			return nil, err
		}
		return NewRuleGroupProvider(
			defaultImpersonationClientForSeed.CreateImpersonatedClient,
			privilegedClient,
		), nil
	}
}

func (r RuleGroupProvider) Create(ctx context.Context, userInfo *provider.UserInfo, ruleGroup *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, r.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}
	err = impersonationClient.Create(ctx, ruleGroup)
	return ruleGroup, err
}

func (r RuleGroupProvider) List(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, options *provider.RuleGroupListOptions) ([]*kubermaticv1.RuleGroup, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, r.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}
	return listRuleGroups(ctx, impersonationClient, cluster.Status.NamespaceName, options)
}

func (r RuleGroupProvider) Get(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, ruleGroupName string) (*kubermaticv1.RuleGroup, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, r.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}
	ruleGroup := &kubermaticv1.RuleGroup{}
	if err := impersonationClient.Get(ctx, types.NamespacedName{
		Name:      ruleGroupName,
		Namespace: cluster.Status.NamespaceName,
	}, ruleGroup); err != nil {
		return nil, err
	}
	return ruleGroup, nil
}

func (r RuleGroupProvider) Update(ctx context.Context, userInfo *provider.UserInfo, newRuleGroup *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, r.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}
	err = impersonationClient.Update(ctx, newRuleGroup)
	return newRuleGroup, err
}

func (r RuleGroupProvider) Delete(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, ruleGroupName string) error {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, r.createSeedImpersonatedClient)
	if err != nil {
		return err
	}
	return impersonationClient.Delete(ctx, &kubermaticv1.RuleGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleGroupName,
			Namespace: cluster.Status.NamespaceName,
		},
	})
}

func (r RuleGroupProvider) GetUnsecured(ctx context.Context, ruleGroupName, namespace string) (*kubermaticv1.RuleGroup, error) {
	ruleGroup := &kubermaticv1.RuleGroup{}
	if err := r.privilegedClient.Get(ctx, types.NamespacedName{
		Name:      ruleGroupName,
		Namespace: namespace,
	}, ruleGroup); err != nil {
		return nil, err
	}
	return ruleGroup, nil
}

func (r RuleGroupProvider) ListUnsecured(ctx context.Context, namespace string, options *provider.RuleGroupListOptions) ([]*kubermaticv1.RuleGroup, error) {
	return listRuleGroups(ctx, r.privilegedClient, namespace, options)
}

func (r RuleGroupProvider) CreateUnsecured(ctx context.Context, ruleGroup *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error) {
	err := r.privilegedClient.Create(ctx, ruleGroup)
	return ruleGroup, err
}

func (r RuleGroupProvider) UpdateUnsecured(ctx context.Context, newRuleGroup *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error) {
	err := r.privilegedClient.Update(ctx, newRuleGroup)
	return newRuleGroup, err
}

func (r RuleGroupProvider) DeleteUnsecured(ctx context.Context, ruleGroupName, namespace string) error {
	return r.privilegedClient.Delete(ctx, &kubermaticv1.RuleGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleGroupName,
			Namespace: namespace,
		},
	})
}

func listRuleGroups(ctx context.Context, client ctrlruntimeclient.Client, namespace string, options *provider.RuleGroupListOptions) ([]*kubermaticv1.RuleGroup, error) {
	if options == nil {
		options = &provider.RuleGroupListOptions{}
	}
	ruleGroupList := &kubermaticv1.RuleGroupList{}
	if err := client.List(ctx, ruleGroupList, ctrlruntimeclient.InNamespace(namespace)); err != nil {
		return nil, err
	}
	var res []*kubermaticv1.RuleGroup
	for _, ruleGroup := range ruleGroupList.Items {
		if len(options.RuleGroupType) == 0 || options.RuleGroupType == ruleGroup.Spec.RuleGroupType {
			res = append(res, ruleGroup.DeepCopy())
		}
	}
	return res, nil
}
