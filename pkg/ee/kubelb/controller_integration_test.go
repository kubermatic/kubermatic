//go:build ee && integration

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2026 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package kubelbcontroller

import (
	"context"
	encodingjson "encoding/json"
	"fmt"
	"reflect"
	"testing"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubelbclusterresources "k8c.io/kubermatic/v2/pkg/ee/kubelb/resources/kubelb-cluster"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestProjectWatchPropagatesTenantDefaultsIntegration(t *testing.T) {
	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{"testdata/tenant-crd.yaml"},
		ErrorIfCRDPathMissing: true,
		CRDs: []*apiextensionsv1.CustomResourceDefinition{
			projectWatchIntegrationCRD("Project", "projects"),
			projectWatchIntegrationCRD("Cluster", "clusters"),
		},
	}
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("start API server: %v", err)
	}
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Errorf("stop API server: %v", err)
		}
	})
	scheme := runtime.NewScheme()
	if err := kubermaticv1.AddToScheme(scheme); err != nil {
		t.Fatalf("register KKP resources: %v", err)
	}
	client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("create direct client: %v", err)
	}
	mgr, err := manager.New(cfg, manager.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: "0"},
		HealthProbeBindAddress: "0",
	})
	if err != nil {
		t.Fatalf("create manager: %v", err)
	}

	// The target controller has no Cluster watch or periodic requeue. Every
	// reconciliation must originate from the production Project mapper and predicate.
	tenantReconciler := &reconciler{Client: mgr.GetClient()}
	_, err = builder.ControllerManagedBy(mgr).
		Named("tenant-project-watch-test").
		Watches(&kubermaticv1.Project{}, tenantReconciler.enqueueClustersForProject(), builder.WithPredicates(projectDefaultsChangedPredicate())).
		Build(reconcile.Func(func(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
			cluster := &kubermaticv1.Cluster{}
			if err := mgr.GetClient().Get(ctx, request.NamespacedName, cluster); err != nil {
				return reconcile.Result{}, err
			}
			project := &kubermaticv1.Project{}
			if err := mgr.GetClient().Get(ctx, ctrlruntimeclient.ObjectKey{Name: cluster.Labels[kubermaticv1.ProjectIDLabelKey]}, project); err != nil {
				return reconcile.Result{}, err
			}
			_, err := tenantReconciler.createOrUpdateKubeLBManagementClusterResources(ctx, client, cluster, project.Spec.DefaultTenantSpec)
			return reconcile.Result{}, err
		}))
	if err != nil {
		t.Fatalf("create target controller: %v", err)
	}

	ctx := context.Background()
	project := &kubermaticv1.Project{
		ObjectMeta: metav1.ObjectMeta{Name: "project"},
		Spec: kubermaticv1.ProjectSpec{
			DefaultTenantSpec: &runtime.RawExtension{Raw: []byte(`{"gatewayAPI":{"class":"eg-initial"}}`)},
		},
	}
	if err := client.Create(ctx, project); err != nil {
		t.Fatalf("create Project: %v", err)
	}
	cluster := projectWatchCluster("project-watch-cluster")
	if err := client.Create(ctx, cluster); err != nil {
		t.Fatalf("create Cluster: %v", err)
	}
	cluster.Status.NamespaceName = "cluster-" + cluster.Name
	if err := client.Status().Update(ctx, cluster); err != nil {
		t.Fatalf("mark Cluster provisioned: %v", err)
	}
	clusterResourceVersion := cluster.ResourceVersion

	managerCtx, cancel := context.WithCancel(ctx)
	managerDone := make(chan error, 1)
	go func() { managerDone <- mgr.Start(managerCtx) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-managerDone:
			if err != nil {
				t.Errorf("manager failed: %v", err)
			}
		case <-time.After(10 * time.Second):
			t.Error("manager did not stop")
		}
	})

	waitForClass := func(want string) {
		t.Helper()
		var lastClass string
		err := wait.PollUntilContextTimeout(ctx, 50*time.Millisecond, 15*time.Second, true, func(ctx context.Context) (bool, error) {
			tenant := integrationTenant(cluster.Name, nil)
			if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(tenant), tenant); err != nil {
				return false, ctrlruntimeclient.IgnoreNotFound(err)
			}
			value, found, err := unstructured.NestedString(tenant.Object, "spec", "gatewayAPI", "class")
			lastClass = value
			return value == want && found == (want != ""), err
		})
		if err != nil {
			t.Fatalf("wait for Project watch to set Tenant class %q: %v (last class %q)", want, err, lastClass)
		}
		unchanged := &kubermaticv1.Cluster{}
		if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cluster), unchanged); err != nil {
			t.Fatalf("get Cluster: %v", err)
		}
		if unchanged.ResourceVersion != clusterResourceVersion {
			t.Fatalf("Cluster changed during Project-only propagation: resourceVersion %s -> %s", clusterResourceVersion, unchanged.ResourceVersion)
		}
	}
	waitForClass("eg-initial")
	project.Spec.DefaultTenantSpec = &runtime.RawExtension{Raw: []byte(`{"gatewayAPI":{"class":"eg-updated"}}`)}
	if err := client.Update(ctx, project); err != nil {
		t.Fatalf("update Project defaults: %v", err)
	}
	waitForClass("eg-updated")
	project.Spec.DefaultTenantSpec = nil
	if err := client.Update(ctx, project); err != nil {
		t.Fatalf("remove Project defaults: %v", err)
	}
	waitForClass("")
}

