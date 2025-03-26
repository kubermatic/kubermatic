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
	"os"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	ctypes "k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
)

func deferredGatherUserClusterLogs(ctx context.Context, log *zap.SugaredLogger, opts *ctypes.Options, cluster *kubermaticv1.Cluster) {
	if err := gatherUserClusterLogs(ctx, log, opts, cluster); err != nil {
		log.Errorw("Failed to gather usercluster logs", zap.Error(err))
	}
}

func gatherUserClusterLogs(ctx context.Context, log *zap.SugaredLogger, opts *ctypes.Options, cluster *kubermaticv1.Cluster) error {
	if opts.LogDirectory == "" {
		return nil
	}

	connProvider := opts.ClusterClientProvider

	log.Debug("Attempting to log usercluster pod events and logs")
	kubeconfig, err := connProvider.GetAdminKubeconfig(ctx, cluster)
	if err != nil {
		return fmt.Errorf("failed to get admin kubeconfig: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "kubecfg")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	_, err = tmpFile.Write(kubeconfig)
	tmpFile.Close()
	if err != nil {
		return fmt.Errorf("failed to write kubeconfig: %w", err)
	}

	cmd := exec.CommandContext(ctx, "protokol",
		"--kubeconfig", tmpFile.Name(),
		"--flat",
		"--output", filepath.Join(opts.LogDirectory, fmt.Sprintf("usercluster-%s", cluster.Name)),
		"--namespace", "kube-*",
	)

	// Start protokol and just let it run in the background. It will just end
	// once we destroy the cluster and the apiserver goes away, at which
	// point it simply stops writing logs files.
	if err := cmd.Start(); err != nil {
		log.Errorw("Failed to start protokol to download usercluster logs", zap.Error(err))
	}

	return nil
}
