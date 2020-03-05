package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"

	k8cuserclusterclient "github.com/kubermatic/kubermatic/api/pkg/cluster/client"
	"github.com/kubermatic/kubermatic/api/pkg/controller/seed-controller-manager/cloud"
	openshiftuserclusterresources "github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/resources/resources/openshift"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kuberneteshelper "github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// UserClusterConnectionProvider offers functions to interact with an user cluster
type UserClusterConnectionProvider interface {
	GetClient(*kubermaticv1.Cluster, ...k8cuserclusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
}

// extractGroupPrefixFunc is a function that knows how to extract a prefix (owners, editors) from "projectID-owners" group,
// group names inside leaf/user clusters don't have projectID in their names
type extractGroupPrefixFunc func(groupName string) string

// NewClusterProvider returns a new cluster provider that respects RBAC policies
// it uses createSeedImpersonatedClient to create a connection that uses user impersonation
func NewClusterProvider(
	cfg *restclient.Config,
	createSeedImpersonatedClient kubermaticImpersonationClient,
	userClusterConnProvider UserClusterConnectionProvider,
	workerName string,
	extractGroupPrefix extractGroupPrefixFunc,
	client ctrlruntimeclient.Client,
	k8sClient kubernetes.Interface,
	oidcKubeConfEndpoint bool) *ClusterProvider {
	return &ClusterProvider{
		createSeedImpersonatedClient: createSeedImpersonatedClient,
		userClusterConnProvider:      userClusterConnProvider,
		workerName:                   workerName,
		extractGroupPrefix:           extractGroupPrefix,
		client:                       client,
		k8sClient:                    k8sClient,
		oidcKubeConfEndpoint:         oidcKubeConfEndpoint,
		seedKubeconfig:               cfg,
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

	oidcKubeConfEndpoint bool
	workerName           string
	extractGroupPrefix   extractGroupPrefixFunc
	client               ctrlruntimeclient.Client
	k8sClient            kubernetes.Interface
	seedKubeconfig       *restclient.Config
}

// New creates a brand new cluster that is bound to the given project
func (p *ClusterProvider) New(project *kubermaticv1.Project, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	if project == nil || userInfo == nil || cluster == nil {
		return nil, errors.New("project and/or userInfo and/or cluster is missing but required")
	}
	// share kubeconfig feature is contrary to cluster OIDC setting
	if p.oidcKubeConfEndpoint && !reflect.DeepEqual(cluster.Spec.OIDC, kubermaticv1.OIDCSettings{}) {
		return nil, errors.New("can not set OIDC for the cluster when share config feature is enabled")
	}

	cluster.Spec.HumanReadableName = strings.TrimSpace(cluster.Spec.HumanReadableName)

	var name string
	if cluster.Name != "" {
		name = cluster.Name
	} else {
		name = rand.String(10)
	}

	newCluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: cluster.Annotations,
			Finalizers:  cluster.Finalizers,
			Labels:      getClusterLabels(cluster.Labels, project.Name, p.workerName),
			Name:        name,
		},
		Spec: cluster.Spec,
		Status: kubermaticv1.ClusterStatus{
			UserEmail:              userInfo.Email,
			NamespaceName:          NamespaceName(name),
			CloudMigrationRevision: cloud.CurrentMigrationRevision,
			KubermaticVersion:      resources.KUBERMATICCOMMIT,
			ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
				Apiserver:                    kubermaticv1.HealthStatusProvisioning,
				Scheduler:                    kubermaticv1.HealthStatusProvisioning,
				Controller:                   kubermaticv1.HealthStatusProvisioning,
				MachineController:            kubermaticv1.HealthStatusProvisioning,
				Etcd:                         kubermaticv1.HealthStatusProvisioning,
				OpenVPN:                      kubermaticv1.HealthStatusProvisioning,
				CloudProviderInfrastructure:  kubermaticv1.HealthStatusProvisioning,
				UserClusterControllerManager: kubermaticv1.HealthStatusProvisioning,
			},
		},
		Address: kubermaticv1.ClusterAddress{},
	}

	seedImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}
	newCluster, err = seedImpersonatedClient.Clusters().Create(newCluster)
	if err != nil {
		return nil, err
	}

	return newCluster, nil
}

func getClusterLabels(specifiedLabels map[string]string, projectName, workerName string) map[string]string {
	resultLabels := map[string]string{}

	if specifiedLabels != nil {
		resultLabels = specifiedLabels
	}

	resultLabels[kubermaticv1.ProjectIDLabelKey] = projectName

	if len(workerName) > 0 {
		resultLabels[kubermaticv1.WorkerNameLabelKey] = workerName
	}

	return resultLabels
}

