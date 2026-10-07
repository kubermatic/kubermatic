//go:build ee

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
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	clusterclient "k8c.io/kubermatic/v2/pkg/cluster/client"
	"k8c.io/kubermatic/v2/pkg/controller/util"
	kubelbclusterresources "k8c.io/kubermatic/v2/pkg/ee/kubelb/resources/kubelb-cluster"
	kubelbuserclusterresources "k8c.io/kubermatic/v2/pkg/ee/kubelb/resources/user-cluster"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/workqueue"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestEnqueueClustersForProjectFiltersClusters(t *testing.T) {
	project := &kubermaticv1.Project{ObjectMeta: metav1.ObjectMeta{Name: "project"}}
	objects := []ctrlruntimeclient.Object{project, projectWatchCluster("active")}
	for _, fixture := range []struct {
		name   string
		modify func(*kubermaticv1.Cluster)
	}{
		{name: "other-project", modify: func(c *kubermaticv1.Cluster) { c.Labels[kubermaticv1.ProjectIDLabelKey] = "other" }},
		{name: "no-project", modify: func(c *kubermaticv1.Cluster) { delete(c.Labels, kubermaticv1.ProjectIDLabelKey) }},
		{name: "paused", modify: func(c *kubermaticv1.Cluster) { c.Spec.Pause = true }},
		{name: "disabled", modify: func(c *kubermaticv1.Cluster) { c.Spec.KubeLB.Enabled = false }},
		{name: "unconfigured", modify: func(c *kubermaticv1.Cluster) { c.Spec.KubeLB = nil }},
		{name: "unprovisioned", modify: func(c *kubermaticv1.Cluster) { c.Status.NamespaceName = "" }},
		{name: "deleting", modify: func(c *kubermaticv1.Cluster) {
			now := metav1.Now()
			c.DeletionTimestamp = &now
			c.Finalizers = []string{CleanupFinalizer}
		}},
		{name: "canary-worker", modify: func(c *kubermaticv1.Cluster) { c.Labels[kubermaticv1.WorkerNameLabelKey] = "canary" }},
		{name: "other-worker", modify: func(c *kubermaticv1.Cluster) { c.Labels[kubermaticv1.WorkerNameLabelKey] = "other" }},
		{name: "empty-worker", modify: func(c *kubermaticv1.Cluster) { c.Labels[kubermaticv1.WorkerNameLabelKey] = "" }},
	} {
		cluster := projectWatchCluster(fixture.name)
		fixture.modify(cluster)
		objects = append(objects, cluster)
	}

	for _, test := range []struct {
		name       string
		workerName string
		want       []string
	}{
		{name: "default worker", want: []string{"active", "empty-worker"}},
		{name: "named worker", workerName: "canary", want: []string{"canary-worker"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[reconcile.Request]())
			t.Cleanup(queue.ShutDown)
			r := &reconciler{
				Client:     fake.NewClientBuilder().WithObjects(objects...).Build(),
				workerName: test.workerName,
			}
			handler := r.enqueueClustersForProject()
			handler.Create(context.Background(), event.CreateEvent{Object: project}, queue)
			handler.Update(context.Background(), event.UpdateEvent{ObjectOld: project, ObjectNew: project.DeepCopy()}, queue)

			var got []string
			for queue.Len() > 0 {
				request, _ := queue.Get()
				got = append(got, request.Name)
				queue.Done(request)
			}
			slices.Sort(got)
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("enqueued clusters = %v, want %v", got, test.want)
			}
		})
	}
}

