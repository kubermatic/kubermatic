/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	k8cuserclusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	utilcluster "k8c.io/kubermatic/v2/pkg/util/cluster"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	kubenetutil "k8s.io/apimachinery/pkg/util/net"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// UserClusterConnectionProvider offers functions to interact with an user cluster.
type UserClusterConnectionProvider interface {
	GetClient(context.Context, *kubermaticv1.Cluster, ...k8cuserclusterclient.ConfigOption) (ctrlruntimeclient.Client, error)
	GetK8sClient(context.Context, *kubermaticv1.Cluster, ...k8cuserclusterclient.ConfigOption) (kubernetes.Interface, error)
	GetClientConfig(context.Context, *kubermaticv1.Cluster, ...k8cuserclusterclient.ConfigOption) (*restclient.Config, error)
}

// extractGroupPrefixFunc is a function that knows how to extract a prefix (owners, editors) from "projectID-owners" group,
// group names inside leaf/user clusters don't have projectID in their names.
type extractGroupPrefixFunc func(groupName string) string

// NewClusterProvider returns a new cluster provider that respects RBAC policies
// it uses createSeedImpersonatedClient to create a connection that uses user impersonation.
func NewClusterProvider(
	cfg *restclient.Config,
	createSeedImpersonatedClient ImpersonationClient,
	userClusterConnProvider UserClusterConnectionProvider,
	workerName string,
	extractGroupPrefix extractGroupPrefixFunc,
	client ctrlruntimeclient.Client,
	k8sClient kubernetes.Interface,
	oidcKubeConfEndpoint bool,
	versions kubermatic.Versions,
	seed *kubermaticv1.Seed) *ClusterProvider {
	return &ClusterProvider{
		createSeedImpersonatedClient: createSeedImpersonatedClient,
		userClusterConnProvider:      userClusterConnProvider,
		workerName:                   workerName,
		extractGroupPrefix:           extractGroupPrefix,
		client:                       client,
		k8sClient:                    k8sClient,
		oidcKubeConfEndpoint:         oidcKubeConfEndpoint,
		seedKubeconfig:               cfg,
		versions:                     versions,
		seed:                         seed,
	}
}

// ClusterProvider struct that holds required components in order to provide
// cluster provided that is RBAC compliant.
type ClusterProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	// whenever a connection to Seed API server is required
	createSeedImpersonatedClient ImpersonationClient

	// userClusterConnProvider used for obtaining a connection to the client's cluster
	userClusterConnProvider UserClusterConnectionProvider

	oidcKubeConfEndpoint bool
	workerName           string
	extractGroupPrefix   extractGroupPrefixFunc
	client               ctrlruntimeclient.Client
	k8sClient            kubernetes.Interface
	seedKubeconfig       *restclient.Config
	versions             kubermatic.Versions
	seed                 *kubermaticv1.Seed
}

var _ provider.ClusterProvider = &ClusterProvider{}
var _ provider.PrivilegedClusterProvider = &ClusterProvider{}

// New creates a brand new cluster that is bound to the given project.
//
// Note that the admin privileges are used to set the cluster status.
func (p *ClusterProvider) New(ctx context.Context, project *kubermaticv1.Project, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	if project == nil || userInfo == nil || cluster == nil {
		return nil, errors.New("project and/or userInfo and/or cluster is missing but required")
	}
	// share kubeconfig feature is contrary to cluster OIDC setting
	if p.oidcKubeConfEndpoint && !reflect.DeepEqual(cluster.Spec.OIDC, kubermaticv1.OIDCSettings{}) {
		return nil, errors.New("can not set OIDC for the cluster when share config feature is enabled")
	}

	newCluster := genAPICluster(project, cluster, p.workerName)

	seedImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}
	if err := seedImpersonatedClient.Create(ctx, newCluster); err != nil {
		return nil, err
	}

	if err := p.waitForCluster(ctx, seedImpersonatedClient, newCluster); err != nil {
		return nil, fmt.Errorf("failed waiting for the new cluster to appear in the cache: %w", err)
	}

	// regular users are not allowed to update the status subresource, so we use the admin client
	err = kubermaticv1helper.UpdateClusterStatus(ctx, p.client, newCluster, func(c *kubermaticv1.Cluster) {
		c.Status.UserEmail = userInfo.Email
	})
	if err != nil {
		return nil, err
	}

	return newCluster, nil
}

