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
	"encoding/json"
	"fmt"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/restmapper"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ExternalClusterProvider struct that holds required components in order to provide connection to the cluster
type ExternalClusterProvider struct {
	// createSeedImpersonatedClient is used as a ground for impersonation
	createMasterImpersonatedClient impersonationClient
	clientPrivileged               ctrlruntimeclient.Client
	restMapperCache                *restmapper.Cache
}

// NewExternalClusterProvider returns an external cluster provider
func NewExternalClusterProvider(createMasterImpersonatedClient impersonationClient, client ctrlruntimeclient.Client) (*ExternalClusterProvider, error) {
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

func (p *ExternalClusterProvider) GenerateClient(cfg *clientcmdapi.Config) (*ctrlruntimeclient.Client, error) {
	iconfig := clientcmd.NewNonInteractiveClientConfig(
		*cfg,
		resources.KubeconfigDefaultContextKey,
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

	client, err := p.restMapperCache.Client(clientConfig)
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func (p *ExternalClusterProvider) CreateOrUpdateKubeconfigSecretForCluster(ctx context.Context, cluster *kubermaticapiv1.ExternalCluster, kubeconfig *clientcmdapi.Config) error {

	rawData, err := json.Marshal(kubeconfig)
	if err != nil {
		return err
	}
	kubeconfigRef, err := p.ensureKubeconfigSecret(ctx, cluster, map[string][]byte{
		resources.ExternalClusterKubeconfig: rawData,
	})
	if err != nil {
		return err
	}
	cluster.Spec.KubeconfigReference = kubeconfigRef
	return nil
}

func (p *ExternalClusterProvider) ensureKubeconfigSecret(ctx context.Context, cluster *kubermaticapiv1.ExternalCluster, secretData map[string][]byte) (*providerconfig.GlobalSecretKeySelector, error) {
	name := cluster.GetKubeconfigSecretName()

	namespacedName := types.NamespacedName{Namespace: resources.KubermaticNamespace, Name: name}
	existingSecret := &corev1.Secret{}

	if err := p.clientPrivileged.Get(ctx, namespacedName, existingSecret); err != nil {
		if !kerrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to probe for secret %q: %v", name, err)
		}
		return createKubeconfigSecret(ctx, p.clientPrivileged, name, secretData)
	}

	return updateKubeconfigSecret(ctx, p.clientPrivileged, existingSecret, secretData)

}

func createKubeconfigSecret(ctx context.Context, client ctrlruntimeclient.Client, name string, secretData map[string][]byte) (*providerconfig.GlobalSecretKeySelector, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: resources.KubermaticNamespace,
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

func updateKubeconfigSecret(ctx context.Context, client ctrlruntimeclient.Client, existingSecret *corev1.Secret, secretData map[string][]byte) (*providerconfig.GlobalSecretKeySelector, error) {
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