// These resources are used only for watching and reading through the cache;
// preserve their typed payloads without installing unrelated KKP validation.
func projectWatchIntegrationCRD(kind, plural string) *apiextensionsv1.CustomResourceDefinition {
	return &apiextensionsv1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: plural + "." + kubermaticv1.SchemeGroupVersion.Group},
		Spec: apiextensionsv1.CustomResourceDefinitionSpec{
			Group: kubermaticv1.SchemeGroupVersion.Group,
			Names: apiextensionsv1.CustomResourceDefinitionNames{Kind: kind, ListKind: kind + "List", Plural: plural},
			Scope: apiextensionsv1.ClusterScoped,
			Versions: []apiextensionsv1.CustomResourceDefinitionVersion{{
				Name: kubermaticv1.SchemeGroupVersion.Version, Served: true, Storage: true,
				Subresources: &apiextensionsv1.CustomResourceSubresources{Status: &apiextensionsv1.CustomResourceSubresourceStatus{}},
				Schema: &apiextensionsv1.CustomResourceValidation{OpenAPIV3Schema: &apiextensionsv1.JSONSchemaProps{
					Type: "object",
					Properties: map[string]apiextensionsv1.JSONSchemaProps{
						"spec":   {Type: "object", XPreserveUnknownFields: ptr.To(true)},
						"status": {Type: "object", XPreserveUnknownFields: ptr.To(true)},
					},
				}},
			}},
		},
	}
}

// These cases need an API server: the fake client's deduced schema does not
// reproduce CRD defaulting or the list-map ownership of GatewayClass mappings.
func TestTenantProjectDefaultsIntegration(t *testing.T) {
	fixture := newTenantIntegrationFixture(t)
	t.Run("update and remove project fields while preserving administrator settings", fixture.updateAndRemoveDefaults)
	t.Run("administrator override conflicts without partially applying defaults", fixture.administratorConflict)
	t.Run("legacy fields are preserved until explicitly adopted", fixture.legacyMigration)
	t.Run("legacy GatewayClass mappings retain omitted entries and fields", fixture.legacyGatewayClassMappings)
	t.Run("legacy administrator edit survives ownership migration", fixture.legacyAdministratorEdit)
	t.Run("migration cannot overwrite concurrent field ownership changes", fixture.concurrentMigration)
	t.Run("unknown previous owner is not migrated", fixture.unknownOwner)
	t.Run("invalid defaults leave existing Tenant unchanged", fixture.invalidDefaults)
	t.Run("Tenant pending deletion is not updated", fixture.deletingTenant)
	t.Run("legacy API defaults survive changed and removed schema defaults", fixture.legacyDefaultProvenance)
}

type tenantIntegrationFixture struct {
	ctx        context.Context
	client     ctrlruntimeclient.Client
	reconciler *reconciler
	sequence   int
}

func newTenantIntegrationFixture(t *testing.T) *tenantIntegrationFixture {
	t.Helper()
	env := &envtest.Environment{
		CRDDirectoryPaths:     []string{"testdata/tenant-crd.yaml"},
		ErrorIfCRDPathMissing: true,
	}
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("start API server: %v", err)
	}
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Errorf("stop API server: %v", err)
		}
	})
	scheme := runtime.NewScheme()
	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		t.Fatalf("register CRDs: %v", err)
	}
	client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("create client: %v", err)
	}
	return &tenantIntegrationFixture{ctx: context.Background(), client: client, reconciler: &reconciler{}}
}

func (f *tenantIntegrationFixture) newCluster() *kubermaticv1.Cluster {
	f.sequence++
	cluster := &kubermaticv1.Cluster{ObjectMeta: metav1.ObjectMeta{
		Name:   fmt.Sprintf("tenant-defaults-%d", f.sequence),
		Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: "test-project"},
	}}
	cluster.Status.Address.ExternalName = "test.example.com"
	return cluster
}

