/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package vsphere

import (
	"context"
	"testing"
	"time"

	"github.com/vmware/govmomi/find"
	_ "github.com/vmware/govmomi/vapi/simulator" // registers the REST endpoints (tagging) in vcsim
	vapitags "github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/vim25/types"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

type tagTestEnv struct {
	t          *testing.T
	ctx        context.Context
	rest       *RESTSession
	manager    *vapitags.Manager
	categoryID string
	// target is an object that tags can be attached to
	target types.ManagedObjectReference
}

func newTagTestEnv(t *testing.T) *tagTestEnv {
	sim := vSphereSimulator{t: t}
	sim.setUp()
	t.Cleanup(sim.tearDown)

	dc := &kubermaticv1.DatacenterSpecVSphere{}
	sim.fillClientInfo(dc)

	ctx := context.Background()
	rest, err := newRESTSession(ctx, dc, sim.username(), sim.password(), nil)
	if err != nil {
		t.Fatalf("failed to create REST session: %v", err)
	}

	session, err := newSession(ctx, dc, sim.username(), sim.password(), nil)
	if err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	datacenter, err := find.NewFinder(session.Client.Client, true).Datacenter(ctx, dc.Datacenter)
	if err != nil {
		t.Fatalf("failed to find datacenter: %v", err)
	}

	manager := vapitags.NewManager(rest.Client)
	categoryID, err := manager.CreateCategory(ctx, &vapitags.Category{
		Name:            "kkp-test",
		Cardinality:     "MULTIPLE",
		AssociableTypes: []string{"Datacenter"},
	})
	if err != nil {
		t.Fatalf("failed to create category: %v", err)
	}

	return &tagTestEnv{t: t, ctx: ctx, rest: rest, manager: manager, categoryID: categoryID, target: datacenter.Reference()}
}

func (e *tagTestEnv) cluster(name string, tags ...string) *kubermaticv1.Cluster {
	return &kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: kubermaticv1.CloudSpec{
				VSphere: &kubermaticv1.VSphereCloudSpec{
					Tags: &kubermaticv1.VSphereTag{CategoryID: e.categoryID, Tags: tags},
				},
			},
		},
	}
}

// reconcile mimics reconcileTags followed by the cloud controller persisting the status.
func (e *tagTestEnv) reconcile(cluster *kubermaticv1.Cluster) {
	e.t.Helper()

	managed, err := syncCreatedClusterTags(e.ctx, e.rest, cluster)
	if err != nil {
		e.t.Fatalf("syncCreatedClusterTags failed: %v", err)
	}

	managed, err = syncDeletedClusterTags(e.ctx, e.rest, cluster, managed)
	if err != nil {
		e.t.Fatalf("syncDeletedClusterTags failed: %v", err)
	}

	setManagedTags(cluster, managed)
}

func (e *tagTestEnv) existingTags() sets.Set[string] {
	e.t.Helper()

	tags, err := e.manager.GetTagsForCategory(e.ctx, e.categoryID)
	if err != nil {
		e.t.Fatalf("failed to list tags: %v", err)
	}

	names := sets.New[string]()
	for _, tag := range tags {
		names.Insert(tag.Name)
	}

	return names
}

func (e *tagTestEnv) assertTags(want ...string) {
	e.t.Helper()

	if got := e.existingTags(); !got.Equal(sets.New(want...)) {
		e.t.Errorf("expected tags %v in category, got %v", sets.List(sets.New(want...)), sets.List(got))
	}
}

// Regression test for https://github.com/kubermatic/kubermatic/issues/13163.
func TestReconcileTagsDoesNotDeleteTagsOfOtherClusters(t *testing.T) {
	env := newTagTestEnv(t)

	first := env.cluster("first", "A", "B")
	second := env.cluster("second", "C", "D")

	env.reconcile(first)
	env.reconcile(second)
	env.assertTags("A", "B", "C", "D")

	// reconciling again must not change anything either
	env.reconcile(first)
	env.reconcile(second)
	env.assertTags("A", "B", "C", "D")
}

