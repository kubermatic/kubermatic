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

	errorMsgs := []string{}

	for _, pod := range pods.Items {
		err := validatePodSecurityContext(&pod)
		if err != nil {
			errorMsgs = append(errorMsgs, fmt.Sprintf("Pod %s/%s invalid: %v", pod.Namespace, pod.Name, err))
		}
	}

	if len(errorMsgs) == 0 {
		return nil
	}

	return errors.New(strings.Join(errorMsgs, "\n"))
}

func validatePodSecurityContext(pod *corev1.Pod) error {
	var privilegedContainers int
	for _, container := range pod.Spec.Containers {
		if container.SecurityContext != nil && container.SecurityContext.Privileged != nil && *container.SecurityContext.Privileged {
			privilegedContainers++
		}
	}

	// all containers in the Pod are running as privileged, we can skip the Pod; privileged mode disables any seccomp profile
	if len(pod.Spec.Containers) == privilegedContainers {
		return nil
	}

	// if the Pod has a valid security context, we're good
	podError := validatePodSpecSecurityContext(&pod.Spec)
	if podError == nil {
		return nil
	}

	// check if all containers have a dedicated security context
	for _, container := range pod.Spec.Containers {
		containerError := validateContainerSecurityContext(&container)
		if containerError != nil {
			return podError
		}
	}

	return nil
}

func validatePodSpecSecurityContext(spec *corev1.PodSpec) error {
	// no security context means no seccomp profile
	if spec.SecurityContext == nil {
		return errors.New("expected security context, got none")
	}

	return validateSeccompProfile(spec.SecurityContext.SeccompProfile)
}

func validateContainerSecurityContext(container *corev1.Container) error {
	// no security context means no seccomp profile
	if container.SecurityContext == nil {
		return errors.New("expected security context, got none")
	}

	return validateSeccompProfile(container.SecurityContext.SeccompProfile)
}

func validateSeccompProfile(sp *corev1.SeccompProfile) error {
	// no seccomp profile means no profile is applied to the containers
	if sp == nil {
		return errors.New("expected seccomp profile, got none")
	}

	// the 'unconfined' profile disables any seccomp filtering
	if sp.Type == corev1.SeccompProfileTypeUnconfined {
		return fmt.Errorf(
			"seccomp profile is '%s', should be '%s' or '%s'",
			corev1.SeccompProfileTypeUnconfined,
			corev1.SeccompProfileTypeRuntimeDefault,
			corev1.SeccompProfileTypeLocalhost,
		)
	}

	return nil
}
