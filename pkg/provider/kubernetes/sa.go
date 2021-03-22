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
	"fmt"
	"strings"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ServiceAccountLabelGroup      = "initialGroup"
	ServiceAccountAnnotationOwner = "owner"
	saPrefix                      = "serviceaccount-"
	mainSAPrefix                  = "main-serviceaccount-"
)

// NewServiceAccountProvider returns a service account provider
func NewServiceAccountProvider(createMasterImpersonatedClient impersonationClient, clientPrivileged ctrlruntimeclient.Client, domain string) *ServiceAccountProvider {
	return &ServiceAccountProvider{
		createMasterImpersonatedClient: createMasterImpersonatedClient,
		clientPrivileged:               clientPrivileged,
		domain:                         domain,
	}
}

// ServiceAccountProvider manages service account resources
type ServiceAccountProvider struct {
	// createMasterImpersonatedClient is used as a ground for impersonation
	createMasterImpersonatedClient impersonationClient

	// treat clientPrivileged as a privileged user and use wisely
	clientPrivileged ctrlruntimeclient.Client

	// domain name on which the server is deployed
	domain string
}

// CreateProjectServiceAccount creates a new service account for the project
func (p *ServiceAccountProvider) CreateProjectServiceAccount(userInfo *provider.UserInfo, project *kubermaticv1.Project, name, group string) (*kubermaticv1.User, error) {
	if project == nil {
		return nil, kerrors.NewBadRequest("Project cannot be nil")
	}
	if len(name) == 0 || len(group) == 0 {
		return nil, kerrors.NewBadRequest("Service account name and group cannot be empty when creating a new SA resource")
	}

	sa := genProjectServiceAccount(project, name, group, p.domain)

	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}

	if err := masterImpersonatedClient.Create(context.Background(), sa); err != nil {
		return nil, err
	}
	sa.Name = removeProjectSAPrefix(sa.Name)
	return sa, nil
}

// CreateUnsecuredProjectServiceAccount creates a new service accounts
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to create the resources
func (p *ServiceAccountProvider) CreateUnsecuredProjectServiceAccount(project *kubermaticv1.Project, name, group string) (*kubermaticv1.User, error) {
	if project == nil {
		return nil, kerrors.NewBadRequest("Project cannot be nil")
	}
	if len(name) == 0 || len(group) == 0 {
		return nil, kerrors.NewBadRequest("Service account name and group cannot be empty when creating a new SA resource")
	}

	sa := genProjectServiceAccount(project, name, group, p.domain)

	if err := p.clientPrivileged.Create(context.Background(), sa); err != nil {
		return nil, err
	}

	sa.Name = removeProjectSAPrefix(sa.Name)
	return sa, nil
}

func genProjectServiceAccount(project *kubermaticv1.Project, name, group, domain string) *kubermaticv1.User {
	uniqueID := rand.String(10)
	uniqueName := fmt.Sprintf("%s%s", saPrefix, uniqueID)

	sa := &kubermaticv1.User{}
	sa.Name = uniqueName
	sa.Spec.Email = fmt.Sprintf("%s@%s", uniqueName, domain)
	sa.Spec.Name = name
	sa.Spec.ID = uniqueID
	sa.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       kubermaticv1.ProjectKindName,
			UID:        project.GetUID(),
			Name:       project.Name,
		},
	}
	sa.Labels = map[string]string{ServiceAccountLabelGroup: group}
	return sa
}

// ListProjectServiceAccount gets service accounts for the project
func (p *ServiceAccountProvider) ListProjectServiceAccount(userInfo *provider.UserInfo, project *kubermaticv1.Project, options *provider.ServiceAccountListOptions) ([]*kubermaticv1.User, error) {
	if userInfo == nil {
		return nil, kerrors.NewBadRequest("userInfo cannot be nil")
	}
	if project == nil {
		return nil, kerrors.NewBadRequest("project cannot be nil")
	}
	if options == nil {
		options = &provider.ServiceAccountListOptions{}
	}

	resultList, err := p.listProjectSA(project)
	if err != nil {
		return nil, err
	}

	// Note:
	// After we get the list of SA we try to get at least one item using unprivileged account to see if the user have read access
	if len(resultList) > 0 {

		masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
		if err != nil {
			return nil, err
		}

		saToGet := resultList[0]
		err = masterImpersonatedClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: saToGet.Name}, &kubermaticv1.User{})
		if err != nil {
			return nil, err
		}

	}

	for _, sa := range resultList {
		sa.Name = removeProjectSAPrefix(sa.Name)
	}

	if len(options.ServiceAccountName) == 0 {
		return resultList, nil
	}

	filteredList := make([]*kubermaticv1.User, 0)
	for _, sa := range resultList {
		if sa.Spec.Name == options.ServiceAccountName {
			filteredList = append(filteredList, sa)
			break
		}
	}

	return filteredList, nil
}