func (f *tenantIntegrationFixture) get(t *testing.T, cluster *kubermaticv1.Cluster) *unstructured.Unstructured {
	t.Helper()
	tenant := integrationTenant(cluster.Name, nil)
	if err := f.client.Get(f.ctx, ctrlruntimeclient.ObjectKeyFromObject(tenant), tenant); err != nil {
		t.Fatalf("get Tenant: %v", err)
	}
	return tenant
}

func (f *tenantIntegrationFixture) reconcile(t *testing.T, cluster *kubermaticv1.Cluster, raw string) {
	t.Helper()
	if _, err := f.reconciler.createOrUpdateKubeLBManagementClusterResources(f.ctx, f.client, cluster, &runtime.RawExtension{Raw: []byte(raw)}); err != nil {
		t.Fatalf("reconcile Tenant: %v", err)
	}
}

func (f *tenantIntegrationFixture) applyAdmin(t *testing.T, cluster *kubermaticv1.Cluster, raw string, force bool) {
	t.Helper()
	tenant := integrationTenant(cluster.Name, integrationTenantSpec(t, raw))
	opts := []ctrlruntimeclient.ApplyOption{ctrlruntimeclient.FieldOwner("tenant-admin")}
	if force {
		opts = append(opts, ctrlruntimeclient.ForceOwnership)
	}
	if err := f.client.Apply(f.ctx, ctrlruntimeclient.ApplyConfigurationFromUnstructured(tenant), opts...); err != nil {
		t.Fatalf("apply administrator settings: %v", err)
	}
}

func (f *tenantIntegrationFixture) setManagerMetadata(t *testing.T, cluster *kubermaticv1.Cluster) {
	t.Helper()
	tenant := f.get(t, cluster)
	base := tenant.DeepCopy()
	tenant.SetFinalizers([]string{"kubelb.k8c.io/cleanup"})
	tenant.SetAnnotations(map[string]string{"test.kubelb.k8c.io/keep": "true"})
	if err := f.client.Patch(f.ctx, tenant, ctrlruntimeclient.MergeFrom(base), ctrlruntimeclient.FieldOwner("kubelb-manager")); err != nil {
		t.Fatalf("set manager metadata: %v", err)
	}
	tenant.Object["status"] = map[string]interface{}{"phase": "Ready"}
	if err := f.client.Status().Update(f.ctx, tenant, ctrlruntimeclient.FieldOwner("kubelb-manager")); err != nil {
		t.Fatalf("set manager status: %v", err)
	}
}

func assertIntegrationManagerMetadata(t *testing.T, tenant *unstructured.Unstructured) {
	t.Helper()
	if !reflect.DeepEqual(tenant.GetFinalizers(), []string{"kubelb.k8c.io/cleanup"}) || tenant.GetAnnotations()["test.kubelb.k8c.io/keep"] != "true" {
		t.Errorf("manager metadata changed: finalizers=%v annotations=%v", tenant.GetFinalizers(), tenant.GetAnnotations())
	}
	if !reflect.DeepEqual(tenant.Object["status"], map[string]interface{}{"phase": "Ready"}) {
		t.Errorf("manager status changed: %v", tenant.Object["status"])
	}
}