func TestReconcileTagsDoesNotDeleteUnmanagedTags(t *testing.T) {
	env := newTagTestEnv(t)

	if _, err := createTag(env.ctx, env.rest, env.categoryID, "unmanaged"); err != nil {
		t.Fatalf("failed to create tag: %v", err)
	}

	cluster := env.cluster("cluster", "A", "B")
	env.reconcile(cluster)
	env.assertTags("A", "B", "unmanaged")

	// removing a managed tag from the spec deletes exactly that tag
	cluster.Spec.Cloud.VSphere.Tags.Tags = []string{"A"}
	env.reconcile(cluster)
	env.assertTags("A", "unmanaged")

	if got := getManagedTags(cluster); got.Len() != 1 {
		t.Errorf("expected exactly one managed tag in the status, got %v", sets.List(got))
	}
}

func TestReconcileTagsKeepsTagsAttachedToObjects(t *testing.T) {
	env := newTagTestEnv(t)

	cluster := env.cluster("cluster", "A", "B")
	env.reconcile(cluster)

	tags, err := env.manager.GetTagsForCategory(env.ctx, env.categoryID)
	if err != nil {
		t.Fatal(err)
	}
	var bID string
	for _, tag := range tags {
		if tag.Name == "B" {
			bID = tag.ID
		}
	}
	if err := env.manager.AttachTag(env.ctx, bID, env.target); err != nil {
		t.Fatalf("failed to attach tag: %v", err)
	}

	cluster.Spec.Cloud.VSphere.Tags.Tags = []string{"A"}
	env.reconcile(cluster)
	env.assertTags("A", "B")

	// B stays managed so that it is retried once it is detached
	if !getManagedTags(cluster).Has(bID) {
		t.Error("expected the attached tag to stay managed")
	}

	if err := env.manager.DetachTag(env.ctx, bID, env.target); err != nil {
		t.Fatalf("failed to detach tag: %v", err)
	}
	env.reconcile(cluster)
	env.assertTags("A")
}

func TestReconcileTagsRecoversFromMissingStatus(t *testing.T) {
	env := newTagTestEnv(t)

	cluster := env.cluster("cluster", "A", "B")
	env.reconcile(cluster)

	// simulate a lost status update: the tags exist in vSphere but are not recorded anywhere
	cluster.Status.CloudProvider = nil
	env.reconcile(cluster)
	if got := getManagedTags(cluster); got.Len() != 2 {
		t.Errorf("expected the existing spec tags to be adopted, got %v", sets.List(got))
	}

	// a tag that was deleted manually in vSphere is dropped from the status
	tags, _ := env.manager.GetTagsForCategory(env.ctx, env.categoryID)
	for i := range tags {
		if tags[i].Name == "A" {
			if err := env.manager.DeleteTag(env.ctx, &tags[i]); err != nil {
				t.Fatal(err)
			}
		}
	}
	cluster.Spec.Cloud.VSphere.Tags.Tags = []string{"B"}
	env.reconcile(cluster)
	env.assertTags("B")
	if got := getManagedTags(cluster); got.Len() != 1 {
		t.Errorf("expected exactly one managed tag, got %v", sets.List(got))
	}
}

func TestCleanupClusterTags(t *testing.T) {
	t.Run("only managed tags of the deleted cluster are removed", func(t *testing.T) {
		env := newTagTestEnv(t)

		first := env.cluster("first", "A", "B")
		second := env.cluster("second", "C", "D")
		env.reconcile(first)
		env.reconcile(second)

		now := metav1.NewTime(time.Now())
		first.DeletionTimestamp = &now
		if err := cleanupClusterTags(env.ctx, env.rest, first); err != nil {
			t.Fatalf("cleanupClusterTags failed: %v", err)
		}

		env.assertTags("C", "D")
	})

	t.Run("clusters without recorded status fall back to their spec tags", func(t *testing.T) {
		env := newTagTestEnv(t)

		legacy := env.cluster("legacy", "A", "B")
		other := env.cluster("other", "C")
		env.reconcile(legacy)
		env.reconcile(other)
		legacy.Status.CloudProvider = nil

		now := metav1.NewTime(time.Now())
		legacy.DeletionTimestamp = &now
		if err := cleanupClusterTags(env.ctx, env.rest, legacy); err != nil {
			t.Fatalf("cleanupClusterTags failed: %v", err)
		}

		env.assertTags("C")
	})
}
