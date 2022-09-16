/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package jig

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterJig struct {
	client   ctrlruntimeclient.Client
	log      *zap.SugaredLogger
	versions kubermatic.Versions

	// user-controller parameters
	projectName  string
	spec         *kubermaticv1.ClusterSpec
	generateName string
	desiredName  string
	ownerEmail   string
	labels       map[string]string
	presetSecret string
	addons       []Addon

	// data about the generated cluster
	clusterName string
}

type Addon struct {
	Name      string
	Variables *runtime.RawExtension
	Labels    map[string]string
}

func NewClusterJig(client ctrlruntimeclient.Client, log *zap.SugaredLogger) *ClusterJig {
	jig := &ClusterJig{
		client:     client,
		log:        log,
		versions:   kubermatic.NewFakeVersions(),
		spec:       &kubermaticv1.ClusterSpec{},
		labels:     map[string]string{},
		ownerEmail: "e2e@test.kubermatic.com",
		addons:     []Addon{},
	}

	jig.WithTestName("e2e")

	if version := ClusterVersion(log); version != "" {
		jig.WithVersion(version)
	}

	return jig
}

func (j *ClusterJig) WithProject(project *kubermaticv1.Project) *ClusterJig {
	return j.WithProjectName(project.Name)
}

func (j *ClusterJig) WithProjectName(projectName string) *ClusterJig {
	j.projectName = projectName
	return j
}

func (j *ClusterJig) WithExistingCluster(clusterName string) *ClusterJig {
	j.clusterName = clusterName
	return j
}

// WithTestName injects the test name into the cluster name. The name should
// be less than 18 characters in length.
func (j *ClusterJig) WithTestName(name string) *ClusterJig {
	return j.WithName(fmt.Sprintf("kkp-%s-%s", name, BuildID()))
}

func (j *ClusterJig) WithName(name string) *ClusterJig {
	j.desiredName = name
	j.generateName = ""
	return j
}

func (j *ClusterJig) WithGenerateName(prefix string) *ClusterJig {
	j.desiredName = ""
	j.generateName = prefix
	return j
}

func (j *ClusterJig) WithHumanReadableName(name string) *ClusterJig {
	j.spec.HumanReadableName = name
	return j
}

func (j *ClusterJig) WithAddons(addons ...Addon) *ClusterJig {
	j.addons = append(j.addons, addons...)
	return j
}

func (j *ClusterJig) WithVersion(version string) *ClusterJig {
	j.spec.Version = *semver.NewSemverOrDie(version)
	return j
}

func (j *ClusterJig) WithLabels(labels map[string]string) *ClusterJig {
	j.labels = labels
	return j
}

func (j *ClusterJig) WithOwnerEmail(email string) *ClusterJig {
	j.ownerEmail = email
	return j
}

func (j *ClusterJig) WithSSHKeyAgent(enabled bool) *ClusterJig {
	j.spec.EnableUserSSHKeyAgent = pointer.Bool(enabled)
	return j
}

func (j *ClusterJig) WithKonnectivity(enabled bool) *ClusterJig {
	j.spec.ClusterNetwork.KonnectivityEnabled = pointer.Bool(enabled)
	return j
}

func (j *ClusterJig) WithOperatingSystemManager(enabled bool) *ClusterJig {
	j.spec.EnableOperatingSystemManager = pointer.Bool(enabled)
	return j
}

func (j *ClusterJig) WithExposeStrategy(strategy kubermaticv1.ExposeStrategy) *ClusterJig {
	j.spec.ExposeStrategy = strategy
	return j
}

func (j *ClusterJig) WithProxyMode(mode string) *ClusterJig {
	j.spec.ClusterNetwork.ProxyMode = mode
	return j
}

func (j *ClusterJig) WithSpec(spec *kubermaticv1.ClusterSpec) *ClusterJig {
	j.spec = spec
	return j
}

func (j *ClusterJig) WithCloudSpec(spec *kubermaticv1.CloudSpec) *ClusterJig {
	j.spec.Cloud = *spec
	return j
}

func (j *ClusterJig) WithCNIPlugin(settings *kubermaticv1.CNIPluginSettings) *ClusterJig {
	j.spec.CNIPlugin = settings
	return j
}