func (f *tenantIntegrationFixture) updateAndRemoveDefaults(t *testing.T) {
	cluster := f.newCluster()
	f.reconcile(t, cluster, `{
		"allowedDomains":["*.example.com"],
		"gatewayAPI":{"classMappings":[{"source":"internal","target":"eg-internal"},{"source":"public","target":"eg-public"}]},
		"timeouts":{"connect":"5s","request":"30s"},
		"defaultAnnotations":{"service":{"project":"old","remove":"old"}}
	}`)
	initial := f.get(t, cluster)
	if initial.GetLabels()["kubermatic.k8c.io/cluster-project-id"] != "test-project" || initial.GetLabels()["kubermatic.k8c.io/cluster-name"] != cluster.Name || initial.GetLabels()["kubermatic.k8c.io/cluster-external-name"] != "test.example.com" {
		t.Fatalf("missing cluster identity labels: %v", initial.GetLabels())
	}
	f.applyAdmin(t, cluster, `{
		"gatewayAPI":{"class":"eg-admin","classMappings":[{"source":"admin","target":"eg-admin"}]},
		"timeouts":{"idleConnection":"1h"},
		"defaultAnnotations":{"service":{"admin":"keep"}}
	}`, false)
	f.setManagerMetadata(t, cluster)
	cluster.Status.Address.ExternalName = "updated.example.com"
	f.reconcile(t, cluster, `{
		"gatewayAPI":{"classMappings":[{"source":"internal","target":"eg-internal-v2"}]},
		"timeouts":{"connect":"10s"},
		"defaultAnnotations":{"service":{"project":"new"}}
	}`)
	tenant := f.get(t, cluster)
	assertIntegrationTenantSpec(t, tenant, `{
		"allowedDomains":["**"],
		"gatewayAPI":{"class":"eg-admin","classMappings":[{"source":"internal","target":"eg-internal-v2"},{"source":"admin","target":"eg-admin"}]},
		"timeouts":{"connect":"10s","idleConnection":"1h"},
		"defaultAnnotations":{"service":{"project":"new","admin":"keep"}}
	}`)
	if tenant.GetUID() != initial.GetUID() || tenant.GetLabels()["kubermatic.k8c.io/cluster-external-name"] != "updated.example.com" {
		t.Errorf("Tenant identity or external-name label incorrect: uid=%s labels=%v", tenant.GetUID(), tenant.GetLabels())
	}
	assertIntegrationManagerMetadata(t, tenant)

	for _, defaults := range []*runtime.RawExtension{nil, {}, {Raw: []byte(`{}`)}, {Raw: []byte(`null`)}} {
		if _, err := f.reconciler.createOrUpdateKubeLBManagementClusterResources(f.ctx, f.client, cluster, defaults); err != nil {
			t.Fatalf("clear project defaults: %v", err)
		}
		tenant = f.get(t, cluster)
		assertIntegrationTenantSpec(t, tenant, `{
			"allowedDomains":["**"],
			"gatewayAPI":{"class":"eg-admin","classMappings":[{"source":"admin","target":"eg-admin"}]},
			"timeouts":{"idleConnection":"1h"},
			"defaultAnnotations":{"service":{"admin":"keep"}}
		}`)
		assertIntegrationManagerMetadata(t, tenant)
	}
}

func (f *tenantIntegrationFixture) administratorConflict(t *testing.T) {
	cluster := f.newCluster()
	f.reconcile(t, cluster, `{"gatewayAPI":{"class":"eg-project"},"loadBalancer":{"limit":5}}`)
	f.applyAdmin(t, cluster, `{"gatewayAPI":{"class":"eg-admin"}}`, true)
	before := f.get(t, cluster)
	existingUsable, err := f.reconciler.createOrUpdateKubeLBManagementClusterResources(f.ctx, f.client, cluster, &runtime.RawExtension{Raw: []byte(`{"gatewayAPI":{"class":"eg-new"},"loadBalancer":{"limit":10}}`)})
	if !existingUsable {
		t.Error("existing Tenant must remain usable after an apply conflict")
	}
	if !apierrors.IsConflict(err) {
		t.Fatalf("expected field ownership conflict, got %v", err)
	}
	after := f.get(t, cluster)
	if !reflect.DeepEqual(before.Object, after.Object) {
		t.Error("conflicting apply changed the existing Tenant")
	}
}

func (f *tenantIntegrationFixture) legacyMigration(t *testing.T) {
	cluster := f.newCluster()
	tenant := integrationTenant(cluster.Name, integrationTenantSpec(t, `{
		"gatewayAPI":{"class":"eg-old"},
		"timeouts":{"connect":"5s","request":"30s"},
		"defaultAnnotations":{"service":{"project":"old","remove":"old"}}
	}`))
	tenant.SetLabels(map[string]string{
		"kubermatic.k8c.io/cluster-name":          cluster.Name,
		"kubermatic.k8c.io/cluster-external-name": "old.example.com",
		"kubermatic.k8c.io/cluster-project-id":    "test-project",
	})
	if err := f.client.Create(f.ctx, tenant, ctrlruntimeclient.FieldOwner("seed-controller-manager")); err != nil {
		t.Fatalf("create legacy Tenant: %v", err)
	}
	f.applyAdmin(t, cluster, `{"timeouts":{"idleConnection":"1h"},"defaultAnnotations":{"service":{"admin":"keep"}}}`, false)
	f.setManagerMetadata(t, cluster)
	f.reconcile(t, cluster, `{"gatewayAPI":{"class":"eg-new"},"timeouts":{"connect":"10s"}}`)
	tenant = f.get(t, cluster)
	assertIntegrationTenantSpec(t, tenant, `{
		"allowedDomains":["**"],
		"gatewayAPI":{"class":"eg-new"},
		"timeouts":{"connect":"10s","request":"30s","idleConnection":"1h"},
		"defaultAnnotations":{"service":{"project":"old","remove":"old","admin":"keep"}}
	}`)
	assertIntegrationManagerMetadata(t, tenant)
	if tenant.GetLabels()["kubermatic.k8c.io/cluster-external-name"] != "test.example.com" {
		t.Error("legacy cluster label was not updated")
	}

	// A later Project update can adopt an initially omitted legacy field.
	// Removing it subsequently then follows normal apply ownership semantics.
	f.reconcile(t, cluster, `{"gatewayAPI":{"class":"eg-later"},"timeouts":{"request":"45s"},"defaultAnnotations":{"service":{"project":"new"}}}`)
	assertIntegrationTenantSpec(t, f.get(t, cluster), `{
		"allowedDomains":["**"],
		"gatewayAPI":{"class":"eg-later"},
		"timeouts":{"request":"45s","idleConnection":"1h"},
		"defaultAnnotations":{"service":{"project":"new","remove":"old","admin":"keep"}}
	}`)
	f.reconcile(t, cluster, `{"gatewayAPI":{"class":"eg-final"}}`)
	assertIntegrationTenantSpec(t, f.get(t, cluster), `{
		"allowedDomains":["**"],
		"gatewayAPI":{"class":"eg-final"},
		"timeouts":{"idleConnection":"1h"},
		"defaultAnnotations":{"service":{"remove":"old","admin":"keep"}}
	}`)
}

