//go:build integration

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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	k8cuserclusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/cni"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	autoscalingv1 "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/apis/autoscaling.k8s.io/v1"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type testUserClusterConnectionProvider struct {
	ctrlruntimeclient.Client
}

func (c *testUserClusterConnectionProvider) GetClient(context.Context, *kubermaticv1.Cluster, ...k8cuserclusterclient.ConfigOption) (ctrlruntimeclient.Client, error) {
	return c, nil
}

func (c *testUserClusterConnectionProvider) Get(ctx context.Context, key ctrlruntimeclient.ObjectKey, obj ctrlruntimeclient.Object, _ ...ctrlruntimeclient.GetOption) error {
	switch x := obj.(type) {
	case *corev1.ServiceAccount:
		x.Secrets = append(x.Secrets, corev1.ObjectReference{
			Name: "token-name",
		})
	case *corev1.Secret:
		x.Data["ca.crt"] = []byte("ca.crtGARBAGE")
		x.Data["token"] = []byte("tokenGARBAGE")
	}
	return nil
}

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

	mgr, err := manager.New(cfg, manager.Options{})
	if err != nil {
		t.Fatalf("failed to construct manager: %v", err)
	}
	if err := autoscalingv1.AddToScheme(mgr.GetScheme()); err != nil {
		t.Fatalf("failed to register vertical pod autoscaler resources to scheme: %v", err)
	}

	crdInstallOpts := envtest.CRDInstallOptions{
		Paths: []string{
			"../../../../charts/kubermatic-operator/crd/k8s.io",
			"../../../crd/k8c.io",
		},
		ErrorIfPathMissing: true,
	}
	if _, err := envtest.InstallCRDs(cfg, crdInstallOpts); err != nil {
		t.Fatalf("failed install crds: %v", err)
	}

	ctx := context.Background()

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

	caBundle := certificates.NewFakeCABundle()
	version := defaulting.DefaultKubernetesVersioning.Default

	testCluster := &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-cluster",
			Labels: map[string]string{
				kubermaticv1.ProjectIDLabelKey: "project",
			},
		},
		Spec: kubermaticv1.ClusterSpec{
			ExposeStrategy: kubermaticv1.ExposeStrategyLoadBalancer,
			ClusterNetwork: kubermaticv1.ClusterNetworkingConfig{
				Pods:                     kubermaticv1.NetworkRanges{CIDRBlocks: []string{"10.241.0.0/16"}},
				Services:                 kubermaticv1.NetworkRanges{CIDRBlocks: []string{"172.193.0.0/20"}},
				DNSDomain:                "cluster.local",
				ProxyMode:                resources.IPVSProxyMode,
				NodeLocalDNSCacheEnabled: ptr.To(true),
			},
			CNIPlugin: &kubermaticv1.CNIPluginSettings{
				Type:    kubermaticv1.CNIPluginTypeCanal,
				Version: cni.GetDefaultCNIPluginVersion(kubermaticv1.CNIPluginTypeCanal),
			},
			Cloud: kubermaticv1.CloudSpec{
				DatacenterName: "my-dc",
				Fake:           &kubermaticv1.FakeCloudSpec{},
			},
			Version: *version,
		},
	}

	clusterNamespace := "cluster-test-cluster"

	// This is used as basis to sync the clusters address which we in turn do
	// before creating any deployments.
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterNamespace,
		},
	}

	lbService := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: clusterNamespace,
			Name:      resources.FrontLoadBalancerServiceName,
		},
		Spec: corev1.ServiceSpec{
			Type:  corev1.ServiceTypeLoadBalancer,
			Ports: []corev1.ServicePort{{Port: 443}},
		},
	}

	caBundleConfigMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: clusterNamespace,
			Name:      resources.CABundleConfigMapName,
		},
		Data: map[string]string{
			resources.CABundleConfigMapKey: caBundle.String(),
		},
	}

	if err := mgr.GetClient().Create(ctx, namespace); err != nil {
		t.Fatalf("failed to create namespace: %v", err)
	}
	if err := mgr.GetClient().Create(ctx, lbService); err != nil {
		t.Fatalf("failed to create the loadbalancer service: %v", err)
	}
	if err := mgr.GetClient().Create(ctx, caBundleConfigMap); err != nil {
		t.Fatalf("failed to create the CA bundle: %v", err)
	}

	if err := mgr.GetClient().Create(ctx, testCluster); err != nil {
		t.Fatalf("failed to create testcluster: %v", err)
	}

	testCluster.Status = kubermaticv1.ClusterStatus{
		UserEmail:     "test@example.com",
		NamespaceName: clusterNamespace,
		ExtendedHealth: kubermaticv1.ExtendedClusterHealth{
			Apiserver:                    kubermaticv1.HealthStatusUp,
			Scheduler:                    kubermaticv1.HealthStatusUp,
			Controller:                   kubermaticv1.HealthStatusUp,
			MachineController:            kubermaticv1.HealthStatusUp,
			Etcd:                         kubermaticv1.HealthStatusUp,
			OpenVPN:                      kubermaticv1.HealthStatusUp,
			CloudProviderInfrastructure:  kubermaticv1.HealthStatusUp,
			UserClusterControllerManager: kubermaticv1.HealthStatusUp,
		},
		Versions: kubermaticv1.ClusterVersionsStatus{
			ControlPlane:      *version,
			Apiserver:         *version,
			ControllerManager: *version,
			Scheduler:         *version,
		},
	}

	if err := mgr.GetClient().Status().Update(ctx, testCluster); err != nil {
		t.Fatalf("failed to update testcluster: %v", err)
	}

	// explicitly set TypeMeta because we need them for setting owner references
	// and in a real-life scenario, the type meta is always set;
	// set this *after* the Create() call, which would remove the TypeMeta for some reason.
	namespace.TypeMeta = metav1.TypeMeta{
		Kind:       "Namespace",
		APIVersion: "v1",
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
		kubermaticImage:      defaulting.DefaultKubermaticImage,
		dnatControllerImage:  defaulting.DefaultDNATControllerImage,
		etcdLauncherImage:    defaulting.DefaultEtcdLauncherImage,
		seedGetter: func() (*kubermaticv1.Seed, error) {
			return &kubermaticv1.Seed{
				Spec: kubermaticv1.SeedSpec{
					Datacenters: map[string]kubermaticv1.Datacenter{
						testCluster.Spec.Cloud.DatacenterName: {},
					},
					NodeportProxy: kubermaticv1.NodeportProxyConfig{
						Envoy: kubermaticv1.NodePortProxyComponentEnvoy{
							NodeportProxyComponent: kubermaticv1.NodeportProxyComponent{
								DockerRepository: defaulting.DefaultEnvoyDockerRepository,
							},
						},
					},
				},
			}, nil
		},
		configGetter: func(_ context.Context) (*kubermaticv1.KubermaticConfiguration, error) {
			kubermaticConfig := &kubermaticv1.KubermaticConfiguration{}
			kubermaticConfig, err := defaulting.DefaultConfiguration(kubermaticConfig, kubermaticlog.Logger)
			if err != nil {
				return nil, err
			}
			return kubermaticConfig, nil
		},
		caBundle:                caBundle,
		userClusterConnProvider: new(testUserClusterConnectionProvider),
		versions:                kubermatic.GetFakeVersions(),
	}

	if _, err := r.ensureResourcesAreDeployed(ctx, testCluster, namespace); err != nil {
		t.Fatalf("Initial resource deployment failed, this indicates that some resources are invalid. Error: %v", err)
	}

	if _, err := r.ensureResourcesAreDeployed(ctx, testCluster, namespace); err != nil {
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

	// stop the manager
	cancel()

	// wait for it to be stopped
	<-running

	// shutdown envtest
	if err := env.Stop(); err != nil {
		t.Errorf("failed to stop testenv: %v", err)
	}
}
