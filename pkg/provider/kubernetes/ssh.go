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
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/uuid"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// PrivilegedSSHKeyProvider represents a data structure that knows how to manage ssh keys in a privileged way.
type PrivilegedSSHKeyProvider struct {
	// treat clientPrivileged as a privileged user and use wisely
	clientPrivileged ctrlruntimeclient.Client
}

var _ provider.PrivilegedSSHKeyProvider = &PrivilegedSSHKeyProvider{}

// NewPrivilegedSSHKeyProvider returns a privileged ssh key provider.
func NewPrivilegedSSHKeyProvider(client ctrlruntimeclient.Client) (*PrivilegedSSHKeyProvider, error) {
	return &PrivilegedSSHKeyProvider{
		clientPrivileged: client,
	}, nil
}

// NewSSHKeyProvider returns a new ssh key provider that respects RBAC policies
// it uses createMasterImpersonatedClient to create a connection that uses User Impersonation.
func NewSSHKeyProvider(createMasterImpersonatedClient ImpersonationClient, client ctrlruntimeclient.Client) *SSHKeyProvider {
	return &SSHKeyProvider{createMasterImpersonatedClient: createMasterImpersonatedClient, client: client}
}

// SSHKeyProvider struct that holds required components in order to provide
// ssh key provider that is RBAC compliant.
type SSHKeyProvider struct {
	// createMasterImpersonatedClient is used as a ground for impersonation
	// whenever a connection to Seed API server is required
	createMasterImpersonatedClient ImpersonationClient

	client ctrlruntimeclient.Client
}

var _ provider.SSHKeyProvider = &SSHKeyProvider{}

// Create creates a ssh key that will belong to the given project.
func (p *SSHKeyProvider) Create(ctx context.Context, userInfo *provider.UserInfo, project *kubermaticv1.Project, keyName, pubKey string) (*kubermaticv1.UserSSHKey, error) {
	if keyName == "" {
		return nil, fmt.Errorf("the ssh key name is missing but required")
	}
	if pubKey == "" {
		return nil, fmt.Errorf("the ssh public part of the key is missing but required")
	}
	if userInfo == nil {
		return nil, errors.New("a userInfo is missing but required")
	}

	sshKey, err := genUserSSHKey(project, keyName, pubKey)
	if err != nil {
		return nil, err
	}

	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	if err := masterImpersonatedClient.Create(ctx, sshKey); err != nil {
		return nil, err
	}
	return sshKey, nil
}

// Create creates a ssh key that belongs to the given project
// This function is unsafe in a sense that it uses privileged account to create the ssh key.
func (p *PrivilegedSSHKeyProvider) CreateUnsecured(ctx context.Context, project *kubermaticv1.Project, keyName, pubKey string) (*kubermaticv1.UserSSHKey, error) {
	if keyName == "" {
		return nil, fmt.Errorf("the ssh key name is missing but required")
	}
	if pubKey == "" {
		return nil, fmt.Errorf("the ssh public part of the key is missing but required")
	}

	sshKey, err := genUserSSHKey(project, keyName, pubKey)
	if err != nil {
		return nil, err
	}

	if err := p.clientPrivileged.Create(ctx, sshKey); err != nil {
		return nil, err
	}
	return sshKey, nil
}

func genUserSSHKey(project *kubermaticv1.Project, keyName, pubKey string) (*kubermaticv1.UserSSHKey, error) {
	pubKeyParsed, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pubKey))
	if err != nil {
		return nil, fmt.Errorf("the provided ssh key is invalid: %w", err)
	}
	sshKeyHash := ssh.FingerprintLegacyMD5(pubKeyParsed)

	keyInternalName := fmt.Sprintf("key-%s-%s", strings.NewReplacer(":", "").Replace(sshKeyHash), uuid.ShortUID(4))
	return &kubermaticv1.UserSSHKey{
		ObjectMeta: metav1.ObjectMeta{
			Name: keyInternalName,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticv1.SchemeGroupVersion.String(),
					Kind:       kubermaticv1.ProjectKindName,
					UID:        project.GetUID(),
					Name:       project.Name,
				},
			},
		},
		Spec: kubermaticv1.SSHKeySpec{
			PublicKey:   pubKey,
			Fingerprint: sshKeyHash,
			Name:        keyName,
			Clusters:    []string{},
		},
	}, nil
}