func TestEnqueueClustersForProjectListFailure(t *testing.T) {
	listCalled := false
	client := fake.NewClientBuilder().WithInterceptorFuncs(interceptor.Funcs{
		List: func(_ context.Context, _ ctrlruntimeclient.WithWatch, list ctrlruntimeclient.ObjectList, _ ...ctrlruntimeclient.ListOption) error {
			listCalled = true
			list.(*kubermaticv1.ClusterList).Items = []kubermaticv1.Cluster{*projectWatchCluster("partial-result")}
			return errors.New("temporary list failure")
		},
	}).Build()
	queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[reconcile.Request]())
	t.Cleanup(queue.ShutDown)
	r := &reconciler{Client: client}
	r.enqueueClustersForProject().Create(context.Background(), event.CreateEvent{
		Object: &kubermaticv1.Project{ObjectMeta: metav1.ObjectMeta{Name: "project"}},
	}, queue)
	if !listCalled {
		t.Fatal("Project event did not list clusters")
	}
	if queue.Len() != 0 {
		t.Fatal("failed list must not enqueue partial results")
	}
}

func TestEnqueueClustersForProjectSkipsInvalidProject(t *testing.T) {
	now := metav1.Now()
	for _, test := range []struct {
		name   string
		object ctrlruntimeclient.Object
	}{
		{name: "nil"},
		{name: "wrong resource", object: projectWatchCluster("active")},
		{name: "terminating", object: &kubermaticv1.Project{ObjectMeta: metav1.ObjectMeta{Name: "project", DeletionTimestamp: &now}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			queue := workqueue.NewTypedRateLimitingQueue(workqueue.DefaultTypedControllerRateLimiter[reconcile.Request]())
			t.Cleanup(queue.ShutDown)
			r := &reconciler{}
			r.enqueueClustersForProject().Create(context.Background(), event.CreateEvent{Object: test.object}, queue)
			if queue.Len() != 0 {
				t.Fatal("invalid or terminating project must not enqueue clusters")
			}
		})
	}
}

func TestProjectDefaultsChangedPredicate(t *testing.T) {
	base := &kubermaticv1.Project{ObjectMeta: metav1.ObjectMeta{Name: "project"}}
	configured := base.DeepCopy()
	configured.Spec.DefaultTenantSpec = &runtime.RawExtension{Raw: []byte(`{"gatewayAPI":{"class":"eg"}}`)}
	changed := configured.DeepCopy()
	changed.Spec.DefaultTenantSpec = &runtime.RawExtension{Raw: []byte(`{"gatewayAPI":{"class":"internal"}}`)}
	metadataOnly := configured.DeepCopy()
	metadataOnly.Labels = map[string]string{"label": "changed"}
	metadataOnly.ResourceVersion = "2"
	statusOnly := configured.DeepCopy()
	statusOnly.Status.Phase = kubermaticv1.ProjectActive
	otherSpec := configured.DeepCopy()
	otherSpec.Spec.Name = "renamed"
	formatted := configured.DeepCopy()
	formatted.Spec.DefaultTenantSpec = &runtime.RawExtension{Raw: []byte("{\n  \"gatewayAPI\": { \"class\": \"eg\" }\n}")}
	terminating := configured.DeepCopy()
	now := metav1.Now()
	terminating.DeletionTimestamp = &now
	p := projectDefaultsChangedPredicate()

	for _, test := range []struct {
		name     string
		old, new *kubermaticv1.Project
		want     bool
	}{
		{name: "defaults added", old: base, new: configured, want: true},
		{name: "defaults changed", old: configured, new: changed, want: true},
		{name: "defaults removed", old: configured, new: base, want: true},
		{name: "unchanged", old: configured, new: configured.DeepCopy()},
		{name: "metadata only", old: configured, new: metadataOnly},
		{name: "status only", old: configured, new: statusOnly},
		{name: "other spec field", old: configured, new: otherSpec},
		{name: "formatting only", old: configured, new: formatted},
		{name: "project terminating", old: base, new: terminating},
		{name: "nil old", new: configured},
		{name: "nil new", old: configured},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := p.Update(event.UpdateEvent{ObjectOld: test.old, ObjectNew: test.new}); got != test.want {
				t.Errorf("predicate = %v, want %v", got, test.want)
			}
		})
	}

	if !p.Create(event.CreateEvent{Object: base}) {
		t.Error("initial event with empty defaults must reconcile changes made while the controller was stopped")
	}
	if !p.Create(event.CreateEvent{Object: configured}) {
		t.Error("initial event with defaults must reconcile existing tenants")
	}
	if p.Create(event.CreateEvent{Object: terminating}) || p.Create(event.CreateEvent{}) {
		t.Error("nil or terminating projects must not enqueue clusters")
	}
	if p.Delete(event.DeleteEvent{Object: configured}) || p.Generic(event.GenericEvent{Object: configured}) {
		t.Error("delete and generic events must not enqueue clusters")
	}
}