func (f *tenantIntegrationFixture) legacyGatewayClassMappings(t *testing.T) {
	// Model a future mapping field that KKP does not include in its desired
	// configuration. The structural schema still tracks that field separately.
	crd := f.tenantCRD(t)
	original := crd.Spec.DeepCopy()
	spec := crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"]
	gateway := spec.Properties["gatewayAPI"]
	mappings := gateway.Properties["classMappings"]
	mappings.Items.Schema.Properties["description"] = apiextensionsv1.JSONSchemaProps{Type: "string"}
	if err := f.client.Update(f.ctx, crd); err != nil {
		t.Fatalf("extend mapping schema: %v", err)
	}
	t.Cleanup(func() {
		current := f.tenantCRD(t)
		current.Spec = *original
		if err := f.client.Update(f.ctx, current); err != nil {
			t.Errorf("restore mapping schema: %v", err)
		}
	})
	f.waitForTenantSchema(t,
		`{"gatewayAPI":{"classMappings":[{"source":"probe","target":"eg-probe","description":"keep"}]}}`,
		`{"allowedDomains":["**"],"gatewayAPI":{"classMappings":[{"source":"probe","target":"eg-probe","description":"keep"}]}}`)

	cluster := f.newCluster()
	tenant := integrationTenant(cluster.Name, integrationTenantSpec(t, `{
		"gatewayAPI":{"classMappings":[
			{"source":"public","target":"eg-old","description":"keep-public"},
			{"source":"internal","target":"eg-internal","description":"keep-internal"}
		]}
	}`))
	if err := f.client.Create(f.ctx, tenant, ctrlruntimeclient.FieldOwner("seed-controller-manager")); err != nil {
		t.Fatalf("create legacy Tenant mappings: %v", err)
	}
	f.reconcile(t, cluster, `{"gatewayAPI":{"classMappings":[{"source":"public","target":"eg-new"}]}}`)
	tenant = f.get(t, cluster)
	assertIntegrationTenantSpec(t, tenant, `{
		"allowedDomains":["**"],
		"gatewayAPI":{"classMappings":[
			{"source":"public","target":"eg-new","description":"keep-public"},
			{"source":"internal","target":"eg-internal","description":"keep-internal"}
		]}
	}`)
	assertIntegrationTenantFieldOwner(t, tenant, ControllerName, true, "f:spec", "f:gatewayAPI", "f:classMappings", `k:{"source":"public"}`, "f:target")
	for _, path := range [][]string{
		{"f:spec", "f:gatewayAPI", "f:classMappings", `k:{"source":"public"}`, "f:description"},
		{"f:spec", "f:gatewayAPI", "f:classMappings", `k:{"source":"internal"}`, "f:target"},
	} {
		assertIntegrationTenantFieldOwner(t, tenant, "seed-controller-manager", true, path...)
		assertIntegrationTenantFieldOwner(t, tenant, ControllerName, false, path...)
	}
}