// List gets a list of ssh keys, by default it will get all the keys that belong to the given project.
// If you want to filter the result please take a look at SSHKeyListOptions
//
// Note:
// After we get the list of the keys we could try to get each individually using unprivileged account to see if the user have read access,
// We don't do this because we assume that if the user was able to get the project (argument) it has to have at least read access.
func (p *SSHKeyProvider) List(ctx context.Context, project *kubermaticv1.Project, options *provider.SSHKeyListOptions) ([]*kubermaticv1.UserSSHKey, error) {
	if project == nil {
		return nil, errors.New("a project is missing but required")
	}
	allKeys := &kubermaticv1.UserSSHKeyList{}
	err := p.client.List(ctx, allKeys)
	if err != nil {
		return nil, err
	}

	projectKeys := []*kubermaticv1.UserSSHKey{}
	for _, key := range allKeys.Items {
		owners := key.GetOwnerReferences()
		for _, owner := range owners {
			if owner.APIVersion == kubermaticv1.SchemeGroupVersion.String() && owner.Kind == kubermaticv1.ProjectKindName && owner.Name == project.Name {
				projectKeys = append(projectKeys, key.DeepCopy())
			}
		}
	}

	if options == nil {
		return projectKeys, nil
	}
	if len(options.ClusterName) == 0 && len(options.SSHKeyName) == 0 {
		return projectKeys, nil
	}

	filteredKeys := []*kubermaticv1.UserSSHKey{}
	for _, key := range projectKeys {
		if len(options.SSHKeyName) != 0 {
			if key.Spec.Name == options.SSHKeyName {
				filteredKeys = append(filteredKeys, key)
			}
		}

		if key.Spec.Clusters == nil {
			continue
		}

		if len(options.ClusterName) != 0 {
			for _, actualClusterName := range key.Spec.Clusters {
				if actualClusterName == options.ClusterName {
					filteredKeys = append(filteredKeys, key)
				}
			}
		}
	}
	return filteredKeys, nil
}

// Get returns a key with the given name.
func (p *SSHKeyProvider) Get(ctx context.Context, userInfo *provider.UserInfo, keyName string) (*kubermaticv1.UserSSHKey, error) {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	userKey := &kubermaticv1.UserSSHKey{}
	if err := masterImpersonatedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: keyName}, userKey); err != nil {
		return nil, err
	}
	return userKey, nil
}

// Delete simply deletes the given key.
func (p *SSHKeyProvider) Delete(ctx context.Context, userInfo *provider.UserInfo, keyName string) error {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return err
	}
	return masterImpersonatedClient.Delete(ctx, &kubermaticv1.UserSSHKey{ObjectMeta: metav1.ObjectMeta{Name: keyName}})
}

// Delete deletes the given ssh key
// This function is unsafe in a sense that it uses privileged account to delete the ssh key.
func (p *PrivilegedSSHKeyProvider) DeleteUnsecured(ctx context.Context, keyName string) error {
	return p.clientPrivileged.Delete(ctx, &kubermaticv1.UserSSHKey{
		ObjectMeta: metav1.ObjectMeta{Name: keyName},
	})
}

// Update simply updates the given key.
func (p *SSHKeyProvider) Update(ctx context.Context, userInfo *provider.UserInfo, newKey *kubermaticv1.UserSSHKey) (*kubermaticv1.UserSSHKey, error) {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	if err := masterImpersonatedClient.Update(ctx, newKey); err != nil {
		return nil, err
	}
	return newKey, nil
}

// UpdateUnsecured update a specific ssh key and returns the updated ssh key
// This function is unsafe in a sense that it uses privileged account to update the ssh key.
func (p *PrivilegedSSHKeyProvider) UpdateUnsecured(ctx context.Context, sshKey *kubermaticv1.UserSSHKey) (*kubermaticv1.UserSSHKey, error) {
	if err := p.clientPrivileged.Update(ctx, sshKey); err != nil {
		return nil, err
	}
	return sshKey, nil
}

// GetUnsecured returns a key with the given name
// This function is unsafe in a sense that it uses privileged account to get the ssh key.
func (p *PrivilegedSSHKeyProvider) GetUnsecured(ctx context.Context, keyName string) (*kubermaticv1.UserSSHKey, error) {
	userSSHKey := &kubermaticv1.UserSSHKey{}
	if err := p.clientPrivileged.Get(ctx, ctrlruntimeclient.ObjectKey{Name: keyName}, userSSHKey); err != nil {
		return nil, err
	}
	return userSSHKey, nil
}
