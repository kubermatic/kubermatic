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

	"github.com/kubermatic/kubermatic/api/pkg/semver"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	autoscalingv1beta2 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1beta2"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestEnsureResourcesAreDeployedIdempotency(t *testing.T) {
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
	if err := autoscalingv1beta2.AddToScheme(mgr.GetScheme()); err != nil {
		t.Fatalf("failed to register vertical pod autoscaler resources to scheme: %v", err)
	}
	crdInstallOpts := envtest.CRDInstallOptions{
		// Results in timeouts, maybe a controller-runtime bug?
		// Paths: []string{"../../../../../config/kubermatic/crd"},
		CRDs: []*apiextensionsv1beta1.CustomResourceDefinition{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "clusters.kubermatic.k8s.io",
				},
				Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
					Group:   "kubermatic.k8s.io",
					Version: "v1",
					Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
						Kind:     "Cluster",
						ListKind: "ClusterList",
						Plural:   "clusters",
						Singular: "cluster",
					},
					Scope: apiextensionsv1beta1.ClusterScoped,
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "verticalpodautoscalers.autoscaling.k8s.io",
				},
				Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
					Group:   "autoscaling.k8s.io",
					Version: "v1beta2",
					Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
						Kind:     "VerticalPodAutoscaler",
						Plural:   "verticalpodautoscalers",
						Singular: "verticalpodautoscaler",
					},
					Scope: apiextensionsv1beta1.NamespaceScoped,
				},
			},
		},
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

	testCluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster",
		},
		Spec: kubermaticv1.ClusterSpec{
			ExposeStrategy: corev1.ServiceTypeLoadBalancer,
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: "my-dc",
			},
			Version: *semver.NewSemverOrDie("1.16.3"),
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
		Status: corev1.ServiceStatus{
			LoadBalancer: corev1.LoadBalancerStatus{
				Ingress: []corev1.LoadBalancerIngress{{
					IP: "1.2.3.4",
				}},
			},
		},
	}

	if err := mgr.GetClient().Create(ctx, testCluster); err != nil {
		t.Fatalf("failed to create testcluster: %v", err)
	}
	if err := mgr.GetClient().Create(ctx, lbService); err != nil {
		t.Fatalf("failed to create the loadbalancer service: %v", err)
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
		seedGetter: func() (*kubermaticv1.Seed, error) {
			return &kubermaticv1.Seed{
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						testCluster.Spec.Cloud.DatacenterName: {},
					},
				},
			}, nil
		},
	}

	if err := r.ensureClusterNetworkDefaults(ctx, testCluster); err != nil {
		t.Fatalf("failed to sync initial network default: %v", err)
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