func (f *tenantIntegrationFixture) legacyDefaultProvenance(t *testing.T) {
	original := f.tenantCRD(t).Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties["allowedDomains"].Default
	t.Cleanup(func() { f.setTenantDefault(t, original) })
	cases := []struct {
		name         string
		defaultValue *apiextensionsv1.JSON
		cluster      *kubermaticv1.Cluster
	}{
		{name: "changed default", defaultValue: &apiextensionsv1.JSON{Raw: []byte(`["*.current.example"]`)}, cluster: f.newCluster()},
		{name: "removed default", cluster: f.newCluster()},
	}
	// Both Tenants were created under the original schema. Their Update manager
	// owns API defaults as well as the fields explicitly supplied by old KKP.
	for _, tc := range cases {
		tenant := integrationTenant(tc.cluster.Name, integrationTenantSpec(t, `{"gatewayAPI":{"class":"eg-old"}}`))
		if err := f.client.Create(f.ctx, tenant, ctrlruntimeclient.FieldOwner("seed-controller-manager")); err != nil {
			t.Fatalf("create legacy Tenant: %v", err)
		}
		assertIntegrationTenantFieldOwner(t, tenant, "seed-controller-manager", true, "f:spec", "f:allowedDomains")
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f.setTenantDefault(t, tc.defaultValue)
			for _, class := range []string{"eg-new", "eg-later"} {
				f.reconcile(t, tc.cluster, fmt.Sprintf(`{"gatewayAPI":{"class":%q}}`, class))
				tenant := f.get(t, tc.cluster)
				assertIntegrationTenantSpec(t, tenant, fmt.Sprintf(`{"allowedDomains":["**"],"gatewayAPI":{"class":%q}}`, class))
				assertIntegrationTenantFieldOwner(t, tenant, "seed-controller-manager", true, "f:spec", "f:allowedDomains")
				assertIntegrationTenantFieldOwner(t, tenant, ControllerName, false, "f:spec", "f:allowedDomains")
			}

			// Only an explicit Project value adopts the field. Later omission can
			// then remove it and let the currently installed schema default apply.
			f.reconcile(t, tc.cluster, `{"gatewayAPI":{"class":"eg-later"},"allowedDomains":["*.project.example"]}`)
			tenant := f.get(t, tc.cluster)
			assertIntegrationTenantSpec(t, tenant, `{"allowedDomains":["*.project.example"],"gatewayAPI":{"class":"eg-later"}}`)
			assertIntegrationTenantFieldOwner(t, tenant, ControllerName, true, "f:spec", "f:allowedDomains")
			assertIntegrationTenantFieldOwner(t, tenant, "seed-controller-manager", false, "f:spec", "f:allowedDomains")
			f.reconcile(t, tc.cluster, `{"gatewayAPI":{"class":"eg-final"}}`)
			expected := `{"gatewayAPI":{"class":"eg-final"}}`
			if tc.defaultValue != nil {
				expected = fmt.Sprintf(`{"gatewayAPI":{"class":"eg-final"},"allowedDomains":%s}`, tc.defaultValue.Raw)
			}
			assertIntegrationTenantSpec(t, f.get(t, tc.cluster), expected)
		})
	}
}

func (f *tenantIntegrationFixture) tenantCRD(t *testing.T) *apiextensionsv1.CustomResourceDefinition {
	t.Helper()
	crd := &apiextensionsv1.CustomResourceDefinition{}
	if err := f.client.Get(f.ctx, ctrlruntimeclient.ObjectKey{Name: "tenants.kubelb.k8c.io"}, crd); err != nil {
		t.Fatalf("get Tenant CRD: %v", err)
	}
	return crd
}

func (f *tenantIntegrationFixture) setTenantDefault(t *testing.T, value *apiextensionsv1.JSON) {
	t.Helper()
	crd := f.tenantCRD(t)
	spec := crd.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"]
	allowedDomains := spec.Properties["allowedDomains"]
	allowedDomains.Default = value
	spec.Properties["allowedDomains"] = allowedDomains
	if err := f.client.Update(f.ctx, crd); err != nil {
		t.Fatalf("update Tenant schema default: %v", err)
	}
	expected := `{"gatewayAPI":{"class":"eg-probe"}}`
	if value != nil {
		expected = fmt.Sprintf(`{"gatewayAPI":{"class":"eg-probe"},"allowedDomains":%s}`, value.Raw)
	}
	f.waitForTenantSchema(t, `{"gatewayAPI":{"class":"eg-probe"}}`, expected)
}

func (f *tenantIntegrationFixture) waitForTenantSchema(t *testing.T, probeSpec, expectedSpec string) {
	t.Helper()
	expected := integrationTenantSpec(t, expectedSpec)
	err := wait.PollUntilContextTimeout(f.ctx, 100*time.Millisecond, 15*time.Second, true, func(ctx context.Context) (bool, error) {
		probe := integrationTenant(f.newCluster().Name, integrationTenantSpec(t, probeSpec))
		if err := f.client.Create(ctx, probe); err != nil {
			return false, err
		}
		if err := f.client.Delete(ctx, probe); err != nil {
			return false, err
		}
		actual, _, err := unstructured.NestedMap(probe.Object, "spec")
		return reflect.DeepEqual(actual, expected), err
	})
	if err != nil {
		t.Fatalf("wait for updated Tenant schema: %v", err)
	}
}

