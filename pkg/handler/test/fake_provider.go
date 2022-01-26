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
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/semver"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
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

type FakeProjectProvider struct {
}

func NewFakeProjectProvider() *FakeProjectProvider {
	return &FakeProjectProvider{}
}

func NewFakePrivilegedProjectProvider() *FakePrivilegedProjectProvider {
	return &FakePrivilegedProjectProvider{}
}

func (f *FakeProjectProvider) New(user []*kubermaticv1.User, name string, labels map[string]string) (*kubermaticv1.Project, error) {
	return nil, errors.New("not implemented")
}

// Delete deletes the given project as the given user
//
// Note:
// Before deletion project's status.phase is set to ProjectTerminating.
func (f *FakeProjectProvider) Delete(userInfo *provider.UserInfo, projectInternalName string) error {
	return errors.New("not implemented")
}

// Get returns the project with the given name.
func (f *FakeProjectProvider) Get(userInfo *provider.UserInfo, projectInternalName string, options *provider.ProjectGetOptions) (*kubermaticv1.Project, error) {
	if NoExistingFakeProjectID == projectInternalName || ForbiddenFakeProjectID == projectInternalName {
		return nil, createError(http.StatusForbidden, ImpersonatedClientErrorMsg)
	}

	return GenProject(ExistingFakeProject, kubermaticv1.ProjectActive, DefaultCreationTimestamp().Add(2*time.Minute)), nil
}

// Update update an existing project and returns it.
func (f *FakeProjectProvider) Update(userInfo *provider.UserInfo, newProject *kubermaticv1.Project) (*kubermaticv1.Project, error) {
	return nil, errors.New("not implemented")
}

// List gets a list of projects, by default it returns all resources.
// If you want to filter the result please set ProjectListOptions
//
// Note that the list is taken from the cache.
func (f *FakeProjectProvider) List(options *provider.ProjectListOptions) ([]*kubermaticv1.Project, error) {
	return nil, errors.New("not implemented")
}

