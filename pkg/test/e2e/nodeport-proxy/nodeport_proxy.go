//go:build e2e

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

package nodeportproxy

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/nodeportproxy"
	npptest "k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/nodeportproxy/test"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NodeportProxy helps setting up nodeport proxy for testing.
type NodeportProxy struct {
	Log       *zap.SugaredLogger
	Namespace string
	Versions  kubermatic.Versions
	Client    ctrlruntimeclient.Client
}

func (d *NodeportProxy) Setup(ctx context.Context) error {
	if d.Namespace == "" {
		d.Namespace = "nodeport-proxy-" + rand.String(5)
	}

	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: d.Namespace,
		},
	}
	d.Log.Debugw("Creating namespace…", "namespace", ns)
	if err := d.Client.Create(ctx, ns); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}

	cfg := d.getConfig()

	// for this test, the seed is meaningless
	seed, err := defaulting.DefaultSeed(&kubermaticv1.Seed{}, cfg, d.Log)
	if err != nil {
		return fmt.Errorf("failed to default seed: %w", err)
	}

	d.Log.Infow("Setting up nodeport-proxy…", "version", d.Versions.KubermaticContainerTag)
	if err = npptest.Deploy(ctx, d.Client, d.Log, d.Namespace, cfg, seed, d.Versions, 5*time.Minute); err != nil {
		if cleanuperr := npptest.Cleanup(ctx, d.Client, d.Log, cfg, 1*time.Minute); cleanuperr != nil {
			d.Log.Errorw("Failed to cleanup", zap.Error(cleanuperr))
		}

		return fmt.Errorf("failed to setup: %w", err)
	}

	d.Log.Info("Nodeport-proxy deployed successfully.")
	return nil
}

func (d *NodeportProxy) Cleanup(ctx context.Context) error {
	return npptest.Cleanup(ctx, d.Client, d.Log, d.getConfig(), 1*time.Minute)
}

func (d *NodeportProxy) GetLoadBalancer(ctx context.Context) (*corev1.Service, error) {
	svc := &corev1.Service{}
	if err := d.Client.Get(ctx, types.NamespacedName{Name: nodeportproxy.ServiceName, Namespace: d.Namespace}, svc); err != nil {
		return nil, err
	}
	return svc, nil
}

func (d *NodeportProxy) getConfig() *kubermaticv1.KubermaticConfiguration {
	return &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: d.Namespace,
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
				EtcdVolumeSize: "500Mi",
			},
		},
	}
}
