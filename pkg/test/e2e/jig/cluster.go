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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterJig struct {
	client   ctrlruntimeclient.Client
	log      *zap.SugaredLogger
	versions kubermatic.Versions

	// user-controller parameters
	projectName  string
	spec         *kubermaticv1.ClusterSpec
	desiredName  string
	ownerEmail   string
	labels       map[string]string
	annotations  map[string]string
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
		client:      client,
		log:         log,
		versions:    kubermatic.GetFakeVersions(),
		spec:        &kubermaticv1.ClusterSpec{},
		annotations: map[string]string{},
		labels:      map[string]string{},
		ownerEmail:  "e2e@test.kubermatic.com",
		addons:      []Addon{},
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

func (j *ClusterJig) WithAnnotations(annotations map[string]string) *ClusterJig {
	j.annotations = annotations
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
	j.spec.EnableUserSSHKeyAgent = ptr.To(enabled)
	return j
}

func (j *ClusterJig) WithKonnectivity(enabled bool) *ClusterJig {
	j.spec.ClusterNetwork.KonnectivityEnabled = ptr.To(enabled) //nolint:staticcheck
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

	cluster := &kubermaticv1.Cluster{}
	if err := j.client.Get(ctx, types.NamespacedName{Name: j.clusterName}, cluster); err != nil {
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	return cluster, nil
}

func (j *ClusterJig) ClusterClient(ctx context.Context) (ctrlruntimeclient.Client, error) {
	cluster, err := j.Cluster(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current cluster: %w", err)
	}

	provider, err := client.NewExternal(j.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create client provider: %w", err)
	}

	var clusterClient ctrlruntimeclient.Client
	err = wait.PollImmediate(ctx, 1*time.Second, 30*time.Second, func(ctx context.Context) (transient error, terminal error) {
		clusterClient, transient = provider.GetClient(ctx, cluster)
		return transient, nil
	})
	if err != nil {
		return nil, fmt.Errorf("cluster did not become available: %w", err)
	}

	return clusterClient, nil
}

func (j *ClusterJig) ClusterRESTConfig(ctx context.Context) (*rest.Config, error) {
	cluster, err := j.Cluster(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get current cluster: %w", err)
	}

	provider, err := client.NewExternal(j.client)
	if err != nil {
		return nil, fmt.Errorf("failed to create client provider: %w", err)
	}

	var clusterClient *rest.Config
	err = wait.PollImmediate(ctx, 1*time.Second, 30*time.Second, func(ctx context.Context) (transient error, terminal error) {
		clusterClient, transient = provider.GetClientConfig(ctx, cluster)
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

	var preset *kubermaticv1.Preset
	if j.presetSecret != "" {
		preset = &kubermaticv1.Preset{}
		if err := j.client.Get(ctx, types.NamespacedName{Name: j.presetSecret}, preset); err != nil {
			return nil, fmt.Errorf("failed to get preset %q: %w", j.presetSecret, err)
		}
	}

	j.log.Infow("Creating cluster...",
		"humanname", j.spec.HumanReadableName,
		"version", j.spec.Version,
		"provider", j.spec.Cloud.ProviderName,
		"datacenter", j.spec.Cloud.DatacenterName,
	)

	creators := []reconciling.NamedClusterReconcilerFactory{
		j.clusterReconcilerFactory(project, preset),
	}

	if err := reconciling.ReconcileClusters(ctx, creators, "", j.client); err != nil {
		return nil, err
	}

	log := j.log.With("cluster", j.desiredName)

	log.Info("Cluster created successfully.")
	j.clusterName = j.desiredName

	if waitForHealthy {
		log.Info("Waiting for cluster to become healthy...")
		if err = j.WaitForHealthyControlPlane(ctx, 5*time.Minute); err != nil {
			return nil, fmt.Errorf("failed to wait for cluster to become healthy: %w", err)
		}
	}

	if err := j.installAddons(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure addons: %w", err)
	}

	return j.Cluster(ctx)
}

func (j *ClusterJig) clusterReconcilerFactory(project *kubermaticv1.Project, preset *kubermaticv1.Preset) reconciling.NamedClusterReconcilerFactory {
	return func() (string, reconciling.ClusterReconciler) {
		return j.desiredName, func(cluster *kubermaticv1.Cluster) (*kubermaticv1.Cluster, error) {
			cluster.Labels = j.labels
			cluster.Annotations = j.annotations
			cluster.Labels[kubermaticv1.ProjectIDLabelKey] = project.Name
			cluster.Spec = *j.spec

			if preset != nil {
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
			}

			return cluster, nil
		}
	}
}

func (j *ClusterJig) WaitForClusterNamespace(ctx context.Context, timeout time.Duration) error {
	if j.clusterName == "" {
		return errors.New("cluster jig has not created a cluster yet")
	}

	return wait.PollLog(ctx, j.log, 5*time.Second, timeout, func(ctx context.Context) (transient error, terminal error) {
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

	return wait.PollLog(ctx, j.log, 5*time.Second, timeout, func(ctx context.Context) (transient error, terminal error) {
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
	unhealthy := sets.New[string]()
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

	return sets.List(unhealthy)
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

	if err := j.client.Delete(ctx, cluster); err != nil {
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	if synchronous {
		log.Info("Waiting for cluster to be gone...")

		err = wait.PollLog(ctx, log, 20*time.Second, 30*time.Minute, func(ctx context.Context) (transient error, terminal error) {
			c := &kubermaticv1.Cluster{}
			err := j.client.Get(ctx, types.NamespacedName{Name: j.clusterName}, c)

			if err == nil {
				return errors.New("cluster still exists"), nil
			}

			return nil, ctrlruntimeclient.IgnoreNotFound(err)
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
	j.log.Infow("Installing addon...", "addon", addon.Name)

	cluster, err := j.Cluster(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current cluster: %w", err)
	}

	addonResource, err := genAddonResource(cluster, addon.Name, addon.Variables, addon.Labels)
	if err != nil {
		return fmt.Errorf("failed to create addon resource: %w", err)
	}

	return ctrlruntimeclient.IgnoreAlreadyExists(j.client.Create(ctx, addonResource))
}

func genAddonResource(cluster *kubermaticv1.Cluster, addonName string, variables *runtime.RawExtension, labels map[string]string) (*kubermaticv1.Addon, error) {
	if cluster.Status.NamespaceName == "" {
		return nil, errors.New("cluster has no namespace name assigned yet")
	}

	if labels == nil {
		labels = map[string]string{}
	}

	return &kubermaticv1.Addon{
		ObjectMeta: metav1.ObjectMeta{
			Name:      addonName,
			Namespace: cluster.Status.NamespaceName,
			Labels:    labels,
		},
		Spec: kubermaticv1.AddonSpec{
			Name:      addonName,
			Variables: variables,
		},
	}, nil
}

func (j *ClusterJig) getProject(ctx context.Context) (*kubermaticv1.Project, error) {
	if j.projectName == "" {
		return nil, errors.New("no parent project specified")
	}

	project := &kubermaticv1.Project{}
	if err := j.client.Get(ctx, types.NamespacedName{Name: j.projectName}, project); err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return project, nil
}
