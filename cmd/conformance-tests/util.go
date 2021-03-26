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

package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func podIsReady(p *corev1.Pod) bool {
	for _, c := range p.Status.Conditions {
		if c.Type == corev1.PodReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}

func retryNAttempts(maxAttempts int, f func(attempt int) error) error {
	var err error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		err = f(attempt)
		if err != nil {
			continue
		}
		return nil
	}
	return fmt.Errorf("function did not succeed after %d attempts: %v", maxAttempts, err)
}

// measuredRetryNAttempts wraps retryNAttempts with code that counts
// the executed number of attempts and the runtimes for each
// attempt.
func measuredRetryNAttempts(
	runtimeMetric *prometheus.GaugeVec,
	//nolint:interfacer
	attemptsMetric prometheus.Gauge,
	log *zap.SugaredLogger,
	maxAttempts int,
	f func(attempt int) error,
) func() error {
	return func() error {
		attempts := 0

		err := retryNAttempts(maxAttempts, func(attempt int) error {
			attempts++
			metric := runtimeMetric.With(prometheus.Labels{"attempt": strconv.Itoa(attempt)})

			return measureTime(metric, log, func() error {
				return f(attempt)
			})
		})

		attemptsMetric.Set(float64(attempts))
		updateMetrics(log)

		return err
	}
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
			if !machineHasNodeRef(machine) {
				log.Infow("Machine has no nodeRef yet", "machine", machine.Name)
				return false, nil
			}
		}
		log.Infow("All machines got a Node", "duration-in-seconds", time.Since(startTime).Seconds())
		return true, nil
	})
	return timeout - time.Since(startTime), err
}

func machineHasNodeRef(machine clusterv1alpha1.Machine) bool {
	return machine.Status.NodeRef != nil && machine.Status.NodeRef.Name != ""
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
			if !nodeIsReady(node) {
				log.Infow("Node is not ready", "node", node.Name)
				return false, nil
			}
		}
		log.Infow("All nodes got ready", "duration-in-seconds", time.Since(startTime).Seconds())
		return true, nil
	})
	return timeout - time.Since(startTime), err
}

func nodeIsReady(node corev1.Node) bool {
	for _, c := range node.Status.Conditions {
		if c.Type == corev1.NodeReady {
			return c.Status == corev1.ConditionTrue
		}
	}
	return false
}
