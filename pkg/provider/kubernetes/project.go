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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/label"
	"k8c.io/kubermatic/v2/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewProjectProvider returns a project provider.
func NewProjectProvider(createMasterImpersonatedClient ImpersonationClient, client ctrlruntimeclient.Client) (*ProjectProvider, error) {
	return &ProjectProvider{
		createMasterImpersonatedClient: createMasterImpersonatedClient,
		clientPrivileged:               client,
	}, nil
}

// NewPrivilegedProjectProvider returns a privileged project provider.
func NewPrivilegedProjectProvider(client ctrlruntimeclient.Client) (*PrivilegedProjectProvider, error) {
	return &PrivilegedProjectProvider{
		clientPrivileged: client,
	}, nil
}

// ProjectProvider represents a data structure that knows how to manage projects.
type ProjectProvider struct {
	// createMasterImpersonatedClient is used as a ground for impersonation
	createMasterImpersonatedClient ImpersonationClient

	// clientPrivileged privileged client
	clientPrivileged ctrlruntimeclient.Client
}

var _ provider.ProjectProvider = &ProjectProvider{}

// PrivilegedProjectProvider represents a data structure that knows how to manage projects in a privileged way.
type PrivilegedProjectProvider struct {
	// treat clientPrivileged as a privileged user and use wisely
	clientPrivileged ctrlruntimeclient.Client
}

var _ provider.PrivilegedProjectProvider = &PrivilegedProjectProvider{}

// New creates a brand new project in the system with the given name
//
// Note:
// a user cannot own more than one project with the given name
// since we get the list of the current projects from a cache (lister) there is a small time window
// during which a user can create more that one project with the given name.
func (p *ProjectProvider) New(ctx context.Context, users []*kubermaticv1.User, projectName string, labels map[string]string) (*kubermaticv1.Project, error) {
	if len(users) == 0 {
		return nil, errors.New("users are missing but required")
	}

	owners := sets.NewString()
	for _, user := range users {
		owners.Insert(user.Name)
	}

	project := &kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:   rand.String(10),
			Labels: labels,
		},
		Spec: kubermaticv1.ProjectSpec{
			Name: projectName,
		},
	}

	if err := p.clientPrivileged.Create(ctx, project); err != nil {
		return nil, err
	}

	return project, nil
}

// Update update a specific project for a specific user and returns the updated project.
func (p *ProjectProvider) Update(ctx context.Context, userInfo *provider.UserInfo, newProject *kubermaticv1.Project) (*kubermaticv1.Project, error) {
	if userInfo == nil {
		return nil, errors.New("a user is missing but required")
	}
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}

	if err := masterImpersonatedClient.Update(ctx, newProject); err != nil {
		return nil, err
	}
	return newProject, nil
}

// Delete deletes the given project as the given user.
func (p *ProjectProvider) Delete(ctx context.Context, userInfo *provider.UserInfo, projectInternalName string) error {
	if userInfo == nil {
		return errors.New("a user is missing but required")
	}
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return err
	}

	existingProject := &kubermaticv1.Project{}
	if err := masterImpersonatedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: projectInternalName}, existingProject); err != nil {
		return err
	}

	return masterImpersonatedClient.Delete(ctx, existingProject)
}

// Get returns the project with the given name.
func (p *ProjectProvider) Get(ctx context.Context, userInfo *provider.UserInfo, projectInternalName string, options *provider.ProjectGetOptions) (*kubermaticv1.Project, error) {
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
	existingProject := &kubermaticv1.Project{}
	if err := masterImpersonatedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: projectInternalName}, existingProject); err != nil {
		return nil, err
	}
	if !options.IncludeUninitialized && existingProject.Status.Phase != kubermaticv1.ProjectActive {
		return nil, kerrors.NewServiceUnavailable("Project is not initialized yet")
	}

	return existingProject, nil
}

// GetUnsecured returns the project with the given name
// This function is unsafe in a sense that it uses privileged account to get project with the given name.
func (p *PrivilegedProjectProvider) GetUnsecured(ctx context.Context, projectInternalName string, options *provider.ProjectGetOptions) (*kubermaticv1.Project, error) {
	if options == nil {
		options = &provider.ProjectGetOptions{IncludeUninitialized: true}
	}
	project := &kubermaticv1.Project{}
	if err := p.clientPrivileged.Get(ctx, ctrlruntimeclient.ObjectKey{Name: projectInternalName}, project); err != nil {
		return nil, err
	}
	if !options.IncludeUninitialized && project.Status.Phase != kubermaticv1.ProjectActive {
		return nil, kerrors.NewServiceUnavailable("Project is not initialized yet")
	}
	return project, nil
}

// DeleteUnsecured deletes any given project
// This function is unsafe in a sense that it uses privileged account to delete project with the given name.
func (p *PrivilegedProjectProvider) DeleteUnsecured(ctx context.Context, projectInternalName string) error {
	existingProject := &kubermaticv1.Project{}
	if err := p.clientPrivileged.Get(ctx, ctrlruntimeclient.ObjectKey{Name: projectInternalName}, existingProject); err != nil {
		return err
	}

	return p.clientPrivileged.Delete(ctx, existingProject)
}

// UpdateUnsecured update a specific project and returns the updated project
// This function is unsafe in a sense that it uses privileged account to update the project.
func (p *PrivilegedProjectProvider) UpdateUnsecured(ctx context.Context, project *kubermaticv1.Project) (*kubermaticv1.Project, error) {
	if err := p.clientPrivileged.Update(ctx, project); err != nil {
		return nil, err
	}
	return project, nil
}

// List gets a list of projects, by default it returns all resources.
// If you want to filter the result please set ProjectListOptions
//
// Note that the list is taken from the cache.
func (p *ProjectProvider) List(ctx context.Context, options *provider.ProjectListOptions) ([]*kubermaticv1.Project, error) {
	if options == nil {
		options = &provider.ProjectListOptions{}
	}
	projects := &kubermaticv1.ProjectList{}
	if err := p.clientPrivileged.List(ctx, projects); err != nil {
		return nil, err
	}

	var ret []*kubermaticv1.Project
	for _, project := range projects.Items {
		// apply list filters
		if len(options.ProjectName) > 0 && project.Spec.Name != options.ProjectName {
			continue
		}

		// filter out restricted labels
		project.Labels = label.FilterLabels(label.ClusterResourceType, project.Labels)

		ret = append(ret, project.DeepCopy())
	}

	return ret, nil
}
