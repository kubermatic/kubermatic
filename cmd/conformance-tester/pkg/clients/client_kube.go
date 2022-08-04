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

package clients

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/scenarios"
	ctypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/resources/cloudcontroller"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	"k8c.io/kubermatic/v2/pkg/version"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// kubeClient uses a regular Kubernetes client to interact with KKP.
type kubeClient struct {
	opts *ctypes.Options
}

var _ Client = &kubeClient{}

func NewKubeClient(opts *ctypes.Options) Client {
	return &kubeClient{
		opts: opts,
	}
}

func (c *kubeClient) Setup(ctx context.Context, log *zap.SugaredLogger) error {
	return nil
}

func (c *kubeClient) CreateProject(ctx context.Context, log *zap.SugaredLogger, name string) (string, error) {
	log.Info("Creating project...")

	project := &kubermaticv1.Project{}
	project.Name = name
	project.Spec.Name = name

	if err := c.opts.SeedClusterClient.Create(ctx, project); err != nil {
		return "", fmt.Errorf("failed to create project: %w", err)
	}

	if err := wait.PollImmediate(2*time.Second, 1*time.Minute, func() (error, error) {
		p := &kubermaticv1.Project{}
		if err := c.opts.SeedClusterClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(project), p); err != nil {
			return nil, fmt.Errorf("failed to get project: %w", err)
		}

		if p.Status.Phase != kubermaticv1.ProjectActive {
			return fmt.Errorf("project is %s", p.Status.Phase), nil
		}

		return nil, nil
	}); err != nil {
		return "", fmt.Errorf("failed to wait for project to be ready: %w", err)
	}

	return name, nil
}

func (c *kubeClient) EnsureSSHKeys(ctx context.Context, log *zap.SugaredLogger) error {
	creators := []reconciling.NamedKubermaticV1UserSSHKeyCreatorGetter{}

	for i, key := range c.opts.PublicKeys {
		log.Infow("Ensuring UserSSHKey...", "pubkey", string(key))

		name := fmt.Sprintf("ssh-key-%s-%d", c.opts.KubermaticProject, i+1)
		creators = append(creators, userSSHKeyCreatorGetter(name, c.opts.KubermaticProject, key))
	}

	return reconciling.ReconcileKubermaticV1UserSSHKeys(ctx, creators, "", c.opts.SeedClusterClient)
}

func userSSHKeyCreatorGetter(keyName string, project string, publicKey []byte) reconciling.NamedKubermaticV1UserSSHKeyCreatorGetter {
	return func() (string, reconciling.KubermaticV1UserSSHKeyCreator) {
		return keyName, func(existing *kubermaticv1.UserSSHKey) (*kubermaticv1.UserSSHKey, error) {
			existing.Spec.Name = "Test SSH Key"
			existing.Spec.Project = project
			existing.Spec.PublicKey = string(publicKey)

			if existing.Spec.Clusters == nil {
				existing.Spec.Clusters = []string{}
			}

			return existing, nil
		}
	}
}

