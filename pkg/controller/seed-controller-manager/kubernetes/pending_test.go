// +build integration

/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package kubernetes

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestEnsureEtcdLauncherFeatureFlag(t *testing.T) {
	tests := []struct {
		name                 string
		clusterFeatures      map[string]bool
		seedEtcdLauncher     bool
		expectedEtcdLauncher bool
	}{
		{
			name:                 "Seed feature gate enabled, cluster has no feature flag",
			clusterFeatures:      nil, // no features set
			seedEtcdLauncher:     true,
			expectedEtcdLauncher: true,
		},
		{
			name: "Seed feature gate enabled, cluster explicitly set to false",
			clusterFeatures: map[string]bool{
				kubermaticv1.ClusterFeatureEtcdLauncher: false,
			},
			seedEtcdLauncher:     true,
			expectedEtcdLauncher: false,
		},
		{
			name: "Seed feature gate disabled, cluster explicitly set to true",
			clusterFeatures: map[string]bool{
				kubermaticv1.ClusterFeatureEtcdLauncher: true,
			},
			seedEtcdLauncher:     false,
			expectedEtcdLauncher: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()
			env := &envtest.Environment{
				// Uncomment this to get the logs from etcd+apiserver
				// AttachControlPlaneOutput: true,
				KubeAPIServerFlags: []string{
					"--etcd-servers={{ if .EtcdURL }}{{ .EtcdURL.String }}{{ end }}",
					"--cert-dir={{ .CertDir }}",
					"--insecure-port={{ if .URL }}{{ .URL.Port }}{{ end }}",
					"--insecure-bind-address={{ if .URL }}{{ .URL.Hostname }}{{ end }}",
					"--secure-port={{ if .SecurePort }}{{ .SecurePort }}{{ end }}",
					"--admission-control=AlwaysAdmit",
					// Upstream does not have `--allow-privileged`,
					"--allow-privileged",
				},
			}
			cfg, err := env.Start()
			if err != nil {
				t.Fatalf("failed to start testenv: %v", err)
			}
			defer func() {
				if err := env.Stop(); err != nil {
					t.Fatalf("failed to stop testenv: %v", err)
				}
			}()

			mgr, err := manager.New(cfg, manager.Options{})
			if err != nil {
				t.Fatalf("failed to construct manager: %v", err)
			}

			crdInstallOpts := envtest.CRDInstallOptions{
				Paths:              []string{"../../../../charts/kubermatic/crd"},
				ErrorIfPathMissing: true,
			}
			if _, err := envtest.InstallCRDs(cfg, crdInstallOpts); err != nil {
				t.Fatalf("failed install crds: %v", err)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go func() {
				if err := mgr.Start(ctx.Done()); err != nil {
					t.Errorf("failed to start manager: %v", err)
				}
			}()

			cluster := &kubermaticv1.Cluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-cluster",
				},
				Spec: kubermaticv1.ClusterSpec{
					Features: test.clusterFeatures,
				},
			}
			if err := mgr.GetClient().Create(ctx, cluster); err != nil {
				t.Fatalf("failed to create testcluster: %v", err)
			}

			r := &Reconciler{
				Client: mgr.GetClient(),
				features: Features{
					EtcdLauncher: test.seedEtcdLauncher,
				},
			}
			if err := r.ensureEtcdLauncherFeatureFlag(ctx, cluster); err != nil {
				t.Fatal(err)
			}
			if cluster.Spec.Features != nil && cluster.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher] != test.expectedEtcdLauncher {
				t.Errorf("expected clsuter flag to be %v , got %v instead", test.expectedEtcdLauncher, cluster.Spec.Features[kubermaticv1.ClusterFeatureEtcdLauncher])
			}
		})
	}

}