// NewUnsecured creates a brand new cluster that is bound to the given project.
//
// Note that the admin privileges are used to create cluster.
func (p *ClusterProvider) NewUnsecured(ctx context.Context, project *kubermaticv1.Project, cluster *kubermaticv1.Cluster, userEmail string) (*kubermaticv1.Cluster, error) {
	if project == nil || cluster == nil {
		return nil, errors.New("project and/or cluster is missing but required")
	}
	// share kubeconfig feature is contrary to cluster OIDC setting
	if p.oidcKubeConfEndpoint && !reflect.DeepEqual(cluster.Spec.OIDC, kubermaticv1.OIDCSettings{}) {
		return nil, errors.New("can not set OIDC for the cluster when share config feature is enabled")
	}

	newCluster := genAPICluster(project, cluster, p.workerName)

	err := p.client.Create(ctx, newCluster)
	if err != nil {
		return nil, err
	}

	if err := p.waitForCluster(ctx, p.client, newCluster); err != nil {
		return nil, fmt.Errorf("failed waiting for the new cluster to appear in the cache: %w", err)
	}

	err = kubermaticv1helper.UpdateClusterStatus(ctx, p.client, newCluster, func(c *kubermaticv1.Cluster) {
		c.Status.UserEmail = userEmail
	})
	if err != nil {
		return nil, err
	}

	return newCluster, nil
}

func (p *ClusterProvider) waitForCluster(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	waiter := reconciling.WaitUntilObjectExistsInCacheConditionFunc(ctx, client, zap.NewNop().Sugar(), ctrlruntimeclient.ObjectKeyFromObject(cluster), cluster)
	if err := wait.Poll(100*time.Millisecond, 5*time.Second, waiter); err != nil {
		return fmt.Errorf("failed waiting for the new cluster to appear in the cache: %w", err)
	}

	return nil
}

func genAPICluster(project *kubermaticv1.Project, cluster *kubermaticv1.Cluster, workerName string) *kubermaticv1.Cluster {
	cluster.Spec.HumanReadableName = strings.TrimSpace(cluster.Spec.HumanReadableName)

	var name string
	if cluster.Name != "" {
		name = cluster.Name
	} else {
		name = utilcluster.MakeClusterName()
	}

	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: cluster.Annotations,
			Finalizers:  cluster.Finalizers,
			Labels:      getClusterLabels(cluster.Labels, project.Name, workerName),
			Name:        name,
		},
		Spec: cluster.Spec,
	}
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
func (p *ClusterProvider) List(ctx context.Context, project *kubermaticv1.Project, options *provider.ClusterListOptions) (*kubermaticv1.ClusterList, error) {
	if project == nil {
		return nil, errors.New("project is missing but required")
	}

	projectClusters := &kubermaticv1.ClusterList{}
	selector := labels.SelectorFromSet(map[string]string{kubermaticv1.ProjectIDLabelKey: project.Name})
	listOpts := &ctrlruntimeclient.ListOptions{LabelSelector: selector}
	if err := p.client.List(ctx, projectClusters, listOpts); err != nil {
		// ignore error if cluster is unreachable
		if kubenetutil.IsConnectionRefused(err) {
			return projectClusters, nil
		}
		return nil, fmt.Errorf("failed to list clusters: %w", err)
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

// Get returns the given cluster, it uses the projectInternalName to determine the group the user belongs to.
func (p *ClusterProvider) Get(ctx context.Context, userInfo *provider.UserInfo, clusterName string, options *provider.ClusterGetOptions) (*kubermaticv1.Cluster, error) {
	if options == nil {
		options = &provider.ClusterGetOptions{}
	}
	seedImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	cluster := &kubermaticv1.Cluster{}
	if err := seedImpersonatedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: clusterName}, cluster); err != nil {
		return nil, err
	}
	if options.CheckInitStatus {
		if !cluster.Status.ExtendedHealth.AllHealthy() {
			return nil, apierrors.NewServiceUnavailable("Cluster components are not ready yet")
		}
	}

	return cluster, nil
}

// IsCluster checks if cluster exist with the given name.
func (p *ClusterProvider) IsCluster(ctx context.Context, clusterName string) bool {
	if err := p.client.Get(ctx, types.NamespacedName{Name: clusterName}, &kubermaticv1.Cluster{}); err != nil {
		return false
	}
	return true
}

// Delete deletes the given cluster.
func (p *ClusterProvider) Delete(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster) error {
	seedImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return err
	}

	// Will delete all child's after the object is gone - otherwise the etcd might be deleted before all machines are gone
	// See https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/#controlling-how-the-garbage-collector-deletes-dependents
	policy := metav1.DeletePropagationBackground
	delOpts := &ctrlruntimeclient.DeleteOptions{
		PropagationPolicy: &policy,
	}
	return seedImpersonatedClient.Delete(ctx, cluster, delOpts)
}