func TestProjectDefaultTenantSpecsEqual(t *testing.T) {
	for _, test := range []struct {
		name     string
		old, new string
		want     bool
	}{
		{
			name: "object key order",
			old:  `{"gatewayAPI":{"class":"eg"},"loadBalancer":{"limit":15}}`,
			new:  `{"loadBalancer":{"limit":15}, "gatewayAPI": {"class":"eg"}}`,
			want: true,
		},
		{
			name: "distinct large integers",
			old:  `{"limit":9007199254740992}`,
			new:  `{"limit":9007199254740993}`,
		},
		{name: "invalid old value", old: `{`, new: `{}`},
		{name: "invalid new value", old: `{}`, new: `{`},
	} {
		t.Run(test.name, func(t *testing.T) {
			oldSpec := &runtime.RawExtension{Raw: []byte(test.old)}
			newSpec := &runtime.RawExtension{Raw: []byte(test.new)}
			if got := projectDefaultTenantSpecsEqual(oldSpec, newSpec); got != test.want {
				t.Errorf("equal = %v, want %v", got, test.want)
			}
		})
	}
}

func projectWatchCluster(name string) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: "project"}},
		Spec:       kubermaticv1.ClusterSpec{KubeLB: &kubermaticv1.KubeLB{Enabled: true}},
		Status:     kubermaticv1.ClusterStatus{NamespaceName: "cluster-" + name},
	}
}

func TestReconcileKubeLBResourcesMaintainsCCMOnTenantError(t *testing.T) {
	conflict := apierrors.NewConflict(schema.GroupResource{Group: "kubelb.k8c.io", Resource: "tenants"}, "active", errors.New("administrator owns requested field"))
	invalid := apierrors.NewBadRequest("strict decoding error: unknown field spec.unknownSetting")
	for _, test := range []struct {
		name       string
		defaults   string
		applyErr   error
		migrateErr error
		wantError  string
	}{
		{name: "apply ownership conflict", defaults: `{}`, applyErr: conflict, wantError: conflict.Error()},
		{name: "unknown Tenant field", defaults: `{"unknownSetting":true}`, applyErr: invalid, wantError: invalid.Error()},
		{name: "invalid defaults object", defaults: `[]`, wantError: "failed to decode project default tenant spec"},
		{name: "migration resource version conflict", defaults: `{}`, migrateErr: conflict, wantError: conflict.Error()},
	} {
		t.Run(test.name, func(t *testing.T) {
			tenant := resourceTestTenant()
			if test.migrateErr != nil {
				tenant.SetLabels(map[string]string{kubelbclusterresources.TenantClusterNameLabelKey: tenant.GetName()})
				tenant.SetManagedFields([]metav1.ManagedFieldsEntry{{
					Manager: "seed-controller-manager", Operation: metav1.ManagedFieldsOperationUpdate,
					APIVersion: kubelbclusterresources.KubelbTenantGVK.GroupVersion().String(), FieldsType: "FieldsV1",
					FieldsV1: metav1.NewFieldsV1(`{"f:metadata":{"f:labels":{"f:kubermatic.k8c.io/cluster-name":{}}}}`),
				}})
			}
			f := newKubeLBResourceFixture(t, tenant, true, interceptor.Funcs{
				Apply: func(context.Context, ctrlruntimeclient.WithWatch, runtime.ApplyConfiguration, ...ctrlruntimeclient.ApplyOption) error {
					if test.applyErr == nil {
						return fmt.Errorf("unexpected Tenant apply")
					}
					return test.applyErr
				},
				Patch: func(context.Context, ctrlruntimeclient.WithWatch, ctrlruntimeclient.Object, ctrlruntimeclient.Patch, ...ctrlruntimeclient.PatchOption) error {
					if test.migrateErr == nil {
						return fmt.Errorf("unexpected Tenant migration")
					}
					return test.migrateErr
				},
			})
			_, err := f.reconcile(t, test.defaults)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("reconciliation error = %v, want %q", err, test.wantError)
			}
			if test.applyErr != nil && !errors.Is(err, test.applyErr) {
				t.Error("Tenant apply error was not retained")
			}
			f.assertCCMImage(t, "quay.io/kubermatic/kubelb-ccm-ee:v1.5.0")
			f.assertUserResources(t)
			f.assertFailedCondition(t)
			unchanged := resourceTestTenant()
			if err := f.management.Get(context.Background(), ctrlruntimeclient.ObjectKeyFromObject(unchanged), unchanged); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(unchanged.Object["spec"], tenant.Object["spec"]) {
				t.Fatal("Tenant spec changed despite the failed defaults update")
			}
		})
	}
}