// List gets all clusters that belong to the given project
// If you want to filter the result please take a look at ClusterListOptions
//
// Note:
// After we get the list of clusters we could try to get each cluster individually using unprivileged account to see if the user have read access,
// We don't do this because we assume that if the user was able to get the project (argument) it has to have at least read access.
func (p *ClusterProvider) List(project *kubermaticv1.Project, options *provider.ClusterListOptions) (*kubermaticv1.ClusterList, error) {
	if project == nil {
		return nil, errors.New("project is missing but required")
	}

	projectClusters := &kubermaticv1.ClusterList{}
	selector := labels.SelectorFromSet(map[string]string{kubermaticv1.ProjectIDLabelKey: project.Name})
	listOpts := &ctrlruntimeclient.ListOptions{LabelSelector: selector}
	if err := p.client.List(context.Background(), projectClusters, listOpts); err != nil {
		return nil, fmt.Errorf("failed to list clusters: %v", err)
	}

	if options == nil || len(options.ClusterSpecName) == 0 {
		return projectClusters, nil
	}

	filteredProjectClusters := &kubermaticv1.ClusterList{}
	for _, projectCluster := range projectClusters.Items {
		if projectCluster.Spec.HumanReadableName == options.ClusterSpecName {
			filteredProjectClusters.Items = append(filteredProjectClusters.Items, projectCluster)
		}
	}

	return filteredProjectClusters, nil
}

// Get returns the given cluster, it uses the projectInternalName to determine the group the user belongs to
func (p *ClusterProvider) Get(userInfo *provider.UserInfo, clusterName string, options *provider.ClusterGetOptions) (*kubermaticv1.Cluster, error) {
	seedImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	cluster, err := seedImpersonatedClient.Clusters().Get(clusterName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	if options.CheckInitStatus {
		if !cluster.Status.ExtendedHealth.AllHealthy() {
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
func (p *ClusterProvider) Update(project *kubermaticv1.Project, userInfo *provider.UserInfo, newCluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	seedImpersonatedClient, err := createKubermaticImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	newCluster.Status.KubermaticVersion = resources.KUBERMATICCOMMIT
	newCluster.Labels = getClusterLabels(newCluster.Labels, project.Name, "") // Do not update worker name.
	return seedImpersonatedClient.Clusters().Update(newCluster)
}

// GetAdminKubeconfigForCustomerCluster returns the admin kubeconfig for the given cluster
func (p *ClusterProvider) GetAdminKubeconfigForCustomerCluster(c *kubermaticv1.Cluster) (*clientcmdapi.Config, error) {
	secret := &corev1.Secret{}
	name := types.NamespacedName{
		Namespace: c.Status.NamespaceName,
		Name:      resources.AdminKubeconfigSecretName,
	}
	if err := p.GetSeedClusterAdminRuntimeClient().Get(context.Background(), name, secret); err != nil {
		return nil, err
	}

	return clientcmd.Load(secret.Data[resources.KubeconfigSecretKey])
}

// GetViewerKubeconfigForCustomerCluster returns the viewer kubeconfig for the given cluster
func (p *ClusterProvider) GetViewerKubeconfigForCustomerCluster(c *kubermaticv1.Cluster) (*clientcmdapi.Config, error) {
	if c.IsOpenshift() {
		return nil, fmt.Errorf("not implemented")
	}
	s := &corev1.Secret{}

	if err := p.GetSeedClusterAdminRuntimeClient().Get(context.Background(), types.NamespacedName{Namespace: c.Status.NamespaceName, Name: resources.ViewerKubeconfigSecretName}, s); err != nil {
		return nil, err
	}

	d := s.Data[resources.KubeconfigSecretKey]
	if len(d) == 0 {
		return nil, fmt.Errorf("no kubeconfig found")
	}

	return clientcmd.Load(d)
}

// RevokeViewerKubeconfig revokes the viewer token and kubeconfig
func (p *ClusterProvider) RevokeViewerKubeconfig(c *kubermaticv1.Cluster) error {
	if c.IsOpenshift() {
		return errors.New("not implemented")
	}
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.ViewerTokenSecretName,
			Namespace: c.Status.NamespaceName,
		},
	}

	if err := p.GetSeedClusterAdminRuntimeClient().Delete(context.Background(), s); err != nil {
		return err
	}
	return nil
}

// RevokeAdminKubeconfig revokes the viewer token and kubeconfig
func (p *ClusterProvider) RevokeAdminKubeconfig(c *kubermaticv1.Cluster) error {
	ctx := context.Background()
	if !c.IsOpenshift() {
		oldCluster := c.DeepCopy()
		c.Address.AdminToken = kuberneteshelper.GenerateToken()
		if err := p.GetSeedClusterAdminRuntimeClient().Patch(ctx, c, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
			return fmt.Errorf("failed to patch cluster with new token: %v", err)
		}
		return nil
	}

	userClusterClient, err := p.GetAdminClientForCustomerCluster(c)
	if err != nil {
		return fmt.Errorf("failed to get usercluster client: %v", err)
	}
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: metav1.NamespaceSystem,
			Name:      openshiftuserclusterresources.TokenOwnerServiceAccountName,
		},
	}
	if err := userClusterClient.Delete(ctx, serviceAccount); err != nil {
		return fmt.Errorf("failed to remove the token owner: %v", err)
	}
	return nil
}

