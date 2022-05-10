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

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/util"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// printEvents and logs for all pods. Include ready pods, because they may still contain useful information.
func printEventsAndLogsForAllPods(ctx context.Context, log *zap.SugaredLogger, client ctrlruntimeclient.Client, k8sclient kubernetes.Interface, namespace string) error {
	log.Infow("Printing logs for all pods", "namespace", namespace)

	pods := &corev1.PodList{}
	if err := client.List(ctx, pods, ctrlruntimeclient.InNamespace(namespace)); err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	var errs []error
	for _, pod := range pods.Items {
		log := log.With("pod", pod.Name)
		if !util.PodIsReady(&pod) {
			log.Error("Pod is not ready")
		}
		log.Info("Logging events for pod")
		if err := logEventsObject(ctx, log, client, pod.Namespace, pod.UID); err != nil {
			log.Errorw("Failed to log events for pod", zap.Error(err))
			errs = append(errs, err)
		}
		log.Info("Printing logs for pod")
		if err := printLogsForPod(ctx, log, k8sclient, &pod); err != nil {
			log.Errorw("Failed to print logs for pod", zap.Error(kerrors.NewAggregate(err)))
			errs = append(errs, err...)
		}
	}

	return kerrors.NewAggregate(errs)
}

func printLogsForPod(ctx context.Context, log *zap.SugaredLogger, k8sclient kubernetes.Interface, pod *corev1.Pod) []error {
	var errs []error
	for _, container := range pod.Spec.Containers {
		containerLog := log.With("container", container.Name)
		containerLog.Info("Printing logs for container")
		if err := printLogsForContainer(ctx, k8sclient, pod, container.Name); err != nil {
			containerLog.Errorw("Failed to print logs for container", zap.Error(err))
			errs = append(errs, err)
		}
	}
	for _, initContainer := range pod.Spec.InitContainers {
		containerLog := log.With("initContainer", initContainer.Name)
		containerLog.Infow("Printing logs for initContainer")
		if err := printLogsForContainer(ctx, k8sclient, pod, initContainer.Name); err != nil {
			containerLog.Errorw("Failed to print logs for initContainer", zap.Error(err))
			errs = append(errs, err)
		}
	}
	return errs
}

func printLogsForContainer(ctx context.Context, client kubernetes.Interface, pod *corev1.Pod, containerName string) error {
	readCloser, err := client.
		CoreV1().
		Pods(pod.Namespace).
		GetLogs(pod.Name, &corev1.PodLogOptions{Container: containerName}).
		Stream(ctx)
	if err != nil {
		return err
	}
	defer readCloser.Close()

	return util.PrintUnbuffered(readCloser)
}

func logUserClusterPodEventsAndLogs(ctx context.Context, log *zap.SugaredLogger, connProvider *clusterclient.Provider, cluster *kubermaticv1.Cluster) {
	log.Info("Attempting to log usercluster pod events and logs")
	cfg, err := connProvider.GetClientConfig(ctx, cluster)
	if err != nil {
		log.Errorw("Failed to get usercluster admin kubeconfig", zap.Error(err))
		return
	}
	k8sClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Errorw("Failed to construct k8sClient for usercluster", zap.Error(err))
		return
	}
	client, err := connProvider.GetClient(ctx, cluster)
	if err != nil {
		log.Errorw("Failed to construct client for usercluster", zap.Error(err))
		return
	}
	if err := printEventsAndLogsForAllPods(ctx, log, client, k8sClient, ""); err != nil {
		log.Errorw("Failed to print events and logs for usercluster pods", zap.Error(err))
	}
}