func TestReconcileKubeLBResourcesBlocksUnusableTenant(t *testing.T) {
	readErr := errors.New("management API unavailable")
	createErr := errors.New("Tenant creation rejected")
	for _, test := range []struct {
		name       string
		absent     bool
		terminates bool
		getErr     error
		wantError  string
	}{
		{name: "failed creation", absent: true, wantError: createErr.Error()},
		{name: "failed existence read", getErr: readErr, wantError: readErr.Error()},
		{name: "terminating Tenant", terminates: true, wantError: "still terminating"},
	} {
		t.Run(test.name, func(t *testing.T) {
			tenant := resourceTestTenant()
			if test.absent {
				tenant = nil
			}
			if test.terminates {
				now := metav1.Now()
				tenant.SetDeletionTimestamp(&now)
				tenant.SetFinalizers([]string{"kubelb.k8c.io/cleanup"})
			}
			f := newKubeLBResourceFixture(t, tenant, true, interceptor.Funcs{
				Get: func(ctx context.Context, c ctrlruntimeclient.WithWatch, key ctrlruntimeclient.ObjectKey, obj ctrlruntimeclient.Object, opts ...ctrlruntimeclient.GetOption) error {
					if test.getErr != nil {
						return test.getErr
					}
					return c.Get(ctx, key, obj, opts...)
				},
				Apply: func(context.Context, ctrlruntimeclient.WithWatch, runtime.ApplyConfiguration, ...ctrlruntimeclient.ApplyOption) error {
					return createErr
				},
			})
			_, err := f.reconcile(t, `{}`)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("reconciliation error = %v, want %q", err, test.wantError)
			}
			f.assertCCMImage(t, "quay.io/kubermatic/kubelb-ccm-ee:v1.4.3")
			roles := &rbacv1.RoleList{}
			if err := f.user.List(context.Background(), roles); err != nil {
				t.Fatal(err)
			}
			if len(roles.Items) != 0 {
				t.Fatal("user resources were reconciled for an unusable Tenant")
			}
			f.assertFailedCondition(t)
		})
	}
}

func TestReconcileKubeLBResourcesWaitsForNewTenantCredentials(t *testing.T) {
	f := newKubeLBResourceFixture(t, nil, false, interceptor.Funcs{
		Apply: func(ctx context.Context, c ctrlruntimeclient.WithWatch, _ runtime.ApplyConfiguration, _ ...ctrlruntimeclient.ApplyOption) error {
			return c.Create(ctx, resourceTestTenant())
		},
	})
	result, err := f.reconcile(t, `{}`)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil || result.RequeueAfter != 15*time.Second {
		t.Fatalf("result = %v, want a retry while manager creates credentials", result)
	}
	if err := f.management.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: f.cluster.Name}, resourceTestTenant()); err != nil {
		t.Fatalf("Tenant must be created before waiting for credentials: %v", err)
	}
	f.assertUserResources(t)
	f.assertCCMImage(t, "quay.io/kubermatic/kubelb-ccm-ee:v1.4.3")
}

