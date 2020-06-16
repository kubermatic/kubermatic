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

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ServiceAccountLabelGroup = "initialGroup"
	saPrefix                 = "serviceaccount-"
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

// Create creates a new service account
func (p *ServiceAccountProvider) Create(userInfo *provider.UserInfo, project *kubermaticv1.Project, name, group string) (*kubermaticv1.User, error) {
	if project == nil {
		return nil, kerrors.NewBadRequest("Project cannot be nil")
	}
	if len(name) == 0 || len(group) == 0 {
		return nil, kerrors.NewBadRequest("Service account name and group cannot be empty when creating a new SA resource")
	}

	sa := genServiceAccount(project, name, group, p.domain)

	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}

	if err := masterImpersonatedClient.Create(context.Background(), sa); err != nil {
		return nil, err
	}
	sa.Name = removeSAPrefix(sa.Name)
	return sa, nil
}

// CreateUnsecured creates a new service accounts
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to create the resources
func (p *ServiceAccountProvider) CreateUnsecured(project *kubermaticv1.Project, name, group string) (*kubermaticv1.User, error) {
	if project == nil {
		return nil, kerrors.NewBadRequest("Project cannot be nil")
	}
	if len(name) == 0 || len(group) == 0 {
		return nil, kerrors.NewBadRequest("Service account name and group cannot be empty when creating a new SA resource")
	}

	sa := genServiceAccount(project, name, group, p.domain)

	if err := p.clientPrivileged.Create(context.Background(), sa); err != nil {
		return nil, err
	}

	sa.Name = removeSAPrefix(sa.Name)
	return sa, nil
}

func genServiceAccount(project *kubermaticv1.Project, name, group, domain string) *kubermaticv1.User {
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

// List gets service accounts for the project
func (p *ServiceAccountProvider) List(userInfo *provider.UserInfo, project *kubermaticv1.Project, options *provider.ServiceAccountListOptions) ([]*kubermaticv1.User, error) {
	if userInfo == nil {
		return nil, kerrors.NewBadRequest("userInfo cannot be nil")
	}
	if project == nil {
		return nil, kerrors.NewBadRequest("project cannot be nil")
	}
	if options == nil {
		options = &provider.ServiceAccountListOptions{}
	}

	resultList, err := p.listSA(project)
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
		sa.Name = removeSAPrefix(sa.Name)
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

// ListUnsecured gets all service accounts
// If you want to filter the result please take a look at ServiceAccountListOptions
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to get the resources
func (p *ServiceAccountProvider) ListUnsecured(project *kubermaticv1.Project, options *provider.ServiceAccountListOptions) ([]*kubermaticv1.User, error) {
	if project == nil {
		return nil, kerrors.NewBadRequest("project cannot be nil")
	}
	if options == nil {
		options = &provider.ServiceAccountListOptions{}
	}

	resultList, err := p.listSA(project)
	if err != nil {
		return nil, err
	}

	for _, sa := range resultList {
		sa.Name = removeSAPrefix(sa.Name)
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

func (p *ServiceAccountProvider) listSA(project *kubermaticv1.Project) ([]*kubermaticv1.User, error) {
	serviceAccounts := &kubermaticv1.UserList{}
	if err := p.clientPrivileged.List(context.Background(), serviceAccounts); err != nil {
		return nil, err
	}

	resultList := make([]*kubermaticv1.User, 0)
	for _, sa := range serviceAccounts.Items {
		if hasSAPrefix(sa.Name) {
			for _, owner := range sa.GetOwnerReferences() {
				if owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() && owner.Kind == kubermaticv1.ProjectKindName && owner.Name == project.Name {
					resultList = append(resultList, sa.DeepCopy())
				}
			}
		}
	}
	return resultList, nil
}

// Get method returns service account with given name
func (p *ServiceAccountProvider) Get(userInfo *provider.UserInfo, name string, options *provider.ServiceAccountGetOptions) (*kubermaticv1.User, error) {
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

	name = addSAPrefix(name)
	serviceAccount := &kubermaticv1.User{}
	if err := masterImpersonatedClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: name}, serviceAccount); err != nil {
		return nil, err
	}

	if options.RemovePrefix {
		serviceAccount.Name = removeSAPrefix(serviceAccount.Name)
	}
	return serviceAccount, nil
}

// GetUnsecured gets all service accounts
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to get the resource
func (p *ServiceAccountProvider) GetUnsecured(name string, options *provider.ServiceAccountGetOptions) (*kubermaticv1.User, error) {
	if len(name) == 0 {
		return nil, kerrors.NewBadRequest("service account name cannot be empty")
	}
	if options == nil {
		options = &provider.ServiceAccountGetOptions{RemovePrefix: true}
	}

	name = addSAPrefix(name)
	serviceAccount := &kubermaticv1.User{}
	if err := p.clientPrivileged.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: name}, serviceAccount); err != nil {
		return nil, err
	}

	if options.RemovePrefix {
		serviceAccount.Name = removeSAPrefix(serviceAccount.Name)
	}
	return serviceAccount, nil
}

