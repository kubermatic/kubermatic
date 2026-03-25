/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package etcd

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"strings"

	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/resources"
)

func (e *Cluster) StartEtcdCmd(ctx context.Context, log *zap.SugaredLogger) (*exec.Cmd, error) {
	if _, err := os.Stat(etcdCommandPath); errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("failed to find etcd executable: %w", err)
	}

	cmd := exec.CommandContext(ctx, etcdCommandPath, etcdCmd(e)...)
	cmd.Env = os.Environ()
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	log.Infof("starting etcd command: %s %s", etcdCommandPath, strings.Join(etcdCmd(e), " "))

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start etcd: %w", err)
	}

	return cmd, nil
}

func etcdCmd(config *Cluster) []string {
	podIPAddr := net.JoinHostPort(config.PodIP, "2379")
	podIPMetricsAddr := net.JoinHostPort(config.PodIP, "2378")

	cmd := []string{
		fmt.Sprintf("--name=%s", config.PodName),
		fmt.Sprintf("--data-dir=%s", config.DataDir),
		fmt.Sprintf("--initial-cluster=%s", strings.Join(config.initialMembers, ",")),
		fmt.Sprintf("--initial-cluster-token=%s", config.Token),
		fmt.Sprintf("--initial-cluster-state=%s", config.initialState),
		fmt.Sprintf("--advertise-client-urls=https://%s.etcd.%s.svc.cluster.local:2379,https://%s", config.PodName, config.namespace, podIPAddr),
		fmt.Sprintf("--listen-client-urls=https://%s,https://127.0.0.1:2379", podIPAddr),
		fmt.Sprintf("--listen-metrics-urls=http://%s,http://127.0.0.1:2378", podIPMetricsAddr),
		"--client-cert-auth",
		fmt.Sprintf("--trusted-ca-file=%s", resources.EtcdTrustedCAFile),
		fmt.Sprintf("--cert-file=%s", resources.EtcdCertFile),
		fmt.Sprintf("--key-file=%s", resources.EtcdKeyFile),
		fmt.Sprintf("--peer-cert-file=%s", resources.EtcdCertFile),
		fmt.Sprintf("--peer-key-file=%s", resources.EtcdKeyFile),
		fmt.Sprintf("--peer-trusted-ca-file=%s", resources.EtcdTrustedCAFile),
		"--auto-compaction-retention=8",
	}

	// set TLS only peer URLs
	if config.usePeerTLSOnly {
		cmd = append(cmd, []string{
			fmt.Sprintf("--listen-peer-urls=https://%s", net.JoinHostPort(config.PodIP, "2381")),
			fmt.Sprintf("--initial-advertise-peer-urls=https://%s.etcd.%s.svc.cluster.local:2381", config.PodName, config.namespace),
			"--peer-client-cert-auth",
		}...)
	} else {
		// 'mixed' mode clusters should start with both plaintext and HTTPS peer ports
		cmd = append(cmd, []string{
			fmt.Sprintf("--listen-peer-urls=http://%s,https://%s", net.JoinHostPort(config.PodIP, "2380"), net.JoinHostPort(config.PodIP, "2381")),
			fmt.Sprintf("--initial-advertise-peer-urls=http://%s.etcd.%s.svc.cluster.local:2380,https://%s.etcd.%s.svc.cluster.local:2381", config.PodName, config.namespace, config.PodName, config.namespace),
		}...)
	}

	if config.EnableCorruptionCheck {
		cmd = append(cmd, []string{
			"--experimental-initial-corrupt-check=true",
			"--experimental-corrupt-check-time=240m",
		}...)
	}

	if config.QuotaBackendGB > 0 {
		bytes, overflow := resources.ConvertGBToBytes(uint64(config.QuotaBackendGB))
		if !overflow {
			cmd = append(cmd, fmt.Sprintf("--quota-backend-bytes=%d", bytes))
		}
	}

	return cmd
}
