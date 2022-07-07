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
	"os"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/semver"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type ClusterJig struct {
	client       ctrlruntimeclient.Client
	log          *zap.SugaredLogger
	kkpNamespace string
	versions     kubermatic.Versions

	// user-controller parameters
	projectName  string
	spec         *kubermaticv1.ClusterSpec
	generateName string
	ownerEmail   string
	labels       map[string]string

	// data about the generated cluster
	clusterName string
}

func NewClusterJig(client ctrlruntimeclient.Client, log *zap.SugaredLogger, kkpNamespace string) *ClusterJig {
	jig := &ClusterJig{
		client:       client,
		log:          log,
		kkpNamespace: kkpNamespace,
		versions:     kubermatic.NewFakeVersions(),
		spec:         &kubermaticv1.ClusterSpec{},
		labels:       map[string]string{},
		generateName: "e2e-",
		ownerEmail:   "e2e@test.kubermatic.com",
	}

	if version := os.Getenv("VERSION_TO_TEST"); version != "" {
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

func (j *ClusterJig) WithGenerateName(prefix string) *ClusterJig {
	j.generateName = prefix
	return j
}

func (j *ClusterJig) WithHumanReadableName(name string) *ClusterJig {
	j.spec.HumanReadableName = name
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

func (j *ClusterJig) WithOperatingSystemManager(enabled bool) *ClusterJig {
	j.spec.EnableOperatingSystemManager = enabled
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
	err = wait.Poll(1*time.Second, 30*time.Second, func() (transient error, terminal error) {
		clusterClient, transient = clusterProvider.GetAdminClientForUserCluster(ctx, cluster)
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
			Labels:       j.labels,
		},
		Spec: *j.spec,
	}

	j.log.Infow("Creating cluster...", "humanname", j.spec.HumanReadableName)
	cluster, err = clusterProvider.NewUnsecured(ctx, project, cluster, j.ownerEmail)
	if err != nil {
		return nil, err
	}

	log := j.log.With("cluster", cluster.Name)

	log.Info("Cluster created successfully.")
	j.clusterName = cluster.Name

	if waitForHealthy {
		log.Info("Waiting for cluster to become healthy...")
		if err = j.WaitForHealthyControlPlane(ctx, 2*time.Minute); err != nil {
			return nil, fmt.Errorf("failed to wait for cluster to become healthy: %w", err)
		}
	}

	// update our local cluster variable with the newly reconciled address values
	if err := j.client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cluster), cluster); err != nil {
		return nil, fmt.Errorf("failed to retrieve cluster: %w", err)
	}

	return cluster, nil
}

func (j *ClusterJig) WaitForHealthyControlPlane(ctx context.Context, timeout time.Duration) error {
	if j.clusterName == "" {
		return errors.New("cluster jig has not created a cluster yet")
	}

	return wait.PollLog(j.log, 5*time.Second, timeout, func() (transient error, terminal error) {
		curCluster := kubermaticv1.Cluster{}
		if err := j.client.Get(ctx, types.NamespacedName{Name: j.clusterName}, &curCluster); err != nil {
			return fmt.Errorf("failed to retrieve cluster: %w", err), nil
		}

		if !curCluster.Status.ExtendedHealth.AllHealthy() {
			return errors.New("cluster is not all healthy"), nil
		}

		return nil, nil
	})
}

func (j *ClusterJig) Delete(ctx context.Context, synchronous bool) error {
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

		err = wait.PollLog(log, 5*time.Second, 10*time.Minute, func() (transient error, terminal error) {
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

func (j *ClusterJig) EnsureAddon(ctx context.Context, addonName string) error {
	cluster, err := j.Cluster(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current cluster: %w", err)
	}

	configGetter, err := provider.DynamicKubermaticConfigurationGetterFactory(j.client, j.kkpNamespace)
	if err != nil {
		return fmt.Errorf("failed to create configGetter: %w", err)
	}

	addonProvider := kubernetes.NewAddonProvider(j.client, nil, configGetter)
	if _, err = addonProvider.NewUnsecured(ctx, cluster, addonName, nil, nil); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to ensure addon: %w", err)
	}

	return nil
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