// ListUnsecuredProjectServiceAccount gets all service accounts for the project
// If you want to filter the result please take a look at ServiceAccountListOptions
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to get the resources
func (p *ServiceAccountProvider) ListUnsecuredProjectServiceAccount(project *kubermaticv1.Project, options *provider.ServiceAccountListOptions) ([]*kubermaticv1.User, error) {
	if project == nil {
		return nil, kerrors.NewBadRequest("project cannot be nil")
	}
	if options == nil {
		options = &provider.ServiceAccountListOptions{}
	}

	resultList, err := p.listProjectSA(project)
	if err != nil {
		return nil, err
	}

	for _, sa := range resultList {
		sa.Name = removeProjectSAPrefix(sa.Name)
	}

	if len(options.ServiceAccountName) == 0 {
		return resultList, nil
	}

	filteredList := make([]*kubermaticv1.User, 0)
	for _, sa := range resultList {
		if sa.Spec.Name == options.ServiceAccountName {
			filteredList = append(filteredList, sa)
			break
		}
	}

	return filteredList, nil
}

func (p *ServiceAccountProvider) listProjectSA(project *kubermaticv1.Project) ([]*kubermaticv1.User, error) {
	serviceAccounts := &kubermaticv1.UserList{}
	if err := p.clientPrivileged.List(context.Background(), serviceAccounts); err != nil {
		return nil, err
	}

	resultList := make([]*kubermaticv1.User, 0)
	for _, sa := range serviceAccounts.Items {
		if hasProjectSAPrefix(sa.Name) {
			for _, owner := range sa.GetOwnerReferences() {
				if owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() && owner.Kind == kubermaticv1.ProjectKindName && owner.Name == project.Name {
					resultList = append(resultList, sa.DeepCopy())
				}
			}
		}
	}
	return resultList, nil
}

// GetProjectServiceAccount method returns project service account with given name
func (p *ServiceAccountProvider) GetProjectServiceAccount(userInfo *provider.UserInfo, name string, options *provider.ServiceAccountGetOptions) (*kubermaticv1.User, error) {
	if userInfo == nil {
		return nil, kerrors.NewBadRequest("userInfo cannot be nil")
	}
	if len(name) == 0 {
		return nil, kerrors.NewBadRequest("service account name cannot be empty")
	}
	if options == nil {
		options = &provider.ServiceAccountGetOptions{RemovePrefix: true}
	}

	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}

	name = addProjectSAPrefix(name)
	serviceAccount := &kubermaticv1.User{}
	if err := masterImpersonatedClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: name}, serviceAccount); err != nil {
		return nil, err
	}

	if options.RemovePrefix {
		serviceAccount.Name = removeProjectSAPrefix(serviceAccount.Name)
	}
	return serviceAccount, nil
}

// GetUnsecuredProjectServiceAccount gets the project service account
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to get the resource
func (p *ServiceAccountProvider) GetUnsecuredProjectServiceAccount(name string, options *provider.ServiceAccountGetOptions) (*kubermaticv1.User, error) {
	if len(name) == 0 {
		return nil, kerrors.NewBadRequest("service account name cannot be empty")
	}
	if options == nil {
		options = &provider.ServiceAccountGetOptions{RemovePrefix: true}
	}

	name = addProjectSAPrefix(name)
	serviceAccount := &kubermaticv1.User{}
	if err := p.clientPrivileged.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: name}, serviceAccount); err != nil {
		return nil, err
	}

	if options.RemovePrefix {
		serviceAccount.Name = removeProjectSAPrefix(serviceAccount.Name)
	}
	return serviceAccount, nil
}

// UpdateProjectServiceAccount simply updates the given project service account
func (p *ServiceAccountProvider) UpdateProjectServiceAccount(userInfo *provider.UserInfo, serviceAccount *kubermaticv1.User) (*kubermaticv1.User, error) {
	if userInfo == nil {
		return nil, kerrors.NewBadRequest("userInfo cannot be nil")
	}
	if serviceAccount == nil {
		return nil, kerrors.NewBadRequest("service account name cannot be nil")
	}

	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}

	serviceAccount.Name = addProjectSAPrefix(serviceAccount.Name)

	if err := masterImpersonatedClient.Update(context.Background(), serviceAccount); err != nil {
		return nil, err
	}
	serviceAccount.Name = removeProjectSAPrefix(serviceAccount.Name)
	return serviceAccount, nil
}

