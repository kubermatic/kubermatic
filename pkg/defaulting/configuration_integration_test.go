//go:build integration

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

package defaulting

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestDefaultValuesMatchSchema(t *testing.T) {
	ctx := context.Background()

	env, mgr, running, cancel, err := setupEnvtest(ctx, t)
	if err != nil {
		t.Fatalf("failed to start testenv: %v", err)
	}

	if err := configTest(ctx, mgr); err != nil {
		t.Errorf("test failed: %v", err)
	}

	// stop the manager
	cancel()

	// wait for it to be stopped
	<-running

	// shutdown envtest
	if err := env.Stop(); err != nil {
		t.Errorf("failed to stop testenv: %v", err)
	}
}

func configTest(ctx context.Context, mgr manager.Manager) error {
	ns := &corev1.Namespace{}
	ns.SetName("foobar")

	if err := mgr.GetClient().Create(ctx, ns); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	// create a vanilla config that should just work
	config := &kubermaticv1.KubermaticConfiguration{}
	config.SetName("working")
	config.SetNamespace(ns.Name)

	defaulted, err := DefaultConfiguration(config, zap.NewNop().Sugar())
	if err != nil {
		return fmt.Errorf("failed to default configuration: %w", err)
	}

	if err := mgr.GetClient().Create(ctx, defaulted); err != nil {
		return fmt.Errorf("failed to create defaulted configuration: %w", err)
	}

	// create a broken config that should be rejected
	config = &kubermaticv1.KubermaticConfiguration{}
	config.SetName("broken")
	config.SetNamespace(ns.Name)
	config.Spec.Versions.ProviderIncompatibilities = []kubermaticv1.Incompatibility{{
		Operation: kubermaticv1.OperationType("this-is-invalid"),
	}}

	if err := mgr.GetClient().Create(ctx, config); err == nil {
		return errors.New("should not have been able to create a KubermaticConfiguration with invalid values, but it succeeded")
	}

	return nil
}

func setupEnvtest(ctx context.Context, t *testing.T) (*envtest.Environment, manager.Manager, chan struct{}, context.CancelFunc, error) {
	env := &envtest.Environment{
		// Uncomment this to get the logs from etcd+apiserver
		// AttachControlPlaneOutput: true,
	}

	cfg, err := env.Start()
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to start testenv: %w", err)
	}

	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed to construct manager: %w", err)
	}

	crdInstallOpts := envtest.CRDInstallOptions{
		Paths: []string{
			"../../charts/kubermatic-operator/crd/k8s.io",
			"../crd/k8c.io",
		},
		ErrorIfPathMissing: true,
	}
	if _, err := envtest.InstallCRDs(cfg, crdInstallOpts); err != nil {
		return nil, nil, nil, nil, fmt.Errorf("failed install crds: %w", err)
	}

	// the manager needs to be stopped because the testenv can be torn down;
	// create a cancellable context to achieve this, plus a channel that signals
	// whether the goroutine is still running (so we can wait for it to stop)
	testCtx, cancel := context.WithCancel(ctx)
	running := make(chan struct{}, 1)

	go func() {
		if err := mgr.Start(testCtx); err != nil {
			t.Errorf("failed to start manager: %v", err)
		}
		close(running)
	}()

	return env, mgr, running, cancel, nil
}
