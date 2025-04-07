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

package tests

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	ctypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestNoK8sGcrImages(ctx context.Context, log *zap.SugaredLogger, opts *ctypes.Options, cluster *kubermaticv1.Cluster) error {
	if !opts.Tests.Has(ctypes.K8sGcrImageTests) {
		log.Info("Tests for k8s.gcr.io disabled, skipping.")
		return nil
	}

	log.Infof("Testing that no k8s.gcr.io images exist in control plane ...")

	pods := &corev1.PodList{}

	if err := opts.SeedClusterClient.List(ctx, pods, &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}); err != nil {
		return fmt.Errorf("failed to list control plane pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return fmt.Errorf("no control plane pods found")
	}

	errorMsgs := []string{}

	for _, pod := range pods.Items {
		for _, container := range pod.Spec.Containers {
			if strings.HasPrefix(container.Image, legacyRegistryK8SGCR) {
				errorMsgs = append(
					errorMsgs,
					fmt.Sprintf("Container %s in Pod %s/%s has image from k8s.gcr.io and should be using registry.k8s.io instead", container.Name, pod.Namespace, pod.Name),
				)
			}
			if strings.HasPrefix(container.Image, fmt.Sprintf("%s/k8s-", legacyRegistryGCR)) ||
				strings.HasPrefix(container.Image, fmt.Sprintf("%s/k8s-", legacyRegistryEUGCR)) ||
				strings.HasPrefix(container.Image, fmt.Sprintf("%s/k8s-", legacyRegistryUSGCR)) {
				errorMsgs = append(
					errorMsgs,
					fmt.Sprintf("Container %s in Pod %s/%s has image from gcr.io/k8s-* and should be using registry.k8s.io instead", container.Name, pod.Namespace, pod.Name),
				)
			}
		}

		for _, initContainer := range pod.Spec.InitContainers {
			if strings.HasPrefix(initContainer.Image, legacyRegistryK8SGCR) {
				errorMsgs = append(
					errorMsgs,
					fmt.Sprintf("InitContainer %s in Pod %s/%s has image from k8s.gcr.io and should be using registry.k8s.io instead", initContainer.Name, pod.Namespace, pod.Name),
				)
			}
			if strings.HasPrefix(initContainer.Image, fmt.Sprintf("%s/k8s-", legacyRegistryGCR)) ||
				strings.HasPrefix(initContainer.Image, fmt.Sprintf("%s/k8s-", legacyRegistryEUGCR)) ||
				strings.HasPrefix(initContainer.Image, fmt.Sprintf("%s/k8s-", legacyRegistryUSGCR)) {
				errorMsgs = append(
					errorMsgs,
					fmt.Sprintf("Container %s in Pod %s/%s has image from gcr.io/k8s-* and should be using registry.k8s.io instead", initContainer.Name, pod.Namespace, pod.Name),
				)
			}
		}
	}

	if len(errorMsgs) == 0 {
		return nil
	}

	return errors.New(strings.Join(errorMsgs, "\n"))
}
