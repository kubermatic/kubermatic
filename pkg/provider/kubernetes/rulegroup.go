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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
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

// NewRuleGroupProvider returns a ruleGroup provider
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

func (r RuleGroupProvider) Create(userInfo *provider.UserInfo, ruleGroup *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, r.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}
	err = impersonationClient.Create(context.Background(), ruleGroup)
	return ruleGroup, err
}

func (r RuleGroupProvider) List(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, options *provider.RuleGroupListOptions) ([]*kubermaticv1.RuleGroup, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, r.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}
	return listRuleGroups(impersonationClient, cluster, options)
}

func (r RuleGroupProvider) Get(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, ruleGroupName string) (*kubermaticv1.RuleGroup, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, r.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}
	ruleGroup := &kubermaticv1.RuleGroup{}
	if err := impersonationClient.Get(context.Background(), types.NamespacedName{
		Name:      ruleGroupName,
		Namespace: cluster.Status.NamespaceName,
	}, ruleGroup); err != nil {
		return nil, err
	}
	return ruleGroup, nil
}

func (r RuleGroupProvider) Update(userInfo *provider.UserInfo, newRuleGroup *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, r.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}
	err = impersonationClient.Update(context.Background(), newRuleGroup)
	return newRuleGroup, err
}

func (r RuleGroupProvider) Delete(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, ruleGroupName string) error {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, r.createSeedImpersonatedClient)
	if err != nil {
		return err
	}
	return impersonationClient.Delete(context.Background(), &kubermaticv1.RuleGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleGroupName,
			Namespace: cluster.Status.NamespaceName,
		},
	})
}

func (r RuleGroupProvider) GetUnsecured(cluster *kubermaticv1.Cluster, ruleGroupName string) (*kubermaticv1.RuleGroup, error) {
	ruleGroup := &kubermaticv1.RuleGroup{}
	if err := r.privilegedClient.Get(context.Background(), types.NamespacedName{
		Name:      ruleGroupName,
		Namespace: cluster.Status.NamespaceName,
	}, ruleGroup); err != nil {
		return nil, err
	}
	return ruleGroup, nil
}

func (r RuleGroupProvider) ListUnsecured(cluster *kubermaticv1.Cluster, options *provider.RuleGroupListOptions) ([]*kubermaticv1.RuleGroup, error) {
	return listRuleGroups(r.privilegedClient, cluster, options)
}

func (r RuleGroupProvider) CreateUnsecured(ruleGroup *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error) {
	err := r.privilegedClient.Create(context.Background(), ruleGroup)
	return ruleGroup, err
}

func (r RuleGroupProvider) UpdateUnsecured(newRuleGroup *kubermaticv1.RuleGroup) (*kubermaticv1.RuleGroup, error) {
	err := r.privilegedClient.Update(context.Background(), newRuleGroup)
	return newRuleGroup, err
}

func (r RuleGroupProvider) DeleteUnsecured(cluster *kubermaticv1.Cluster, ruleGroupName string) error {
	return r.privilegedClient.Delete(context.Background(), &kubermaticv1.RuleGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      ruleGroupName,
			Namespace: cluster.Status.NamespaceName,
		},
	})
}

func listRuleGroups(client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster, options *provider.RuleGroupListOptions) ([]*kubermaticv1.RuleGroup, error) {
	if options == nil {
		options = &provider.RuleGroupListOptions{}
	}
	ruleGroupList := &kubermaticv1.RuleGroupList{}
	if err := client.List(context.Background(), ruleGroupList, ctrlruntimeclient.InNamespace(cluster.Status.NamespaceName)); err != nil {
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
