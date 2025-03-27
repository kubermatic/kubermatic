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
	"flag"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/e2e/jig"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/utils/ptr"
)

// Options holds the e2e test options.
var (
	credentials jig.BYOCredentials

	logOptions  = utils.DefaultLogOptions
	skipCleanup = false
)

func init() {
	flag.BoolVar(&skipCleanup, "skip-cleanup", false, "Skip clean-up of resources")
	credentials.AddFlags(flag.CommandLine)
	jig.AddFlags(flag.CommandLine)
	logOptions.AddFlags(flag.CommandLine)
}

func TestExposeKubernetesApiserver(t *testing.T) {
	ctx := context.Background()
	logger := log.NewFromOptions(logOptions).Sugar()

	if err := credentials.Parse(); err != nil {
		t.Fatalf("Failed to get credentials: %v", err)
	}

	seedClient, seedConfig, err := utils.GetClients()
	if err != nil {
		t.Fatalf("failed to get client for seed cluster: %v", err)
	}

	// create test environment
	testJig := jig.NewBYOCluster(seedClient, logger, credentials)
	testJig.ClusterJig.
		WithTestName("expose-strategy").
		WithExposeStrategy(kubermaticv1.ExposeStrategyTunneling).
		WithPatch(func(cs *kubermaticv1.ClusterSpec) *kubermaticv1.ClusterSpec {
			cs.ComponentsOverride.Apiserver.EndpointReconcilingDisabled = ptr.To(true)
			return cs
		}).
		WithPatch(func(cs *kubermaticv1.ClusterSpec) *kubermaticv1.ClusterSpec {
			cs.ClusterNetwork.TunnelingAgentIP = resources.DefaultTunnelingAgentIP
			return cs
		})

	_, cluster, err := testJig.Setup(ctx, jig.WaitForNothing)
	defer testJig.Cleanup(ctx, t, false)
	if err != nil {
		t.Fatalf("failed to setup test environment: %v", err)
	}

	agentConfig := &AgentConfig{
		Log:       logger,
		Client:    seedClient,
		Namespace: cluster.Status.NamespaceName,
		Versions:  kubermatic.GetFakeVersions(),
	}
	if err := agentConfig.DeployAgentPod(ctx); err != nil {
		t.Fatalf("Failed to deploy agent: %v", err)
	}

	client := &clientJig{utils.TestPodConfig{
		Log:           logger,
		Namespace:     cluster.Status.NamespaceName,
		Client:        seedClient,
		Config:        seedConfig,
		CreatePodFunc: newClientPod,
	}}
	if err := client.DeployTestPod(ctx, logger); err != nil {
		t.Fatalf("Failed to deploy Pod: %v", err)
	}

	t.Run("Testing SNI when Kubeconfig is used e.g. Kubelet", func(t *testing.T) {
		if err := client.VerifyApiserverVersion(ctx, "", false, jig.ClusterSemver(logger)); err != nil {
			t.Fatalf("Apiserver should be reachable passing from the SNI entrypoint in nodeport proxy, but: %v", err)
		}
	})

	t.Run("Tunneling requests using HTTP/2 CONNECT when no SNI is present e.g. pods relying on kubernetes service in default namespace", func(t *testing.T) {
		if err := client.VerifyApiserverVersion(ctx, agentConfig.GetKASHostPort(), true, jig.ClusterSemver(logger)); err != nil {
			t.Fatalf("Apiserver should be reachable passing from the SNI entrypoint in nodeport proxy, but: %v", err)
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
