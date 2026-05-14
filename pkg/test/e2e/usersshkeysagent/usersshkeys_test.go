//go:build e2e

/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package usersshkeysagent

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	defaultTimeout     = 5 * time.Minute
	defaultInterval    = 10 * time.Second
	authorizedKeysPath = "/home/ubuntu/.ssh/authorized_keys"
)

var (
	credentials jig.AWSCredentials
	logOptions  = utils.DefaultLogOptions
)

func init() {
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

type runner struct {
	seedClient ctrlruntimeclient.Client
	userClient ctrlruntimeclient.Client
	restConfig *rest.Config
	logger     *zap.SugaredLogger
	testJig    *jig.TestJig
}

func TestUserSSHKeysAgent(t *testing.T) {
	ctx := context.Background()
	rawLogger := log.NewFromOptions(logOptions)
	logger := rawLogger.Sugar()
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLogger.WithOptions(zap.AddCallerSkip(1))))

	if err := credentials.Parse(); err != nil {
		t.Fatalf("failed to parse AWS credentials: %v", err)
	}

	seedClient, _, err := utils.GetClients()
	if err != nil {
		t.Fatalf("failed to get seed client: %v", err)
	}

	// kMd travels through cloud-init at first boot, before the agent runs.
	kMD := generateSSHKey(t)

	testJig := jig.NewAWSCluster(seedClient, logger, credentials, 1, nil)
	testJig.ClusterJig.
		WithTestName("sshkeys").
		WithSSHKeyAgent(true)
	testJig.MachineJig.AddSSHPublicKey(kMD)

	t.Cleanup(func() { testJig.Cleanup(ctx, t, true) })

	logger.Info("setting up the cluster (this takes several minutes)")
	project, cluster, err := testJig.Setup(ctx, jig.WaitForReadyNodes)
	if err != nil {
		t.Fatalf("failed to set up cluster + machines: %v", err)
	}

	userClient, err := testJig.ClusterClient(ctx)
	if err != nil {
		t.Fatalf("failed to get user-cluster client: %v", err)
	}

	restCfg, err := testJig.ClusterRESTConfig(ctx)
	if err != nil {
		t.Fatalf("failed to get user-cluster REST config: %v", err)
	}

	r := &runner{
		seedClient: seedClient,
		userClient: userClient,
		restConfig: restCfg,
		logger:     logger,
		testJig:    testJig,
	}

	t.Run("KKPKeyMaterializes", r.testKKPKeyMaterializes(ctx, project, cluster, kMD))
	t.Run("KKPKeyRemoved", r.testKKPKeyRemoved(ctx, project, cluster, kMD))
	t.Run("KKPKeyAddedAlongsideMD", r.testKKPKeyAddedAlongsideMD(ctx, project, cluster, kMD))
	t.Run("MultipleKKPKeysSortedDeterministically", r.testMultipleKKPKeysSortedDeterministically(ctx, project, cluster, kMD))
	t.Run("Dedup", r.testDedup(ctx, project, cluster, kMD))
}

func (r *runner) testKKPKeyMaterializes(ctx context.Context, project *kubermaticv1.Project, cluster *kubermaticv1.Cluster, kMD string) func(*testing.T) {
	return func(t *testing.T) {
		kKKP := generateSSHKey(t)
		key := newUserSSHKey("e2e-kkp-1", "e2e-kkp-1", project.Name, cluster.Name, kKKP)

		if err := r.seedClient.Create(ctx, key); err != nil {
			t.Fatalf("failed to create UserSSHKey: %v", err)
		}

		t.Cleanup(func() {
			if err := r.seedClient.Delete(ctx, key); err != nil {
				t.Logf("warning: failed to delete UserSSHKey %q during cleanup: %v", key.Name, err)
			}
		})

		r.expectOnEveryNode(ctx, t, func(view authorizedKeysView) error {
			if got, ok := view.Managed[key.Name]; !ok || got != kKKP {
				return fmt.Errorf("expected managed[%q]=%q, got %q (ok=%v)", key.Name, kKKP, got, ok)
			}

			if !containsLine(view.External, kMD) {
				return fmt.Errorf("cloud-init key %q not preserved as external line, got %v", kMD, view.External)
			}

			return nil
		})
	}
}

