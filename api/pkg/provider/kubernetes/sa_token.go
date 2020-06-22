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
	"fmt"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	v1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	kubev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/rest"
)

const (
	labelTokenName = "token"
	tokenPrefix    = "sa-token-"
)

// NewServiceAccountProvider returns a service account provider
func NewServiceAccountTokenProvider(kubernetesImpersonationClient kubernetesImpersonationClient, secretLister kubev1.SecretLister) (*ServiceAccountTokenProvider, error) {
	kubernetesClient, err := kubernetesImpersonationClient(rest.ImpersonationConfig{})
	if err != nil {
		return nil, err
	}

	return &ServiceAccountTokenProvider{
		kubernetesImpersonationClient: kubernetesImpersonationClient,
		secretLister:                  secretLister,
		kubernetesClientPrivileged:    kubernetesClient,
	}, nil
}

// ServiceAccountProvider manages service account resources
type ServiceAccountTokenProvider struct {
	// kubernetesImpersonationClient is used as a ground for impersonation
	kubernetesImpersonationClient kubernetesImpersonationClient

	secretLister kubev1.SecretLister

	// treat kubernetesClientPrivileged as a privileged user and use wisely
	kubernetesClientPrivileged kubernetes.Interface
}

// Create creates a new token for service account
func (p *ServiceAccountTokenProvider) Create(userInfo *provider.UserInfo, sa *kubermaticv1.User, projectID, tokenName, tokenID, token string) (*v1.Secret, error) {
	if userInfo == nil {
		return nil, kerrors.NewBadRequest("userInfo cannot be nil")
	}
	if sa == nil {
		return nil, kerrors.NewBadRequest("service account cannot be nil")
	}

	secret := genToken(sa, projectID, tokenName, tokenID, token)

	kubernetesImpersonatedClient, err := createKubernetesImpersonationClientWrapperFromUserInfo(userInfo, p.kubernetesImpersonationClient)
	if err != nil {
		return nil, kerrors.NewInternalError(err)
	}

	createdToken, err := kubernetesImpersonatedClient.CoreV1().Secrets(resources.KubermaticNamespace).Create(secret)
	if err != nil {
		return nil, err
	}
	createdToken.Name = removeTokenPrefix(createdToken.Name)
	return createdToken, nil
}

// CreateUnsecured creates a new token
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to create the resource
func (p *ServiceAccountTokenProvider) CreateUnsecured(sa *kubermaticv1.User, projectID, tokenName, tokenID, token string) (*v1.Secret, error) {
	if sa == nil {
		return nil, kerrors.NewBadRequest("service account cannot be nil")
	}

	secret := genToken(sa, projectID, tokenName, tokenID, token)

	kubernetesImpersonatedClient, err := p.kubernetesImpersonationClient(rest.ImpersonationConfig{})
	if err != nil {
		return nil, kerrors.NewInternalError(err)
	}

	createdToken, err := kubernetesImpersonatedClient.CoreV1().Secrets(resources.KubermaticNamespace).Create(secret)
	if err != nil {
		return nil, err
	}
	createdToken.Name = removeTokenPrefix(createdToken.Name)
	return createdToken, nil
}

func genToken(sa *kubermaticv1.User, projectID, tokenName, tokenID, token string) *v1.Secret {
	secret := &v1.Secret{}
	secret.Name = addTokenPrefix(tokenID)
	secret.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       kubermaticv1.UserKindName,
			UID:        sa.GetUID(),
			Name:       sa.Name,
		},
	}
	secret.Labels = map[string]string{
		kubermaticv1.ProjectIDLabelKey: projectID,
		"name":                         tokenName,
	}
	secret.Data = make(map[string][]byte)
	secret.Data[labelTokenName] = []byte(token)
	secret.Type = "Opaque"
	return secret
}

