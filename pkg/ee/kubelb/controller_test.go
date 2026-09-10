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
	"reflect"
	"slices"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
