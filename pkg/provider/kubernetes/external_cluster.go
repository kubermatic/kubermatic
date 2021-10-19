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
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	ksemver "k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/util/restmapper"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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

// ExternalClusterProvider struct that holds required components in order to provide connection to the cluster
type ExternalClusterProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	createMasterImpersonatedClient ImpersonationClient
	clientPrivileged               ctrlruntimeclient.Client
	restMapperCache                *restmapper.Cache
}

// NewExternalClusterProvider returns an external cluster provider
func NewExternalClusterProvider(createMasterImpersonatedClient ImpersonationClient, client ctrlruntimeclient.Client) (*ExternalClusterProvider, error) {
	return &ExternalClusterProvider{
		createMasterImpersonatedClient: createMasterImpersonatedClient,
		clientPrivileged:               client,
		restMapperCache:                restmapper.New(),
	}, nil
}

// New creates a brand new external cluster in the system with the given name
func (p *ExternalClusterProvider) New(userInfo *provider.UserInfo, project *kubermaticapiv1.Project, cluster *kubermaticapiv1.ExternalCluster) (*kubermaticapiv1.ExternalCluster, error) {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	addProjectReference(project, cluster)
	if err := masterImpersonatedClient.Create(context.Background(), cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

// NewUnsecured creates a brand new external cluster in the system with the given name
//
// Note that this function:
// is unsafe in a sense that it uses privileged account to create the resource
func (p *ExternalClusterProvider) NewUnsecured(project *kubermaticapiv1.Project, cluster *kubermaticapiv1.ExternalCluster) (*kubermaticapiv1.ExternalCluster, error) {
	addProjectReference(project, cluster)
	if err := p.clientPrivileged.Create(context.Background(), cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

// Get returns the given cluster
func (p *ExternalClusterProvider) Get(userInfo *provider.UserInfo, clusterName string) (*kubermaticapiv1.ExternalCluster, error) {

	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}

	cluster := &kubermaticapiv1.ExternalCluster{}
	if err := masterImpersonatedClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: clusterName}, cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}

// Delete deletes the given cluster
func (p *ExternalClusterProvider) Delete(userInfo *provider.UserInfo, cluster *kubermaticapiv1.ExternalCluster) error {
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
	return masterImpersonatedClient.Delete(context.Background(), cluster, delOpts)
}

// DeleteUnsecured deletes an external cluster.
//
// Note that the admin privileges are used to delete cluster
func (p *ExternalClusterProvider) DeleteUnsecured(cluster *kubermaticapiv1.ExternalCluster) error {
	// Will delete all child's after the object is gone - otherwise the etcd might be deleted before all machines are gone
	// See https://kubernetes.io/docs/concepts/workloads/controllers/garbage-collection/#controlling-how-the-garbage-collector-deletes-dependents
	policy := metav1.DeletePropagationBackground
	delOpts := &ctrlruntimeclient.DeleteOptions{
		PropagationPolicy: &policy,
	}
	return p.clientPrivileged.Delete(context.Background(), cluster, delOpts)
}

// GetUnsecured returns an external cluster for the project and given name.
//
// Note that the admin privileges are used to get cluster
func (p *ExternalClusterProvider) GetUnsecured(clusterName string) (*kubermaticapiv1.ExternalCluster, error) {

	cluster := &kubermaticapiv1.ExternalCluster{}
	if err := p.clientPrivileged.Get(context.Background(), types.NamespacedName{Name: clusterName}, cluster); err != nil {
		return nil, err
	}

	return cluster, nil
}

// List gets all external clusters that belong to the given project
func (p *ExternalClusterProvider) List(project *kubermaticapiv1.Project) (*kubermaticapiv1.ExternalClusterList, error) {
	if project == nil {
		return nil, errors.New("project is missing but required")
	}

	projectClusters := &kubermaticapiv1.ExternalClusterList{}
	selector := labels.SelectorFromSet(map[string]string{kubermaticapiv1.ProjectIDLabelKey: project.Name})
	listOpts := &ctrlruntimeclient.ListOptions{LabelSelector: selector}
	if err := p.clientPrivileged.List(context.Background(), projectClusters, listOpts); err != nil {
		return nil, fmt.Errorf("failed to list clusters: %v", err)
	}

	return projectClusters, nil
}

// Update updates the given cluster
func (p *ExternalClusterProvider) UpdateUnsecured(cluster *kubermaticapiv1.ExternalCluster) (*kubermaticapiv1.ExternalCluster, error) {
	if err := p.clientPrivileged.Update(context.Background(), cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

// Update updates the given cluster
func (p *ExternalClusterProvider) Update(userInfo *provider.UserInfo, cluster *kubermaticapiv1.ExternalCluster) (*kubermaticapiv1.ExternalCluster, error) {
	masterImpersonatedClient, err := createImpersonationClientWrapperFromUserInfo(userInfo, p.createMasterImpersonatedClient)
	if err != nil {
		return nil, err
	}
	if err := masterImpersonatedClient.Update(context.Background(), cluster); err != nil {
		return nil, err
	}
	return cluster, nil
}

func addProjectReference(project *kubermaticapiv1.Project, cluster *kubermaticapiv1.ExternalCluster) {
	if cluster.Labels == nil {
		cluster.Labels = make(map[string]string)
	}
	cluster.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: kubermaticapiv1.SchemeGroupVersion.String(),
			Kind:       kubermaticapiv1.ProjectKindName,
			UID:        project.GetUID(),
			Name:       project.Name,
		},
	}
	cluster.Labels[kubermaticapiv1.ProjectIDLabelKey] = project.Name
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

func (p *ExternalClusterProvider) GetClient(cluster *kubermaticapiv1.ExternalCluster) (ctrlruntimeclient.Client, error) {
	secretKeyGetter := provider.SecretKeySelectorValueFuncFactory(context.Background(), p.clientPrivileged)
	rawKubeconfig, err := secretKeyGetter(cluster.Spec.KubeconfigReference, resources.KubeconfigSecretKey)
	if err != nil {
		return nil, err
	}
	kubeconfig, err := base64.StdEncoding.DecodeString(rawKubeconfig)
	if err != nil {
		return nil, err
	}
	cfg, err := clientcmd.Load(kubeconfig)
	if err != nil {
		return nil, err
	}
	return p.GenerateClient(cfg)
}

func (p *ExternalClusterProvider) GetVersion(cluster *kubermaticapiv1.ExternalCluster) (*ksemver.Semver, error) {
	secretKeyGetter := provider.SecretKeySelectorValueFuncFactory(context.Background(), p.clientPrivileged)
	rawKubeconfig, err := secretKeyGetter(cluster.Spec.KubeconfigReference, resources.KubeconfigSecretKey)
	if err != nil {
		return nil, err
	}
	kubeconfig, err := base64.StdEncoding.DecodeString(rawKubeconfig)
	if err != nil {
		return nil, err
	}
	cfg, err := clientcmd.Load(kubeconfig)
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

func (p *ExternalClusterProvider) CreateOrUpdateKubeconfigSecretForCluster(ctx context.Context, cluster *kubermaticapiv1.ExternalCluster, kubeconfig string) error {
	kubeconfigRef, err := p.ensureKubeconfigSecret(ctx, cluster, map[string][]byte{
		resources.ExternalClusterKubeconfig: []byte(kubeconfig),
	})
	if err != nil {
		return err
	}
	cluster.Spec.KubeconfigReference = kubeconfigRef
	return nil
}

func (p *ExternalClusterProvider) ListNodes(cluster *kubermaticapiv1.ExternalCluster) (*corev1.NodeList, error) {
	client, err := p.GetClient(cluster)
	if err != nil {
		return nil, err
	}

	nodes := &corev1.NodeList{}
	if err := client.List(context.Background(), nodes); err != nil {
		return nil, err
	}

	return nodes, nil
}

func (p *ExternalClusterProvider) GetNode(cluster *kubermaticapiv1.ExternalCluster, nodeName string) (*corev1.Node, error) {
	client, err := p.GetClient(cluster)
	if err != nil {
		return nil, err
	}

	node := &corev1.Node{}
	if err := client.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: nodeName}, node); err != nil {
		return nil, err
	}

	return node, nil
}

func (p *ExternalClusterProvider) IsMetricServerAvailable(cluster *kubermaticapiv1.ExternalCluster) (bool, error) {
	client, err := p.GetClient(cluster)
	if err != nil {
		return false, err
	}

	allNodeMetricsList := &v1beta1.NodeMetricsList{}
	if err := client.List(context.Background(), allNodeMetricsList); err != nil {
		if _, ok := err.(*meta.NoKindMatchError); ok {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (p *ExternalClusterProvider) ensureKubeconfigSecret(ctx context.Context, cluster *kubermaticapiv1.ExternalCluster, secretData map[string][]byte) (*providerconfig.GlobalSecretKeySelector, error) {
	name := cluster.GetKubeconfigSecretName()

	if cluster.Labels == nil {
		return nil, fmt.Errorf("missing cluster labels")
	}
	projectID, ok := cluster.Labels[kubermaticapiv1.ProjectIDLabelKey]
	if !ok {
		return nil, fmt.Errorf("missing cluster projectID label")
	}

	namespacedName := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: name}
	existingSecret := &corev1.Secret{}

	if err := p.clientPrivileged.Get(ctx, namespacedName, existingSecret); err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to probe for secret %q: %v", name, err)
		}
		return createKubeconfigSecret(ctx, p.clientPrivileged, name, projectID, secretData)
	}

	return updateKubeconfigSecret(ctx, p.clientPrivileged, existingSecret, projectID, secretData)

}

func createKubeconfigSecret(ctx context.Context, client ctrlruntimeclient.Client, name, projectID string, secretData map[string][]byte) (*providerconfig.GlobalSecretKeySelector, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: resources.KubermaticNamespace,
			Labels:    map[string]string{kubermaticapiv1.ProjectIDLabelKey: projectID},
		},
		Type: corev1.SecretTypeOpaque,
		Data: secretData,
	}
	if err := client.Create(ctx, secret); err != nil {
		return nil, fmt.Errorf("failed to create kubeconfig secret: %v", err)
	}
	return &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      name,
			Namespace: resources.KubermaticNamespace,
		},
	}, nil
}

