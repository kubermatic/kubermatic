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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/seed/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	features "k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	e2eutils "k8c.io/kubermatic/v2/pkg/test/e2e/utils"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// wait time between poll attempts of a Service vip and/or nodePort.
	// coupled with testTries to produce a net timeout value.
	hitEndpointRetryDelay = 2 * time.Second
	podReadinessTimeout   = 2 * time.Minute
)

// Deployer helps setting up nodeport proxy for testing.
type Deployer struct {
	Log       *zap.SugaredLogger
	Namespace string
	Versions  kubermatic.Versions
	Client    ctrlruntimeclient.Client

	resources []ctrlruntimeclient.Object
}

func (d *Deployer) SetUp(ctx context.Context) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: d.Namespace,
		},
	}
	if d.Namespace == "" {
		ns.ObjectMeta.GenerateName = "nodeport-proxy-"
	}
	d.Log.Debugw("Creating namespace", "service", ns)
	if err := d.Client.Create(ctx, ns); err != nil {
		return fmt.Errorf("failed to create namespace: %w", err)
	}
	d.Namespace = ns.Name
	d.resources = append(d.resources, ns)

	cfg := &kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kubermatic",
			Namespace: d.Namespace,
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			FeatureGates: map[string]bool{
				features.TunnelingExposeStrategy: true,
			},
			UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{
				EtcdVolumeSize: "500Mi",
			},
		},
	}

	recorderFunc := func(create reconciling.ObjectCreator) reconciling.ObjectCreator {
		return func(existing ctrlruntimeclient.Object) (ctrlruntimeclient.Object, error) {
			obj, err := create(existing)
			if err != nil {
				return nil, err
			}

			d.resources = append(d.resources, obj)
			return existing, nil
		}
	}

	seed, err := defaulting.DefaultSeed(&kubermaticv1.Seed{}, cfg, d.Log)
	if err != nil {
		return fmt.Errorf("failed to default seed: %w", err)
	}

	if err := reconciling.ReconcileServiceAccounts(ctx,
		[]reconciling.NamedServiceAccountCreatorGetter{
			nodeportproxy.ServiceAccountCreator(cfg),
		}, d.Namespace, d.Client, recorderFunc); err != nil {
		return fmt.Errorf("failed to reconcile ServiceAcconts: %w", err)
	}
	if err := reconciling.ReconcileRoles(ctx,
		[]reconciling.NamedRoleCreatorGetter{
			nodeportproxy.RoleCreator(),
		}, d.Namespace, d.Client, recorderFunc); err != nil {
		return fmt.Errorf("failed to reconcile Role: %w", err)
	}
	if err := reconciling.ReconcileRoleBindings(ctx,
		[]reconciling.NamedRoleBindingCreatorGetter{
			nodeportproxy.RoleBindingCreator(cfg),
		}, d.Namespace, d.Client, recorderFunc); err != nil {
		return fmt.Errorf("failed to reconcile RoleBinding: %w", err)
	}
	if err := reconciling.ReconcileClusterRoles(ctx,
		[]reconciling.NamedClusterRoleCreatorGetter{
			nodeportproxy.ClusterRoleCreator(cfg),
		}, "", d.Client, recorderFunc); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRole: %w", err)
	}
	if err := reconciling.ReconcileClusterRoleBindings(ctx,
		[]reconciling.NamedClusterRoleBindingCreatorGetter{
			nodeportproxy.ClusterRoleBindingCreator(cfg),
		}, "", d.Client, recorderFunc); err != nil {
		return fmt.Errorf("failed to reconcile ClusterRoleBinding: %w", err)
	}
	if err := reconciling.ReconcileServices(ctx,
		[]reconciling.NamedServiceCreatorGetter{
			nodeportproxy.ServiceCreator(seed)},
		d.Namespace, d.Client, recorderFunc); err != nil {
		return fmt.Errorf("failed to reconcile Services: %w", err)
	}
	if err := reconciling.ReconcileDeployments(ctx,
		[]reconciling.NamedDeploymentCreatorGetter{
			nodeportproxy.EnvoyDeploymentCreator(cfg, seed, false, d.Versions),
			nodeportproxy.UpdaterDeploymentCreator(cfg, seed, d.Versions),
		}, d.Namespace, d.Client, recorderFunc); err != nil {
		return fmt.Errorf("failed to reconcile Kubermatic Deployments: %w", err)
	}

	// Wait for pods to be ready
	for _, o := range d.resources {
		if dep, ok := o.(*appsv1.Deployment); ok {
			pods, err := d.waitForPodsCreated(ctx, dep)
			if err != nil {
				return fmt.Errorf("failed to create pods: %w", err)
			}
			if err := d.waitForPodsReady(ctx, pods...); err != nil {
				return fmt.Errorf("failed waiting for pods to be running: %w", err)
			}
		}
	}
	d.Log.Debugw("deployed nodeport-proxy", "version", d.Versions.Kubermatic)
	return nil
}

// CleanUp deletes the resources.
func (d *Deployer) CleanUp(ctx context.Context) error {
	for _, o := range d.resources {
		// TODO handle better errors
		_ = d.Client.Delete(ctx, o)
	}
	return nil
}

// GetLbService returns the service used to expose the nodeport proxy pods.
func (d *Deployer) GetLbService(ctx context.Context) *corev1.Service {
	svc := corev1.Service{}
	if err := d.Client.Get(ctx, types.NamespacedName{Name: nodeportproxy.ServiceName, Namespace: d.Namespace}, &svc); err != nil {
		return nil
	}
	return &svc
}

func (d *Deployer) waitForPodsCreated(ctx context.Context, dep *appsv1.Deployment) ([]string, error) {
	return e2eutils.WaitForPodsCreated(ctx, d.Client, d.Log, int(*dep.Spec.Replicas), dep.Namespace, dep.Spec.Selector.MatchLabels)
}

func (d *Deployer) waitForPodsReady(ctx context.Context, pods ...string) error {
	if !e2eutils.CheckPodsRunningReady(ctx, d.Client, d.Log, d.Namespace, pods, podReadinessTimeout) {
		return fmt.Errorf("timeout waiting for %d pods to be ready", len(pods))
	}
	return nil
}