func (r *runner) testKKPKeyRemoved(ctx context.Context, project *kubermaticv1.Project, cluster *kubermaticv1.Cluster, kMD string) func(*testing.T) {
	return func(t *testing.T) {
		kKKP := generateSSHKey(t)
		key := newUserSSHKey("e2e-kkp-removed", "e2e-kkp-removed", project.Name, cluster.Name, kKKP)

		if err := r.seedClient.Create(ctx, key); err != nil {
			t.Fatalf("create UserSSHKey: %v", err)
		}

		r.expectOnEveryNode(ctx, t, func(view authorizedKeysView) error {
			if view.Managed[key.Name] != kKKP {
				return fmt.Errorf("KKP key not yet on node")
			}

			return nil
		})

		if err := r.seedClient.Delete(ctx, key); err != nil {
			t.Fatalf("delete UserSSHKey: %v", err)
		}

		r.expectOnEveryNode(ctx, t, func(view authorizedKeysView) error {
			if _, present := view.Managed[key.Name]; present {
				return fmt.Errorf("KKP key still present in managed view: %v", view.Managed)
			}

			if !containsLine(view.External, kMD) {
				return fmt.Errorf("cloud-init key %q no longer present after KKP key removal, got %v", kMD, view.External)
			}

			return nil
		})
	}
}

func (r *runner) testDedup(ctx context.Context, project *kubermaticv1.Project, cluster *kubermaticv1.Cluster, kMD string) func(*testing.T) {
	return func(t *testing.T) {
		key := newUserSSHKey("e2e-dedup", "e2e-dedup", project.Name, cluster.Name, kMD)

		if err := r.seedClient.Create(ctx, key); err != nil {
			t.Fatalf("create UserSSHKey: %v", err)
		}
		t.Cleanup(func() {
			if err := r.seedClient.Delete(ctx, key); err != nil {
				t.Logf("warning: failed to delete UserSSHKey %q during cleanup: %v", key.Name, err)
			}
		})

		r.expectOnEveryNode(ctx, t, func(view authorizedKeysView) error {
			if view.Managed[key.Name] != kMD {
				return fmt.Errorf("expected cloud-init key to appear as managed[%q], got %v", key.Name, view.Managed)
			}
			if containsLine(view.External, kMD) {
				return fmt.Errorf("cloud-init key still present as external line (dedup failed): %v", view.External)
			}
			return nil
		})
	}
}

func (r *runner) testKKPKeyAddedAlongsideMD(ctx context.Context, project *kubermaticv1.Project, cluster *kubermaticv1.Cluster, kMD string) func(*testing.T) {
	return func(t *testing.T) {
		kA := generateSSHKey(t)
		kB := generateSSHKey(t)
		keyA := newUserSSHKey("e2e-pair-a", "e2e-pair-a", project.Name, cluster.Name, kA)
		keyB := newUserSSHKey("e2e-pair-b", "e2e-pair-b", project.Name, cluster.Name, kB)

		if err := r.seedClient.Create(ctx, keyA); err != nil {
			t.Fatalf("create keyA: %v", err)
		}
		t.Cleanup(func() {
			if err := r.seedClient.Delete(ctx, keyA); err != nil {
				t.Logf("warning: failed to delete UserSSHKey %q during cleanup: %v", keyA.Name, err)
			}
		})

		if err := r.seedClient.Create(ctx, keyB); err != nil {
			t.Fatalf("create keyB: %v", err)
		}
		t.Cleanup(func() {
			if err := r.seedClient.Delete(ctx, keyB); err != nil {
				t.Logf("warning: failed to delete UserSSHKey %q during cleanup: %v", keyB.Name, err)
			}
		})

		r.expectOnEveryNode(ctx, t, func(view authorizedKeysView) error {
			if view.Managed[keyA.Name] != kA {
				return fmt.Errorf("expected keyA in managed[%q], got %v", keyA.Name, view.Managed)
			}

			if view.Managed[keyB.Name] != kB {
				return fmt.Errorf("expected keyB in managed[%q], got %v", keyB.Name, view.Managed)
			}

			if !containsLine(view.External, kMD) {
				return fmt.Errorf("cloud-init key %q not preserved, got %v", kMD, view.External)
			}

			return nil
		})
	}
}