func updateKubeconfigSecret(ctx context.Context, client ctrlruntimeclient.Client, existingSecret *corev1.Secret, projectID string, secretData map[string][]byte) (*providerconfig.GlobalSecretKeySelector, error) {
	if existingSecret.Data == nil {
		existingSecret.Data = map[string][]byte{}
	}

	requiresUpdate := false

	for k, v := range secretData {
		if !bytes.Equal(v, existingSecret.Data[k]) {
			requiresUpdate = true
			break
		}
	}

	if existingSecret.Labels == nil {
		existingSecret.Labels = map[string]string{kubermaticapiv1.ProjectIDLabelKey: projectID}
		requiresUpdate = true
	}

	if requiresUpdate {
		existingSecret.Data = secretData
		if err := client.Update(ctx, existingSecret); err != nil {
			return nil, fmt.Errorf("failed to update kubeconfig secret: %v", err)
		}
	}

	return &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{
			Name:      existingSecret.Name,
			Namespace: resources.KubermaticNamespace,
		},
	}, nil
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
	cluster := &kubermaticapiv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   clusterID,
			Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: projectID},
		},
		Spec: kubermaticapiv1.ClusterSpec{
			Cloud: kubermaticapiv1.CloudSpec{},
		},
	}
	if cloud.GKE != nil {
		cluster.Spec.Cloud.GCP = &kubermaticapiv1.GCPCloudSpec{
			ServiceAccount: cloud.GKE.ServiceAccount,
		}
		err := CreateOrUpdateCredentialSecretForCluster(ctx, p.clientPrivileged, cluster)
		if err != nil {
			return nil, err
		}
		return cluster.Spec.Cloud.GCP.CredentialsReference, nil
	}

	return nil, fmt.Errorf("can't create credential secret for unsupported provider")
}
