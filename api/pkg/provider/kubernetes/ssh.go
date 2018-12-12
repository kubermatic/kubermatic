package kubernetes

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"

	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/uuid"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

// NewSSHKeyProvider returns a new ssh key provider that respects RBAC policies
// it uses createMasterImpersonatedClient to create a connection that uses User Impersonation
func NewSSHKeyProvider(createMasterImpersonatedClient kubermaticImpersonationClient, keyLister kubermaticv1lister.UserSSHKeyLister) *SSHKeyProvider {
	return &SSHKeyProvider{createMasterImpersonatedClient: createMasterImpersonatedClient, keyLister: keyLister}
}

// SSHKeyProvider struct that holds required components in order to provide
// ssh key provider that is RBAC compliant
type SSHKeyProvider struct {
	// createMasterImpersonatedClient is used as a ground for impersonation
	// whenever a connection to Seed API server is required
	createMasterImpersonatedClient kubermaticImpersonationClient

	// keyLister provide access to local cache that stores ssh keys objects
	keyLister kubermaticv1lister.UserSSHKeyLister
}

// Create creates a ssh key that will belong to the given project
func (p *SSHKeyProvider) Create(userInfo *provider.UserInfo, project *kubermaticapiv1.Project, keyName, pubKey string) (*kubermaticapiv1.UserSSHKey, error) {
	if keyName == "" {
		return nil, fmt.Errorf("the ssh key name is missing but required")
	}
	if pubKey == "" {
		return nil, fmt.Errorf("the ssh public part of the key is missing but required")
	}
	if userInfo == nil {
		return nil, errors.New("a userInfo is missing but required")
	}

	pubKeyParsed, _, _, _, err := ssh.ParseAuthorizedKey([]byte(pubKey))
	if err != nil {
		return nil, fmt.Errorf("the provided ssh key is invalid due to = %v", err)
	}
	sshKeyHash := ssh.FingerprintLegacyMD5(pubKeyParsed)

	keyInternalName := fmt.Sprintf("key-%s-%s", strings.NewReplacer(":", "").Replace(sshKeyHash), uuid.ShortUID(4))
	sshKey := &kubermaticapiv1.UserSSHKey{
		ObjectMeta: metav1.ObjectMeta{
			Name: keyInternalName,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
					Kind:       kubermaticapiv1.ProjectKindName,
					UID:        project.GetUID(),
					Name:       project.Name,
				},
			},
		},
		Spec: kubermaticapiv1.SSHKeySpec{
			PublicKey:   pubKey,
			Fingerprint: sshKeyHash,
			Name:        keyName,
			Clusters:    []string{},
		},
	}

	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	return masterImpersonatedClient.UserSSHKeys().Create(sshKey)
}

// List gets a list of ssh keys, by default it will get all the keys that belong to the given project.
// If you want to filter the result please take a look at SSHKeyListOptions
//
// Note:
// After we get the list of the keys we could try to get each individually using unprivileged account to see if the user have read access,
// We don't do this because we assume that if the user was able to get the project (argument) it has to have at least read access.
func (p *SSHKeyProvider) List(project *kubermaticapiv1.Project, options *provider.SSHKeyListOptions) ([]*kubermaticapiv1.UserSSHKey, error) {
	if project == nil {
		return nil, errors.New("a project is missing but required")
	}
	allKeys, err := p.keyLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	projectKeys := []*kubermaticapiv1.UserSSHKey{}
	for _, key := range allKeys {
		owners := key.GetOwnerReferences()
		for _, owner := range owners {
			if owner.APIVersion == kubermaticapiv1.SchemeGroupVersion.String() && owner.Kind == kubermaticapiv1.ProjectKindName && owner.Name == project.Name {
				projectKeys = append(projectKeys, key)
			}
		}
	}

	if options == nil {
		return projectKeys, nil
	}
	if len(options.ClusterName) == 0 && len(options.SSHKeyName) == 0 {
		return projectKeys, nil
	}

	filteredKeys := []*kubermaticapiv1.UserSSHKey{}
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

// Get returns a key with the given name
func (p *SSHKeyProvider) Get(userInfo *provider.UserInfo, keyName string) (*kubermaticapiv1.UserSSHKey, error) {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	return masterImpersonatedClient.UserSSHKeys().Get(keyName, metav1.GetOptions{})
}

// Delete simply deletes the given key
func (p *SSHKeyProvider) Delete(userInfo *provider.UserInfo, keyName string) error {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return err
	}
	return masterImpersonatedClient.UserSSHKeys().Delete(keyName, &metav1.DeleteOptions{})
}

// Update simply updates the given key
func (p *SSHKeyProvider) Update(userInfo *provider.UserInfo, newKey *kubermaticapiv1.UserSSHKey) (*kubermaticapiv1.UserSSHKey, error) {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	return masterImpersonatedClient.UserSSHKeys().Update(newKey)
}
