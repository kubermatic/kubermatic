package kubernetes

import (
	"fmt"

	"github.com/golang/glog"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	client "github.com/kubermatic/kubermatic/api/pkg/crd/client/master/clientset/versioned"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/ssh"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
)

const (
	userLabelKey      = "user"
	userEmailLabelKey = "email"
	userIDLabelKey    = "id"

	sshKeyKind = "ssh-key"
)

type kubernetesProvider struct {
	client client.Interface

	cps        map[string]provider.CloudProvider
	dcs        map[string]provider.DatacenterMeta
	workerName string
}

// NewKubernetesProvider creates a new kubernetes provider object
func NewKubernetesProvider(
	crdClient client.Interface,
	cps map[string]provider.CloudProvider,
	workerName string,
	dcs map[string]provider.DatacenterMeta,
) provider.DataProvider {
	return &kubernetesProvider{
		cps:        cps,
		client:     crdClient,
		workerName: workerName,
		dcs:        dcs,
	}
}

func (p *kubernetesProvider) NewClusterWithCloud(user apiv1.User, spec *kubermaticv1.ClusterSpec) (*kubermaticv1.Cluster, error) {
	clusters, err := p.Clusters(user)
	if err != nil {
		return nil, err
	}

	// sanity checks for a fresh cluster
	switch {
	case user.ID == "":
		return nil, errors.NewBadRequest("cluster user is required")
	case spec.HumanReadableName == "":
		return nil, errors.NewBadRequest("cluster humanReadableName is required")
	}

	clusterName := rand.String(9)

	for _, c := range clusters {
		if c.Spec.HumanReadableName == spec.HumanReadableName {
			return nil, errors.NewAlreadyExists("cluster", spec.HumanReadableName)
		}
	}

	if spec.SeedDatacenterName == "" {
		dc, found := p.dcs[spec.Cloud.DatacenterName]
		if !found {
			return nil, errors.NewBadRequest("Unknown datacenter")
		}
		spec.SeedDatacenterName = dc.Seed
	}

	c := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterName,
			Labels: map[string]string{
				kubermaticv1.WorkerNameLabelKey: p.workerName,
				userLabelKey:                    user.ID,
			},
		},
		Spec: *spec,
		Status: kubermaticv1.ClusterStatus{
			Seed:          spec.SeedDatacenterName,
			NamespaceName: NamespaceName(clusterName),
			UserEmail:     user.Email,
			UserName:      user.Name,
		},
		Address: &kubermaticv1.ClusterAddress{},
	}

	c.Spec.WorkerName = p.workerName
	_, prov, err := provider.ClusterCloudProvider(p.cps, c)
	if err != nil {
		return nil, err
	}

	if err = prov.ValidateCloudSpec(c.Spec.Cloud); err != nil {
		return nil, fmt.Errorf("cloud provider data could not be validated successfully: %v", err)
	}

	c, err = p.client.KubermaticV1().Clusters().Create(c)
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (p *kubernetesProvider) Cluster(user apiv1.User, cluster string) (*kubermaticv1.Cluster, error) {
	c, err := p.client.KubermaticV1().Clusters().Get(cluster, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if c.Labels[userLabelKey] != user.ID && !auth.IsAdmin(&user) {
		return nil, errors.NewNotAuthorized()
	}
	return c, nil
}

func (p *kubernetesProvider) Clusters(user apiv1.User) ([]*kubermaticv1.Cluster, error) {
	filter := map[string]string{}
	if !auth.IsAdmin(&user) {
		filter[userLabelKey] = user.ID
	}
	selector := labels.SelectorFromSet(labels.Set(filter)).String()
	options := metav1.ListOptions{LabelSelector: selector, FieldSelector: labels.Everything().String()}
	clusterList, err := p.client.KubermaticV1().Clusters().List(options)
	if err != nil {
		return nil, err
	}
	res := []*kubermaticv1.Cluster{}
	for i := range clusterList.Items {
		res = append(res, &clusterList.Items[i])
	}
	return res, nil
}

func (p *kubernetesProvider) DeleteCluster(user apiv1.User, cluster string) error {
	// check permission by getting the cluster first
	c, err := p.Cluster(user, cluster)
	if err != nil {
		return err
	}

	return p.client.KubermaticV1().Clusters().Delete(c.Name, &metav1.DeleteOptions{})
}

func (p *kubernetesProvider) InitiateClusterUpgrade(user apiv1.User, name, version string) (*kubermaticv1.Cluster, error) {
	c, err := p.Cluster(user, name)
	if err != nil {
		return nil, err
	}

	c.Spec.MasterVersion = version
	return p.client.KubermaticV1().Clusters().Update(c)
}

func (p *kubernetesProvider) assignSSHKeyToCluster(user apiv1.User, name, cluster string) error {
	k, err := p.SSHKey(user, name)
	if err != nil {
		return err
	}
	k.AddToCluster(cluster)
	_, err = p.client.KubermaticV1().UserSSHKeies().Update(k)
	return err
}

// AssignSSHKeysToCluster assigns a ssh key to a cluster
func (p *kubernetesProvider) AssignSSHKeysToCluster(user apiv1.User, names []string, cluster string) error {
	for _, name := range names {
		if err := p.assignSSHKeyToCluster(user, name, cluster); err != nil {
			return fmt.Errorf("failed to assign key %s to cluster: %v", name, err)
		}
	}
	return nil
}

// ClusterSSHKeys returns the ssh keys of a cluster
func (p *kubernetesProvider) ClusterSSHKeys(user apiv1.User, cluster string) ([]*kubermaticv1.UserSSHKey, error) {
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
func (p *kubernetesProvider) SSHKeys(user apiv1.User) ([]*kubermaticv1.UserSSHKey, error) {
	opts := metav1.ListOptions{}
	var err error
	if !auth.IsAdmin(&user) {
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
func (p *kubernetesProvider) SSHKey(user apiv1.User, name string) (*kubermaticv1.UserSSHKey, error) {
	k, err := p.client.KubermaticV1().UserSSHKeies().Get(name, metav1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			return nil, errors.NewNotFound(sshKeyKind, name)
		}
		return nil, err
	}
	if k.Spec.Owner == user.ID || auth.IsAdmin(&user) {
		return k, nil
	}
	return nil, errors.NewNotFound(sshKeyKind, name)
}

// CreateSSHKey creates a ssh key
func (p *kubernetesProvider) CreateSSHKey(name, pubkey string, user apiv1.User) (*kubermaticv1.UserSSHKey, error) {
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
func (p *kubernetesProvider) DeleteSSHKey(name string, user apiv1.User) error {
	k, err := p.SSHKey(user, name)
	if err != nil {
		return err
	}
	err = p.client.KubermaticV1().UserSSHKeies().Delete(k.Name, &metav1.DeleteOptions{})
	if kerrors.IsNotFound(err) {

		return errors.NewNotFound(sshKeyKind, k.Name)
	}
	return err
}

func (p *kubernetesProvider) UserByEmail(email string) (*kubermaticv1.User, error) {
	selector, err := labels.Parse(fmt.Sprintf("%s=%s", userEmailLabelKey, kubernetes.ToLabelValue(email)))
	if err != nil {
		return nil, err
	}

	userList, err := p.client.KubermaticV1().Users().List(metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return nil, err
	}
	if len(userList.Items) != 1 || userList.Items[0].Spec.Email != email {
		return nil, provider.ErrNotFound
	}

	return &userList.Items[0], nil
}

// CreateUser creates a user
func (p *kubernetesProvider) CreateUser(id, name, email string) (*kubermaticv1.User, error) {
	user := kubermaticv1.User{}
	user.Labels = map[string]string{
		userEmailLabelKey: kubernetes.ToLabelValue(email),
		userIDLabelKey:    kubernetes.ToLabelValue(id),
	}
	user.GenerateName = "user-"
	user.Spec.Email = email
	user.Spec.Name = name
	user.Spec.ID = id
	user.Spec.Groups = []string{}

	return p.client.KubermaticV1().Users().Create(&user)
}