// Update updates a cluster.
func (p *ClusterProvider) Update(ctx context.Context, project *kubermaticv1.Project, userInfo *provider.UserInfo, newCluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	seedImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createSeedImpersonatedClient)
	if err != nil {
		return nil, err
	}

	newCluster.Labels = getClusterLabels(newCluster.Labels, project.Name, "") // Do not update worker name.
	if err := seedImpersonatedClient.Update(ctx, newCluster); err != nil {
		return nil, err
	}
	return newCluster, nil
}

// GetAdminKubeconfigForCustomerCluster returns the admin kubeconfig for the given cluster.
func (p *ClusterProvider) GetAdminKubeconfigForCustomerCluster(ctx context.Context, c *kubermaticv1.Cluster) (*clientcmdapi.Config, error) {
	secret := &corev1.Secret{}
	name := types.NamespacedName{
		Namespace: c.Status.NamespaceName,
		Name:      resources.AdminKubeconfigSecretName,
	}
	if err := p.GetSeedClusterAdminRuntimeClient().Get(ctx, name, secret); err != nil {
		return nil, err
	}

	return clientcmd.Load(secret.Data[resources.KubeconfigSecretKey])
}

// GetViewerKubeconfigForCustomerCluster returns the viewer kubeconfig for the given cluster.
func (p *ClusterProvider) GetViewerKubeconfigForCustomerCluster(ctx context.Context, c *kubermaticv1.Cluster) (*clientcmdapi.Config, error) {
	s := &corev1.Secret{}

	if err := p.GetSeedClusterAdminRuntimeClient().Get(ctx, types.NamespacedName{Namespace: c.Status.NamespaceName, Name: resources.ViewerKubeconfigSecretName}, s); err != nil {
		return nil, err
	}

	d := s.Data[resources.KubeconfigSecretKey]
	if len(d) == 0 {
		return nil, fmt.Errorf("no kubeconfig found")
	}

	return clientcmd.Load(d)
}

// RevokeViewerKubeconfig revokes the viewer token and kubeconfig.
func (p *ClusterProvider) RevokeViewerKubeconfig(ctx context.Context, c *kubermaticv1.Cluster) error {
	s := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      resources.ViewerTokenSecretName,
			Namespace: c.Status.NamespaceName,
		},
	}

	if err := p.GetSeedClusterAdminRuntimeClient().Delete(ctx, s); err != nil {
		return err
	}
	return nil
}

// RevokeAdminKubeconfig revokes the viewer token and kubeconfig.
func (p *ClusterProvider) RevokeAdminKubeconfig(ctx context.Context, c *kubermaticv1.Cluster) error {
	// during the KKP 2.20->2.21 migration, the cluster address was moved
	// and we need to handle current seeds (using cluster.Status.Address)
	// and older seeds (using cluster.Address)
	newToken := kuberneteshelper.GenerateToken()

	// seed is on KKP 2.21 and so it has the new fields available; update the
	// status address first, as its the primary source of truth for all other
	// reconciling (see the GetAddress() function on Clusters)
	if p.seed.IsUpToDate(p.versions) {
		oldCluster := c.DeepCopy()
		c.Status.Address.AdminToken = newToken

		if err := p.GetSeedClusterAdminRuntimeClient().Status().Patch(ctx, c, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
			return fmt.Errorf("failed to patch cluster with new token: %w", err)
		}
	}

	// KKP 2.21 also still contains the old field, so we can always reset it
	oldCluster := c.DeepCopy()
	c.Address.AdminToken = newToken

	if err := p.GetSeedClusterAdminRuntimeClient().Patch(ctx, c, ctrlruntimeclient.MergeFrom(oldCluster)); err != nil {
		return fmt.Errorf("failed to patch cluster with new token: %w", err)
	}

	return nil
}

// GetAdminClientForCustomerCluster returns a client to interact with all resources in the given cluster
//
// Note that the client you will get has admin privileges.
func (p *ClusterProvider) GetAdminClientForCustomerCluster(ctx context.Context, c *kubermaticv1.Cluster) (ctrlruntimeclient.Client, error) {
	return p.userClusterConnProvider.GetClient(ctx, c)
}

// GetAdminK8sClientForCustomerCluster returns a k8s go client to interact with all resources in the given cluster
//
// Note that the client you will get has admin privileges.
func (p *ClusterProvider) GetAdminK8sClientForCustomerCluster(ctx context.Context, c *kubermaticv1.Cluster) (kubernetes.Interface, error) {
	return p.userClusterConnProvider.GetK8sClient(ctx, c)
}