// GetUnsecured returns the project with the given name
// This function is unsafe in a sense that it uses privileged account to get project with the given name.
func (f *FakePrivilegedProjectProvider) GetUnsecured(projectInternalName string, options *provider.ProjectGetOptions) (*kubermaticv1.Project, error) {
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
func (f *FakePrivilegedProjectProvider) DeleteUnsecured(projectInternalName string) error {
	return nil
}

// UpdateUnsecured update an existing project and returns it
// This function is unsafe in a sense that it uses privileged account to update project.
func (f *FakePrivilegedProjectProvider) UpdateUnsecured(project *kubermaticv1.Project) (*kubermaticv1.Project, error) {
	return project, nil
}

func createError(status int32, message string) error {
	return &kerrors.StatusError{ErrStatus: metav1.Status{
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

func (p *FakeExternalClusterProvider) CreateOrUpdateCredentialSecretForCluster(ctx context.Context, cloud *apiv2.ExternalClusterCloudSpec, projectID, clusterID string) (*providerconfig.GlobalSecretKeySelector, error) {
	return p.Provider.CreateOrUpdateCredentialSecretForCluster(ctx, cloud, projectID, clusterID)
}

func (p *FakeExternalClusterProvider) IsMetricServerAvailable(cluster *kubermaticv1.ExternalCluster) (bool, error) {
	return true, nil
}

func (p *FakeExternalClusterProvider) GetNode(cluster *kubermaticv1.ExternalCluster, nodeName string) (*corev1.Node, error) {
	node := &corev1.Node{}
	if err := p.FakeClient.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: nodeName}, node); err != nil {
		return nil, err
	}

	return node, nil
}

func (p *FakeExternalClusterProvider) ListNodes(cluster *kubermaticv1.ExternalCluster) (*corev1.NodeList, error) {
	nodes := &corev1.NodeList{}
	if err := p.FakeClient.List(context.Background(), nodes); err != nil {
		return nil, err
	}

	return nodes, nil
}

func (p *FakeExternalClusterProvider) Update(userInfo *provider.UserInfo, cluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error) {
	return p.Provider.Update(userInfo, cluster)
}

func (p *FakeExternalClusterProvider) GetVersion(cluster *kubermaticv1.ExternalCluster) (*semver.Semver, error) {
	return semver.NewSemver(DefaultKubernetesVersion)
}

func (p *FakeExternalClusterProvider) GetClient(cluster *kubermaticv1.ExternalCluster) (ctrlruntimeclient.Client, error) {
	return p.FakeClient, nil
}

func (p *FakeExternalClusterProvider) List(project *kubermaticv1.Project) (*kubermaticv1.ExternalClusterList, error) {
	return p.Provider.List(project)
}

func (p *FakeExternalClusterProvider) Get(userInfo *provider.UserInfo, clusterName string) (*kubermaticv1.ExternalCluster, error) {
	return p.Provider.Get(userInfo, clusterName)
}

func (p *FakeExternalClusterProvider) Delete(userInfo *provider.UserInfo, cluster *kubermaticv1.ExternalCluster) error {
	return p.Provider.Delete(userInfo, cluster)
}

func (p *FakeExternalClusterProvider) GenerateClient(cfg *clientcmdapi.Config) (ctrlruntimeclient.Client, error) {
	return p.FakeClient, nil
}

func (p *FakeExternalClusterProvider) CreateOrUpdateKubeconfigSecretForCluster(ctx context.Context, cluster *kubermaticv1.ExternalCluster, kubeconfig string) error {
	return p.Provider.CreateOrUpdateKubeconfigSecretForCluster(ctx, cluster, kubeconfig)
}

func (p *FakeExternalClusterProvider) New(userInfo *provider.UserInfo, project *kubermaticv1.Project, cluster *kubermaticv1.ExternalCluster) (*kubermaticv1.ExternalCluster, error) {
	return p.Provider.New(userInfo, project, cluster)
}

type FakeConstraintTemplateProvider struct {
	Provider   *kubernetes.ConstraintTemplateProvider
	FakeClient ctrlruntimeclient.Client
}

func (p *FakeConstraintTemplateProvider) List() (*kubermaticv1.ConstraintTemplateList, error) {
	return p.Provider.List()
}

func (p *FakeConstraintTemplateProvider) Get(name string) (*kubermaticv1.ConstraintTemplate, error) {
	return p.Provider.Get(name)
}

func (p *FakeConstraintTemplateProvider) Create(ct *kubermaticv1.ConstraintTemplate) (*kubermaticv1.ConstraintTemplate, error) {
	return p.Provider.Create(ct)
}

func (p *FakeConstraintTemplateProvider) Update(ct *kubermaticv1.ConstraintTemplate) (*kubermaticv1.ConstraintTemplate, error) {
	return p.Provider.Update(ct)
}

func (p *FakeConstraintTemplateProvider) Delete(ct *kubermaticv1.ConstraintTemplate) error {
	return p.Provider.Delete(ct)
}

type FakeConstraintProvider struct {
	Provider   *kubernetes.ConstraintProvider
	FakeClient ctrlruntimeclient.Client
}

func (p *FakeConstraintProvider) List(cluster *kubermaticv1.Cluster) (*kubermaticv1.ConstraintList, error) {
	return p.Provider.List(cluster)
}

func (p *FakeConstraintProvider) Get(cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.Constraint, error) {
	return p.Provider.Get(cluster, name)
}

func (p *FakeConstraintProvider) Delete(cluster *kubermaticv1.Cluster, userInfo *provider.UserInfo, name string) error {
	return p.Provider.Delete(cluster, userInfo, name)
}

func (p *FakeConstraintProvider) Create(userInfo *provider.UserInfo, constraint *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error) {
	return p.Provider.Create(userInfo, constraint)
}

func (p *FakeConstraintProvider) Update(userInfo *provider.UserInfo, constraint *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error) {
	return p.Provider.Update(userInfo, constraint)
}

type FakeDefaultConstraintProvider struct {
	Provider   *kubernetes.DefaultConstraintProvider
	FakeClient ctrlruntimeclient.Client
}

func (p *FakeDefaultConstraintProvider) Create(ct *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error) {
	return p.Provider.Create(ct)
}

func (p *FakeDefaultConstraintProvider) List() (*kubermaticv1.ConstraintList, error) {
	return p.Provider.List()
}

func (p *FakeDefaultConstraintProvider) Get(name string) (*kubermaticv1.Constraint, error) {
	return p.Provider.Get(name)
}

func (p *FakeDefaultConstraintProvider) Delete(name string) error {
	return p.Provider.Delete(name)
}

func (p *FakeDefaultConstraintProvider) Update(constraint *kubermaticv1.Constraint) (*kubermaticv1.Constraint, error) {
	return p.Provider.Update(constraint)
}

type FakePrivilegedAllowedRegistryProvider struct {
	Provider   *kubernetes.PrivilegedAllowedRegistryProvider
	FakeClient ctrlruntimeclient.Client
}

func (p *FakePrivilegedAllowedRegistryProvider) CreateUnsecured(wr *kubermaticv1.AllowedRegistry) (*kubermaticv1.AllowedRegistry, error) {
	return p.Provider.CreateUnsecured(wr)
}

func (p *FakePrivilegedAllowedRegistryProvider) GetUnsecured(name string) (*kubermaticv1.AllowedRegistry, error) {
	return p.Provider.GetUnsecured(name)
}

func (p *FakePrivilegedAllowedRegistryProvider) ListUnsecured() (*kubermaticv1.AllowedRegistryList, error) {
	return p.Provider.ListUnsecured()
}

func (p *FakePrivilegedAllowedRegistryProvider) UpdateUnsecured(wr *kubermaticv1.AllowedRegistry) (*kubermaticv1.AllowedRegistry, error) {
	return p.Provider.UpdateUnsecured(wr)
}

func (p *FakePrivilegedAllowedRegistryProvider) DeleteUnsecured(name string) error {
	return p.Provider.DeleteUnsecured(name)
}

type FakeEtcdBackupConfigProvider struct {
	Provider   *kubernetes.EtcdBackupConfigProvider
	FakeClient ctrlruntimeclient.Client
}

func (p *FakeEtcdBackupConfigProvider) Create(userInfo *provider.UserInfo, ebc *kubermaticv1.EtcdBackupConfig) (*kubermaticv1.EtcdBackupConfig, error) {
	return p.Provider.Create(userInfo, ebc)
}

func (p *FakeEtcdBackupConfigProvider) Get(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.EtcdBackupConfig, error) {
	return p.Provider.Get(userInfo, cluster, name)
}

func (p *FakeEtcdBackupConfigProvider) List(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster) (*kubermaticv1.EtcdBackupConfigList, error) {
	return p.Provider.List(userInfo, cluster)
}

func (p *FakeEtcdBackupConfigProvider) Delete(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, name string) error {
	return p.Provider.Delete(userInfo, cluster, name)
}

func (p *FakeEtcdBackupConfigProvider) Patch(userInfo *provider.UserInfo, oldConfig, newConfig *kubermaticv1.EtcdBackupConfig) (*kubermaticv1.EtcdBackupConfig, error) {
	return p.Provider.Patch(userInfo, oldConfig, newConfig)
}

type FakeEtcdRestoreProvider struct {
	Provider   *kubernetes.EtcdRestoreProvider
	FakeClient ctrlruntimeclient.Client
}

func (p *FakeEtcdRestoreProvider) Create(userInfo *provider.UserInfo, ebc *kubermaticv1.EtcdRestore) (*kubermaticv1.EtcdRestore, error) {
	return p.Provider.Create(userInfo, ebc)
}

func (p *FakeEtcdRestoreProvider) Get(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, name string) (*kubermaticv1.EtcdRestore, error) {
	return p.Provider.Get(userInfo, cluster, name)
}

func (p *FakeEtcdRestoreProvider) List(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster) (*kubermaticv1.EtcdRestoreList, error) {
	return p.Provider.List(userInfo, cluster)
}

func (p *FakeEtcdRestoreProvider) Delete(userInfo *provider.UserInfo, cluster *kubermaticv1.Cluster, name string) error {
	return p.Provider.Delete(userInfo, cluster, name)
}
