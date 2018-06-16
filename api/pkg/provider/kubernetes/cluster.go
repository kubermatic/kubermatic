package kubernetes

import (
	"strings"
	"time"

	"github.com/golang/glog"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticclientset "github.com/kubermatic/kubermatic/api/pkg/crd/client/clientset/versioned"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	machineclientset "github.com/kubermatic/machine-controller/pkg/client/clientset/versioned"

	corev1 "k8s.io/api/core/v1"
	kuberrrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// UserClusterConnectionProvider offers functions to interact with a user cluster
type UserClusterConnectionProvider interface {
	GetClient(*kubermaticv1.Cluster) (kubernetes.Interface, error)
	GetMachineClient(*kubermaticv1.Cluster) (machineclientset.Interface, error)
	GetAdminKubeconfig(c *kubermaticv1.Cluster) ([]byte, error)
}

// NewClusterProvider returns a datacenter specific cluster provider
func NewClusterProvider(
	client kubermaticclientset.Interface,
	userClusterConnProvider UserClusterConnectionProvider,
	clusterLister kubermaticv1lister.ClusterLister,
	workerName string,
	isAdmin func(apiv1.User) bool,
	addons []string) *ClusterProvider {
	return &ClusterProvider{
		client:                  client,
		userClusterConnProvider: userClusterConnProvider,
		clusterLister:           clusterLister,
		workerName:              workerName,
		isAdmin:                 isAdmin,
		addons:                  addons,
	}
}

// ClusterProvider handles actions to create/modify/delete clusters in a specific kubernetes cluster
type ClusterProvider struct {
	client                  kubermaticclientset.Interface
	userClusterConnProvider UserClusterConnectionProvider
	clusterLister           kubermaticv1lister.ClusterLister
	isAdmin                 func(apiv1.User) bool
	addons                  []string

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

	name := rand.String(10)
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				kubermaticv1.WorkerNameLabelKey: p.workerName,
				userLabelKey:                    user.ID,
			},
			Name: name,
		},
		Spec: *spec,
		Status: kubermaticv1.ClusterStatus{
			UserEmail:     user.Email,
			UserName:      user.Name,
			NamespaceName: NamespaceName(name),
		},
		Address: kubermaticv1.ClusterAddress{},
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

	go func() {
		gv := kubermaticv1.SchemeGroupVersion
		ownerRef := *metav1.NewControllerRef(cluster, gv.WithKind("Cluster"))
		err = wait.Poll(50*time.Millisecond, 60*time.Second, func() (done bool, err error) {
			for _, addon := range p.addons {
				_, err = p.client.KubermaticV1().Addons(cluster.Status.NamespaceName).Create(&kubermaticv1.Addon{
					ObjectMeta: metav1.ObjectMeta{
						Name:            addon,
						Namespace:       cluster.Status.NamespaceName,
						OwnerReferences: []metav1.OwnerReference{ownerRef},
					},
					Spec: kubermaticv1.AddonSpec{
						Name: addon,
						Cluster: corev1.ObjectReference{
							Name:       cluster.Name,
							Namespace:  "",
							UID:        cluster.UID,
							APIVersion: cluster.APIVersion,
							Kind:       "Cluster",
						},
					},
				})
				if err != nil {
					if kuberrrors.IsAlreadyExists(err) {
						continue
					}
					glog.V(0).Infof("failed to create initial adddon %s for cluster %s: %v", addon, cluster.Name, err)
					return false, nil
				}
			}

			return true, nil
		})
		if err != nil {
			glog.V(0).Infof("failed to create initial addons in cluster %s: %v", cluster.Name, err)
		}
	}()

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

// UpdateCluster updates a cluster
func (p *ClusterProvider) UpdateCluster(user apiv1.User, newCluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	_, err := p.Cluster(user, newCluster.Name)
	if err != nil {
		return nil, err
	}

	return p.client.KubermaticV1().Clusters().Update(newCluster)
}

// GetAdminKubeconfig returns the admin kubeconfig for the given cluster
func (p *ClusterProvider) GetAdminKubeconfig(c *kubermaticv1.Cluster) (*clientcmdapi.Config, error) {
	b, err := p.userClusterConnProvider.GetAdminKubeconfig(c)
	if err != nil {
		return nil, err
	}

	return clientcmd.Load(b)
}

// GetMachineClient returns a client to interact with machine resources in the given cluster
func (p *ClusterProvider) GetMachineClient(c *kubermaticv1.Cluster) (machineclientset.Interface, error) {
	return p.userClusterConnProvider.GetMachineClient(c)
}

// GetClient returns a client to interact with the given cluster
func (p *ClusterProvider) GetClient(c *kubermaticv1.Cluster) (kubernetes.Interface, error) {
	return p.userClusterConnProvider.GetClient(c)
}
