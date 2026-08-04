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

package admingroupcontroller

import (
	"context"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// TestEventFiltersIgnoreLabels proves the controller's predicates gate on
// fields only, never on labels: a worker-name label neither triggers nor
// blocks processing. This is why running the controller locally requires no
// worker-name labelling of Users or the KubermaticSetting.
func TestEventFiltersIgnoreLabels(t *testing.T) {
	userFilter := withUserEventFilter()
	settingsFilter := withSettingsEventFilter()

	base := testUser("u", "u@acme.com", []string{"dev"}, false, "")

	// Label-only change: no relevant field moved -> no requeue.
	labeled := base.DeepCopy()
	labeled.Labels = map[string]string{"worker-name": "somelocaltest"}
	if userFilter.Update(event.UpdateEvent{ObjectOld: base, ObjectNew: labeled}) {
		t.Error("user filter requeued on a label-only change")
	}

	// Relevant field change on a labelled object: the label must not block it.
	joined := labeled.DeepCopy()
	joined.Spec.Groups = append(joined.Spec.Groups, "admins")
	if !userFilter.Update(event.UpdateEvent{ObjectOld: labeled, ObjectNew: joined}) {
		t.Error("user filter ignored a groups change because a worker-name label was present")
	}

	// Losing the global viewer flag makes a matching user eligible for promotion,
	// so it has to requeue on its own.
	viewer := globalViewer(joined.DeepCopy())
	if !userFilter.Update(event.UpdateEvent{ObjectOld: viewer, ObjectNew: joined}) {
		t.Error("user filter ignored a global viewer change")
	}

	oldSettings := genSettings([]string{"admins"})
	labeledSettings := oldSettings.DeepCopy()
	labeledSettings.Labels = map[string]string{"worker-name": "somelocaltest"}
	if settingsFilter.Update(event.UpdateEvent{ObjectOld: oldSettings, ObjectNew: labeledSettings}) {
		t.Error("settings filter requeued on a label-only change")
	}

	changedSettings := labeledSettings.DeepCopy()
	changedSettings.Spec.AdminGroups = []string{"admins", "platform"}
	if !settingsFilter.Update(event.UpdateEvent{ObjectOld: labeledSettings, ObjectNew: changedSettings}) {
		t.Error("settings filter ignored an adminGroups change because a worker-name label was present")
	}
}

// TestTwoControllerInstancesConverge simulates a locally-run controller next to
// a deployed one carrying the same logic: two reconciler instances over the
// same objects. The interleaved reconciles must converge — the first writer
// promotes, every later pass (from either instance) is a no-op, proven by the
// resourceVersion not moving.
func TestTwoControllerInstancesConverge(t *testing.T) {
	const userName = "user-shared"
	ctx := context.Background()

	client := fake.NewClientBuilder().WithObjects(
		testUser(userName, "bob@acme.com", []string{"admins"}, false, ""),
		genSettings([]string{"admins"}),
	).Build()

	newInstance := func() *reconciler {
		return &reconciler{
			log:             kubermaticlog.Logger,
			recorder:        &events.FakeRecorder{},
			masterClient:    client,
			masterAPIReader: client,
		}
	}
	local, deployed := newInstance(), newInstance()
	request := reconcile.Request{NamespacedName: types.NamespacedName{Name: userName}}

	if _, err := local.Reconcile(ctx, request); err != nil {
		t.Fatalf("first reconcile: %v", err)
	}

	promoted := &kubermaticv1.User{}
	if err := client.Get(ctx, request.NamespacedName, promoted); err != nil {
		t.Fatalf("get user: %v", err)
	}
	if !promoted.Spec.IsAdmin || promoted.Annotations[annotationKey] != "admins" {
		t.Fatalf("expected promotion by first instance, got admin=%v annotations=%v", promoted.Spec.IsAdmin, promoted.Annotations)
	}
	settledRV := promoted.ResourceVersion

	// Interleave the two instances; none of these passes may write anything.
	for i, r := range []*reconciler{deployed, local, deployed} {
		if _, err := r.Reconcile(ctx, request); err != nil {
			t.Fatalf("follow-up reconcile %d: %v", i, err)
		}
	}

	after := &kubermaticv1.User{}
	if err := client.Get(ctx, request.NamespacedName, after); err != nil {
		t.Fatalf("get user: %v", err)
	}
	if after.ResourceVersion != settledRV {
		t.Errorf("a second instance modified the user (resourceVersion %s -> %s); reconcile is not convergent", settledRV, after.ResourceVersion)
	}
	if !after.Spec.IsAdmin || after.Annotations[annotationKey] != "admins" {
		t.Errorf("state drifted after interleaved reconciles: admin=%v annotations=%v", after.Spec.IsAdmin, after.Annotations)
	}
}