func (c *kubeClient) CreateCluster(ctx context.Context, log *zap.SugaredLogger, scenario scenarios.Scenario) (*kubermaticv1.Cluster, error) {
	log.Info("Creating cluster...")

	name := fmt.Sprintf("%s-%s", c.opts.NamePrefix, rand.String(5))

	// The cluster humanReadableName must be unique per project;
	// we build up a readable humanReadableName with the various cli parameters annd
	// add a random string in the end to ensure we really have a unique humanReadableName.
	humanReadableName := ""
	if c.opts.NamePrefix != "" {
		humanReadableName = c.opts.NamePrefix + "-"
	}
	humanReadableName += scenario.Name() + "-"
	humanReadableName += rand.String(8)

	cluster := &kubermaticv1.Cluster{}
	cluster.Name = name
	cluster.Labels = map[string]string{
		kubermaticv1.ProjectIDLabelKey: c.opts.KubermaticProject,
	}

	cluster.Spec = *scenario.Cluster(c.opts.Secrets)
	cluster.Spec.HumanReadableName = humanReadableName
	cluster.Spec.UsePodSecurityPolicyAdmissionPlugin = c.opts.PspEnabled
	cluster.Spec.EnableOperatingSystemManager = pointer.Bool(c.opts.OperatingSystemManagerEnabled)

	if cloudcontroller.ExternalCloudControllerFeatureSupported(scenario.Datacenter(), cluster, version.NewFromConfiguration(c.opts.KubermaticConfiguration).GetIncompatibilities()...) {
		cluster.Spec.Features = map[string]bool{kubermaticv1.ClusterFeatureExternalCloudProvider: true}
		if cloudcontroller.ExternalCloudControllerClusterName(cluster) {
			cluster.Spec.Features[kubermaticv1.ClusterFeatureCCMClusterName] = true
		}
		if cluster.Spec.Cloud.VSphere != nil {
			cluster.Spec.Features[kubermaticv1.ClusterFeatureVsphereCSIClusterID] = true
		}
	}

	if err := c.opts.SeedClusterClient.Create(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	waiter := reconciling.WaitUntilObjectExistsInCacheConditionFunc(ctx, c.opts.SeedClusterClient, zap.NewNop().Sugar(), ctrlruntimeclient.ObjectKeyFromObject(cluster), cluster)
	if err := wait.Poll(100*time.Millisecond, 5*time.Second, func() (error, error) {
		success, err := waiter()
		if err != nil {
			return nil, err
		}
		if !success {
			return errors.New("object is not yet in cache"), nil
		}

		return nil, nil
	}); err != nil {
		return nil, fmt.Errorf("failed waiting for the new cluster to appear in the cache: %w", err)
	}

	// In the future, this hack should not be required anymore, until then we sadly have
	// to manually ensure that the owner email is set correctly
	err := kubermaticv1helper.UpdateClusterStatus(ctx, c.opts.SeedClusterClient, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.UserEmail = "e2e@kubermatic.com"
	})
	if err != nil {
		return nil, err
	}

	// fetch all existing SSH keys
	keyList := &kubermaticv1.UserSSHKeyList{}
	if err := c.opts.SeedClusterClient.List(ctx, keyList); err != nil {
		return nil, fmt.Errorf("failed to list SSH keys: %w", err)
	}

	// get all the keys in the current project
	projectKeys := []kubermaticv1.UserSSHKey{}
	for _, key := range keyList.Items {
		if key.Spec.Project == c.opts.KubermaticProject {
			projectKeys = append(projectKeys, key)
		}
	}

	// assign them to the new cluster
	for _, key := range projectKeys {
		if err := wait.PollImmediate(100*time.Millisecond, 10*time.Second, func() (transient error, terminal error) {
			k := &kubermaticv1.UserSSHKey{}
			if err := c.opts.SeedClusterClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(&key), k); err != nil {
				return err, nil
			}

			k.AddToCluster(name)

			return c.opts.SeedClusterClient.Update(ctx, k), nil
		}); err != nil {
			return nil, fmt.Errorf("failed to assign SSH key to cluster: %w", err)
		}
	}

	log.Infof("Successfully created cluster %s", cluster.Name)

	return cluster, nil
}

func (c *kubeClient) CreateNodeDeployments(ctx context.Context, log *zap.SugaredLogger, scenario scenarios.Scenario, userClusterClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	log.Info("Getting existing MachineDeployments...")

	mdList := &clusterv1alpha1.MachineDeploymentList{}
	if err := userClusterClient.List(ctx, mdList); err != nil {
		return fmt.Errorf("failed to list existing MachineDeployments: %w", err)
	}

	existingReplicas := 0
	for _, md := range mdList.Items {
		existingReplicas += int(*md.Spec.Replicas)
	}
	log.Infof("Found %d pre-existing node replicas", existingReplicas)

	nodeCount := c.opts.NodeCount - existingReplicas
	if nodeCount < 0 {
		return fmt.Errorf("found %d existing replicas and want %d, scale-down not supported", existingReplicas, c.opts.NodeCount)
	}
	if nodeCount == 0 {
		return nil
	}

	log.Info("Preparing MachineDeployments...")

	var mds []clusterv1alpha1.MachineDeployment
	if err := wait.PollImmediate(3*time.Second, time.Minute, func() (transient error, terminal error) {
		mds, transient = scenario.MachineDeployments(ctx, nodeCount, c.opts.Secrets, cluster)
		return transient, nil
	}); err != nil {
		return fmt.Errorf("didn't get NodeDeployments from scenario within a minute: %w", err)
	}

	log.Info("Creating MachineDeployments...")
	for _, md := range mds {
		if err := wait.PollImmediateLog(log, 5*time.Second, time.Minute, func() (error, error) {
			return userClusterClient.Create(ctx, &md), nil
		}); err != nil {
			return fmt.Errorf("didn't get NodeDeployments from scenario within a minute: %w", err)
		}
	}

	log.Infof("Successfully created MachineDeployments with %d replicas in total", nodeCount)
	return nil
}

func (c *kubeClient) DeleteCluster(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, timeout time.Duration) error {
	// if there is no timeout, we do not wait for the cluster to be gone
	if timeout == 0 {
		log.Info("Deleting user cluster now...")

		return ctrlruntimeclient.IgnoreNotFound(c.opts.SeedClusterClient.Delete(ctx, cluster))
	}

	return wait.PollImmediate(1*time.Second, timeout, func() (error, error) {
		cl := &kubermaticv1.Cluster{}
		err := c.opts.SeedClusterClient.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cluster), cl)

		// gone already!
		if apierrors.IsNotFound(err) {
			return nil, nil
		}

		if err != nil {
			return fmt.Errorf("failed to get cluster: %w", err), nil
		}

		if cl.DeletionTimestamp == nil {
			log.Info("Deleting user cluster now...")

			if err := c.opts.SeedClusterClient.Delete(ctx, cl); err != nil {
				return fmt.Errorf("failed to delete cluster: %w", err), nil
			}

			return errors.New("cluster was deleted"), nil
		}

		return errors.New("cluster still exists"), nil
	})
}