func (j *ClusterJig) WithPatch(patcher func(c *kubermaticv1.ClusterSpec) *kubermaticv1.ClusterSpec) *ClusterJig {
	j.spec = patcher(j.spec)
	return j
}

func (j *ClusterJig) WithPreset(presetSecret string) *ClusterJig {
	j.presetSecret = presetSecret
	return j
}

func (j *ClusterJig) WithFeatures(features map[string]bool) *ClusterJig {
	j.spec.Features = features
	return j
}

func (j *ClusterJig) ClusterName() string {
	return j.clusterName
}

func (j *ClusterJig) Cluster(ctx context.Context) (*kubermaticv1.Cluster, error) {
	if j.clusterName == "" {
		return nil, errors.New("no cluster created yet")
	}

	clusterProvider, err := j.getClusterProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster provider: %w", err)
	}

	project, err := j.getProject(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return clusterProvider.GetUnsecured(ctx, project, j.clusterName, nil)
}

func (j *ClusterJig) ClusterClient(ctx context.Context) (ctrlruntimeclient.Client, error) {
	clusterProvider, err := j.getClusterProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster provider: %w", err)
	}

	cluster, err := j.Cluster(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current cluster: %w", err)
	}

	var clusterClient ctrlruntimeclient.Client
	err = wait.Poll(ctx, 1*time.Second, 30*time.Second, func() (transient error, terminal error) {
		clusterClient, transient = clusterProvider.GetAdminClientForUserCluster(ctx, cluster)
		return transient, nil
	})
	if err != nil {
		return nil, fmt.Errorf("cluster did not become available: %w", err)
	}

	return clusterClient, nil
}

func (j *ClusterJig) ClusterRESTConfig(ctx context.Context) (*rest.Config, error) {
	clusterProvider, err := j.getClusterProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster provider: %w", err)
	}

	cluster, err := j.Cluster(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current cluster: %w", err)
	}

	var clusterClient *rest.Config
	err = wait.Poll(ctx, 1*time.Second, 30*time.Second, func() (transient error, terminal error) {
		clusterClient, transient = clusterProvider.GetAdminClientConfigForUserCluster(ctx, cluster)
		return transient, nil
	})
	if err != nil {
		return nil, fmt.Errorf("cluster did not become available: %w", err)
	}

	return clusterClient, nil
}

func (j *ClusterJig) Create(ctx context.Context, waitForHealthy bool) (*kubermaticv1.Cluster, error) {
	if j.clusterName != "" {
		return nil, errors.New("cluster was already created; delete it first or use a different cluster jig")
	}

	if j.projectName == "" {
		return nil, errors.New("no project specified")
	}

	project, err := j.getProject(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get parent project: %w", err)
	}

	clusterProvider, err := j.getClusterProvider()
	if err != nil {
		return nil, fmt.Errorf("failed to create cluster provider: %w", err)
	}

	cluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: j.generateName,
			Name:         j.desiredName,
			Labels:       j.labels,
		},
		Spec: *j.spec,
	}

	// Normally this label is injected by the cluster provider, but if
	// you're applying a preset first, the code for takinng the information
	// and putting it into a credential Secret will refuse to work properly
	// if the label isn't set yet. So this is kind of a workaround.
	cluster.Labels[kubermaticv1.ProjectIDLabelKey] = project.Name

	if j.presetSecret != "" {
		cluster, err = j.applyPreset(ctx, cluster)
		if err != nil {
			return nil, fmt.Errorf("failed to apply preset: %w", err)
		}
	}

	j.log.Infow("Creating cluster...",
		"humanname", j.spec.HumanReadableName,
		"version", cluster.Spec.Version,
		"provider", cluster.Spec.Cloud.ProviderName,
		"datacenter", cluster.Spec.Cloud.DatacenterName,
	)
	cluster, err = clusterProvider.NewUnsecured(ctx, project, cluster, j.ownerEmail)
	if err != nil {
		return nil, err
	}

	log := j.log.With("cluster", cluster.Name)

	log.Info("Cluster created successfully.")
	j.clusterName = cluster.Name

	if waitForHealthy {
		log.Info("Waiting for cluster to become healthy...")
		if err = j.WaitForHealthyControlPlane(ctx, 5*time.Minute); err != nil {
			return nil, fmt.Errorf("failed to wait for cluster to become healthy: %w", err)
		}
	}

	if err := j.installAddons(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure addons: %w", err)
	}

	// update our local cluster variable with the newly reconciled address values
	if err := j.client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cluster), cluster); err != nil {
		return nil, fmt.Errorf("failed to retrieve cluster: %w", err)
	}

	return cluster, nil
}

