package kubermatic

import (
	"fmt"

	"github.com/golang/glog"
	mastercrdclient "github.com/kubermatic/kubermatic/api/pkg/crd/client/master/clientset/versioned"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/ssh"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type kubermaticProvider struct {
	client mastercrdclient.Interface
}

// New creates a new kubermaticProvider object
func New(client mastercrdclient.Interface) provider.DataProvider {
	return &kubermaticProvider{
		client: client,
	}
}

func (p *kubermaticProvider) assignSSHKeyToCluster(user auth.User, name, cluster string) error {
	k, err := p.SSHKey(user, name)
	if err != nil {
		return err
	}
	k.AddToCluster(cluster)
	_, err = p.client.KubermaticV1().UserSSHKeies().Update(k)
	return err
}

// AssignSSHKeysToCluster assigns a ssh key to a cluster
func (p *kubermaticProvider) AssignSSHKeysToCluster(user auth.User, names []string, cluster string) error {
	for _, name := range names {
		if err := p.assignSSHKeyToCluster(user, name, cluster); err != nil {
			return fmt.Errorf("failed to assign key %s to cluster: %v", name, err)
		}
	}
	return nil
}

// ClusterSSHKeys returns the ssh keys of a cluster
func (p *kubermaticProvider) ClusterSSHKeys(user auth.User, cluster string) ([]*kubermaticv1.UserSSHKey, error) {
	keys, err := p.SSHKeys(user)
	if err != nil {
		return nil, err
	}

	clusterKeys := []*kubermaticv1.UserSSHKey{}
	for _, k := range keys {
		if k.IsUsedByCluster(cluster) {
			clusterKeys = append(clusterKeys, k)
		}
	}
	return clusterKeys, nil
}

// SSHKeys returns the user ssh keys
func (p *kubermaticProvider) SSHKeys(user auth.User) ([]*kubermaticv1.UserSSHKey, error) {
	opts := metav1.ListOptions{}
	var err error
	if !user.IsAdmin() {
		opts, err = ssh.UserListOptions(user.ID)
		if err != nil {
			return nil, err
		}
	}

	glog.V(7).Infof("searching for users SSH keys with label selector: (%s)", opts.LabelSelector)
	list, err := p.client.KubermaticV1().UserSSHKeies().List(opts)
	if err != nil {
		return nil, err
	}

	res := []*kubermaticv1.UserSSHKey{}
	for i := range list.Items {
		res = append(res, &list.Items[i])
	}
	return res, nil
}

// SSHKey returns a ssh key by name
func (p *kubermaticProvider) SSHKey(user auth.User, name string) (*kubermaticv1.UserSSHKey, error) {
	k, err := p.client.KubermaticV1().UserSSHKeies().Get(name, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, errors.NewNotFound("ssh-key", k.Name)
		}
		return nil, err
	}
	if k.Spec.Owner == user.ID || user.IsAdmin() {
		return k, nil
	}
	return nil, errors.NewNotFound("ssh-key", name)
}

// CreateSSHKey creates a ssh key
func (p *kubermaticProvider) CreateSSHKey(name, pubkey string, user auth.User) (*kubermaticv1.UserSSHKey, error) {
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
func (p *kubermaticProvider) DeleteSSHKey(name string, user auth.User) error {
	k, err := p.SSHKey(user, name)
	if err != nil {
		return err
	}
	err = p.client.KubermaticV1().UserSSHKeies().Delete(k.Name, &metav1.DeleteOptions{})
	if kerrors.IsNotFound(err) {
		return errors.NewNotFound("ssh-key", k.Name)
	}
	return err
}
