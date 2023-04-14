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
	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v3/cmd/conformance-tester/pkg/scenarios"
	ctypes "k8c.io/kubermatic/v3/cmd/conformance-tester/pkg/types"
	"k8c.io/kubermatic/v3/pkg/kubernetes"
	kkpreconciling "k8c.io/kubermatic/v3/pkg/resources/reconciling"
	"k8c.io/kubermatic/v3/pkg/util/wait"
	"k8c.io/reconciler/pkg/reconciling"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/sets"
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

func (c *kubeClient) log(log *zap.SugaredLogger) *zap.SugaredLogger {
	return log.With("client", "kube")
}

func (c *kubeClient) EnsureSSHKeys(ctx context.Context, log *zap.SugaredLogger) error {
	creators := []kkpreconciling.NamedUserSSHKeyReconcilerFactory{}

	for i, key := range c.opts.PublicKeys {
		c.log(log).Infow("Ensuring UserSSHKey...", "pubkey", string(key))

		name := fmt.Sprintf("ssh-key-%d", i+1)
		creators = append(creators, userSSHKeyReconcilerFactory(name, key))
	}

	return kkpreconciling.ReconcileUserSSHKeys(ctx, creators, "", c.opts.SeedClusterClient)
}

func userSSHKeyReconcilerFactory(keyName string, publicKey []byte) kkpreconciling.NamedUserSSHKeyReconcilerFactory {
	return func() (string, kkpreconciling.UserSSHKeyReconciler) {
		return keyName, func(existing *kubermaticv1.UserSSHKey) (*kubermaticv1.UserSSHKey, error) {
			existing.Spec.Name = "Test SSH Key"
			existing.Spec.PublicKey = string(publicKey)

			if existing.Spec.Clusters == nil {
				existing.Spec.Clusters = []string{}
			}

			return existing, nil
		}
	}
}

func (c *kubeClient) CreateCluster(ctx context.Context, log *zap.SugaredLogger, scenario scenarios.Scenario) (*kubermaticv1.Cluster, error) {
	c.log(log).Info("Creating cluster...")

	name := fmt.Sprintf("%s-%s", c.opts.NamePrefix, rand.String(5))

	// we build up a readable humanReadableName with the various cli parameters and
	// add a random string in the end to ensure we really have a unique humanReadableName.
	humanReadableName := ""
	if c.opts.NamePrefix != "" {
		humanReadableName = c.opts.NamePrefix + "-"
	}
	humanReadableName += scenario.Name() + "-"
	humanReadableName += rand.String(8)

	cluster := &kubermaticv1.Cluster{}
	cluster.Name = name

	cluster.Spec = *scenario.Cluster(c.opts.Secrets)
	cluster.Spec.HumanReadableName = humanReadableName
	cluster.Spec.EnableOperatingSystemManager = pointer.Bool(c.opts.OperatingSystemManagerEnabled)
	cluster.Spec.ClusterNetwork.KonnectivityEnabled = pointer.Bool(c.opts.KonnectivityEnabled)

	if c.opts.DualStackEnabled {
		cluster.Spec.ClusterNetwork.IPFamily = kubermaticv1.IPFamilyDualStack
	}

	if err := c.opts.SeedClusterClient.Create(ctx, cluster); err != nil {
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	waiter := reconciling.WaitUntilObjectExistsInCacheConditionFunc(ctx, c.opts.SeedClusterClient, zap.NewNop().Sugar(), ctrlruntimeclient.ObjectKeyFromObject(cluster), cluster)
	if err := wait.Poll(ctx, 100*time.Millisecond, 5*time.Second, func() (error, error) {
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
	err := kubernetes.UpdateClusterStatus(ctx, c.opts.SeedClusterClient, cluster, func(c *kubermaticv1.Cluster) {
		c.Status.UserEmail = "e2e@kubermatic.com"
	})
	if err != nil {
		return nil, err
	}

	// get all the keys
	sshKeys, err := c.getExistingSSHKeys(ctx)
	if err != nil {
		return nil, err
	}

	// assign them to the new cluster
	for _, key := range sshKeys {
		if err := wait.PollImmediate(ctx, 100*time.Millisecond, 10*time.Second, func() (transient error, terminal error) {
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

	c.log(log).Infof("Successfully created cluster %s", cluster.Name)

	return cluster, nil
}

func (c *kubeClient) getExistingSSHKeys(ctx context.Context) ([]kubermaticv1.UserSSHKey, error) {
	keyList := &kubermaticv1.UserSSHKeyList{}
	if err := c.opts.SeedClusterClient.List(ctx, keyList); err != nil {
		return nil, fmt.Errorf("failed to list SSH keys: %w", err)
	}

	return keyList.Items, nil
}

func (c *kubeClient) CreateMachineDeployments(ctx context.Context, log *zap.SugaredLogger, scenario scenarios.Scenario, userClusterClient ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	c.log(log).Info("Getting existing MachineDeployments...")

	mdList := &clusterv1alpha1.MachineDeploymentList{}
	if err := userClusterClient.List(ctx, mdList); err != nil {
		return fmt.Errorf("failed to list existing MachineDeployments: %w", err)
	}

	existingReplicas := 0
	for _, md := range mdList.Items {
		existingReplicas += int(*md.Spec.Replicas)
	}
	c.log(log).Infof("Found %d pre-existing node replicas", existingReplicas)

	nodeCount := c.opts.NodeCount - existingReplicas
	if nodeCount < 0 {
		return fmt.Errorf("found %d existing replicas and want %d, scale-down not supported", existingReplicas, c.opts.NodeCount)
	}
	if nodeCount == 0 {
		return nil
	}

	// get all the keys
	sshKeys, err := c.getExistingSSHKeys(ctx)
	if err != nil {
		return err
	}

	publicKeys := sets.NewString()
	for _, key := range sshKeys {
		publicKeys.Insert(key.Spec.PublicKey)
	}

	c.log(log).Info("Preparing MachineDeployments...")
	var mds []clusterv1alpha1.MachineDeployment
	if err := wait.PollImmediate(ctx, 3*time.Second, time.Minute, func() (transient error, terminal error) {
		mds, transient = scenario.MachineDeployments(ctx, nodeCount, c.opts.Secrets, cluster, publicKeys.List())
		return transient, nil
	}); err != nil {
		return fmt.Errorf("failed to create MachineDeployments from scenario: %w", err)
	}

	c.log(log).Info("Creating MachineDeployments...")
	for _, md := range mds {
		if err := wait.PollImmediateLog(ctx, log, 5*time.Second, time.Minute, func() (error, error) {
			return userClusterClient.Create(ctx, &md), nil
		}); err != nil {
			return fmt.Errorf("failed to apply MachineDeployments: %w", err)
		}
	}

	c.log(log).Infof("Successfully created MachineDeployments with %d replicas in total", nodeCount)
	return nil
}

func (c *kubeClient) DeleteCluster(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster, timeout time.Duration) error {
	// if there is no timeout, we do not wait for the cluster to be gone
	if timeout == 0 {
		c.log(log).Info("Deleting user cluster now...")

		return ctrlruntimeclient.IgnoreNotFound(c.opts.SeedClusterClient.Delete(ctx, cluster))
	}

	return wait.PollImmediate(ctx, 1*time.Second, timeout, func() (error, error) {
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
			c.log(log).Info("Deleting user cluster now...")

			if err := c.opts.SeedClusterClient.Delete(ctx, cl); err != nil {
				return fmt.Errorf("failed to delete cluster: %w", err), nil
			}

			return errors.New("cluster was deleted"), nil
		}

		return errors.New("cluster still exists"), nil
	})
}