// GetAdminClientConfigForCustomerCluster returns a client config
//
// Note that the client you will get has admin privileges.
func (p *ClusterProvider) GetAdminClientConfigForCustomerCluster(ctx context.Context, c *kubermaticv1.Cluster) (*restclient.Config, error) {
	return p.userClusterConnProvider.GetClientConfig(ctx, c)
}

// GetClientForCustomerCluster returns a client to interact with all resources in the given cluster
//
// Note that the client doesn't use admin account instead it authn/authz as userInfo(email, group)
// This implies that you have to make sure the user has the appropriate permissions inside the user cluster.
func (p *ClusterProvider) GetClientForCustomerCluster(ctx context.Context, userInfo *provider.UserInfo, c *kubermaticv1.Cluster) (ctrlruntimeclient.Client, error) {
	return p.userClusterConnProvider.GetClient(ctx, c, p.withImpersonation(userInfo))
}

func (p *ClusterProvider) GetTokenForCustomerCluster(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster) (string, error) {
	parts := strings.Split(userInfo.Group, "-")
	switch parts[0] {
	case "editors":
		return cluster.GetAddress().AdminToken, nil
	case "owners":
		return cluster.GetAddress().AdminToken, nil
	case "viewers":
		s := &corev1.Secret{}
		name := types.NamespacedName{Namespace: cluster.Status.NamespaceName, Name: resources.ViewerTokenSecretName}

		if err := p.GetSeedClusterAdminRuntimeClient().Get(ctx, name, s); err != nil {
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
// Note that the admin privileges are used to get cluster.
func (p *ClusterProvider) GetUnsecured(ctx context.Context, project *kubermaticv1.Project, clusterName string, options *provider.ClusterGetOptions) (*kubermaticv1.Cluster, error) {
	if project == nil {
		return nil, errors.New("project is missing but required")
	}
	if options == nil {
		options = &provider.ClusterGetOptions{}
	}

	cluster := &kubermaticv1.Cluster{}
	if err := p.client.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
		return nil, err
	}
	if cluster.Labels[kubermaticv1.ProjectIDLabelKey] == project.Name {
		if options.CheckInitStatus {
			if !cluster.Status.ExtendedHealth.AllHealthy() {
				return nil, apierrors.NewServiceUnavailable("Cluster components are not ready yet")
			}
		}
		return cluster, nil
	}

	return nil, apierrors.NewNotFound(schema.GroupResource{}, clusterName)
}

// UpdateUnsecured updates a cluster.
//
// Note that the admin privileges are used to update cluster.
func (p *ClusterProvider) UpdateUnsecured(ctx context.Context, project *kubermaticv1.Project, cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	if project == nil {
		return nil, errors.New("project is missing but required")
	}
	cluster.Labels = getClusterLabels(cluster.Labels, project.Name, "") // Do not update worker name.
	err := p.client.Update(ctx, cluster)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

// DeleteUnsecured deletes a cluster.
//
// Note that the admin privileges are used to delete cluster.
func (p *ClusterProvider) DeleteUnsecured(ctx context.Context, cluster *kubermaticv1.Cluster) error {
	// Will delete all child's after the object is gone - otherwise the etcd might be deleted before all machines are gone
	// See https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/#controlling-how-the-garbage-collector-deletes-dependents
	policy := metav1.DeletePropagationBackground
	delOpts := &ctrlruntimeclient.DeleteOptions{
		PropagationPolicy: &policy,
	}
	return p.client.Delete(ctx, cluster, delOpts)
}

// SeedAdminConfig return an admin kubeconfig for the seed. This function does not perform any kind
// of access control. Try to not use it.
func (p *ClusterProvider) SeedAdminConfig() *restclient.Config {
	return p.seedKubeconfig
}

// ListAll gets all clusters
//
// Note that the admin privileges are used to list all clusters.
func (p *ClusterProvider) ListAll(ctx context.Context, labelSelector labels.Selector) (*kubermaticv1.ClusterList, error) {
	optionsLabelSelector := labels.Everything()
	if labelSelector != nil {
		optionsLabelSelector = labelSelector
	}

	projectClusters := &kubermaticv1.ClusterList{}
	if err := p.client.List(ctx, projectClusters, ctrlruntimeclient.MatchingLabelsSelector{
		Selector: optionsLabelSelector,
	}); err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	return projectClusters, nil
}

// GetSeedName gets the seed name of the cluster.
func (p *ClusterProvider) GetSeedName() string {
	return p.seed.Name
}
