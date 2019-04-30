package kubernetes

import (
	"errors"
	"strings"

	k8cuserclusterclient "github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	kubermaticv1lister "github.com/kubermatic/kubermatic/api/pkg/crd/client/listers/kubermatic/v1"
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// UserClusterConnectionProvider offers functions to interact with an user cluster
type UserClusterConnectionProvider interface {
	GetDynamicClient(*kubermaticapiv1.Cluster, ...k8cuserclusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
	GetAdminKubeconfig(*kubermaticapiv1.Cluster) ([]byte, error)
}

// extractGroupPrefixFunc is a function that knows how to extract a prefix (owners, editors) from "projectID-owners" group,
// group names inside leaf/user clusters don't have projectID in their names
type extractGroupPrefixFunc func(groupName string) string

// NewClusterProvider returns a new cluster provider that respects RBAC policies
// it uses createSeedImpersonatedClient to create a connection that uses user impersonation
func NewClusterProvider(
	createSeedImpersonatedClient kubermaticImpersonationClient,
	userClusterConnProvider UserClusterConnectionProvider,
	clusterLister kubermaticv1lister.ClusterLister,
	workerName string,
	extractGroupPrefix extractGroupPrefixFunc,
	client ctrlruntimeclient.Client,
	k8sClient kubernetes.Interface) *ClusterProvider {
	return &ClusterProvider{
		createSeedImpersonatedClient: createSeedImpersonatedClient,
		userClusterConnProvider:      userClusterConnProvider,
		clusterLister:                clusterLister,
		workerName:                   workerName,
		extractGroupPrefix:           extractGroupPrefix,
		client:                       client,
		k8sClient:                    k8sClient,
	}
}

// ClusterProvider struct that holds required components in order to provide
// cluster provided that is RBAC compliant
type ClusterProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	// whenever a connection to Seed API server is required
	createSeedImpersonatedClient kubermaticImpersonationClient

	// userClusterConnProvider used for obtaining a connection to the client's cluster
	userClusterConnProvider UserClusterConnectionProvider

	// clusterLister provide access to local cache that stores cluster objects
	clusterLister kubermaticv1lister.ClusterLister

	workerName         string
	extractGroupPrefix extractGroupPrefixFunc
	client             ctrlruntimeclient.Client
	k8sClient          kubernetes.Interface
}

// New creates a brand new cluster that is bound to the given project
func (p *ClusterProvider) New(project *kubermaticapiv1.Project, userInfo *provider.UserInfo, partialCluster *provider.PartialCluster) (*kubermaticapiv1.Cluster, error) {
	if project == nil || userInfo == nil || partialCluster == nil {
		return nil, errors.New("project and/or userInfo and/or partialCluster is missing but required")
	}
	partialCluster.ClusterSpec.HumanReadableName = strings.TrimSpace(partialCluster.ClusterSpec.HumanReadableName)

	labels := map[string]string{
		kubermaticapiv1.ProjectIDLabelKey: project.Name,
	}
	if len(p.workerName) > 0 {
		labels[kubermaticapiv1.WorkerNameLabelKey] = p.workerName
	}

	name := rand.String(10)
	cluster := &kubermaticapiv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: partialCluster.Annotations,
			Labels:      labels,
			Name:        name,
		},
		Spec: *partialCluster.ClusterSpec,
		Status: kubermaticapiv1.ClusterStatus{
			UserEmail:     userInfo.Email,
			NamespaceName: NamespaceName(name),
		},
		Address: kubermaticapiv1.ClusterAddress{},
	}

	seedImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
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
func (p *ClusterProvider) List(project *kubermaticapiv1.Project, options *provider.ClusterListOptions) ([]*kubermaticapiv1.Cluster, error) {
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
			projectClusters = append(projectClusters, cluster.DeepCopy())
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
func (p *ClusterProvider) Get(userInfo *provider.UserInfo, clusterName string, options *provider.ClusterGetOptions) (*kubermaticapiv1.Cluster, error) {
	seedImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	cluster, err := seedImpersonatedClient.Clusters().Get(clusterName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if options.CheckInitStatus {
		if !cluster.Status.Health.AllHealthy() {
			return nil, kerrors.NewServiceUnavailable("Cluster components are not ready yet")
		}
	}
	return cluster, nil
}

// Delete deletes the given cluster
func (p *ClusterProvider) Delete(userInfo *provider.UserInfo, clusterName string) error {
	seedImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return err
	}

	// Will delete all child's after the object is gone - otherwise the etcd might be deleted before all machines are gone
	// See https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/#controlling-how-the-garbage-collector-deletes-dependents
	policy := metav1.DeletePropagationBackground
	opts := metav1.DeleteOptions{PropagationPolicy: &policy}
	return seedImpersonatedClient.Clusters().Delete(clusterName, &opts)
}

// Update updates a cluster
func (p *ClusterProvider) Update(userInfo *provider.UserInfo, newCluster *kubermaticapiv1.Cluster) (*kubermaticapiv1.Cluster, error) {
	seedImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	return seedImpersonatedClient.Clusters().Update(newCluster)
}

// GetAdminKubeconfigForCustomerCluster returns the admin kubeconfig for the given cluster
func (p *ClusterProvider) GetAdminKubeconfigForCustomerCluster(c *kubermaticapiv1.Cluster) (*clientcmdapi.Config, error) {
	b, err := p.userClusterConnProvider.GetAdminKubeconfig(c)
	if err != nil {
		return nil, err
	}

	return clientcmd.Load(b)
}

// GetAdminClientForCustomerCluster returns a client to interact with all resources in the given cluster
//
// Note that the client you will get has admin privileges
func (p *ClusterProvider) GetAdminClientForCustomerCluster(c *kubermaticapiv1.Cluster) (ctrlruntimeclient.Client, error) {
	return p.userClusterConnProvider.GetDynamicClient(c)
}

// GetClientForCustomerCluster returns a client to interact with all resources in the given cluster
//
// Note that the client doesn't use admin account instead it authn/authz as userInfo(email, group)
// This implies that you have to make sure the user has the appropriate permissions inside the user cluster
func (p *ClusterProvider) GetClientForCustomerCluster(userInfo *provider.UserInfo, c *kubermaticapiv1.Cluster) (ctrlruntimeclient.Client, error) {
	return p.userClusterConnProvider.GetDynamicClient(c, p.withImpersonation(userInfo))
}

// GetSeedClusterAdminRuntimeClient returns a runtime client to interact with the seed cluster resources.
//
// Note that this client has admin privileges in the seed cluster.
func (p *ClusterProvider) GetSeedClusterAdminRuntimeClient() ctrlruntimeclient.Client {
	return p.client
}

// GetSeedClusterAdminClient returns a kubernetes client to interact with the seed cluster resources.
//
// Note that this client has admin privileges in the seed cluster.
func (p *ClusterProvider) GetSeedClusterAdminClient() kubernetes.Interface {
	return p.k8sClient
}

func (p *ClusterProvider) withImpersonation(userInfo *provider.UserInfo) k8cuserclusterclient.ConfigOption {
	return func(cfg *restclient.Config) *restclient.Config {
		cfg.Impersonate = restclient.ImpersonationConfig{
			UserName: userInfo.Email,
			Groups:   []string{p.extractGroupPrefix(userInfo.Group), "system:authenticated"},
		}
		return cfg
	}
}