func TestReconcileKubeLBResourcesPreservesDownstreamErrors(t *testing.T) {
	tenantErr := errors.New("Tenant defaults rejected")
	downstreamErr := errors.New("resource write failed")
	for _, stage := range []string{"user", "seed"} {
		t.Run(stage, func(t *testing.T) {
			f := newKubeLBResourceFixture(t, resourceTestTenant(), true, interceptor.Funcs{
				Apply: func(context.Context, ctrlruntimeclient.WithWatch, runtime.ApplyConfiguration, ...ctrlruntimeclient.ApplyOption) error {
					return tenantErr
				},
			})
			if stage == "user" {
				f.r.userClusterConnectionProvider = resourceTestUserClients{interceptor.NewClient(f.user, interceptor.Funcs{
					Create: func(context.Context, ctrlruntimeclient.WithWatch, ctrlruntimeclient.Object, ...ctrlruntimeclient.CreateOption) error {
						return downstreamErr
					},
				})}
			} else {
				f.r.Client = interceptor.NewClient(f.seed, interceptor.Funcs{
					Update: func(ctx context.Context, c ctrlruntimeclient.WithWatch, obj ctrlruntimeclient.Object, opts ...ctrlruntimeclient.UpdateOption) error {
						if _, ok := obj.(*appsv1.Deployment); ok {
							return downstreamErr
						}
						return c.Update(ctx, obj, opts...)
					},
				})
			}
			_, err := f.reconcile(t, `{}`)
			if !errors.Is(err, tenantErr) || !errors.Is(err, downstreamErr) {
				t.Fatalf("reconciliation error = %v, want both Tenant and %s errors", err, stage)
			}
			f.assertCCMImage(t, "quay.io/kubermatic/kubelb-ccm-ee:v1.4.3")
			f.assertFailedCondition(t)
		})
	}
}

type kubeLBResourceFixture struct {
	r          *reconciler
	cluster    *kubermaticv1.Cluster
	seed       ctrlruntimeclient.WithWatch
	user       ctrlruntimeclient.WithWatch
	management ctrlruntimeclient.WithWatch
	kubeconfig []byte
}

func newKubeLBResourceFixture(t *testing.T, tenant *unstructured.Unstructured, credentialsReady bool, managementInterceptors interceptor.Funcs) *kubeLBResourceFixture {
	t.Helper()
	cluster := projectWatchCluster("active")
	cluster.UID = "cluster-uid"
	cluster.Status.Address.InternalName = "apiserver.example.test"
	seed := fake.NewClientBuilder().WithObjects(cluster,
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: resources.KubeLBDeploymentName, Namespace: cluster.Status.NamespaceName},
			Spec: appsv1.DeploymentSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: resources.KubeLBDeploymentName, Image: "quay.io/kubermatic/kubelb-ccm-ee:v1.4.3"}},
			}}},
		},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: resources.KubeLBCCMKubeconfigSecretName, Namespace: cluster.Status.NamespaceName}},
	).Build()
	scheme := fake.NewScheme()
	if err := apiextensionsv1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	user := fake.NewClientBuilder().WithScheme(scheme).Build()
	kubeconfig, err := clientcmd.Write(clientcmdapi.Config{Clusters: map[string]*clientcmdapi.Cluster{
		"management": {Server: "https://kubelb.example.test"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	var managementObjects []ctrlruntimeclient.Object
	if tenant != nil {
		managementObjects = append(managementObjects, tenant)
	}
	if credentialsReady {
		managementObjects = append(managementObjects, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: kubeLBCCMKubeconfigSecretName, Namespace: "tenant-" + cluster.Name},
			Data:       map[string][]byte{kubeconfigSecretKey: kubeconfig},
		})
	}
	management := fake.NewClientBuilder().WithObjects(managementObjects...).WithReturnManagedFields().WithInterceptorFuncs(managementInterceptors).Build()
	r := &reconciler{
		Client: seed, log: zap.NewNop().Sugar(),
		userClusterConnectionProvider: resourceTestUserClients{user},
		configGetter: func(context.Context) (*kubermaticv1.KubermaticConfiguration, error) {
			return &kubermaticv1.KubermaticConfiguration{Spec: kubermaticv1.KubermaticConfigurationSpec{
				UserCluster: kubermaticv1.KubermaticUserClusterConfiguration{KubeLB: kubermaticv1.KubeLBConfiguration{ImageTag: "v1.5.0"}},
			}}, nil
		},
	}
	return &kubeLBResourceFixture{r: r, cluster: cluster, seed: seed, user: user, management: management, kubeconfig: kubeconfig}
}

func (f *kubeLBResourceFixture) reconcile(t *testing.T, defaults string) (*reconcile.Result, error) {
	t.Helper()
	return util.ClusterReconcileWrapper(context.Background(), f.r.Client, "", f.cluster, kubermatic.Versions{GitVersion: "test"},
		kubermaticv1.ClusterConditionKubeLBControllerReconcilingSuccess, func() (*reconcile.Result, error) {
			return f.r.reconcileKubeLBResources(context.Background(), f.management, f.kubeconfig, f.cluster, kubermaticv1.Datacenter{}, &runtime.RawExtension{Raw: []byte(defaults)})
		})
}

func (f *kubeLBResourceFixture) assertCCMImage(t *testing.T, image string) {
	t.Helper()
	deployment := &appsv1.Deployment{}
	if err := f.seed.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: resources.KubeLBDeploymentName, Namespace: f.cluster.Status.NamespaceName}, deployment); err != nil {
		t.Fatal(err)
	}
	if len(deployment.Spec.Template.Spec.Containers) != 1 || deployment.Spec.Template.Spec.Containers[0].Image != image {
		t.Errorf("CCM containers = %v, want image %s", deployment.Spec.Template.Spec.Containers, image)
	}
}

