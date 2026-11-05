//go:build integration

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

package clustertemplatecontroller

import (
	"context"
	"testing"
	"time"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/generator"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

const (
	dev2WorkerName = "dev2"
	timeout        = time.Second * 15
	interval       = time.Second * 1
)

// Test_two_managers_one_instance_one_cluster reproduces issue #16724 end to
// end: two seed-controller-managers sharing one seed, the regular one with an
// empty worker name and a second one started with -worker-name=dev2, must turn
// a single ClusterTemplateInstance into exactly one Cluster, created by the
// regular manager and carrying no worker-name label. The controller is
// registered through Add on both managers, so the watch predicates and the
// in-reconcile guard execute exactly as in production.
func Test_two_managers_one_instance_one_cluster(t *testing.T) {
	ctx := context.Background()

	rawLog := kubermaticlog.New(true, kubermaticlog.FormatJSON)
	kubermaticlog.Logger = rawLog.Sugar()

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	// Bootstrapping test environment.
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{"../../../crd/k8c.io"},
		ErrorIfCRDPathMissing: true,
	}

	cfg, err := testEnv.Start()
	if err != nil {
		t.Fatalf("failed to start testEnv: %s", err)
	}

	t.Cleanup(func() {
		if err := testEnv.Stop(); err != nil {
			t.Fatalf("failed to stop testEnv: %s", err)
		}
	})

	if err := kubermaticv1.AddToScheme(scheme.Scheme); err != nil {
		t.Fatalf("failed to add kubermaticv1 scheme: %s", err)
	}

	// two managers against the same API server: the seed's regular
	// controller manager (empty worker name) and a developer copy started
	// with -worker-name=dev2; metrics are disabled so both managers can
	// run inside a single test process
	managerOptions := ctrl.Options{
		Scheme:  scheme.Scheme,
		Metrics: metricsserver.Options{BindAddress: "0"},
	}

	regularMgr, err := ctrl.NewManager(cfg, managerOptions)
	if err != nil {
		t.Fatalf("failed to create regular manager: %s", err)
	}

	dev2Mgr, err := ctrl.NewManager(cfg, managerOptions)
	if err != nil {
		t.Fatalf("failed to create dev2 manager: %s", err)
	}

	client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Scheme: scheme.Scheme})
	if err != nil {
		t.Fatalf("failed to create client: %s", err)
	}

	if err := Add(regularMgr, kubermaticlog.Logger, "", "kubermatic", 2); err != nil {
		t.Fatalf("failed to add controller to regular manager: %s", err)
	}
	if err := Add(dev2Mgr, kubermaticlog.Logger, dev2WorkerName, "kubermatic", 2); err != nil {
		t.Fatalf("failed to add controller to dev2 manager: %s", err)
	}

	mgrCtx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	go func() {
		if err := regularMgr.Start(mgrCtx); err != nil {
			t.Errorf("failed to start regular manager: %s", err)
		}
	}()

	go func() {
		if err := dev2Mgr.Start(mgrCtx); err != nil {
			t.Errorf("failed to start dev2 manager: %s", err)
		}
	}()

	// generator.GenDefaultProject's name ("my-first-project-ID") is not a valid
	// RFC 1123 object name, and the instance name is derived from the project
	// name; use a lowercase project so envtest's API server accepts the objects
	projectName := "testproject"

	template := generator.GenClusterTemplate("ct-e2e", "ctide2e", projectName, kubermaticv1.GlobalClusterTemplateScope, "bob@acme.com")
	// on a real seed the API's defaulting webhooks fill these fields; envtest
	// runs no webhooks, so the template must carry the values the CRD schema
	// requires (KKP's defaults: NodePort expose strategy, ipvs proxy mode)
	template.Spec.ExposeStrategy = kubermaticv1.ExposeStrategyNodePort
	template.Spec.ClusterNetwork = kubermaticv1.ClusterNetworkingConfig{
		Pods:      kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.25.0.0/16"}},
		Services:  kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.240.16.0/20"}},
		ProxyMode: "ipvs",
	}
	if err := client.Create(ctx, template); err != nil {
		t.Fatalf("failed to create cluster template: %s", err)
	}

	instance := generator.GenClusterTemplateInstance(projectName, "ctide2e", "bob@acme.com", 1)
	if err := client.Create(ctx, instance); err != nil {
		t.Fatalf("failed to create cluster template instance: %s", err)
	}

	// exactly one cluster must carry the instance's label; a second one
	// means the dev2 manager processed the foreign instance (issue #16724)
	var createdCluster *kubermaticv1.Cluster
	deadline := time.Now().Add(timeout)
	for {
		clusters := &kubermaticv1.ClusterList{}
		if err := client.List(ctx, clusters, ctrlruntimeclient.MatchingLabels{kubermaticv1.ClusterTemplateInstanceLabelKey: instance.Name}); err != nil {
			t.Fatalf("failed to list clusters: %s", err)
		}

		if len(clusters.Items) > 1 {
			t.Fatalf("expected exactly 1 cluster for instance %s, found %d", instance.Name, len(clusters.Items))
		}

		if len(clusters.Items) == 1 {
			createdCluster = clusters.Items[0].DeepCopy()
			break
		}

		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for the cluster of instance %s to be created", instance.Name)
		}

		time.Sleep(interval)
	}

	// settle: give a misbehaving dev2 manager the chance to add its duplicate
	time.Sleep(2 * interval)

	clusters := &kubermaticv1.ClusterList{}
	if err := client.List(ctx, clusters, ctrlruntimeclient.MatchingLabels{kubermaticv1.ClusterTemplateInstanceLabelKey: instance.Name}); err != nil {
		t.Fatalf("failed to list clusters: %s", err)
	}
	if len(clusters.Items) != 1 {
		t.Fatalf("expected exactly 1 cluster for instance %s after settling, found %d", instance.Name, len(clusters.Items))
	}

	if workerName, ok := createdCluster.Labels[kubermaticv1.WorkerNameLabelKey]; ok {
		t.Errorf("expected the created cluster to carry no worker-name label, got %q", workerName)
	}

	// the regular manager deletes the temporary instance once its cluster exists
	deadline = time.Now().Add(timeout)
	for {
		current := &kubermaticv1.ClusterTemplateInstance{}
		err := client.Get(ctx, types.NamespacedName{Name: instance.Name}, current)
		if apierrors.IsNotFound(err) {
			return
		}
		if err != nil {
			t.Fatalf("failed to get cluster template instance: %s", err)
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for instance %s to be deleted", instance.Name)
		}
		time.Sleep(interval)
	}
}
