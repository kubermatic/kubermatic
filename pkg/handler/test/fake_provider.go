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

package test

import (
	"context"
	"errors"
	"net/http"
	"time"

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/semver"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	NoExistingFakeProject      = "NoExistingFakeProject"
	NoExistingFakeProjectID    = "NoExistingFakeProject-ID"
	ForbiddenFakeProject       = "ForbiddenFakeProject"
	ForbiddenFakeProjectID     = "ForbiddenFakeProject-ID"
	ExistingFakeProject        = "ExistingFakeProject"
	ExistingFakeProjectID      = "ExistingFakeProject-ID"
	ImpersonatedClientErrorMsg = "forbidden"
)

type FakePrivilegedProjectProvider struct {
}

var _ provider.PrivilegedProjectProvider = &FakePrivilegedProjectProvider{}

type FakeProjectProvider struct {
}

var _ provider.ProjectProvider = &FakeProjectProvider{}

func NewFakeProjectProvider() *FakeProjectProvider {
	return &FakeProjectProvider{}
}

func NewFakePrivilegedProjectProvider() *FakePrivilegedProjectProvider {
	return &FakePrivilegedProjectProvider{}
}

func (f *FakeProjectProvider) New(ctx context.Context, name string, labels map[string]string) (*kubermaticv1.Project, error) {
	return nil, errors.New("not implemented")
}

// Delete deletes the given project as the given user
//
// Note:
// Before deletion project's status.phase is set to ProjectTerminating.
func (f *FakeProjectProvider) Delete(ctx context.Context, userInfo *provider.UserInfo, projectInternalName string) error {
	return errors.New("not implemented")
}

// Get returns the project with the given name.
func (f *FakeProjectProvider) Get(ctx context.Context, userInfo *provider.UserInfo, projectInternalName string, options *provider.ProjectGetOptions) (*kubermaticv1.Project, error) {
	if NoExistingFakeProjectID == projectInternalName || ForbiddenFakeProjectID == projectInternalName {
		return nil, createError(http.StatusForbidden, ImpersonatedClientErrorMsg)
	}

	return GenProject(ExistingFakeProject, kubermaticv1.ProjectActive, DefaultCreationTimestamp().Add(2*time.Minute)), nil
}

// Update update an existing project and returns it.
func (f *FakeProjectProvider) Update(ctx context.Context, userInfo *provider.UserInfo, newProject *kubermaticv1.Project) (*kubermaticv1.Project, error) {
	return nil, errors.New("not implemented")
}

// List gets a list of projects, by default it returns all resources.
// If you want to filter the result please set ProjectListOptions
//
// Note that the list is taken from the cache.
func (f *FakeProjectProvider) List(ctx context.Context, options *provider.ProjectListOptions) ([]*kubermaticv1.Project, error) {
	return nil, errors.New("not implemented")
}

// GetUnsecured returns the project with the given name
// This function is unsafe in a sense that it uses privileged account to get project with the given name.
func (f *FakePrivilegedProjectProvider) GetUnsecured(ctx context.Context, projectInternalName string, options *provider.ProjectGetOptions) (*kubermaticv1.Project, error) {
	if NoExistingFakeProjectID == projectInternalName {
		return nil, createError(http.StatusNotFound, "")
	}
	if ForbiddenFakeProjectID == projectInternalName {
		return nil, createError(http.StatusForbidden, "")
	}

	return nil, nil
}

// DeleteUnsecured deletes any given project
// This function is unsafe in a sense that it uses privileged account to delete project with the given name.
func (f *FakePrivilegedProjectProvider) DeleteUnsecured(ctx context.Context, projectInternalName string) error {
	return nil
}

// UpdateUnsecured update an existing project and returns it
// This function is unsafe in a sense that it uses privileged account to update project.
func (f *FakePrivilegedProjectProvider) UpdateUnsecured(ctx context.Context, project *kubermaticv1.Project) (*kubermaticv1.Project, error) {
	return project, nil
}

func createError(status int32, message string) error {
	return &apierrors.StatusError{ErrStatus: metav1.Status{
		Status:  metav1.StatusFailure,
		Code:    status,
		Reason:  metav1.StatusReasonBadRequest,
		Message: message,
	}}
}

type FakeExternalClusterProvider struct {
	Provider   *kubernetes.ExternalClusterProvider
	FakeClient ctrlruntimeclient.Client
}

var _ provider.ExternalClusterProvider = &FakeExternalClusterProvider{}

func (p *FakeExternalClusterProvider) CreateOrUpdateCredentialSecretForCluster(ctx context.Context, cloud *apiv2.ExternalClusterCloudSpec, projectID, clusterID string) (*providerconfig.GlobalSecretKeySelector, error) {
	return p.Provider.CreateOrUpdateCredentialSecretForCluster(ctx, cloud, projectID, clusterID)
}

