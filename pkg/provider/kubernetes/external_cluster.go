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
	"encoding/base64"
	"errors"
	"fmt"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	ksemver "k8c.io/kubermatic/v2/pkg/semver"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
	"k8c.io/kubermatic/v2/pkg/util/restmapper"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/metrics/pkg/apis/metrics/v1beta1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ExternalClusterProvider struct that holds required components in order to provide connection to the cluster.
type ExternalClusterProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	createMasterImpersonatedClient ImpersonationClient
	clientPrivileged               ctrlruntimeclient.Client
	restMapperCache                *restmapper.Cache
}

var _ provider.ExternalClusterProvider = &ExternalClusterProvider{}
var _ provider.PrivilegedExternalClusterProvider = &ExternalClusterProvider{}

// NewExternalClusterProvider returns an external cluster provider.
func NewExternalClusterProvider(createMasterImpersonatedClient ImpersonationClient, client ctrlruntimeclient.Client) (*ExternalClusterProvider, error) {
	return &ExternalClusterProvider{
		createMasterImpersonatedClient: createMasterImpersonatedClient,
		clientPrivileged:               client,
		restMapperCache:                restmapper.New(),
	}, nil
}

// New creates a brand new external cluster in the system with the given name.
func (p *ExternalClusterProvider) New(ctx context.Context, userInfo *provider.UserInfo, project *kubermaticv1.Project, cluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error) {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	addProjectReference(project, cluster)
	if err := masterImpersonatedClient.Create(ctx, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

// NewUnsecured creates a brand new external cluster in the system with the given name
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to create the resource.
func (p *ExternalClusterProvider) NewUnsecured(ctx context.Context, project *kubermaticv1.Project, cluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error) {
	addProjectReference(project, cluster)
	if err := p.clientPrivileged.Create(ctx, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

// Get returns the given cluster.
func (p *ExternalClusterProvider) Get(ctx context.Context, userInfo *provider.UserInfo, clusterName string) (*kubermaticv1.ExternalCluster, error) {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}

	cluster := &kubermaticv1.ExternalCluster{}
	if err := masterImpersonatedClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: clusterName}, cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}

// Delete deletes the given cluster.
func (p *ExternalClusterProvider) Delete(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.ExternalCluster) error {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return err
	}

	// Will delete all child's after the object is gone - otherwise the etcd might be deleted before all machines are gone
	// See https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/#controlling-how-the-garbage-collector-deletes-dependents
	policy := metav1.DeletePropagationBackground
	delOpts := &ctrlruntimeclient.DeleteOptions{
		PropagationPolicy: &policy,
	}
	return masterImpersonatedClient.Delete(ctx, cluster, delOpts)
}

// DeleteUnsecured deletes an external cluster.
//
// Note that the admin privileges are used to delete cluster.
func (p *ExternalClusterProvider) DeleteUnsecured(ctx context.Context, cluster *kubermaticv1.ExternalCluster) error {
	// Will delete all child's after the object is gone - otherwise the etcd might be deleted before all machines are gone
	// See https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/#controlling-how-the-garbage-collector-deletes-dependents
	policy := metav1.DeletePropagationBackground
	delOpts := &ctrlruntimeclient.DeleteOptions{
		PropagationPolicy: &policy,
	}
	return p.clientPrivileged.Delete(ctx, cluster, delOpts)
}

// GetUnsecured returns an external cluster for the project and given name.
//
// Note that the admin privileges are used to get cluster.
func (p *ExternalClusterProvider) GetUnsecured(ctx context.Context, clusterName string) (*kubermaticv1.ExternalCluster, error) {
	cluster := &kubermaticv1.ExternalCluster{}
	if err := p.clientPrivileged.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}

// List gets all external clusters that belong to the given project.
func (p *ExternalClusterProvider) List(ctx context.Context, project *kubermaticv1.Project) (*kubermaticv1.ExternalClusterList, error) {
	if project == nil {
		return nil, errors.New("project is missing but required")
	}

	projectClusters := &kubermaticv1.ExternalClusterList{}
	selector := labels.SelectorFromSet(map[string]string{kubermaticv1.ProjectIDLabelKey: project.Name})
	listOpts := &ctrlruntimeclient.ListOptions{LabelSelector: selector}
	if err := p.clientPrivileged.List(ctx, projectClusters, listOpts); err != nil {
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	return projectClusters, nil
}

// Update updates the given cluster.
func (p *ExternalClusterProvider) UpdateUnsecured(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error) {
	if err := p.clientPrivileged.Update(ctx, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

// Update updates the given cluster.
func (p *ExternalClusterProvider) Update(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error) {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	if err := masterImpersonatedClient.Update(ctx, cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

func addProjectReference(project *kubermaticv1.Project, cluster *kubermaticv1.ExternalCluster) {
	if cluster.Labels == nil {
		cluster.Labels = make(map[string]string)
	}
	cluster.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: kubermaticv1.SchemeGroupVersion.String(),
			Kind:       kubermaticv1.ProjectKindName,
			UID:        project.GetUID(),
			Name:       project.Name,
		},
	}
	cluster.Labels[kubermaticv1.ProjectIDLabelKey] = project.Name
}

func (p *ExternalClusterProvider) GenerateClient(cfg *clientcmdapi.Config) (ctrlruntimeclient.Client, error) {
	clientConfig, err := getRestConfig(cfg)
	if err != nil {
		return nil, err
	}
	client, err := p.restMapperCache.Client(clientConfig)
	if err != nil {
		return nil, err
	}
	return client, nil
}

func (p *ExternalClusterProvider) GetClient(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (ctrlruntimeclient.Client, error) {
	secretKeyGetter := provider.SecretKeySelectorValueFuncFactory(ctx, p.clientPrivileged)
	rawKubeconfig, err := secretKeyGetter(cluster.Spec.KubeconfigReference, resources.KubeconfigSecretKey)
	if err != nil {
		return nil, err
	}
	cfg, err := clientcmd.Load([]byte(rawKubeconfig))
	if err != nil {
		return nil, err
	}
	return p.GenerateClient(cfg)
}

func (p *ExternalClusterProvider) GetVersion(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (*ksemver.Semver, error) {
	secretKeyGetter := provider.SecretKeySelectorValueFuncFactory(ctx, p.clientPrivileged)
	rawKubeconfig, err := secretKeyGetter(cluster.Spec.KubeconfigReference, resources.KubeconfigSecretKey)
	if err != nil {
		return nil, err
	}
	cfg, err := clientcmd.Load([]byte(rawKubeconfig))
	if err != nil {
		return nil, err
	}
	clientConfig, err := getRestConfig(cfg)
	if err != nil {
		return nil, err
	}
	masterClient, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, err
	}

	version, err := masterClient.DiscoveryClient.ServerVersion()
	if err != nil {
		return nil, err
	}
	v, err := ksemver.NewSemver(version.GitVersion)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (p *ExternalClusterProvider) VersionsEndpoint(ctx context.Context, configGetter provider.KubermaticConfigurationGetter, providerType kubermaticv1.ExternalClusterProviderType) ([]apiv1.MasterVersion, error) {
	masterVersions := []apiv1.MasterVersion{}
	config, err := configGetter(ctx)
	if err != nil {
		return nil, err
	}

	versions := config.Spec.Versions.ExternalClusters[providerType]
	for _, version := range versions.Versions {
		masterVersions = append(masterVersions, apiv1.MasterVersion{
			Version: version.Semver(),
			Default: versions.Default != nil && version.Equal(versions.Default),
		})
	}
	return masterVersions, nil
}

func (p *ExternalClusterProvider) ValidateKubeconfig(ctx context.Context, kubeconfig []byte) error {
	cfg, err := clientcmd.Load(kubeconfig)
	if err != nil {
		return utilerrors.NewFromKubernetesError(err)
	}

	cli, err := p.GenerateClient(cfg)
	if err != nil {
		return fmt.Errorf("cannot connect to the kubernetes cluster: %w", err)
	}
	// check if kubeconfig can automatically authenticate and get resources.
	if err := cli.List(ctx, &corev1.PodList{}); err != nil {
		return fmt.Errorf("can not retrieve data, check your kubeconfig: %w", err)
	}
	return nil
}

func (p *ExternalClusterProvider) CreateOrUpdateKubeconfigSecretForCluster(ctx context.Context,
	cluster *kubermaticv1.ExternalCluster,
	kubeconfig []byte, namespace string) error {
	kubeconfigRef, err := p.ensureKubeconfigSecret(ctx, cluster, namespace, map[string][]byte{
		resources.ExternalClusterKubeconfig: kubeconfig,
	})
	if err != nil {
		return err
	}
	cluster.Spec.KubeconfigReference = kubeconfigRef
	return nil
}

func (p *ExternalClusterProvider) ListNodes(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (*corev1.NodeList, error) {
	client, err := p.GetClient(ctx, cluster)
	if err != nil {
		return nil, err
	}

	nodes := &corev1.NodeList{}
	if err := client.List(ctx, nodes); err != nil {
		return nil, err
	}

	return nodes, nil
}

func (p *ExternalClusterProvider) GetNode(ctx context.Context, cluster *kubermaticv1.ExternalCluster, nodeName string) (*corev1.Node, error) {
	client, err := p.GetClient(ctx, cluster)
	if err != nil {
		return nil, err
	}

	node := &corev1.Node{}
	if err := client.Get(ctx, ctrlruntimeclient.ObjectKey{Name: nodeName}, node); err != nil {
		return nil, err
	}

	return node, nil
}

func (p *ExternalClusterProvider) IsMetricServerAvailable(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (bool, error) {
	client, err := p.GetClient(ctx, cluster)
	if err != nil {
		return false, err
	}

	allNodeMetricsList := &v1beta1.NodeMetricsList{}
	if err := client.List(ctx, allNodeMetricsList); err != nil {
		if meta.IsNoMatchError(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (p *ExternalClusterProvider) ensureKubeconfigSecret(ctx context.Context, cluster *kubermaticv1.ExternalCluster, namespace string, secretData map[string][]byte) (*providerconfig.GlobalSecretKeySelector, error) {
	creator, err := kubeconfigSecretCreatorGetter(cluster, secretData)
	if err != nil {
		return nil, err
	}

	if err := reconciling.ReconcileSecrets(ctx, []reconciling.NamedSecretCreatorGetter{creator}, namespace, p.clientPrivileged); err != nil {
		return nil, err
	}

	return &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      cluster.GetKubeconfigSecretName(),
			Namespace: namespace,
		},
	}, nil
}

func kubeconfigSecretCreatorGetter(cluster *kubermaticv1.ExternalCluster, secretData map[string][]byte) (reconciling.NamedSecretCreatorGetter, error) {
	projectID := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
	if len(projectID) == 0 {
		return nil, fmt.Errorf("external cluster is missing '%s' label", kubermaticv1.ProjectIDLabelKey)
	}

	return func() (name string, create reconciling.SecretCreator) {
		return cluster.GetKubeconfigSecretName(), func(existing *corev1.Secret) (*corev1.Secret, error) {
			if existing.Labels == nil {
				existing.Labels = map[string]string{}
			}

			existing.Labels[kubermaticv1.ProjectIDLabelKey] = projectID
			existing.Data = secretData

			return existing, nil
		}
	}, nil
}

func (p *ExternalClusterProvider) GetProviderPoolNodes(ctx context.Context,
	cluster *kubermaticv1.ExternalCluster,
	providerNodeLabel, providerNodePoolName string,
) ([]corev1.Node, error) {
	nodes, err := p.ListNodes(ctx, cluster)
	if err != nil {
		return nil, utilerrors.NewFromKubernetesError(err)
	}

	var clusterNodes []corev1.Node
	for _, node := range nodes.Items {
		if node.Labels[providerNodeLabel] == providerNodePoolName {
			clusterNodes = append(clusterNodes, node)
		}
	}

	return clusterNodes, err
}

func getRestConfig(cfg *clientcmdapi.Config) (*rest.Config, error) {
	iconfig := clientcmd.NewNonInteractiveClientConfig(
		*cfg,
		"",
		&clientcmd.ConfigOverrides{},
		nil,
	)

	clientConfig, err := iconfig.ClientConfig()
	if err != nil {
		return nil, err
	}

	// Avoid blocking of the controller by increasing the QPS for user cluster interaction
	clientConfig.QPS = 20
	clientConfig.Burst = 50

	return clientConfig, nil
}

func (p *ExternalClusterProvider) CreateOrUpdateCredentialSecretForCluster(ctx context.Context, cloud *apiv2.ExternalClusterCloudSpec, projectID, clusterID string) (*providerconfig.GlobalSecretKeySelector, error) {
	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   clusterID,
			Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: projectID},
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: kubermaticv1.CloudSpec{},
		},
	}
	if cloud.GKE != nil {
		cluster.Spec.Cloud.GCP = &kubermaticv1.GCPCloudSpec{
			ServiceAccount: cloud.GKE.ServiceAccount,
		}
		err := CreateOrUpdateCredentialSecretForCluster(ctx, p.clientPrivileged, cluster)
		if err != nil {
			return nil, err
		}
		return cluster.Spec.Cloud.GCP.CredentialsReference, nil
	}
	if cloud.EKS != nil {
		cluster.Spec.Cloud.AWS = &kubermaticv1.AWSCloudSpec{
			AccessKeyID:     cloud.EKS.AccessKeyID,
			SecretAccessKey: cloud.EKS.SecretAccessKey,
		}
		err := CreateOrUpdateCredentialSecretForCluster(ctx, p.clientPrivileged, cluster)
		if err != nil {
			return nil, err
		}
		return cluster.Spec.Cloud.AWS.CredentialsReference, nil
	}
	if cloud.AKS != nil {
		cluster.Spec.Cloud.Azure = &kubermaticv1.AzureCloudSpec{
			TenantID:       cloud.AKS.TenantID,
			SubscriptionID: cloud.AKS.SubscriptionID,
			ClientID:       cloud.AKS.ClientID,
			ClientSecret:   cloud.AKS.ClientSecret,
		}
		err := CreateOrUpdateCredentialSecretForCluster(ctx, p.clientPrivileged, cluster)
		if err != nil {
			return nil, err
		}
		return cluster.Spec.Cloud.Azure.CredentialsReference, nil
	}

	return nil, fmt.Errorf("can't create credential secret for unsupported provider")
}

func (p *ExternalClusterProvider) GetMasterClient() ctrlruntimeclient.Client {
	return p.clientPrivileged
}

func (p *ExternalClusterProvider) CreateOrUpdateKubeOneManifestSecret(ctx context.Context, encodedManifest string, externalCluster *kubermaticv1.ExternalCluster) error {
	secretName := resources.KubeOneManifestSecretName
	manifest, err := base64.StdEncoding.DecodeString(encodedManifest)
	if err != nil {
		return fmt.Errorf("failed to decode kubeone manifest: %w", err)
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialKubeOneSecret(ctx, p.clientPrivileged, externalCluster, secretName, map[string][]byte{
		resources.KubeOneManifest: manifest,
	})
	if err != nil {
		return err
	}

	// add secret key selectors to cluster object
	externalCluster.Spec.CloudSpec.KubeOne.ManifestReference = *credentialRef

	return nil
}

func (p *ExternalClusterProvider) CreateOrUpdateKubeOneSSHSecret(ctx context.Context, sshKey apiv2.KubeOneSSHKey, externalCluster *kubermaticv1.ExternalCluster) error {
	secretName := resources.KubeOneSSHSecretName
	privateKey, err := base64.StdEncoding.DecodeString(sshKey.PrivateKey)
	if err != nil {
		return fmt.Errorf("failed to decode kubeone ssh key: %w", err)
	}
	data := map[string][]byte{
		resources.KubeOneSSHPrivateKey: privateKey,
	}
	if sshKey.Passphrase != "" {
		data[resources.KubeOneSSHPassphrase] = []byte(sshKey.Passphrase)
	}

	// move credentials into dedicated Secret
	credentialRef, err := ensureCredentialKubeOneSecret(ctx, p.clientPrivileged, externalCluster, secretName, data)
	if err != nil {
		return err
	}

	// add secret key selectors to cluster object
	externalCluster.Spec.CloudSpec.KubeOne.SSHReference = *credentialRef

	return nil
}

func ExternalClusterPausedChecker(ctx context.Context, externalClusterName string, masterClient ctrlruntimeclient.Client) (bool, error) {
	externalCluster := &kubermaticv1.ExternalCluster{}
	if err := masterClient.Get(ctx, types.NamespacedName{Name: externalClusterName}, externalCluster); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}

		return false, fmt.Errorf("failed to get external cluster %q: %w", externalClusterName, err)
	}

	return externalCluster.Spec.Pause, nil
}
