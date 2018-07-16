package kubernetes

import (
	"fmt"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/ssh"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	sshKeyKind = "ssh-key"
)

// NewSSHKeyProvider returns a ssh key provider
func NewSSHKeyProvider(client kubermaticclientset.Interface, sshKeyLister kubermaticv1lister.UserSSHKeyLister, isAdmin func(apiv1.User) bool) *SSHKeyProvider {
	return &SSHKeyProvider{
		client:       client,
		sshKeyLister: sshKeyLister,
		isAdmin:      isAdmin,
	}
}

// SSHKeyProvider manages ssh key resources
type SSHKeyProvider struct {
	client       kubermaticclientset.Interface
	sshKeyLister kubermaticv1lister.UserSSHKeyLister
	isAdmin      func(apiv1.User) bool
}

// SSHKey returns a ssh key by name
func (p *SSHKeyProvider) SSHKey(user apiv1.User, name string) (*kubermaticv1.UserSSHKey, error) {
	k, err := p.sshKeyLister.Get(name)
	if err != nil {
		return nil, err
	}
	if k.Spec.Owner == user.ID || p.isAdmin(user) {
		return k, nil
	}
	return nil, errors.NewNotFound(sshKeyKind, name)
}

// SSHKeys returns the user ssh keys
func (p *SSHKeyProvider) SSHKeys(user apiv1.User) ([]*kubermaticv1.UserSSHKey, error) {
	allKeys, err := p.sshKeyLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}
	if p.isAdmin(user) {
		return allKeys, err
	}

	userkeys := []*kubermaticv1.UserSSHKey{}
	for _, key := range allKeys {
		if key.Spec.Owner == user.ID {
			userkeys = append(userkeys, key)
		}
	}

	return userkeys, nil
}

func (p *SSHKeyProvider) assignSSHKeyToCluster(user apiv1.User, name, cluster string) error {
	k, err := p.SSHKey(user, name)
	if err != nil {
		return err
	}
	k.AddToCluster(cluster)

	_, err = p.client.KubermaticV1().UserSSHKeies().Update(k)
	return err
}

// AssignSSHKeysToCluster assigns a ssh key to a cluster
func (p *SSHKeyProvider) AssignSSHKeysToCluster(user apiv1.User, names []string, cluster string) error {
	for _, name := range names {
		if err := p.assignSSHKeyToCluster(user, name, cluster); err != nil {
			return fmt.Errorf("failed to assign key %s to cluster: %v", name, err)
		}
	}
	return nil
}

// ClusterSSHKeys returns the ssh keys of a cluster
func (p *SSHKeyProvider) ClusterSSHKeys(user apiv1.User, cluster string) ([]*kubermaticv1.UserSSHKey, error) {
	keys, err := p.SSHKeys(user)
	if err != nil {
		return nil, err
	}

	var clusterKeys []*kubermaticv1.UserSSHKey
	for _, k := range keys {
		if k.IsUsedByCluster(cluster) {
			clusterKeys = append(clusterKeys, k)
		}
	}
	return clusterKeys, nil
}

// CreateSSHKey creates a ssh key
func (p *SSHKeyProvider) CreateSSHKey(name, pubkey string, user apiv1.User) (*kubermaticv1.UserSSHKey, error) {
	key, err := ssh.NewUserSSHKeyBuilder().
		SetName(name).
		SetOwner(user.ID).
		SetRawKey(pubkey).
		Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build key: %v", err)
	}

	return p.client.KubermaticV1().UserSSHKeies().Create(key)
}

// DeleteSSHKey deletes a ssh key
func (p *SSHKeyProvider) DeleteSSHKey(name string, user apiv1.User) error {
	k, err := p.SSHKey(user, name)
	if err != nil {
		return err
	}
	return p.client.KubermaticV1().UserSSHKeies().Delete(k.Name, &metav1.DeleteOptions{})
}
