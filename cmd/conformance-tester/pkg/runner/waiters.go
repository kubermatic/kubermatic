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

package runner

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	ctypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/util"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func waitForControlPlane(ctx context.Context, log *zap.SugaredLogger, opts *ctypes.Options, clusterName string) (*kubermaticv1.Cluster, error) {
	log.Debug("Waiting for control plane to become ready...")
	started := time.Now()
	namespacedClusterName := types.NamespacedName{Name: clusterName}

	err := wait.Poll(3*time.Second, opts.ControlPlaneReadyWaitTimeout, func() (done bool, err error) {
		newCluster := &kubermaticv1.Cluster{}

		if err := opts.SeedClusterClient.Get(ctx, namespacedClusterName, newCluster); err != nil {
			if kerrors.IsNotFound(err) {
				return false, nil
			}
		}

		// Check for this first, because otherwise we instantly return as the cluster-controller did not
		// create any pods yet
		if !newCluster.Status.ExtendedHealth.AllHealthy() {
			return false, nil
		}

		controlPlanePods := &corev1.PodList{}
		if err := opts.SeedClusterClient.List(
			ctx,
			controlPlanePods,
			&ctrlruntimeclient.ListOptions{Namespace: newCluster.Status.NamespaceName},
		); err != nil {
			return false, fmt.Errorf("failed to list controlplane pods: %w", err)
		}

		for _, pod := range controlPlanePods.Items {
			if !util.PodIsReady(&pod) {
				return false, nil
			}
		}

		return true, nil
	})
	// Timeout or other error
	if err != nil {
		return nil, err
	}

	// Get copy of latest version
	cluster := &kubermaticv1.Cluster{}
	if err := opts.SeedClusterClient.Get(ctx, namespacedClusterName, cluster); err != nil {
		return nil, err
	}

	log.Debugf("Control plane became ready after %.2f seconds", time.Since(started).Seconds())
	return cluster, nil
}

// podFailedKubeletAdmissionDueToNodeAffinityPredicate detects a condition in
// which a pod is scheduled but fails kubelet admission due to a race condition
// between scheduler and kubelet.
// see: https://github.com/kubernetes/kubernetes/issues/93338
func podFailedKubeletAdmissionDueToNodeAffinityPredicate(p *corev1.Pod, log *zap.SugaredLogger) bool {
	failedAdmission := p.Status.Phase == "Failed" && p.Status.Reason == "NodeAffinity"
	if failedAdmission {
		log.Infow("pod failed kubelet admission due to NodeAffinity predicate", "pod", *p)
	}

	return failedAdmission
}

func waitUntilAllPodsAreReady(ctx context.Context, log *zap.SugaredLogger, opts *ctypes.Options, userClusterClient ctrlruntimeclient.Client, timeout time.Duration) error {
	log.Debug("Waiting for all pods to be ready...")
	started := time.Now()

	err := wait.Poll(opts.UserClusterPollInterval, timeout, func() (done bool, err error) {
		podList := &corev1.PodList{}
		if err := userClusterClient.List(ctx, podList); err != nil {
			log.Warnw("Failed to load pod list while waiting until all pods are running", zap.Error(err))
			return false, nil
		}

		for _, pod := range podList.Items {
			// Ignore pods failing kubelet admission (KKP #6185)
			if !util.PodIsReady(&pod) && !podFailedKubeletAdmissionDueToNodeAffinityPredicate(&pod, log) {
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		return err
	}

	log.Debugf("All pods became ready after %.2f seconds", time.Since(started).Seconds())
	return nil
}

// waitForMachinesToJoinCluster waits for machines to join the cluster. It does so by checking
// if the machines have a nodeRef. It does not check if the nodeRef is valid.
// All errors are swallowed, only the timeout error is returned.
func waitForMachinesToJoinCluster(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, timeout time.Duration) (time.Duration, error) {
	startTime := time.Now()

	err := wait.Poll(10*time.Second, timeout, func() (bool, error) {
		machineList := &clusterv1alpha1.MachineList{}
		if err := client.List(ctx, machineList); err != nil {
			log.Warnw("Failed to list machines", zap.Error(err))
			return false, nil
		}

		for _, machine := range machineList.Items {
			if machine.Status.NodeRef == nil || machine.Status.NodeRef.Name == "" {
				log.Infow("Machine has no nodeRef yet", "machine", machine.Name)
				return false, nil
			}
		}

		log.Infow("All machines got a Node", "duration-in-seconds", time.Since(startTime).Seconds())
		return true, nil
	})

	return timeout - time.Since(startTime), err
}

// WaitForNodesToBeReady waits for all nodes to be ready. It does so by checking the Nodes "Ready"
// condition. It swallows all errors except for the timeout.
func waitForNodesToBeReady(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, timeout time.Duration) (time.Duration, error) {
	startTime := time.Now()

	err := wait.Poll(10*time.Second, timeout, func() (bool, error) {
		nodeList := &corev1.NodeList{}
		if err := client.List(ctx, nodeList); err != nil {
			log.Warnw("Failed to list nodes", zap.Error(err))
			return false, nil
		}

		for _, node := range nodeList.Items {
			if !util.NodeIsReady(node) {
				log.Infow("Node is not ready", "node", node.Name)
				return false, nil
			}
		}

		log.Infow("All nodes got ready", "duration-in-seconds", time.Since(startTime).Seconds())
		return true, nil
	})

	return timeout - time.Since(startTime), err
}