// UpdateUnsecuredProjectServiceAccount updated the project service account
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to update the resource
func (p *ServiceAccountProvider) UpdateUnsecuredProjectServiceAccount(serviceAccount *kubermaticv1.User) (*kubermaticv1.User, error) {
	if serviceAccount == nil {
		return nil, kerrors.NewBadRequest("service account name cannot be nil")
	}

	serviceAccount.Name = addProjectSAPrefix(serviceAccount.Name)

	if err := p.clientPrivileged.Update(context.Background(), serviceAccount); err != nil {
		return nil, err
	}
	serviceAccount.Name = removeProjectSAPrefix(serviceAccount.Name)
	return serviceAccount, nil
}

// DeleteProjectServiceAccount simply deletes the given project service account
func (p *ServiceAccountProvider) DeleteProjectServiceAccount(userInfo *provider.UserInfo, name string) error {
	if userInfo == nil {
		return kerrors.NewBadRequest("userInfo cannot be nil")
	}
	if len(name) == 0 {
		return kerrors.NewBadRequest("service account name cannot be empty")
	}

	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return err
	}

	name = addProjectSAPrefix(name)
	return masterImpersonatedClient.Delete(context.Background(), &kubermaticv1.User{ObjectMeta: metav1.ObjectMeta{Name: name}})
}

// DeleteUnsecuredProjectServiceAccount deletes project service account
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to delete the resource
func (p *ServiceAccountProvider) DeleteUnsecuredProjectServiceAccount(name string) error {
	if len(name) == 0 {
		return kerrors.NewBadRequest("service account name cannot be empty")
	}

	name = addProjectSAPrefix(name)
	return p.clientPrivileged.Delete(context.Background(), &kubermaticv1.User{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	})
}

// IsProjectServiceAccount determines whether the given email address
// belongs to a project service account
func IsProjectServiceAccount(email string) bool {
	return hasProjectSAPrefix(email)
}

// removeProjectSAPrefix removes "serviceaccount-" from a SA's ID,
// for example given "serviceaccount-7d4b5695vb" it returns "7d4b5695vb"
func removeProjectSAPrefix(id string) string {
	return strings.TrimPrefix(id, saPrefix)
}

// addProjectSAPrefix adds "serviceaccount-" prefix to a SA's ID,
// for example given "7d4b5695vb" it returns "serviceaccount-7d4b5695vb"
func addProjectSAPrefix(id string) string {
	if !hasProjectSAPrefix(id) {
		return fmt.Sprintf("%s%s", saPrefix, id)
	}
	return id
}

// hasProjectSAPrefix checks if the given id has "serviceaccount-" prefix
func hasProjectSAPrefix(sa string) bool {
	return strings.HasPrefix(sa, saPrefix)
}

// CreateMainServiceAccount creates a new main service account
func (p *ServiceAccountProvider) CreateMainServiceAccount(userInfo *provider.UserInfo, name, group string) (*kubermaticv1.User, error) {
	if len(name) == 0 || len(group) == 0 {
		return nil, kerrors.NewBadRequest("Service account name and group cannot be empty when creating a new main service account resource")
	}

	sa := genMainServiceAccount(name, group, p.domain, userInfo.Email)

	if err := p.clientPrivileged.Create(context.Background(), sa); err != nil {
		return nil, err
	}
	sa.Name = removeMainSAPrefix(sa.Name)
	return sa, nil
}

// ListMainServiceAccounts gets main service accounts
func (p *ServiceAccountProvider) ListMainServiceAccounts(userInfo *provider.UserInfo, options *provider.ServiceAccountListOptions) ([]*kubermaticv1.User, error) {
	if userInfo == nil {
		return nil, kerrors.NewBadRequest("userInfo cannot be nil")
	}
	if options == nil {
		options = &provider.ServiceAccountListOptions{}
	}

	serviceAccounts := &kubermaticv1.UserList{}
	if err := p.clientPrivileged.List(context.Background(), serviceAccounts); err != nil {
		return nil, err
	}

	resultList := make([]*kubermaticv1.User, 0)
	for _, sa := range serviceAccounts.Items {
		if hasMainSAPrefix(sa.Name) && sa.Annotations != nil {
			if strings.EqualFold(sa.Annotations[ServiceAccountAnnotationOwner], userInfo.Email) {
				resultList = append(resultList, sa.DeepCopy())
			}
		}
	}

	for _, sa := range resultList {
		sa.Name = removeMainSAPrefix(sa.Name)
	}

	if len(options.ServiceAccountName) == 0 {
		return resultList, nil
	}

	filteredList := make([]*kubermaticv1.User, 0)
	for _, sa := range resultList {
		if sa.Spec.Name == options.ServiceAccountName {
			filteredList = append(filteredList, sa)
			break
		}
	}

	return filteredList, nil
}