func (f *kubeLBResourceFixture) assertUserResources(t *testing.T) {
	t.Helper()
	role := &rbacv1.ClusterRole{}
	if err := f.user.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: "system:kubermatic-kubelb-ccm"}, role); err != nil {
		t.Fatal(err)
	}
	foundWAF := false
	for _, rule := range role.Rules {
		if slices.Contains(rule.Resources, "tenantwafpolicies") && slices.Contains(rule.Verbs, "watch") {
			foundWAF = true
		}
	}
	if !foundWAF {
		t.Error("user RBAC lacks current CCM TenantWAFPolicy permissions")
	}
	for _, name := range []string{kubelbuserclusterresources.SyncSecretCRDName, kubelbuserclusterresources.TenantWAFPolicyCRDName} {
		crd := &apiextensionsv1.CustomResourceDefinition{}
		if err := f.user.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: name}, crd); err != nil {
			t.Fatal(err)
		}
		if crd.Spec.Group != "kubelb.k8c.io" || len(crd.Spec.Versions) == 0 {
			t.Errorf("CRD %s has no KubeLB schema", name)
		}
	}
}

func (f *kubeLBResourceFixture) assertFailedCondition(t *testing.T) {
	t.Helper()
	cluster := &kubermaticv1.Cluster{}
	if err := f.seed.Get(context.Background(), ctrlruntimeclient.ObjectKeyFromObject(f.cluster), cluster); err != nil {
		t.Fatal(err)
	}
	if !cluster.Status.HasConditionValue(kubermaticv1.ClusterConditionKubeLBControllerReconcilingSuccess, corev1.ConditionFalse) {
		t.Error("failed Tenant reconciliation must retain a failed controller condition")
	}
}

func resourceTestTenant() *unstructured.Unstructured {
	tenant := &unstructured.Unstructured{Object: map[string]any{"spec": map[string]any{"allowedDomains": []any{"example.test"}}}}
	tenant.SetGroupVersionKind(kubelbclusterresources.KubelbTenantGVK)
	tenant.SetName("active")
	return tenant
}

type resourceTestUserClients struct{ client ctrlruntimeclient.Client }

func (p resourceTestUserClients) GetClient(context.Context, *kubermaticv1.Cluster, ...clusterclient.ConfigOption) (ctrlruntimeclient.Client, error) {
	return p.client, nil
}
