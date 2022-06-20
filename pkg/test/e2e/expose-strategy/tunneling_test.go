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

	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/semver"
	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	"k8s.io/apimachinery/pkg/util/rand"
)

// Options holds the e2e test options.
type testOptions struct {
	skipCleanup       bool
	logOptions        log.Options
	datacenter        string
	kubernetesVersion semver.Semver
}

var (
	options = testOptions{
		kubernetesVersion: semver.Semver(e2eutils.KubernetesVersion()),
		logOptions:        e2eutils.DefaultLogOptions,
	}
)

func init() {
	flag.StringVar(&options.datacenter, "datacenter", "byo-kubernetes", "Name of the datacenter used by the user clusters created for the test.")
	flag.Var(&options.kubernetesVersion, "kubernetes-version", "Kubernetes version for the user cluster")
	flag.BoolVar(&options.skipCleanup, "skip-cleanup", false, "Skip clean-up of resources")

	options.logOptions.AddFlags(flag.CommandLine)
}

func TestExposeKubernetesApiserver(t *testing.T) {
	ctx := context.Background()
	logger := log.NewFromOptions(options.logOptions).Sugar()
	k8scli, restCli, restConf := e2eutils.GetClientsOrDie()

	clusterJig := &ClusterJig{
		Log:            logger,
		Client:         k8scli,
		Name:           "c" + rand.String(9),
		DatacenterName: options.datacenter,
		Version:        options.kubernetesVersion,
	}
	if err := clusterJig.SetUp(ctx); err != nil {
		t.Errorf("Failed to setup usercluster: %v", err)
		return
	}

	agentConfig := &AgentConfig{
		Log:       logger,
		Client:    k8scli,
		Namespace: clusterJig.Cluster.Status.NamespaceName,
		Versions:  kubermatic.NewDefaultVersions(),
	}
	if err := agentConfig.DeployAgentPod(ctx); err != nil {
		t.Errorf("Failed to deploy agent: %v", err)
		return
	}

	client := &clientJig{e2eutils.TestPodConfig{
		Log:           logger,
		Namespace:     clusterJig.Cluster.Status.NamespaceName,
		Client:        k8scli,
		PodRestClient: restCli,
		Config:        restConf,
		CreatePodFunc: newClientPod,
	}}
	if err := client.DeployTestPod(ctx, logger); err != nil {
		t.Errorf("Failed to deploy Pod: %v", err)
		return
	}

	logger.Debug("Testing SNI when Kubeconfig is used e.g. Kubelet")
	if result := client.QueryApiserverVersion("", false, options.kubernetesVersion, 5, 4); result != true {
		t.Error("Apiserver should be reachable passing from the SNI entrypoint in nodeport proxy")
	}

	logger.Debug("Tunneling requests using HTTP/2 CONNECT when no SNI is present e.g. pods relying on kubernetes service in default namespace")
	if result := client.QueryApiserverVersion(agentConfig.GetKASHostPort(), true, options.kubernetesVersion, 5, 4); result != true {
		t.Error("Apiserver should be reachable passing from the SNI entrypoint in nodeport proxy")
	}

	if !options.skipCleanup {
		defer func() {
			if err := client.CleanUp(ctx); err != nil {
				t.Errorf("Failed to cleanup: %v", err)
			}
			if err := agentConfig.CleanUp(ctx); err != nil {
				t.Errorf("Failed to cleanup: %v", err)
			}
			if err := clusterJig.CleanUp(ctx); err != nil {
				t.Errorf("Failed to cleanup: %v", err)
			}
		}()
	}
}