// List  gets tokens for the given service account and project
func (p *ServiceAccountTokenProvider) List(userInfo *provider.UserInfo, project *kubermaticv1.Project, sa *kubermaticv1.User, options *provider.ServiceAccountTokenListOptions) ([]*v1.Secret, error) {
	if userInfo == nil {
		return nil, kerrors.NewBadRequest("userInfo cannot be nil")
	}
	if project == nil {
		return nil, kerrors.NewBadRequest("project cannot be nil")
	}
	if sa == nil {
		return nil, kerrors.NewBadRequest("sa cannot be nil")
	}
	if options == nil {
		options = &provider.ServiceAccountTokenListOptions{}
	}

	labelSelector, err := labels.Parse(fmt.Sprintf("%s=%s", kubermaticv1.ProjectIDLabelKey, project.Name))
	if err != nil {
		return nil, err
	}
	allSecrets, err := p.secretLister.List(labelSelector)
	if err != nil {
		return nil, err
	}

	resultList := make([]*v1.Secret, 0)
	for _, secret := range allSecrets {
		if isToken(secret) {
			for _, owner := range secret.GetOwnerReferences() {
				if owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() && owner.Kind == kubermaticv1.UserKindName &&
					owner.Name == sa.Name && owner.UID == sa.UID {
					resultList = append(resultList, secret.DeepCopy())
				}
			}
		}
	}

	// Note:
	// After we get the list of tokens we try to get at least one item using unprivileged account to see if the token have read access
	if len(resultList) > 0 {
		kubernetesImpersonatedClient, err := createKubernetesImpersonationClientWrapperFromUserInfo(userInfo, p.kubernetesImpersonationClient)
		if err != nil {
			return nil, err
		}

		tokenToGet := resultList[0]
		_, err = kubernetesImpersonatedClient.CoreV1().Secrets(resources.KubermaticNamespace).Get(tokenToGet.Name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
	}

	for _, token := range resultList {
		token.Name = removeTokenPrefix(token.Name)
	}

	if len(options.TokenID) == 0 {
		return resultList, nil
	}

	filteredList := make([]*v1.Secret, 0)
	for _, token := range resultList {
		name, ok := token.Labels["name"]
		if ok {
			if name == options.TokenID {
				filteredList = append(filteredList, token)
				break
			}
		}
	}

	return filteredList, nil
}

// ListUnsecured returns all tokens in kubermatic namespace
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to get the resource
// gets resources from the cache
func (p *ServiceAccountTokenProvider) ListUnsecured(options *provider.ServiceAccountTokenListOptions) ([]*v1.Secret, error) {
	labelSelector := labels.Everything()
	if options != nil {
		if options.LabelSelector != nil {
			labelSelector = options.LabelSelector
		}
	}
	allSecrets, err := p.secretLister.List(labelSelector)
	if err != nil {
		return nil, err
	}
	allTokens := []*v1.Secret{}
	for _, secret := range allSecrets {
		if isToken(secret) {
			sCpy := secret.DeepCopy()
			sCpy.Name = removeTokenPrefix(sCpy.Name)
			allTokens = append(allTokens, sCpy)
		}
	}
	if options == nil {
		return allTokens, nil
	}
	if options.TokenID != "" {
		for _, token := range allTokens {
			if token.Name == options.TokenID {
				return []*v1.Secret{token}, nil
			}
		}
		return nil, kerrors.NewNotFound(v1.SchemeGroupVersion.WithResource("secret").GroupResource(), options.TokenID)
	}

	if options.ServiceAccountID != "" {
		resultList := make([]*v1.Secret, 0)
		for _, token := range allTokens {
			for _, owner := range token.GetOwnerReferences() {
				if owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() && owner.Kind == kubermaticv1.UserKindName &&
					owner.Name == options.ServiceAccountID {
					resultList = append(resultList, token.DeepCopy())
				}
			}
		}
		return resultList, nil
	}

	return allTokens, nil
}

func isToken(secret *v1.Secret) bool {
	if secret == nil {
		return false
	}
	return strings.HasPrefix(secret.Name, "sa-token")
}

// Get method returns token by name
func (p *ServiceAccountTokenProvider) Get(userInfo *provider.UserInfo, name string) (*v1.Secret, error) {
	if userInfo == nil {
		return nil, kerrors.NewBadRequest("userInfo cannot be nil")
	}
	if len(name) == 0 {
		return nil, kerrors.NewBadRequest("token name cannot be empty")
	}
	name = addTokenPrefix(name)

	kubernetesImpersonatedClient, err := createKubernetesImpersonationClientWrapperFromUserInfo(userInfo, p.kubernetesImpersonationClient)
	if err != nil {
		return nil, kerrors.NewInternalError(err)
	}

	token, err := kubernetesImpersonatedClient.CoreV1().Secrets(resources.KubermaticNamespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	token.Name = removeTokenPrefix(token.Name)
	return token, nil
}

// GetUnsecured gets the token by name
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to get the resource
func (p *ServiceAccountTokenProvider) GetUnsecured(name string) (*v1.Secret, error) {
	if len(name) == 0 {
		return nil, kerrors.NewBadRequest("token name cannot be empty")
	}
	name = addTokenPrefix(name)

	kubernetesImpersonatedClient, err := p.kubernetesImpersonationClient(rest.ImpersonationConfig{})
	if err != nil {
		return nil, kerrors.NewInternalError(err)
	}

	token, err := kubernetesImpersonatedClient.CoreV1().Secrets(resources.KubermaticNamespace).Get(name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	token.Name = removeTokenPrefix(token.Name)
	return token, nil
}

// Update method updates given token
func (p *ServiceAccountTokenProvider) Update(userInfo *provider.UserInfo, secret *v1.Secret) (*v1.Secret, error) {
	if userInfo == nil {
		return nil, kerrors.NewBadRequest("userInfo cannot be nil")
	}
	if secret == nil {
		return nil, kerrors.NewBadRequest("secret cannot be empty")
	}
	secretCpy := *secret
	secretCpy.Name = addTokenPrefix(secretCpy.Name)

	kubernetesImpersonatedClient, err := createKubernetesImpersonationClientWrapperFromUserInfo(userInfo, p.kubernetesImpersonationClient)
	if err != nil {
		return nil, kerrors.NewInternalError(err)
	}

	updatedToken, err := kubernetesImpersonatedClient.CoreV1().Secrets(resources.KubermaticNamespace).Update(&secretCpy)
	if err != nil {
		return nil, err
	}
	updatedToken.Name = removeTokenPrefix(updatedToken.Name)
	return updatedToken, nil
}

// UpdateUnsecured updates the token
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to get the resource
func (p *ServiceAccountTokenProvider) UpdateUnsecured(secret *v1.Secret) (*v1.Secret, error) {
	if secret == nil {
		return nil, kerrors.NewBadRequest("secret cannot be empty")
	}
	secretCpy := *secret
	secretCpy.Name = addTokenPrefix(secretCpy.Name)

	kubernetesImpersonatedClient, err := p.kubernetesImpersonationClient(rest.ImpersonationConfig{})
	if err != nil {
		return nil, kerrors.NewInternalError(err)
	}

	updatedToken, err := kubernetesImpersonatedClient.CoreV1().Secrets(resources.KubermaticNamespace).Update(&secretCpy)
	if err != nil {
		return nil, err
	}
	updatedToken.Name = removeTokenPrefix(updatedToken.Name)
	return updatedToken, nil
}

// Delete method deletes given token
func (p *ServiceAccountTokenProvider) Delete(userInfo *provider.UserInfo, name string) error {
	if userInfo == nil {
		return kerrors.NewBadRequest("userInfo cannot be nil")
	}
	if len(name) == 0 {
		return kerrors.NewBadRequest("token name cannot be empty")
	}
	name = addTokenPrefix(name)

	kubernetesImpersonatedClient, err := createKubernetesImpersonationClientWrapperFromUserInfo(userInfo, p.kubernetesImpersonationClient)
	if err != nil {
		return kerrors.NewInternalError(err)
	}
	return kubernetesImpersonatedClient.CoreV1().Secrets(resources.KubermaticNamespace).Delete(name, &metav1.DeleteOptions{})
}

// DeleteUnsecured deletes the token
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to delete the resource
func (p *ServiceAccountTokenProvider) DeleteUnsecured(name string) error {
	if len(name) == 0 {
		return kerrors.NewBadRequest("token name cannot be empty")
	}
	name = addTokenPrefix(name)

	kubernetesImpersonatedClient, err := p.kubernetesImpersonationClient(rest.ImpersonationConfig{})
	if err != nil {
		return kerrors.NewInternalError(err)
	}
	return kubernetesImpersonatedClient.CoreV1().Secrets(resources.KubermaticNamespace).Delete(name, &metav1.DeleteOptions{})
}

// removeTokenPrefix removes "sa-token-" from a token's ID
// for example given "sa-token-gmtzqz692d" it returns "gmtzqz692d"
func removeTokenPrefix(id string) string {
	return strings.TrimPrefix(id, tokenPrefix)
}

// addTokenPrefix adds "sa-token-" prefix to a token's ID,
// for example given "gmtzqz692d" it returns "sa-token-gmtzqz692d"
func addTokenPrefix(id string) string {
	if !hasTokenPrefix(id) {
		return fmt.Sprintf("%s%s", tokenPrefix, id)
	}
	return id
}

// hasTokenPrefix checks if the given id has "sa-token-" prefix
func hasTokenPrefix(token string) bool {
	return strings.HasPrefix(token, tokenPrefix)
}
