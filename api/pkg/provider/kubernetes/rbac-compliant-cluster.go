package kubernetes

import (
	"errors"
	"strings"

	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	clusterv1alpha1clientset "sigs.k8s.io/cluster-api/pkg/client/clientset_generated/clientset"
)

// NewRBACCompliantClusterProvider returns a new cluster provider that respects RBAC policies
// it uses createSeedImpersonatedClient to create a connection that uses user impersonation
func NewRBACCompliantClusterProvider(
	createSeedImpersonatedClient kubermaticImpersonationClient,
	userClusterConnProvider UserClusterConnectionProvider,
	clusterLister kubermaticv1lister.ClusterLister,
	workerName string) *RBACCompliantClusterProvider {
	return &RBACCompliantClusterProvider{
		createSeedImpersonatedClient: createSeedImpersonatedClient,
		userClusterConnProvider:      userClusterConnProvider,
		clusterLister:                clusterLister,
		workerName:                   workerName,
	}
}

// RBACCompliantClusterProvider struct that holds required components in order to provide
// cluster provided that is RBAC compliant
type RBACCompliantClusterProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	// whenever a connection to Seed API server is required
	createSeedImpersonatedClient kubermaticImpersonationClient

	// userClusterConnProvider used for obtaining a connection to the client's cluster
	userClusterConnProvider UserClusterConnectionProvider

	// clusterLister provide access to local cache that stores cluster objects
	clusterLister kubermaticv1lister.ClusterLister

	workerName string
}

// New creates a brand new cluster that is bound to the given project
func (p *RBACCompliantClusterProvider) New(project *kubermaticapiv1.Project, userInfo *provider.UserInfo, spec *kubermaticapiv1.ClusterSpec) (*kubermaticapiv1.Cluster, error) {
	if project == nil || userInfo == nil || spec == nil {
		return nil, errors.New("project and/or userInfo and/or spec is missing but required")
	}
	spec.HumanReadableName = strings.TrimSpace(spec.HumanReadableName)

	labels := map[string]string{
		kubermaticapiv1.ProjectIDLabelKey: project.Name,
	}
	if len(p.workerName) > 0 {
		labels[kubermaticapiv1.WorkerNameLabelKey] = p.workerName
	}

	name := rand.String(10)
	cluster := &kubermaticapiv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Labels: labels,
			Name:   name,
		},
		Spec: *spec,
		Status: kubermaticapiv1.ClusterStatus{
			UserEmail:     userInfo.Email,
			NamespaceName: NamespaceName(name),
		},
		Address: kubermaticapiv1.ClusterAddress{},
	}

	seedImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}
	cluster, err = seedImpersonatedClient.Clusters().Create(cluster)
	if err != nil {
		return nil, err
	}

	return cluster, nil
}

// List gets all clusters that belong to the given project
// If you want to filter the result please take a look at ClusterListOptions
//
// Note:
// After we get the list of clusters we could try to get each cluster individually using unprivileged account to see if the user have read access,
// We don't do this because we assume that if the user was able to get the project (argument) it has to have at least read access.
func (p *RBACCompliantClusterProvider) List(project *kubermaticapiv1.Project, options *provider.ClusterListOptions) ([]*kubermaticapiv1.Cluster, error) {
	if project == nil {
		return nil, errors.New("project is missing but required")
	}
	clusters, err := p.clusterLister.List(labels.Everything())
	if err != nil {
		return nil, err
	}

	projectClusters := []*kubermaticapiv1.Cluster{}
	for _, cluster := range clusters {
		if clusterProject := cluster.GetLabels()[kubermaticapiv1.ProjectIDLabelKey]; clusterProject == project.Name {
			projectClusters = append(projectClusters, cluster)
		}
	}

	if options == nil {
		return projectClusters, nil
	}
	if len(options.ClusterSpecName) == 0 {
		return projectClusters, nil
	}

	filteredProjectClusters := []*kubermaticapiv1.Cluster{}
	for _, projectCluster := range projectClusters {
		if projectCluster.Spec.HumanReadableName == options.ClusterSpecName {
			filteredProjectClusters = append(filteredProjectClusters, projectCluster)
		}
	}

	return filteredProjectClusters, nil
}

// Get returns the given cluster, it uses the projectInternalName to determine the group the user belongs to
func (p *RBACCompliantClusterProvider) Get(userInfo *provider.UserInfo, clusterName string, options *provider.ClusterGetOptions) (*kubermaticapiv1.Cluster, error) {
	seedImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	cluster, err := seedImpersonatedClient.Clusters().Get(clusterName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if options.CheckInitStatus {
		isHealthy := cluster.Status.Health.Apiserver &&
			cluster.Status.Health.Scheduler &&
			cluster.Status.Health.Controller &&
			cluster.Status.Health.MachineController &&
			cluster.Status.Health.Etcd
		if !isHealthy {
			return nil, kerrors.NewServiceUnavailable("Cluster components are not ready yet")
		}
	}
	return cluster, nil
}

// Delete deletes the given cluster
func (p *RBACCompliantClusterProvider) Delete(userInfo *provider.UserInfo, clusterName string) error {
	seedImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return err
	}

	return seedImpersonatedClient.Clusters().Delete(clusterName, &metav1.DeleteOptions{})
}

// Update updates a cluster
func (p *RBACCompliantClusterProvider) Update(userInfo *provider.UserInfo, newCluster *kubermaticapiv1.Cluster) (*kubermaticapiv1.Cluster, error) {
	seedImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	return seedImpersonatedClient.Clusters().Update(newCluster)
}

// GetAdminKubeconfigForCustomerCluster returns the admin kubeconfig for the given cluster
func (p *RBACCompliantClusterProvider) GetAdminKubeconfigForCustomerCluster(c *kubermaticapiv1.Cluster) (*clientcmdapi.Config, error) {
	b, err := p.userClusterConnProvider.GetAdminKubeconfig(c)
	if err != nil {
		return nil, err
	}

	return clientcmd.Load(b)
}

// GetMachineClientForCustomerCluster returns a client to interact with machine resources in the given cluster
//
// Note that the client you will get has admin privileges
func (p *RBACCompliantClusterProvider) GetMachineClientForCustomerCluster(c *kubermaticapiv1.Cluster) (clusterv1alpha1clientset.Interface, error) {
	return p.userClusterConnProvider.GetMachineClient(c)
}

// GetKubernetesClientForCustomerCluster returns a client to interact with the given cluster
//
// Note that the client you will get has admin privileges
func (p *RBACCompliantClusterProvider) GetKubernetesClientForCustomerCluster(c *kubermaticapiv1.Cluster) (kubernetes.Interface, error) {
	return p.userClusterConnProvider.GetClient(c)
}
