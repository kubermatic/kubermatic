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
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ClusterTemplateLabelKey         = "template-id"
	ClusterTemplateInstanceLabelKey = "template-instance-id"
)

// AlertmanagerProvider struct that holds required components in order to manage alertmanager objects.
type ClusterTemplateInstanceProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	// whenever a connection to Seed API server is required
	createSeedImpersonatedClient impersonationClient

	// privilegedClient is used for admins
	privilegedClient ctrlruntimeclient.Client
}

// ClusterTemplateInstanceProvider returns provider
func NewClusterTemplateInstanceProvider(createSeedImpersonatedClient impersonationClient, privilegedClient ctrlruntimeclient.Client) *ClusterTemplateInstanceProvider {
	return &ClusterTemplateInstanceProvider{
		createSeedImpersonatedClient: createSeedImpersonatedClient,
		privilegedClient:             privilegedClient,
	}
}

func ClusterTemplateInstanceProviderFactory(mapper meta.RESTMapper, seedKubeconfigGetter provider.SeedKubeconfigGetter) provider.ClusterTemplateInstanceProviderGetter {
	return func(seed *kubermaticv1.Seed) (provider.ClusterTemplateInstanceProvider, error) {
		cfg, err := seedKubeconfigGetter(seed)
		if err != nil {
			return nil, err
		}
		defaultImpersonationClientForSeed := NewImpersonationClient(cfg, mapper)
		privilegedClient, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Mapper: mapper})
		if err != nil {
			return nil, err
		}
		return NewClusterTemplateInstanceProvider(
			defaultImpersonationClientForSeed.CreateImpersonatedClient,
			privilegedClient,
		), nil
	}
}

func (r ClusterTemplateInstanceProvider) Create(userInfo *provider.UserInfo, template *kubermaticv1.ClusterTemplate, project *kubermaticv1.Project, replicas int64) (*kubermaticv1.ClusterTemplateInstance, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, r.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}
	instance := createClusterTemplateInstance(template, project, replicas)

	err = impersonationClient.Create(context.Background(), instance)
	return instance, err
}

func (r ClusterTemplateInstanceProvider) CreateUnsecured(template *kubermaticv1.ClusterTemplate, project *kubermaticv1.Project, replicas int64) (*kubermaticv1.ClusterTemplateInstance, error) {

	instance := createClusterTemplateInstance(template, project, replicas)

	err := r.privilegedClient.Create(context.Background(), instance)
	return instance, err
}

func createClusterTemplateInstance(template *kubermaticv1.ClusterTemplate, project *kubermaticv1.Project, replicas int64) *kubermaticv1.ClusterTemplateInstance {
	instance := &kubermaticv1.ClusterTemplateInstance{
		ObjectMeta: metav1.ObjectMeta{
			Name:   GetClusterTemplateInstanceName(project.Name, template.Name),
			Labels: map[string]string{ClusterTemplateLabelKey: template.Name},
		},
		Spec: kubermaticv1.ClusterTemplateInstanceSpec{
			ProjectID:           project.Name,
			ClusterTemplateID:   template.Name,
			ClusterTemplateName: template.Spec.HumanReadableName,
			Replicas:            replicas,
		},
	}

	addProjectReferenceForClusterTemplateInstance(project, instance)
	return instance
}

func (r ClusterTemplateInstanceProvider) GetUnsecured(name string) (*kubermaticv1.ClusterTemplateInstance, error) {
	instance := &kubermaticv1.ClusterTemplateInstance{}
	if err := r.privilegedClient.Get(context.Background(), types.NamespacedName{
		Name: name,
	}, instance); err != nil {
		return nil, err
	}
	return instance, nil
}

func (r ClusterTemplateInstanceProvider) Get(userInfo *provider.UserInfo, name string) (*kubermaticv1.ClusterTemplateInstance, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, r.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}
	instance := &kubermaticv1.ClusterTemplateInstance{}
	if err := impersonationClient.Get(context.Background(), types.NamespacedName{
		Name: name,
	}, instance); err != nil {
		return nil, err
	}
	return instance, nil
}

func (r ClusterTemplateInstanceProvider) List(userInfo *provider.UserInfo, options provider.ClusterTemplateInstanceListOptions) (*kubermaticv1.ClusterTemplateInstanceList, error) {
	instanceList := &kubermaticv1.ClusterTemplateInstanceList{}

	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, r.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	labelSelector := ctrlruntimeclient.MatchingLabels{}

	if options.ProjectID != "" {
		labelSelector[kubermaticv1.ClusterTemplateProjectLabelKey] = options.ProjectID
	}
	if options.TemplateID != "" {
		labelSelector[ClusterTemplateLabelKey] = options.TemplateID
	}

	if err := impersonationClient.List(context.Background(), instanceList, labelSelector); err != nil {
		return nil, err
	}

	return instanceList, nil
}

func (r ClusterTemplateInstanceProvider) Patch(userInfo *provider.UserInfo, instance *kubermaticv1.ClusterTemplateInstance) (*kubermaticv1.ClusterTemplateInstance, error) {
	impersonationClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, r.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	oldInstance, err := r.Get(userInfo, instance.Name)
	if err != nil {
		return nil, err
	}
	oldInstance = oldInstance.DeepCopy()

	if err := impersonationClient.Patch(context.Background(), instance, ctrlruntimeclient.MergeFrom(oldInstance)); err != nil {
		return nil, err
	}

	return instance, nil
}

func (r ClusterTemplateInstanceProvider) PatchUnsecured(instance *kubermaticv1.ClusterTemplateInstance) (*kubermaticv1.ClusterTemplateInstance, error) {
	oldInstance, err := r.GetUnsecured(instance.Name)
	if err != nil {
		return nil, err
	}
	oldInstance = oldInstance.DeepCopy()
	if err := r.privilegedClient.Patch(context.Background(), instance, ctrlruntimeclient.MergeFrom(oldInstance)); err != nil {
		return nil, err
	}

	return instance, nil
}

func GetClusterTemplateInstanceName(projectID, templateID string) string {
	return fmt.Sprintf("%s-%s-%s", projectID, templateID, rand.String(10))
}

func addProjectReferenceForClusterTemplateInstance(project *kubermaticv1.Project, templateInstance *kubermaticv1.ClusterTemplateInstance) {
	if templateInstance.Labels == nil {
		templateInstance.Labels = make(map[string]string)
	}
	templateInstance.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       kubermaticv1.ProjectKindName,
			UID:        project.GetUID(),
			Name:       project.Name,
		},
	}
	templateInstance.Labels[kubermaticv1.ProjectIDLabelKey] = project.Name
}