func (j *ClusterJig) WaitForClusterNamespace(ctx context.Context, timeout time.Duration) error {
	if j.clusterName == "" {
		return errors.New("cluster jig has not created a cluster yet")
	}

	return wait.PollLog(ctx, j.log, 5*time.Second, timeout, func() (transient error, terminal error) {
		curCluster := kubermaticv1.Cluster{}
		if err := j.client.Get(ctx, types.NamespacedName{Name: j.clusterName}, &curCluster); err != nil {
			return fmt.Errorf("failed to retrieve cluster: %w", err), nil
		}

		if curCluster.Status.NamespaceName == "" {
			return errors.New("cluster has no namespace yet"), nil
		}

		ns := corev1.Namespace{}
		if err := j.client.Get(ctx, types.NamespacedName{Name: curCluster.Status.NamespaceName}, &ns); err != nil {
			return fmt.Errorf("cluster namespace does not exist yet: %w", err), nil
		}

		return nil, nil
	})
}

func (j *ClusterJig) WaitForHealthyControlPlane(ctx context.Context, timeout time.Duration) error {
	if j.clusterName == "" {
		return errors.New("cluster jig has not created a cluster yet")
	}

	return wait.PollLog(ctx, j.log, 5*time.Second, timeout, func() (transient error, terminal error) {
		curCluster := kubermaticv1.Cluster{}
		if err := j.client.Get(ctx, types.NamespacedName{Name: j.clusterName}, &curCluster); err != nil {
			return fmt.Errorf("failed to retrieve cluster: %w", err), nil
		}

		if !curCluster.Status.ExtendedHealth.AllHealthy() {
			return fmt.Errorf("cluster is unhealthy: %v", getUnhealthyComponents(curCluster.Status.ExtendedHealth)), nil
		}

		return nil, nil
	})
}

func getUnhealthyComponents(health kubermaticv1.ExtendedClusterHealth) []string {
	unhealthy := sets.NewString()
	handle := func(key string, s kubermaticv1.HealthStatus) {
		if s == kubermaticv1.HealthStatusUp {
			return
		}

		// use custom formatting to keep the error message shorter and more readable
		switch s {
		case kubermaticv1.HealthStatusDown:
			s = "down"
		case kubermaticv1.HealthStatusProvisioning:
			s = "provisioning"
		}

		unhealthy.Insert(fmt.Sprintf("%s=%s", key, s))
	}

	// control plane health
	handle("etcd", health.Etcd)
	handle("controller", health.Controller)
	handle("apiserver", health.Apiserver)
	handle("scheduler", health.Scheduler)

	// "all healthy" additions to the control plane
	handle("machineController", health.MachineController)
	handle("cloudProviderInfrastructure", health.CloudProviderInfrastructure)
	handle("userClusterControllerManager", health.UserClusterControllerManager)

	return unhealthy.List()
}

func (j *ClusterJig) Delete(ctx context.Context, synchronous bool) error {
	if j.clusterName == "" {
		return nil
	}

	cluster, err := j.Cluster(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current cluster: %w", err)
	}

	log := j.log.With("cluster", j.clusterName)
	log.Info("Deleting cluster...")

	clusterProvider, err := j.getClusterProvider()
	if err != nil {
		return fmt.Errorf("failed to create cluster provider: %w", err)
	}

	if err := clusterProvider.DeleteUnsecured(ctx, cluster); err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	if synchronous {
		log.Info("Waiting for cluster to be gone...")

		project, err := j.getProject(ctx)
		if err != nil {
			return fmt.Errorf("failed to get project: %w", err)
		}

		err = wait.PollLog(ctx, log, 10*time.Second, 10*time.Minute, func() (transient error, terminal error) {
			_, err := clusterProvider.GetUnsecured(ctx, project, j.clusterName, nil)

			if err == nil {
				return errors.New("cluster still exists"), nil
			}
			if !apierrors.IsNotFound(err) {
				return nil, err
			}

			return nil, nil
		})

		if err != nil {
			return fmt.Errorf("failed to wait for cluster to be gone: %w", err)
		}
	}

	j.clusterName = ""

	return nil
}

