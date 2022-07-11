//go:build e2e

/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package exposestrategy

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// Options holds the e2e test options.
var (
	logOptions  = log.NewDefaultOptions()
	skipCleanup = false
)

func init() {
	flag.BoolVar(&skipCleanup, "skip-cleanup", false, "Skip clean-up of resources")
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

type testJig struct {
	client ctrlruntimeclient.Client
	log    *zap.SugaredLogger

	projectJig *jig.ProjectJig
	clusterJig *jig.ClusterJig
}

func newTestJig(client ctrlruntimeclient.Client, log *zap.SugaredLogger) *testJig {
	return &testJig{
		client: client,
		log:    log,
	}
}

func (j *testJig) Setup(ctx context.Context) (*kubermaticv1.Project, *kubermaticv1.Cluster, error) {
	// create dummy project
	j.projectJig = jig.NewProjectJig(j.client, j.log)

	project, err := j.projectJig.Create(ctx, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create project: %w", err)
	}

	// create test cluster
	j.clusterJig = jig.NewClusterJig(j.client, j.log)
	cluster, err := j.clusterJig.
		WithProject(project).
		WithGenerateName("e2e-expose-strategy-").
		WithHumanReadableName("Expose strategy test cluster").
		WithSSHKeyAgent(false).
		WithExposeStrategy(kubermaticv1.ExposeStrategyTunneling).
		WithCloudSpec(&kubermaticv1.CloudSpec{
			DatacenterName: jig.DatacenterName(),
			ProviderName:   string(kubermaticv1.BringYourOwnCloudProvider),
			BringYourOwn:   &kubermaticv1.BringYourOwnCloudSpec{},
		}).
		WithPatch(func(cs *kubermaticv1.ClusterSpec) *kubermaticv1.ClusterSpec {
			cs.ComponentsOverride.Apiserver.EndpointReconcilingDisabled = pointer.Bool(true)
			return cs
		}).
		Create(ctx, true)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	return project, cluster, nil
}

func (j *testJig) WaitForHealthyControlPlane(ctx context.Context, timeout time.Duration) error {
	if j.clusterJig != nil {
		return j.clusterJig.WaitForHealthyControlPlane(ctx, timeout)
	}

	return errors.New("no cluster created yet")
}

func (j *testJig) Cleanup(ctx context.Context, t *testing.T) {
	if j.clusterJig != nil {
		if err := j.clusterJig.Delete(ctx, false); err != nil {
			t.Errorf("Failed to delete cluster: %v", err)
		}
	}

	if j.projectJig != nil {
		if err := j.projectJig.Delete(ctx, false); err != nil {
			t.Errorf("Failed to delete project: %v", err)
		}
	}
}

func TestExposeKubernetesApiserver(t *testing.T) {
	ctx := context.Background()
	logger := log.NewFromOptions(logOptions).Sugar()

	seedClient, restConfig, seedConfig, err := e2eutils.GetClients()
	if err != nil {
		t.Fatalf("failed to get client for seed cluster: %v", err)
	}

	// setup a dummy project & cluster
	testJig := newTestJig(seedClient, logger)
	if !skipCleanup {
		defer testJig.Cleanup(ctx, t)
	}

	_, cluster, err := testJig.Setup(ctx)
	if err != nil {
		t.Fatalf("Failed to setup test environment: %v", err)
	}

	agentConfig := &AgentConfig{
		Log:       logger,
		Client:    seedClient,
		Namespace: cluster.Status.NamespaceName,
		Versions:  kubermatic.NewDefaultVersions(),
	}
	if err := agentConfig.DeployAgentPod(ctx); err != nil {
		t.Fatalf("Failed to deploy agent: %v", err)
	}

	client := &clientJig{e2eutils.TestPodConfig{
		Log:           logger,
		Namespace:     cluster.Status.NamespaceName,
		Client:        seedClient,
		PodRestClient: restConfig,
		Config:        seedConfig,
		CreatePodFunc: newClientPod,
	}}
	if err := client.DeployTestPod(ctx, logger); err != nil {
		t.Fatalf("Failed to deploy Pod: %v", err)
	}

	t.Run("Testing SNI when Kubeconfig is used e.g. Kubelet", func(t *testing.T) {
		if !client.QueryApiserverVersion("", false, jig.ClusterSemver(), 5, 4) {
			t.Fatal("Apiserver should be reachable passing from the SNI entrypoint in nodeport proxy")
		}
	})

	t.Run("Tunneling requests using HTTP/2 CONNECT when no SNI is present e.g. pods relying on kubernetes service in default namespace", func(t *testing.T) {
		if !client.QueryApiserverVersion(agentConfig.GetKASHostPort(), true, jig.ClusterSemver(), 5, 4) {
			t.Fatal("Apiserver should be reachable passing from the SNI entrypoint in nodeport proxy")
		}
	})

	if !skipCleanup {
		defer func() {
			if err := client.CleanUp(ctx); err != nil {
				t.Errorf("Failed to cleanup: %v", err)
			}
			if err := agentConfig.CleanUp(ctx); err != nil {
				t.Errorf("Failed to cleanup: %v", err)
			}
		}()
	}
}