func assertIntegrationTenantFieldOwner(t *testing.T, tenant *unstructured.Unstructured, manager string, want bool, path ...string) {
	t.Helper()
	owned := false
	for _, entry := range tenant.GetManagedFields() {
		if entry.Manager != manager || entry.FieldsV1 == nil || entry.Subresource != "" {
			continue
		}
		fields := map[string]interface{}{}
		if err := encodingjson.NewDecoder(entry.FieldsV1.GetRawReader()).Decode(&fields); err != nil {
			t.Fatalf("decode field ownership: %v", err)
		}
		_, found, err := unstructured.NestedFieldNoCopy(fields, path...)
		if err != nil {
			t.Fatalf("read field ownership: %v", err)
		}
		owned = owned || found
	}
	if owned != want {
		t.Errorf("manager %q ownership of %v: got %t, want %t", manager, path, owned, want)
	}
}

func (f *tenantIntegrationFixture) legacyAdministratorEdit(t *testing.T) {
	cluster := f.newCluster()
	tenant := integrationTenant(cluster.Name, integrationTenantSpec(t, `{"gatewayAPI":{"class":"eg-old"},"loadBalancer":{"limit":5}}`))
	if err := f.client.Create(f.ctx, tenant, ctrlruntimeclient.FieldOwner("seed-controller-manager")); err != nil {
		t.Fatalf("create legacy Tenant: %v", err)
	}
	if err := f.client.Patch(f.ctx, tenant, ctrlruntimeclient.RawPatch(types.MergePatchType, []byte(`{"spec":{"gatewayAPI":{"class":"eg-admin"}}}`)), ctrlruntimeclient.FieldOwner("tenant-admin")); err != nil {
		t.Fatalf("edit legacy Tenant: %v", err)
	}
	_, err := f.reconciler.createOrUpdateKubeLBManagementClusterResources(f.ctx, f.client, cluster, &runtime.RawExtension{Raw: []byte(`{"gatewayAPI":{"class":"eg-project"},"loadBalancer":{"limit":10}}`)})
	if !apierrors.IsConflict(err) {
		t.Fatalf("expected conflict with the administrator's legacy edit, got %v", err)
	}
	assertIntegrationTenantSpec(t, f.get(t, cluster), `{"allowedDomains":["**"],"gatewayAPI":{"class":"eg-admin"},"loadBalancer":{"limit":5}}`)
}

func (f *tenantIntegrationFixture) concurrentMigration(t *testing.T) {
	cluster := f.newCluster()
	tenant := integrationTenant(cluster.Name, integrationTenantSpec(t, `{"gatewayAPI":{"class":"eg-old"}}`))
	if err := f.client.Create(f.ctx, tenant, ctrlruntimeclient.FieldOwner("seed-controller-manager")); err != nil {
		t.Fatalf("create legacy Tenant: %v", err)
	}
	desired := integrationTenant(cluster.Name, integrationTenantSpec(t, `{"gatewayAPI":{"class":"eg-project"}}`))
	migration, err := kubelbclusterresources.TenantManagedFieldsMigrationPatch(tenant, desired, ControllerName)
	if err != nil || migration == nil {
		t.Fatalf("prepare ownership migration: patch=%s error=%v", migration, err)
	}
	if err := f.client.Patch(f.ctx, tenant, ctrlruntimeclient.RawPatch(types.MergePatchType, []byte(`{"spec":{"gatewayAPI":{"class":"eg-admin"}}}`)), ctrlruntimeclient.FieldOwner("tenant-admin")); err != nil {
		t.Fatalf("concurrent administrator edit: %v", err)
	}
	before := f.get(t, cluster)
	if err := f.client.Patch(f.ctx, tenant, ctrlruntimeclient.RawPatch(types.JSONPatchType, migration)); !apierrors.IsConflict(err) {
		t.Fatalf("expected stale ownership migration to conflict, got %v", err)
	}
	if !reflect.DeepEqual(before.Object, f.get(t, cluster).Object) {
		t.Error("stale migration changed the Tenant or its field ownership")
	}
}

