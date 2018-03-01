package kubernetes

import (
	"strings"
	"time"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	kuberrrors "k8s.io/apimachinery/pkg/api/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/wait"
)

// NewClusterProvider returns a datacenter specific cluster provider
func NewClusterProvider(
	client kubermaticclientset.Interface,
	clusterLister kubermaticv1lister.ClusterLister,
	workerName string,
	isAdmin func(apiv1.User) bool) *ClusterProvider {
	return &ClusterProvider{
		client:        client,
		clusterLister: clusterLister,
		workerName:    workerName,
		isAdmin:       isAdmin,
	}
}

// ClusterProvider handles actions to create/modify/delete clusters in a specific kubernetes cluster
type ClusterProvider struct {
	client        kubermaticclientset.Interface
	clusterLister kubermaticv1lister.ClusterLister
	isAdmin       func(apiv1.User) bool

	workerName string
}

// NewCluster creates a new Cluster with the given ClusterSpec for the given user
func (p *ClusterProvider) NewCluster(user apiv1.User, spec *kubermaticv1.ClusterSpec) (*kubermaticv1.Cluster, error) {
	spec.HumanReadableName = strings.TrimSpace(spec.HumanReadableName)
	spec.WorkerName = p.workerName

	clusters, err := p.Clusters(user)
	if err != nil {
		return nil, err
	}

	for _, c := range clusters {
		if c.Spec.HumanReadableName == spec.HumanReadableName {
			return nil, errors.NewAlreadyExists("cluster", spec.HumanReadableName)
		}
	}

	// sanity checks for a fresh cluster
	switch {
	case user.ID == "":
		return nil, errors.NewBadRequest("user id is required")
	case spec.HumanReadableName == "":
		return nil, errors.NewBadRequest("cluster humanReadableName is required")
	}

	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				kubermaticv1.WorkerNameLabelKey: p.workerName,
				userLabelKey:                    user.ID,
			},
		},
		Spec: *spec,
		Status: kubermaticv1.ClusterStatus{
			UserEmail: user.Email,
			UserName:  user.Name,
		},
		Address: &kubermaticv1.ClusterAddress{},
	}

	cluster, err = p.client.KubermaticV1().Clusters().Create(cluster)
	if err != nil {
		if kuberrrors.IsAlreadyExists(err) {
			return nil, provider.ErrAlreadyExists
		}
		return nil, err
	}

	//We wait until the cluster exists in the lister so we can use this instead of doing api calls
	existsInLister := func() (bool, error) {
		_, err := p.clusterLister.Get(cluster.Name)
		if err != nil {
			return false, nil
		}
		return true, nil
	}

	return cluster, wait.Poll(10*time.Millisecond, 30*time.Second, existsInLister)
}

// Cluster returns the given cluster
func (p *ClusterProvider) Cluster(user apiv1.User, name string) (*kubermaticv1.Cluster, error) {
	cluster, err := p.clusterLister.Get(name)
	if err != nil {
		if kuberrrors.IsNotFound(err) {
			return nil, provider.ErrNotFound
		}
		return nil, err
	}
	if cluster.Labels[userLabelKey] == user.ID || p.isAdmin(user) {
		return cluster, nil
	}

	return nil, errors.NewNotAuthorized()
}

// Clusters returns all clusters for the given user
func (p *ClusterProvider) Clusters(user apiv1.User) ([]*kubermaticv1.Cluster, error) {
	var selector labels.Selector

	if p.isAdmin(user) {
		selector = labels.Everything()
	} else {
		filter := map[string]string{}
		filter[userLabelKey] = user.ID
		selector = labels.SelectorFromSet(labels.Set(filter))
	}

	return p.clusterLister.List(selector)
}

// DeleteCluster deletes the given cluster
func (p *ClusterProvider) DeleteCluster(user apiv1.User, name string) error {
	cluster, err := p.Cluster(user, name)
	if err != nil {
		return err
	}

	return p.client.KubermaticV1().Clusters().Delete(cluster.Name, &metav1.DeleteOptions{})
}

// InitiateClusterUpgrade sets the version of the given cluster
func (p *ClusterProvider) InitiateClusterUpgrade(user apiv1.User, name, version string) (*kubermaticv1.Cluster, error) {
	cluster, err := p.Cluster(user, name)
	if err != nil {
		return nil, err
	}

	cluster.Spec.MasterVersion = version

	return p.client.KubermaticV1().Clusters().Update(cluster)
}
