// +build integration

package kubernetes

import (
	"context"
	"testing"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"

	corev1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func TestEnsureResourcesAreDeployedIdempotency(t *testing.T) {
	kubermaticlog.Logger = kubermaticlog.New(true, kubermaticlog.FormatJSON).Sugar()
	env := &envtest.Environment{
		//AttachControlPlaneOutput: true,
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
		// Results in timeouts, maybe a controller-runtime bug?
		// Paths: []string{"../../../../../config/kubermatic/crd"},
		CRDs: []*apiextensionsv1beta1.CustomResourceDefinition{{
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
		}},
	}
	if _, err := envtest.InstallCRDs(cfg, crdInstallOpts); err != nil {
		t.Fatalf("failed install crds: %v", err)
	}

	go func() {
		if err := mgr.Start(make(chan struct{})); err != nil {
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
		},
		Status: kubermaticv1.ClusterStatus{
			NamespaceName: "cluster-test-cluster",
		},
	}

	ctx := context.Background()
	if err := mgr.GetClient().Create(ctx, testCluster); err != nil {
		t.Fatalf("failed to create testcluster: %v", err)
	}

	r := &Reconciler{
		log:    kubermaticlog.Logger,
		Client: mgr.GetClient(),
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

	if err := r.ensureResourcesAreDeployed(ctx, testCluster); err != nil {
		t.Fatalf("Initial resource deployment failed, this indicates that some resources are invalid. Error: %v", err)
	}

	if err := r.ensureResourcesAreDeployed(ctx, testCluster); err != nil {
		t.Fatalf("The second resource reconciliation failed, indicating we don't properly default some fields. Check the `Object differs from generated one` error for the object for which we timed out. Original error: %v", err)
	}
}