func (r *runner) testMultipleKKPKeysSortedDeterministically(ctx context.Context, project *kubermaticv1.Project, cluster *kubermaticv1.Cluster, kMD string) func(*testing.T) {
	return func(t *testing.T) {
		kA := generateSSHKey(t)
		kB := generateSSHKey(t)
		keyA := newUserSSHKey("e2e-sort-a", "e2e-sort-a", project.Name, cluster.Name, kA)
		keyB := newUserSSHKey("e2e-sort-b", "e2e-sort-b", project.Name, cluster.Name, kB)

		if err := r.seedClient.Create(ctx, keyA); err != nil {
			t.Fatalf("create keyA: %v", err)
		}
		t.Cleanup(func() {
			if err := r.seedClient.Delete(ctx, keyA); err != nil {
				t.Logf("warning: failed to delete UserSSHKey %q during cleanup: %v", keyA.Name, err)
			}
		})

		if err := r.seedClient.Create(ctx, keyB); err != nil {
			t.Fatalf("create keyB: %v", err)
		}

		t.Cleanup(func() {
			if err := r.seedClient.Delete(ctx, keyB); err != nil {
				t.Logf("warning: failed to delete UserSSHKey %q during cleanup: %v", keyB.Name, err)
			}
		})

		r.expectOnEveryNode(ctx, t, func(view authorizedKeysView) error {
			if len(view.Managed) < 2 {
				return fmt.Errorf("expected at least 2 managed keys, got %d", len(view.Managed))
			}

			if _, ok := view.Managed[keyA.Name]; !ok {
				return fmt.Errorf("keyA %q not found in managed keys", keyA.Name)
			}

			if _, ok := view.Managed[keyB.Name]; !ok {
				return fmt.Errorf("keyB %q not found in managed keys", keyB.Name)
			}

			return nil
		})
	}
}

func (r *runner) expectOnEveryNode(ctx context.Context, t *testing.T, check func(authorizedKeysView) error) {
	t.Helper()
	err := wait.PollLog(ctx, r.logger, defaultInterval, defaultTimeout,
		func(ctx context.Context) (transient error, terminal error) {
			nodes := corev1.NodeList{}
			if err := r.userClient.List(ctx, &nodes); err != nil {
				return fmt.Errorf("list nodes: %w", err), nil
			}

			if len(nodes.Items) == 0 {
				return errors.New("no nodes yet"), nil
			}

			for _, node := range nodes.Items {
				if !kubernetes.IsNodeReady(&node) {
					return fmt.Errorf("node %q not Ready", node.Name), nil
				}

				pod, err := findAgentPodForNode(ctx, r.userClient, node.Name)
				if err != nil {
					return fmt.Errorf("find agent pod on %q: %w", node.Name, err), nil
				}

				if pod == nil {
					return fmt.Errorf("no agent pod yet on %q", node.Name), nil
				}

				content, err := readAuthorizedKeys(ctx, r.restConfig, pod, authorizedKeysPath)
				if err != nil {
					return fmt.Errorf("read authorized_keys on %q: %w", node.Name, err), nil
				}

				if err := check(parseAuthorizedKeys(content)); err != nil {
					return fmt.Errorf("node %s: %w", node.Name, err), nil
				}
			}
			return nil, nil
		})
	if err != nil {
		t.Fatalf("expectation never converged: %v", err)
	}
}

func containsLine(lines []string, want string) bool {
	want = strings.TrimSpace(want)
	for _, l := range lines {
		if strings.TrimSpace(l) == want {
			return true
		}
	}

	return false
}
