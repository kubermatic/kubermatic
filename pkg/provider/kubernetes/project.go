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

package kubernetes

import (
	"context"
	"errors"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/label"
	"k8c.io/kubermatic/v2/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewProjectProvider returns a project provider
func NewProjectProvider(createMasterImpersonatedClient ImpersonationClient, client ctrlruntimeclient.Client) (*ProjectProvider, error) {

	return &ProjectProvider{
		createMasterImpersonatedClient: createMasterImpersonatedClient,
		clientPrivileged:               client,
	}, nil
}

// NewPrivilegedProjectProvider returns a privileged project provider
func NewPrivilegedProjectProvider(client ctrlruntimeclient.Client) (*PrivilegedProjectProvider, error) {
	return &PrivilegedProjectProvider{
		clientPrivileged: client,
	}, nil
}

// ProjectProvider represents a data structure that knows how to manage projects
type ProjectProvider struct {
	// createMasterImpersonatedClient is used as a ground for impersonation
	createMasterImpersonatedClient ImpersonationClient

	// clientPrivileged privileged client
	clientPrivileged ctrlruntimeclient.Client
}

// PrivilegedProjectProvider represents a data structure that knows how to manage projects in a privileged way
type PrivilegedProjectProvider struct {
	// treat clientPrivileged as a privileged user and use wisely
	clientPrivileged ctrlruntimeclient.Client
}

// New creates a brand new project in the system with the given name
//
// Note:
// a user cannot own more than one project with the given name
// since we get the list of the current projects from a cache (lister) there is a small time window
// during which a user can create more that one project with the given name.
func (p *ProjectProvider) New(users []*kubermaticapiv1.User, projectName string, labels map[string]string) (*kubermaticapiv1.Project, error) {
	if len(users) == 0 {
		return nil, errors.New("users are missing but required")
	}

	project := &kubermaticapiv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			OwnerReferences: []metav1.OwnerReference{},
			Name:            rand.String(10),
			Labels:          labels,
		},
		Spec: kubermaticapiv1.ProjectSpec{
			Name: projectName,
		},
		Status: kubermaticapiv1.ProjectStatus{
			Phase: kubermaticapiv1.ProjectInactive,
		},
	}

	for _, user := range users {
		project.OwnerReferences = append(project.OwnerReferences, metav1.OwnerReference{
			APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
			Kind:       kubermaticapiv1.UserKindName,
			UID:        user.GetUID(),
			Name:       user.Name,
		})
	}

	if err := p.clientPrivileged.Create(context.Background(), project); err != nil {
		return nil, err
	}
	return project, nil
}

// Update update a specific project for a specific user and returns the updated project
func (p *ProjectProvider) Update(userInfo *provider.UserInfo, newProject *kubermaticapiv1.Project) (*kubermaticapiv1.Project, error) {
	if userInfo == nil {
		return nil, errors.New("a user is missing but required")
	}
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}

	if err := masterImpersonatedClient.Update(context.Background(), newProject); err != nil {
		return nil, err
	}
	return newProject, nil
}

// Delete deletes the given project as the given user
//
// Note:
// Before deletion project's status.phase is set to ProjectTerminating
func (p *ProjectProvider) Delete(userInfo *provider.UserInfo, projectInternalName string) error {
	if userInfo == nil {
		return errors.New("a user is missing but required")
	}
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return err
	}

	existingProject := &kubermaticapiv1.Project{}
	if err := masterImpersonatedClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: projectInternalName}, existingProject); err != nil {
		return err
	}

	existingProject.Status.Phase = kubermaticapiv1.ProjectTerminating
	if err := masterImpersonatedClient.Update(context.Background(), existingProject); err != nil {
		return err
	}

	return masterImpersonatedClient.Delete(context.Background(), existingProject)
}

// Get returns the project with the given name
func (p *ProjectProvider) Get(userInfo *provider.UserInfo, projectInternalName string, options *provider.ProjectGetOptions) (*kubermaticapiv1.Project, error) {
	if userInfo == nil {
		return nil, errors.New("a user is missing but required")
	}
	if options == nil {
		options = &provider.ProjectGetOptions{IncludeUninitialized: true}
	}
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	existingProject := &kubermaticapiv1.Project{}
	if err := masterImpersonatedClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: projectInternalName}, existingProject); err != nil {
		return nil, err
	}
	if !options.IncludeUninitialized && existingProject.Status.Phase != kubermaticapiv1.ProjectActive {
		return nil, kerrors.NewServiceUnavailable("Project is not initialized yet")
	}

	return existingProject, nil
}

// GetUnsecured returns the project with the given name
// This function is unsafe in a sense that it uses privileged account to get project with the given name
func (p *PrivilegedProjectProvider) GetUnsecured(projectInternalName string, options *provider.ProjectGetOptions) (*kubermaticapiv1.Project, error) {
	if options == nil {
		options = &provider.ProjectGetOptions{IncludeUninitialized: true}
	}
	project := &kubermaticapiv1.Project{}
	if err := p.clientPrivileged.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: projectInternalName}, project); err != nil {
		return nil, err
	}
	if !options.IncludeUninitialized && project.Status.Phase != kubermaticapiv1.ProjectActive {
		return nil, kerrors.NewServiceUnavailable("Project is not initialized yet")
	}
	return project, nil
}

// DeleteUnsecured deletes any given project
// This function is unsafe in a sense that it uses privileged account to delete project with the given name
//
// Note:
// Before deletion project's status.phase is set to ProjectTerminating
func (p *PrivilegedProjectProvider) DeleteUnsecured(projectInternalName string) error {
	existingProject := &kubermaticapiv1.Project{}
	if err := p.clientPrivileged.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: projectInternalName}, existingProject); err != nil {
		return err
	}
	existingProject.Status.Phase = kubermaticapiv1.ProjectTerminating
	if err := p.clientPrivileged.Update(context.Background(), existingProject); err != nil {
		return err
	}

	return p.clientPrivileged.Delete(context.Background(), existingProject)
}

// UpdateUnsecured update a specific project and returns the updated project
// This function is unsafe in a sense that it uses privileged account to update the project
func (p *PrivilegedProjectProvider) UpdateUnsecured(project *kubermaticapiv1.Project) (*kubermaticapiv1.Project, error) {
	if err := p.clientPrivileged.Update(context.Background(), project); err != nil {
		return nil, err
	}
	return project, nil
}

// List gets a list of projects, by default it returns all resources.
// If you want to filter the result please set ProjectListOptions
//
// Note that the list is taken from the cache
func (p *ProjectProvider) List(options *provider.ProjectListOptions) ([]*kubermaticapiv1.Project, error) {
	if options == nil {
		options = &provider.ProjectListOptions{}
	}
	projects := &kubermaticapiv1.ProjectList{}
	if err := p.clientPrivileged.List(context.Background(), projects); err != nil {
		return nil, err
	}

	var ret []*kubermaticapiv1.Project
	for _, project := range projects.Items {
		if len(options.ProjectName) > 0 && project.Spec.Name != options.ProjectName {
			continue
		}
		if len(options.OwnerUID) > 0 {
			owners := project.GetOwnerReferences()
			for _, owner := range owners {
				if owner.UID == options.OwnerUID {
					ret = append(ret, project.DeepCopy())
					continue
				}
			}
			continue
		}

		ret = append(ret, project.DeepCopy())
	}

	// Filter out restricted labels
	for i, project := range ret {
		project.Labels = label.FilterLabels(label.ClusterResourceType, project.Labels)
		ret[i] = project
	}

	return ret, nil
}
