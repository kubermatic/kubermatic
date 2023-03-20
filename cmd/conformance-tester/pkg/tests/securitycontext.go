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
	"fmt"
	"strings"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	ctypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestUserClusterControlPlaneSecurityContext(ctx context.Context, log *zap.SugaredLogger, opts *ctypes.Options, cluster *kubermaticv1.Cluster) error {
	if !opts.Tests.Has(ctypes.SecurityContextTests) {
		log.Info("Security context tests disabled, skipping.")
		return nil
	}

	log.Infof("Testing security context of control plane components in Seed cluster...")

	pods := &corev1.PodList{}

	if err := opts.SeedClusterClient.List(ctx, pods, &ctrlruntimeclient.ListOptions{Namespace: cluster.Status.NamespaceName}); err != nil {
		return fmt.Errorf("failed to list control plane pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return fmt.Errorf("no control plane pods found")
	}

	errors := []string{}

	for _, pod := range pods.Items {
		var privilegedContainers int
		for _, container := range pod.Spec.Containers {
			if container.SecurityContext != nil && container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
				privilegedContainers++
			}
		}

		// all containers in the Pod are running as privileged, we can skip the Pod; privileged mode disables any seccomp profile
		if len(pod.Spec.Containers) == privilegedContainers {
			continue
		}

		// no security context means no seccomp profile
		if pod.Spec.SecurityContext == nil {
			errors = append(
				errors,
				fmt.Sprintf("expected security context on Pod %s/%s, got none", pod.Namespace, pod.Name),
			)
			continue
		}

		// no seccomp profile means no profile is applied to the containers
		if pod.Spec.SecurityContext.SeccompProfile == nil {
			errors = append(
				errors,
				fmt.Sprintf("expected seccomp profile on Pod %s/%s, got none", pod.Namespace, pod.Name),
			)
			continue
		}

		// the 'unconfined' profile disables any seccomp filtering
		if pod.Spec.SecurityContext.SeccompProfile.Type == corev1.SeccompProfileTypeUnconfined {
			errors = append(
				errors,
				fmt.Sprintf(
					"seccomp profile of Pod %s/%s is '%s', should be '%s' or '%s'", pod.Namespace, pod.Name,
					corev1.SeccompProfileTypeUnconfined, corev1.SeccompProfileTypeRuntimeDefault, corev1.SeccompProfileTypeLocalhost,
				),
			)
			continue
		}
	}

	if len(errors) == 0 {
		return nil
	}

	return fmt.Errorf(strings.Join(errors, "\n"))
}