func (p *FakeExternalClusterProvider) IsMetricServerAvailable(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (bool, error) {
	return true, nil
}

func (p *FakeExternalClusterProvider) GetNode(ctx context.Context, cluster *kubermaticv1.ExternalCluster, nodeName string) (*corev1.Node, error) {
	node := &corev1.Node{}
	if err := p.FakeClient.Get(ctx, ctrlruntimeclient.ObjectKey{Name: nodeName}, node); err != nil {
		return nil, err
	}

	return node, nil
}

func (p *FakeExternalClusterProvider) ListNodes(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (*corev1.NodeList, error) {
	nodes := &corev1.NodeList{}
	if err := p.FakeClient.List(ctx, nodes); err != nil {
		return nil, err
	}

	return nodes, nil
}

func (p *FakeExternalClusterProvider) Update(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error) {
	return p.Provider.Update(ctx, userInfo, cluster)
}

func (p *FakeExternalClusterProvider) GetVersion(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (*semver.Semver, error) {
	return defaulting.DefaultKubernetesVersioning.Default, nil
}

func (p *FakeExternalClusterProvider) VersionsEndpoint(ctx context.Context, configGetter provider.KubermaticConfigurationGetter, providerType kubermaticv1.ExternalClusterProviderType) ([]apiv1.MasterVersion, error) {
	return p.Provider.VersionsEndpoint(ctx, configGetter, providerType)
}

func (p *FakeExternalClusterProvider) GetClient(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (ctrlruntimeclient.Client, error) {
	return p.FakeClient, nil
}

func (p *FakeExternalClusterProvider) List(ctx context.Context, project *kubermaticv1.Project) (*kubermaticv1.ExternalClusterList, error) {
	return p.Provider.List(ctx, project)
}

func (p *FakeExternalClusterProvider) Get(ctx context.Context, userInfo *provider.UserInfo, clusterName string) (*kubermaticv1.ExternalCluster, error) {
	return p.Provider.Get(ctx, userInfo, clusterName)
}

func (p *FakeExternalClusterProvider) Delete(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.ExternalCluster) error {
	return p.Provider.Delete(ctx, userInfo, cluster)
}

func (p *FakeExternalClusterProvider) GenerateClient(cfg *clientcmdapi.Config) (ctrlruntimeclient.Client, error) {
	return p.FakeClient, nil
}

func (p *FakeExternalClusterProvider) ValidateKubeconfig(_ context.Context, _ []byte) error {
	return nil
}

func (p *FakeExternalClusterProvider) CreateOrUpdateKubeconfigSecretForCluster(ctx context.Context, cluster *kubermaticv1.ExternalCluster, kubeconfig []byte) error {
	return p.Provider.CreateOrUpdateKubeconfigSecretForCluster(ctx, cluster, kubeconfig)
}

func (p *FakeExternalClusterProvider) CreateOrUpdateKubeOneSSHSecret(ctx context.Context, sshKey apiv2.KubeOneSSHKey, externalCluster *kubermaticv1.ExternalCluster) error {
	return p.Provider.CreateOrUpdateKubeOneSSHSecret(ctx, sshKey, externalCluster)
}

func (p *FakeExternalClusterProvider) CreateKubeOneClusterNamespace(ctx context.Context, externalCluster *kubermaticv1.ExternalCluster) error {
	return p.Provider.CreateKubeOneClusterNamespace(ctx, externalCluster)
}

func (p *FakeExternalClusterProvider) CreateOrUpdateKubeOneManifestSecret(ctx context.Context, manifest string, externalCluster *kubermaticv1.ExternalCluster) error {
	return p.Provider.CreateOrUpdateKubeOneManifestSecret(ctx, manifest, externalCluster)
}

func (p *FakeExternalClusterProvider) CreateOrUpdateKubeOneCredentialSecret(ctx context.Context, cloud apiv2.KubeOneCloudSpec, externalCluster *kubermaticv1.ExternalCluster) error {
	return p.Provider.CreateOrUpdateKubeOneCredentialSecret(ctx, cloud, externalCluster)
}

func (p *FakeExternalClusterProvider) New(ctx context.Context, userInfo *provider.UserInfo, project *kubermaticv1.Project, cluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error) {
	return p.Provider.New(ctx, userInfo, project, cluster)
}

func (p *FakeExternalClusterProvider) GetProviderPoolNodes(ctx context.Context, cluster *kubermaticv1.ExternalCluster, providerNodeLabel, providerNodePoolName string) ([]corev1.Node, error) {
	return p.Provider.GetProviderPoolNodes(ctx, cluster, providerNodeLabel, providerNodePoolName)
}