// GetMainServiceAccount gets the main service account
func (p *ServiceAccountProvider) GetMainServiceAccount(userInfo *provider.UserInfo, name string, options *provider.ServiceAccountGetOptions) (*kubermaticv1.User, error) {
	if len(name) == 0 {
		return nil, kerrors.NewBadRequest("service account name cannot be empty")
	}
	if options == nil {
		options = &provider.ServiceAccountGetOptions{RemovePrefix: true}
	}

	name = addMainSAPrefix(name)
	serviceAccount := &kubermaticv1.User{}
	if err := p.clientPrivileged.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: name}, serviceAccount); err != nil {
		return nil, err
	}

	if options.RemovePrefix {
		serviceAccount.Name = removeMainSAPrefix(serviceAccount.Name)
	}

	if serviceAccount.Annotations == nil {
		return nil, kerrors.NewInternalError(fmt.Errorf("missing owner annotation"))
	}

	if !strings.EqualFold(serviceAccount.Annotations[ServiceAccountAnnotationOwner], userInfo.Email) {
		return nil, kerrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("actual user %s is not the owner of the service account %s", userInfo.Email, name))
	}

	return serviceAccount, nil
}

// UpdateUnsecuredProjectServiceAccount updated the project service account
func (p *ServiceAccountProvider) UpdateMainServiceAccount(userInfo *provider.UserInfo, serviceAccount *kubermaticv1.User) (*kubermaticv1.User, error) {
	if serviceAccount == nil {
		return nil, kerrors.NewBadRequest("service account name cannot be nil")
	}

	if serviceAccount.Annotations == nil {
		return nil, kerrors.NewInternalError(fmt.Errorf("missing owner annotation"))
	}

	if !strings.EqualFold(serviceAccount.Annotations[ServiceAccountAnnotationOwner], userInfo.Email) {
		return nil, kerrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("actual user %s is not the owner of the service account %s", userInfo.Email, serviceAccount.Name))
	}

	serviceAccount.Name = addMainSAPrefix(serviceAccount.Name)

	if err := p.clientPrivileged.Update(context.Background(), serviceAccount); err != nil {
		return nil, err
	}
	serviceAccount.Name = removeMainSAPrefix(serviceAccount.Name)
	return serviceAccount, nil
}

// DeleteMainServiceAccount deletes main service account
func (p *ServiceAccountProvider) DeleteMainServiceAccount(userInfo *provider.UserInfo, serviceAccount *kubermaticv1.User) error {
	if serviceAccount == nil {
		return kerrors.NewBadRequest("service account cannot be nil")
	}

	if serviceAccount.Annotations == nil {
		return kerrors.NewInternalError(fmt.Errorf("missing owner annotation"))
	}

	if !strings.EqualFold(serviceAccount.Annotations[ServiceAccountAnnotationOwner], userInfo.Email) {
		return kerrors.NewForbidden(schema.GroupResource{}, userInfo.Email, fmt.Errorf("actual user %s is not the owner of the service account %s", userInfo.Email, serviceAccount.Name))
	}

	serviceAccount.Name = addMainSAPrefix(serviceAccount.Name)
	return p.clientPrivileged.Delete(context.Background(), &kubermaticv1.User{
		ObjectMeta: metav1.ObjectMeta{Name: serviceAccount.Name},
	})
}

func genMainServiceAccount(name, group, domain, owner string) *kubermaticv1.User {
	uniqueID := rand.String(10)
	uniqueName := fmt.Sprintf("%s%s", mainSAPrefix, uniqueID)

	sa := &kubermaticv1.User{}
	sa.Name = uniqueName
	sa.Spec.Email = fmt.Sprintf("%s@%s", uniqueName, domain)
	sa.Spec.Name = name
	sa.Spec.ID = uniqueID
	sa.Annotations = map[string]string{ServiceAccountAnnotationOwner: owner}
	sa.Labels = map[string]string{ServiceAccountLabelGroup: group}
	return sa
}

// removeProjectSAPrefix removes "main-serviceaccount-" from a SA's ID,
// for example given "main-serviceaccount-7d4b5695vb" it returns "7d4b5695vb"
func removeMainSAPrefix(id string) string {
	return strings.TrimPrefix(id, mainSAPrefix)
}

// hasMainSAPrefix checks if the given id has "main-serviceaccount-" prefix
func hasMainSAPrefix(sa string) bool {
	return strings.HasPrefix(sa, mainSAPrefix)
}

// addMainSAPrefix adds "main-serviceaccount-" prefix to a SA's ID,
// for example given "7d4b5695vb" it returns "main-serviceaccount-7d4b5695vb"
func addMainSAPrefix(id string) string {
	if !hasMainSAPrefix(id) {
		return fmt.Sprintf("%s%s", mainSAPrefix, id)
	}
	return id
}