func (j *ClusterJig) installAddons(ctx context.Context) error {
	if err := j.WaitForClusterNamespace(ctx, 30*time.Second); err != nil {
		return fmt.Errorf("failed to wait for cluster namespace: %w", err)
	}

	for _, addon := range j.addons {
		if err := j.EnsureAddon(ctx, addon); err != nil {
			return fmt.Errorf("failed to install %q addon: %w", addon.Name, err)
		}
	}

	return nil
}

func (j *ClusterJig) EnsureAddon(ctx context.Context, addon Addon) error {
	j.log.Info("Installing addon...", "addon", addon.Name)

	cluster, err := j.Cluster(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current cluster: %w", err)
	}

	configGetter, err := kubernetes.DynamicKubermaticConfigurationGetterFactory(j.client, KubermaticNamespace())
	if err != nil {
		return fmt.Errorf("failed to create configGetter: %w", err)
	}

	addonProvider := kubernetes.NewAddonProvider(j.client, nil, configGetter)
	if _, err = addonProvider.NewUnsecured(ctx, cluster, addon.Name, addon.Variables, addon.Labels); ctrlruntimeclient.IgnoreAlreadyExists(err) != nil {
		return fmt.Errorf("failed to ensure addon: %w", err)
	}

	return nil
}

func (j *ClusterJig) applyPreset(ctx context.Context, cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
	preset := kubermaticv1.Preset{}
	key := types.NamespacedName{Name: j.presetSecret}
	if err := j.client.Get(ctx, key, &preset); err != nil {
		return nil, fmt.Errorf("failed to get preset %q: %w", j.presetSecret, err)
	}

	switch cluster.Spec.Cloud.ProviderName {
	case string(kubermaticv1.AWSCloudProvider):
		cluster.Spec.Cloud.AWS.AccessKeyID = preset.Spec.AWS.AccessKeyID
		cluster.Spec.Cloud.AWS.SecretAccessKey = preset.Spec.AWS.SecretAccessKey
		cluster.Spec.Cloud.AWS.VPCID = preset.Spec.AWS.VPCID
		cluster.Spec.Cloud.AWS.SecurityGroupID = preset.Spec.AWS.SecurityGroupID
		cluster.Spec.Cloud.AWS.InstanceProfileName = preset.Spec.AWS.InstanceProfileName

	case string(kubermaticv1.HetznerCloudProvider):
		cluster.Spec.Cloud.Hetzner.Token = preset.Spec.Hetzner.Token
		cluster.Spec.Cloud.Hetzner.Network = preset.Spec.Hetzner.Network

	default:
		return nil, fmt.Errorf("provider %q is not yet supported, please implement", cluster.Spec.Cloud.ProviderName)
	}

	return cluster, nil
}

func (j *ClusterJig) getClusterProvider() (*kubernetes.ClusterProvider, error) {
	userClusterConnectionProvider, err := client.NewExternal(j.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create userClusterConnectionProvider: %w", err)
	}

	clusterProvider := kubernetes.NewClusterProvider(
		nil,
		nil,
		userClusterConnectionProvider,
		"",
		nil,
		j.client,
		nil,
		false,
		j.versions,
		nil,
	)

	return clusterProvider, nil
}

func (j *ClusterJig) getProject(ctx context.Context) (*kubermaticv1.Project, error) {
	if j.projectName == "" {
		return nil, errors.New("no parent project specified")
	}

	projectProvider, err := kubernetes.NewPrivilegedProjectProvider(j.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create project provider: %w", err)
	}

	return projectProvider.GetUnsecured(ctx, j.projectName, &provider.ProjectGetOptions{IncludeUninitialized: true})
}
