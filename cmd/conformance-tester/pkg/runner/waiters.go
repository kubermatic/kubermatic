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
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	ctypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/util"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func waitForControlPlane(ctx context.Context, log *zap.SugaredLogger, opts *ctypes.Options, clusterName string) (*kubermaticv1.Cluster, error) {
	started := time.Now()
	namespacedClusterName := types.NamespacedName{Name: clusterName}

	log.Infow("Waiting for control plane to become ready...", "timeout", opts.ControlPlaneReadyWaitTimeout)

	err := wait.PollLog(ctx, log, 5*time.Second, opts.ControlPlaneReadyWaitTimeout, func(ctx context.Context) (transient error, terminal error) {
		newCluster := &kubermaticv1.Cluster{}

		if err := opts.SeedClusterClient.Get(ctx, namespacedClusterName, newCluster); err != nil {
			if apierrors.IsNotFound(err) {
				return err, nil
			}
		}

		// Check for this first, because otherwise we instantly return as the cluster-controller did not
		// create any pods yet
		if !newCluster.Status.ExtendedHealth.AllHealthy() {
			return errors.New("cluster is not all healthy"), nil
		}

		controlPlanePods := &corev1.PodList{}
		if err := opts.SeedClusterClient.List(
			ctx,
			controlPlanePods,
			&ctrlruntimeclient.ListOptions{Namespace: newCluster.Status.NamespaceName},
		); err != nil {
			return nil, fmt.Errorf("failed to list controlplane pods: %w", err)
		}

		unready := sets.New[string]()
		for _, pod := range controlPlanePods.Items {
			if !util.PodIsReady(&pod) {
				unready.Insert(pod.Name)
			}
		}

		if unready.Len() == 0 {
			return nil, nil
		}

		return fmt.Errorf("not all Pods are ready: %v", sets.List(unready)), nil
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

	log.Infow("Control plane is ready", "duration", time.Since(started).Round(time.Second))
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
	started := time.Now()

	log.Infow("Waiting for all Pods to be ready...", "timeout", timeout)

	err := wait.PollLog(ctx, log, opts.UserClusterPollInterval, timeout, func(ctx context.Context) (transient error, terminal error) {
		podList := &corev1.PodList{}
		if err := userClusterClient.List(ctx, podList); err != nil {
			return fmt.Errorf("failed to list Pods in user cluster: %w", err), nil
		}

		unready := sets.New[string]()
		for _, pod := range podList.Items {
			// Ignore pods failing kubelet admission (KKP #6185)
			if !util.PodIsReady(&pod) && !podFailedKubeletAdmissionDueToNodeAffinityPredicate(&pod, log) && !util.PodIsCompleted(&pod) {
				unready.Insert(pod.Name)
			}
		}

		if unready.Len() == 0 {
			return nil, nil
		}

		return fmt.Errorf("not all Pods are ready: %v", sets.List(unready)), nil
	})
	if err != nil {
		return err
	}

	log.Infow("All pods became ready", "duration", time.Since(started).Round(time.Second))

	return nil
}

// waitForMachinesToJoinCluster waits for machines to join the cluster. It does so by checking
// if the machines have a nodeRef. It does not check if the nodeRef is valid.
// All errors are swallowed, only the timeout error is returned.
func waitForMachinesToJoinCluster(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, timeout time.Duration) (time.Duration, error) {
	startTime := time.Now()

	log.Infow("Waiting for machines to join cluster...", "timeout", timeout)

	err := wait.PollLog(ctx, log, 10*time.Second, timeout, func(ctx context.Context) (transient error, terminal error) {
		machineList := &clusterv1alpha1.MachineList{}
		if err := client.List(ctx, machineList); err != nil {
			return fmt.Errorf("failed to list machines: %w", err), nil
		}

		missingMachines := sets.New[string]()
		for _, machine := range machineList.Items {
			if machine.Status.NodeRef == nil || machine.Status.NodeRef.Name == "" {
				missingMachines.Insert(machine.Name)
			}
		}

		if missingMachines.Len() == 0 {
			return nil, nil
		}

		return fmt.Errorf("not all machines have joined: %v", sets.List(missingMachines)), nil
	})

	elapsed := time.Since(startTime)

	if err == nil {
		log.Infow("All machines joined the cluster", "duration", elapsed.Round(time.Second))
	}

	return timeout - elapsed, err
}

// WaitForNodesToBeReady waits for all nodes to be ready. It does so by checking the Nodes "Ready"
// condition. It swallows all errors except for the timeout.
func waitForNodesToBeReady(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, timeout time.Duration) (time.Duration, error) {
	startTime := time.Now()

	log.Infow("Waiting for nodes to become ready...", "timeout", timeout)

	err := wait.PollLog(ctx, log, 10*time.Second, timeout, func(ctx context.Context) (transient error, terminal error) {
		nodeList := &corev1.NodeList{}
		if err := client.List(ctx, nodeList); err != nil {
			return fmt.Errorf("failed to list nodes: %w", err), nil
		}

		unready := sets.New[string]()
		for _, node := range nodeList.Items {
			if !util.NodeIsReady(node) {
				unready.Insert(node.Name)
			}
		}

		if unready.Len() == 0 {
			return nil, nil
		}

		return fmt.Errorf("not all nodes are ready: %v", sets.List(unready)), nil
	})

	elapsed := time.Since(startTime)

	if err == nil {
		log.Infow("All nodes became ready", "duration", elapsed.Round(time.Second))
	}

	return timeout - elapsed, err
}
