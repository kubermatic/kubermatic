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
	"go.uber.org/zap"
	ctypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// TestMetering checks if metering components are deployed.
// This test does not test metering reporting
func TestMetering(ctx context.Context, log *zap.SugaredLogger, opts *ctypes.Options) error {
	log.Info("Testing metering availability...")

	key := types.NamespacedName{
		Namespace: "metering",
		Name:      "metering-prometheus-0",
	}

	prometheusPod := corev1.Pod{}
	if err := opts.SeedClusterClient.Get(ctx, key, &prometheusPod); err != nil {
		return errors.New("no metering prometheus Pod found")
	}

	if prometheusPod.Status.Phase != corev1.PodRunning {
		return errors.New("metering prometheus Pod is not running")
	}

	log.Info("Successfully validated metering.")
	return nil
}