func (f *tenantIntegrationFixture) unknownOwner(t *testing.T) {
	cluster := f.newCluster()
	tenant := integrationTenant(cluster.Name, integrationTenantSpec(t, `{"gatewayAPI":{"class":"eg-existing"}}`))
	if err := f.client.Create(f.ctx, tenant, ctrlruntimeclient.FieldOwner("external-installer")); err != nil {
		t.Fatalf("create externally managed Tenant: %v", err)
	}
	before := f.get(t, cluster)
	_, err := f.reconciler.createOrUpdateKubeLBManagementClusterResources(f.ctx, f.client, cluster, &runtime.RawExtension{Raw: []byte(`{"gatewayAPI":{"class":"eg-project"}}`)})
	if !apierrors.IsConflict(err) {
		t.Fatalf("expected existing owner conflict, got %v", err)
	}
	if !reflect.DeepEqual(before.Object, f.get(t, cluster).Object) {
		t.Error("reconciliation changed an externally managed Tenant after a conflict")
	}
}

func (f *tenantIntegrationFixture) invalidDefaults(t *testing.T) {
	cluster := f.newCluster()
	f.reconcile(t, cluster, `{"gatewayAPI":{"class":"eg-stable"}}`)
	for _, raw := range []string{`{`, `[]`, `"text"`, `true`, `42`, `{"unknownSetting":true}`, `{"loadBalancer":{"limit":"invalid"}}`} {
		t.Run(raw, func(t *testing.T) {
			before := f.get(t, cluster)
			existingUsable, err := f.reconciler.createOrUpdateKubeLBManagementClusterResources(f.ctx, f.client, cluster, &runtime.RawExtension{Raw: []byte(raw)})
			if !existingUsable {
				t.Error("existing Tenant must remain usable after invalid defaults")
			}
			if err == nil {
				t.Fatal("expected invalid defaults to be rejected")
			}
			if !reflect.DeepEqual(before.Object, f.get(t, cluster).Object) {
				t.Error("invalid defaults changed the existing Tenant")
			}
		})
	}
}

func (f *tenantIntegrationFixture) deletingTenant(t *testing.T) {
	cluster := f.newCluster()
	f.reconcile(t, cluster, `{"gatewayAPI":{"class":"eg-original"}}`)
	f.setManagerMetadata(t, cluster)
	tenant := f.get(t, cluster)
	if err := f.client.Delete(f.ctx, tenant); err != nil {
		t.Fatalf("delete Tenant: %v", err)
	}
	before := f.get(t, cluster)
	if before.GetDeletionTimestamp().IsZero() {
		t.Fatal("Tenant should be pending finalization")
	}
	existingUsable, err := f.reconciler.createOrUpdateKubeLBManagementClusterResources(f.ctx, f.client, cluster, &runtime.RawExtension{Raw: []byte(`{"gatewayAPI":{"class":"eg-new"}}`)})
	if err == nil {
		t.Fatal("expected reconciliation of a deleting Tenant to fail")
	}
	if existingUsable {
		t.Error("a terminating Tenant must not be considered usable")
	}
	if !reflect.DeepEqual(before.Object, f.get(t, cluster).Object) {
		t.Error("reconciliation changed a Tenant pending deletion")
	}
}

func integrationTenant(name string, spec map[string]interface{}) *unstructured.Unstructured {
	tenant := &unstructured.Unstructured{}
	tenant.SetGroupVersionKind(kubelbclusterresources.KubelbTenantGVK)
	tenant.SetName(name)
	if spec != nil {
		tenant.Object["spec"] = spec
	}
	return tenant
}

func integrationTenantSpec(t *testing.T, raw string) map[string]interface{} {
	t.Helper()
	var spec map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		t.Fatalf("decode test Tenant spec: %v", err)
	}
	return spec
}

func assertIntegrationTenantSpec(t *testing.T, tenant *unstructured.Unstructured, raw string) {
	t.Helper()
	expected := integrationTenantSpec(t, raw)
	actual, _, err := unstructured.NestedMap(tenant.Object, "spec")
	if err != nil {
		t.Fatalf("read Tenant spec: %v", err)
	}
	// The API server may reorder a map list when merging independently owned
	// entries. Compare mappings by source, since ordering has no significance.
	for _, spec := range []map[string]interface{}{expected, actual} {
		mappings, found, err := unstructured.NestedSlice(spec, "gatewayAPI", "classMappings")
		if err != nil {
			t.Fatalf("read GatewayClass mappings: %v", err)
		}
		if found {
			bySource := map[string]interface{}{}
			for _, entry := range mappings {
				mapping := entry.(map[string]interface{})
				bySource[mapping["source"].(string)] = mapping
			}
			if err := unstructured.SetNestedMap(spec, bySource, "gatewayAPI", "classMappings"); err != nil {
				t.Fatalf("normalize GatewayClass mappings: %v", err)
			}
		}
	}
	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Tenant spec mismatch:\nexpected: %#v\nactual:   %#v", expected, actual)
	}
}
