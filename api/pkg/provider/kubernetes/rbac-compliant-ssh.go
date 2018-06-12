package kubernetes

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"

	kubermaticclientv1 "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned/typed/kubermatic/v1"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"github.com/kubermatic/kubermatic/api/pkg/uuid"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	restclient "k8s.io/client-go/rest"
)

// NewRBACCompliantSSHKeyProvider returns a new ssh key provider that respects RBAC policies
// it uses createMasterImpersonatedClient to create a connection that uses User Impersonation
func NewRBACCompliantSSHKeyProvider(
	createMasterImpersonatedClient kubermaticImpersonationClient) (*RBACCompliantSSHKeyProvider, error) {
	return &RBACCompliantSSHKeyProvider{
		createMasterImpersonatedClient: createMasterImpersonatedClient,
	}, nil
}

// RBACCompliantSSHKeyProvider struct that holds required components in order to provide
// ssh key provider that is RBAC compliant
type RBACCompliantSSHKeyProvider struct {
	// createMasterImpersonatedClient is used as a ground for impersonation
	// whenever a connection to Seed API server is required
	createMasterImpersonatedClient kubermaticImpersonationClient

	// keyLister provide access to local cache that stores ssh keys objects
	keyLister kubermaticv1lister.UserSSHKeyLister
}

// Create creates a ssh key that will belong to the given project
func (p *RBACCompliantSSHKeyProvider) Create(keyName, pubKey string, user *kubermaticapiv1.User, project *kubermaticapiv1.Project) (*kubermaticapiv1.UserSSHKey, error) {
	if keyName == "" {
		return nil, fmt.Errorf("the ssh key name is missing but required")
	}
	if pubKey == "" {
		return nil, fmt.Errorf("the ssh public part of the key is missing but required")
	}
	if user == nil {
		return nil, errors.New("a user is missing but required")
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

	masterImpersonatedClient, err := p.createMasterImpersonationClientWrapper(user, project)
	if err != nil {
		return nil, err
	}
	return masterImpersonatedClient.UserSSHKeies().Create(sshKey)
}

// ListOptions allows to set filters that will be applied
// to filter the result of List method.
type ListOptions struct {
	// ClusterName gets the keys that are being used by the given cluster name
	ClusterName string
}

// List gets a list of ssh keys, by default it will get all the keys that belong to the given project.
// If you want to filter the result please take a look at ListOptions
//
// Note:
// After we get the list of the keys we could try to get each individually using unprivileged account to see if the user have read access,
// We don't do this because we assume that if the user was able to get the project (argument) it has to have at least read access.
func (p *RBACCompliantSSHKeyProvider) List(project *kubermaticapiv1.Project, user *kubermaticapiv1.User, options *ListOptions) ([]*kubermaticapiv1.UserSSHKey, error) {
	if project == nil || user == nil {
		return nil, errors.New("a project or/and a user is missing but required")
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
	if options != nil && len(options.ClusterName) == 0 {
		return projectKeys, nil
	}

	filteredKeys := []*kubermaticapiv1.UserSSHKey{}
	for _, key := range projectKeys {

		if key.Spec.Clusters == nil {
			continue
		}

		for _, actualClusterName := range key.Spec.Clusters {
			if actualClusterName == options.ClusterName {
				filteredKeys = append(filteredKeys, key)
			}
		}
	}
	return filteredKeys, nil

}

// Get returns a key with the given name
func (p *RBACCompliantSSHKeyProvider) Get(project *kubermaticapiv1.Project, user *kubermaticapiv1.User, keyName string) (*kubermaticapiv1.UserSSHKey, error) {
	masterImpersonatedClient, err := p.createMasterImpersonationClientWrapper(user, project)
	if err != nil {
		return nil, err
	}
	return masterImpersonatedClient.UserSSHKeies().Get(keyName, metav1.GetOptions{})
}

// Delete simply deletes the given key
func (p *RBACCompliantSSHKeyProvider) Delete(project *kubermaticapiv1.Project, user *kubermaticapiv1.User, keyName string) error {
	masterImpersonatedClient, err := p.createMasterImpersonationClientWrapper(user, project)
	if err != nil {
		return err
	}
	return masterImpersonatedClient.UserSSHKeies().Delete(keyName, &metav1.DeleteOptions{})
}

// Update simply updates the given key
func (p *RBACCompliantSSHKeyProvider) Update(user *kubermaticapiv1.User, project *kubermaticapiv1.Project, newKey *kubermaticapiv1.UserSSHKey) (*kubermaticapiv1.UserSSHKey, error) {
	masterImpersonatedClient, err := p.createMasterImpersonationClientWrapper(user, project)
	if err != nil {
		return nil, err
	}
	return masterImpersonatedClient.UserSSHKeies().Update(newKey)
}

// createMasterImpersonationClientWrapper is a helper method that spits back kubermatic client that uses user impersonation
func (p *RBACCompliantSSHKeyProvider) createMasterImpersonationClientWrapper(user *kubermaticapiv1.User, project *kubermaticapiv1.Project) (kubermaticclientv1.KubermaticV1Interface, error) {
	if user == nil || project == nil {
		return nil, errors.New("a project and/or a user is missing but required")
	}
	groupName, err := user.GroupForProject(project.Name)
	if err != nil {
		return nil, kerrors.NewForbidden(schema.GroupResource{}, project.Name, err)
	}
	impersonationCfg := restclient.ImpersonationConfig{
		UserName: user.Spec.Email,
		Groups:   []string{groupName},
	}
	return p.createMasterImpersonatedClient(impersonationCfg)
}
