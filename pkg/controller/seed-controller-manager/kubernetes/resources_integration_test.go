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
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/semver"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	"k8s.io/utils/pointer"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestEnsureResourcesAreDeployedIdempotency(t *testing.T) {
	kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()
	env := &envtest.Environment{
		// Uncomment this to get the logs from etcd+apiserver
		// AttachControlPlaneOutput: true,
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
	if err := autoscalingv1beta2.AddToScheme(mgr.GetScheme()); err != nil {
		t.Fatalf("failed to register vertical pod autoscaler resources to scheme: %v", err)
	}
	crdInstallOpts := envtest.CRDInstallOptions{
		Paths:              []string{"../../../../charts/kubermatic/crd"},
		ErrorIfPathMissing: true,
	}
	if _, err := envtest.InstallCRDs(cfg, crdInstallOpts); err != nil {
		t.Fatalf("failed install crds: %v", err)
	}

	ctx := context.Background()

	go func() {
		if err := mgr.Start(ctx); err != nil {
			t.Errorf("failed to start manager: %v", err)
		}
	}()

	caBundle := certificates.NewFakeCABundle()

	testCluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster",
		},
		Spec: kubermaticv1.ClusterSpec{
			ExposeStrategy: kubermaticv1.ExposeStrategyLoadBalancer,
			ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.193.0.0/20"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                resources.IPVSProxyMode,
				NodeLocalDNSCacheEnabled: pointer.BoolPtr(true),
			},
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: "my-dc",
			},
			Version: *semver.NewSemverOrDie("1.18.9"),
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: "cluster-test-cluster",
			ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
				CloudProviderInfrastructure: kubermaticv1.HealthStatusUp,
			},
		},
	}

	// This is used as basis to sync the clusters address which we in turn do
	// before creating any deployments.
	lbService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testCluster.Status.NamespaceName,
			Name:      resources.FrontLoadBalancerServiceName,
		},
		Spec: corev1.ServiceSpec{
			Type:  corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{{Port: 443}},
		},
	}

	caBundleConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testCluster.Status.NamespaceName,
			Name:      resources.CABundleConfigMapName,
		},
		Data: map[string]string{
			resources.CABundleConfigMapKey: caBundle.String(),
		},
	}

	if err := mgr.GetClient().Create(ctx, testCluster); err != nil {
		t.Fatalf("failed to create testcluster: %v", err)
	}
	if err := mgr.GetClient().Create(ctx, lbService); err != nil {
		t.Fatalf("failed to create the loadbalancer service: %v", err)
	}
	if err := mgr.GetClient().Create(ctx, caBundleConfigMap); err != nil {
		t.Fatalf("failed to create the CA bundle: %v", err)
	}

	// Status must be set *after* the Service has been created, because
	// the Create() call would reset it to nil.
	lbService.Status = corev1.ServiceStatus{
		LoadBalancer: corev1.LoadBalancerStatus{
			Ingress: []corev1.LoadBalancerIngress{{
				IP: "1.2.3.4",
			}},
		},
	}

	// Status is a subresource for services and we need the IP to be set, else
	// the reconciliation returns early
	if err := mgr.GetClient().Status().Update(ctx, lbService); err != nil {
		t.Fatalf("failed to set lb service status: %v", err)
	}

	r := &Reconciler{
		log:                  kubermaticlog.Logger,
		Client:               mgr.GetClient(),
		dockerPullConfigJSON: []byte("{}"),
		nodeAccessNetwork:    kubermaticv1.DefaultNodeAccessNetwork,
		kubermaticImage:      resources.DefaultKubermaticImage,
		dnatControllerImage:  resources.DefaultDNATControllerImage,
		etcdLauncherImage:    resources.DefaultEtcdLauncherImage,
		seedGetter: func() (*kubermaticv1.Seed, error) {
			return &kubermaticv1.Seed{
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						testCluster.Spec.Cloud.DatacenterName: {},
					},
				},
			}, nil
		},
		caBundle: caBundle,
	}

	if err := r.ensureResourcesAreDeployed(ctx, testCluster); err != nil {
		t.Fatalf("Initial resource deployment failed, this indicates that some resources are invalid. Error: %v", err)
	}

	if err := r.ensureResourcesAreDeployed(ctx, testCluster); err != nil {
		t.Fatalf("The second resource reconciliation failed, indicating we don't properly default some fields. Check the `Object differs from generated one` error for the object for which we timed out. Original error: %v", err)
	}

	// A very basic sanity check that we actually deployed something
	deploymentList := &appsv1.DeploymentList{}
	if err := mgr.GetAPIReader().List(ctx, deploymentList, ctrlruntimeclient.InNamespace(testCluster.Status.NamespaceName)); err != nil {
		t.Fatalf("failed to list deployments: %v", err)
	}
	if len(deploymentList.Items) == 0 {
		t.Error("expected to find at least one deployment, got zero")
	}
}