// GetAdminClientForCustomerCluster returns a client to interact with all resources in the given cluster
//
// Note that the client you will get has admin privileges
func (p *ClusterProvider) GetAdminClientForCustomerCluster(c *kubermaticv1.Cluster) (ctrlruntimeclient.Client, error) {
	return p.userClusterConnProvider.GetClient(c)
}

// GetClientForCustomerCluster returns a client to interact with all resources in the given cluster
//
// Note that the client doesn't use admin account instead it authn/authz as userInfo(email, group)
// This implies that you have to make sure the user has the appropriate permissions inside the user cluster
func (p *ClusterProvider) GetClientForCustomerCluster(userInfo *provider.UserInfo, c *kubermaticv1.Cluster) (ctrlruntimeclient.Client, error) {
	return p.userClusterConnProvider.GetClient(c, p.withImpersonation(userInfo))
}

func (p *ClusterProvider) GetTokenForCustomerCluster(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster) (string, error) {
	parts := strings.Split(userInfo.Group, "-")
	switch parts[0] {
	case "editors":
		return cluster.Address.AdminToken, nil
	case "owners":
		return cluster.Address.AdminToken, nil
	case "viewers":
		s := &corev1.Secret{}
		name := types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: resources.ViewerTokenSecretName}

		if err := p.GetSeedClusterAdminRuntimeClient().Get(context.Background(), name, s); err != nil {
			return "", err
		}

		return string(s.Data[resources.ViewerTokenSecretKey]), nil
	default:
		return "", fmt.Errorf("user group %s not supported", userInfo.Group)
	}
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

// GetUnsecured returns a cluster for the project and given name.
//
// Note that the admin privileges are used to get cluster
func (p *ClusterProvider) GetUnsecured(project *kubermaticv1.Project, clusterName string) (*kubermaticv1.Cluster, error) {
	if project == nil {
		return nil, errors.New("project is missing but required")
	}

	cluster := &kubermaticv1.Cluster{}
	if err := p.client.Get(context.Background(), types.NamespacedName{Name: clusterName}, cluster); err != nil {
		return nil, err
	}
	if cluster.Labels[kubermaticv1.ProjectIDLabelKey] == project.Name {
		return cluster, nil
	}

	return nil, kerrors.NewNotFound(schema.GroupResource{}, clusterName)
}

// UpdateUnsecured updates a cluster.
//
// Note that the admin privileges are used to update cluster
func (p *ClusterProvider) UpdateUnsecured(project *kubermaticv1.Project, cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	if project == nil {
		return nil, errors.New("project is missing but required")
	}
	cluster.Status.KubermaticVersion = resources.KUBERMATICCOMMIT
	cluster.Labels = getClusterLabels(cluster.Labels, project.Name, "") // Do not update worker name.
	err := p.client.Update(context.Background(), cluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// DeleteUnsecured deletes a cluster.
//
// Note that the admin privileges are used to delete cluster
func (p *ClusterProvider) DeleteUnsecured(cluster *kubermaticv1.Cluster) error {
	// Will delete all child's after the object is gone - otherwise the etcd might be deleted before all machines are gone
	// See https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/#controlling-how-the-garbage-collector-deletes-dependents
	policy := metav1.DeletePropagationBackground
	delOpts := &ctrlruntimeclient.DeleteOptions{
		PropagationPolicy: &policy,
	}
	return p.client.Delete(context.Background(), cluster, delOpts)
}

// SeedAdminConfig return an admin kubeconfig for the seed. This function does not perform any kind
// of access control. Try to not use it.
func (p *ClusterProvider) SeedAdminConfig() *restclient.Config {
	return p.seedKubeconfig
}
