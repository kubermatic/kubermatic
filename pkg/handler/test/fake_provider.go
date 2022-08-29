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
	k8csemverv1 "k8c.io/kubermatic/v2/pkg/semver/v1"

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

func (p *FakeExternalClusterProvider) VersionsEndpoint(ctx context.Context, configGetter provider.KubermaticConfigurationGetter, providerType kubermaticv1.ExternalClusterProviderType) ([]apiv1.MasterVersion, error) {
	return p.Provider.VersionsEndpoint(ctx, configGetter, providerType)
}

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

func (p *FakeExternalClusterProvider) GetVersion(ctx context.Context, cluster *kubermaticv1.ExternalCluster) (*k8csemverv1.Semver, error) {
	return defaulting.DefaultKubernetesVersioning.Default, nil
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

type FakeConstraintTemplateProvider struct {
	Provider   *kubernetes.ConstraintTemplateProvider
	FakeClient ctrlruntimeclient.Client
}

var _ provider.ConstraintTemplateProvider = &FakeConstraintTemplateProvider{}

func (p *FakeConstraintTemplateProvider) List(ctx context.Context) (*kubermaticv1.ConstraintTemplateList, error) {
	return p.Provider.List(ctx)
}

func (p *FakeConstraintTemplateProvider) Get(ctx context.Context, name string) (*kubermaticv1.ConstraintTemplate, error) {
	return p.Provider.Get(ctx, name)
}

func (p *FakeConstraintTemplateProvider) Create(ctx context.Context, ct *kubermaticv1.ConstraintTemplate) (*kubermaticv1.ConstraintTemplate, error) {
	return p.Provider.Create(ctx, ct)
}

func (p *FakeConstraintTemplateProvider) Update(ctx context.Context, ct *kubermaticv1.ConstraintTemplate) (*kubermaticv1.ConstraintTemplate, error) {
	return p.Provider.Update(ctx, ct)
}

func (p *FakeConstraintTemplateProvider) Delete(ctx context.Context, ct *kubermaticv1.ConstraintTemplate) error {
	return p.Provider.Delete(ctx, ct)
}

type FakeConstraintProvider struct {
	Provider   *kubernetes.ConstraintProvider
	FakeClient ctrlruntimeclient.Client
}

var _ provider.ConstraintProvider = &FakeConstraintProvider{}

func (p *FakeConstraintProvider) List(ctx context.Context, cluster *kubermaticv1.Cluster) (*kubermaticv1.ConstraintList, error) {
	return p.Provider.List(ctx, cluster)
}

func (p *FakeConstraintProvider) Get(ctx context.Context, cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.Constraint, error) {
	return p.Provider.Get(ctx, cluster, name)
}

func (p *FakeConstraintProvider) Delete(ctx context.Context, cluster *kubermaticv1.Cluster, userInfo *provider.UserInfo, name string) error {
	return p.Provider.Delete(ctx, cluster, userInfo, name)
}

func (p *FakeConstraintProvider) Create(ctx context.Context, userInfo *provider.UserInfo, constraint *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error) {
	return p.Provider.Create(ctx, userInfo, constraint)
}

func (p *FakeConstraintProvider) Update(ctx context.Context, userInfo *provider.UserInfo, constraint *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error) {
	return p.Provider.Update(ctx, userInfo, constraint)
}

type FakeDefaultConstraintProvider struct {
	Provider   *kubernetes.DefaultConstraintProvider
	FakeClient ctrlruntimeclient.Client
}

var _ provider.DefaultConstraintProvider = &FakeDefaultConstraintProvider{}

func (p *FakeDefaultConstraintProvider) Create(ctx context.Context, ct *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error) {
	return p.Provider.Create(ctx, ct)
}

func (p *FakeDefaultConstraintProvider) List(ctx context.Context) (*kubermaticv1.ConstraintList, error) {
	return p.Provider.List(ctx)
}

func (p *FakeDefaultConstraintProvider) Get(ctx context.Context, name string) (*kubermaticv1.Constraint, error) {
	return p.Provider.Get(ctx, name)
}

func (p *FakeDefaultConstraintProvider) Delete(ctx context.Context, name string) error {
	return p.Provider.Delete(ctx, name)
}

func (p *FakeDefaultConstraintProvider) Update(ctx context.Context, constraint *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error) {
	return p.Provider.Update(ctx, constraint)
}

type FakePrivilegedAllowedRegistryProvider struct {
	Provider   *kubernetes.PrivilegedAllowedRegistryProvider
	FakeClient ctrlruntimeclient.Client
}

func (p *FakePrivilegedAllowedRegistryProvider) CreateUnsecured(ctx context.Context, wr *kubermaticv1.AllowedRegistry) (*kubermaticv1.AllowedRegistry, error) {
	return p.Provider.CreateUnsecured(ctx, wr)
}

func (p *FakePrivilegedAllowedRegistryProvider) GetUnsecured(ctx context.Context, name string) (*kubermaticv1.AllowedRegistry, error) {
	return p.Provider.GetUnsecured(ctx, name)
}

func (p *FakePrivilegedAllowedRegistryProvider) ListUnsecured(ctx context.Context) (*kubermaticv1.AllowedRegistryList, error) {
	return p.Provider.ListUnsecured(ctx)
}

func (p *FakePrivilegedAllowedRegistryProvider) UpdateUnsecured(ctx context.Context, wr *kubermaticv1.AllowedRegistry) (*kubermaticv1.AllowedRegistry, error) {
	return p.Provider.UpdateUnsecured(ctx, wr)
}

func (p *FakePrivilegedAllowedRegistryProvider) DeleteUnsecured(ctx context.Context, name string) error {
	return p.Provider.DeleteUnsecured(ctx, name)
}

type FakeEtcdBackupConfigProvider struct {
	Provider   *kubernetes.EtcdBackupConfigProvider
	FakeClient ctrlruntimeclient.Client
}

var _ provider.EtcdBackupConfigProvider = &FakeEtcdBackupConfigProvider{}

func (p *FakeEtcdBackupConfigProvider) Create(ctx context.Context, userInfo *provider.UserInfo, ebc *kubermaticv1.EtcdBackupConfig) (*kubermaticv1.EtcdBackupConfig, error) {
	return p.Provider.Create(ctx, userInfo, ebc)
}

func (p *FakeEtcdBackupConfigProvider) Get(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.EtcdBackupConfig, error) {
	return p.Provider.Get(ctx, userInfo, cluster, name)
}

func (p *FakeEtcdBackupConfigProvider) List(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster) (*kubermaticv1.EtcdBackupConfigList, error) {
	return p.Provider.List(ctx, userInfo, cluster)
}

func (p *FakeEtcdBackupConfigProvider) Delete(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, name string) error {
	return p.Provider.Delete(ctx, userInfo, cluster, name)
}

func (p *FakeEtcdBackupConfigProvider) Patch(ctx context.Context, userInfo *provider.UserInfo, oldConfig, newConfig *kubermaticv1.EtcdBackupConfig) (*kubermaticv1.EtcdBackupConfig, error) {
	return p.Provider.Patch(ctx, userInfo, oldConfig, newConfig)
}

type FakeEtcdRestoreProvider struct {
	Provider   *kubernetes.EtcdRestoreProvider
	FakeClient ctrlruntimeclient.Client
}

var _ provider.EtcdRestoreProvider = &FakeEtcdRestoreProvider{}

func (p *FakeEtcdRestoreProvider) Create(ctx context.Context, userInfo *provider.UserInfo, ebc *kubermaticv1.EtcdRestore) (*kubermaticv1.EtcdRestore, error) {
	return p.Provider.Create(ctx, userInfo, ebc)
}

func (p *FakeEtcdRestoreProvider) Get(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.EtcdRestore, error) {
	return p.Provider.Get(ctx, userInfo, cluster, name)
}

func (p *FakeEtcdRestoreProvider) List(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster) (*kubermaticv1.EtcdRestoreList, error) {
	return p.Provider.List(ctx, userInfo, cluster)
}

func (p *FakeEtcdRestoreProvider) Delete(ctx context.Context, userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, name string) error {
	return p.Provider.Delete(ctx, userInfo, cluster, name)
}