// Update simply updates the given service account
func (p *ServiceAccountProvider) Update(userInfo *provider.UserInfo, serviceAccount *kubermaticv1.User) (*kubermaticv1.User, error) {
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

	serviceAccount.Name = addSAPrefix(serviceAccount.Name)

	if err := masterImpersonatedClient.Update(context.Background(), serviceAccount); err != nil {
		return nil, err
	}
	serviceAccount.Name = removeSAPrefix(serviceAccount.Name)
	return serviceAccount, nil
}

// UpdateUnsecured gets all service accounts
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to update the resource
func (p *ServiceAccountProvider) UpdateUnsecured(serviceAccount *kubermaticv1.User) (*kubermaticv1.User, error) {
	if serviceAccount == nil {
		return nil, kerrors.NewBadRequest("service account name cannot be nil")
	}

	serviceAccount.Name = addSAPrefix(serviceAccount.Name)

	if err := p.clientPrivileged.Update(context.Background(), serviceAccount); err != nil {
		return nil, err
	}
	serviceAccount.Name = removeSAPrefix(serviceAccount.Name)
	return serviceAccount, nil
}

// Delete simply deletes the given service account
func (p *ServiceAccountProvider) Delete(userInfo *provider.UserInfo, name string) error {
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

	name = addSAPrefix(name)
	return masterImpersonatedClient.Delete(context.Background(), &kubermaticv1.User{ObjectMeta: metav1.ObjectMeta{Name: name}})
}

// DeleteUnsecured gets all service accounts
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to delete the resource
func (p *ServiceAccountProvider) DeleteUnsecured(name string) error {
	if len(name) == 0 {
		return kerrors.NewBadRequest("service account name cannot be empty")
	}

	name = addSAPrefix(name)
	return p.clientPrivileged.Delete(context.Background(), &kubermaticv1.User{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	})
}

// IsServiceAccounts determines whether the given email address
// belongs to a service account
func IsServiceAccount(email string) bool {
	return hasSAPrefix(email)
}

// removeSAPrefix removes "serviceaccount-" from a SA's ID,
// for example given "serviceaccount-7d4b5695vb" it returns "7d4b5695vb"
func removeSAPrefix(id string) string {
	return strings.TrimPrefix(id, saPrefix)
}

// addSAPrefix adds "serviceaccount-" prefix to a SA's ID,
// for example given "7d4b5695vb" it returns "serviceaccount-7d4b5695vb"
func addSAPrefix(id string) string {
	if !hasSAPrefix(id) {
		return fmt.Sprintf("%s%s", saPrefix, id)
	}
	return id
}

// hasSAPrefix checks if the given id has "serviceaccount-" prefix
func hasSAPrefix(sa string) bool {
	return strings.HasPrefix(sa, saPrefix)
}
