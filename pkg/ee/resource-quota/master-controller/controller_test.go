//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

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

package mastercontroller

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/machine/accelerator"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"
	"k8c.io/kubermatic/v2/pkg/test/generator"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const rqName = "resourceQuota"

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name          string
		requestName   string
		expectedUsage kubermaticv1.ResourceDetails
		masterClient  ctrlruntimeclient.Client
		seedClients   map[string]ctrlruntimeclient.Client
	}{
		{
			name:          "scenario 1: calculate rq global usage",
			requestName:   rqName,
			expectedUsage: *genResourceDetails("7", "7G", "18G"),
			masterClient: fake.
				NewClientBuilder().
				WithObjects(genResourceQuota(rqName, kubermaticv1.ResourceDetails{}), generator.GenTestSeed()).
				Build(),
			seedClients: map[string]ctrlruntimeclient.Client{
				"first": fake.
					NewClientBuilder().
					WithObjects(func() *kubermaticv1.ResourceQuota {
						quota := genResourceQuota(rqName, *genResourceDetails("2", "5G", "10G"))
						quota.Status.LocalUsage.Accelerators = acceleratorQuota("99")
						return quota
					}()).
					Build(),
				"second": fake.
					NewClientBuilder().
					WithObjects(genResourceQuota(rqName, *genResourceDetails("5", "2G", "8G"))).
					Build(),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := &reconciler{
				log:          kubermaticlog.Logger,
				recorder:     &events.FakeRecorder{},
				masterClient: tc.masterClient,
				seedClients:  tc.seedClients,
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			rq := &kubermaticv1.ResourceQuota{}
			err := tc.masterClient.Get(ctx, request.NamespacedName, rq)

			if err != nil {
				t.Fatalf("failed to get resource quota: %v", err)
			}

			if !diff.SemanticallyEqual(tc.expectedUsage, rq.Status.GlobalUsage) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedUsage, rq.Status.GlobalUsage))
			}
		})
	}
}

func TestReconcileAcceleratorAccounting(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	accelerators := acceleratorQuota("4")
	digest := kubermaticv1.AcceleratorQuotaDigestFor(accelerators)
	revision := kubermaticv1.AcceleratorAccountingRevision("revision-1")

	masterResourceQuota := genResourceQuota(rqName, kubermaticv1.ResourceDetails{})
	activateResourceQuota(masterResourceQuota)
	masterResourceQuota.Spec.Quota.Accelerators = accelerators
	masterResourceQuota.Status.GlobalAcceleratorAccounting = &kubermaticv1.ResourceQuotaGlobalAcceleratorAccountingStatus{
		ObservedAccountingRevision: revision,
		ObservedQuotaDigest:        digest,
	}

	first := genResourceQuota(rqName, *genResourceDetails("2", "5G", "10G"))
	first.Status.LocalUsage.Accelerators = acceleratorQuota("1")
	first.Status.LocalAcceleratorAccounting = readyLocalAccounting(revision, digest, now.Add(-time.Minute))
	second := genResourceQuota(rqName, *genResourceDetails("5", "2G", "8G"))
	second.Status.LocalUsage.Accelerators = acceleratorQuota("2")
	second.Status.LocalAcceleratorAccounting = readyLocalAccounting(revision, digest, now.Add(-2*time.Minute))

	masterClient := fake.NewClientBuilder().WithObjects(masterResourceQuota).Build()
	r := &reconciler{
		log:          kubermaticlog.Logger,
		recorder:     &events.FakeRecorder{},
		masterClient: masterClient,
		seedClients: map[string]ctrlruntimeclient.Client{
			"first":  fake.NewClientBuilder().WithObjects(first).Build(),
			"second": fake.NewClientBuilder().WithObjects(second).Build(),
		},
		now: func() time.Time { return now },
	}

	request := reconcile.Request{NamespacedName: types.NamespacedName{Name: rqName}}
	result, err := r.Reconcile(ctx, request)
	if err != nil {
		t.Fatalf("reconciling failed: %v", err)
	}
	if result.RequeueAfter != 3*time.Minute {
		t.Fatalf("expected reconciliation at oldest heartbeat expiry, got %v", result.RequeueAfter)
	}

	got := &kubermaticv1.ResourceQuota{}
	if err := masterClient.Get(ctx, request.NamespacedName, got); err != nil {
		t.Fatalf("getting ResourceQuota: %v", err)
	}
	expectedUsage := genResourceDetails("7", "7G", "18G")
	expectedUsage.Accelerators = acceleratorQuota("3")
	if !diff.SemanticallyEqual(*expectedUsage, got.Status.GlobalUsage) {
		t.Fatalf("global usage differs:\n%v", diff.ObjectDiff(*expectedUsage, got.Status.GlobalUsage))
	}
	expectedAccounting := &kubermaticv1.ResourceQuotaGlobalAcceleratorAccountingStatus{
		ActivationPhase:            kubermaticv1.AcceleratorAccountingPhaseReady,
		ObservedAccountingRevision: revision,
		ObservedQuotaDigest:        digest,
		ObservedAt:                 metav1.NewTime(now.Add(-2 * time.Minute)),
		Ready:                      true,
	}
	if !diff.SemanticallyEqual(expectedAccounting, got.Status.GlobalAcceleratorAccounting) {
		t.Fatalf("global accounting differs:\n%v", diff.ObjectDiff(expectedAccounting, got.Status.GlobalAcceleratorAccounting))
	}
}

func TestAcceleratorAccountingRevisionTransitions(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	revisionNumber := 0
	r := &reconciler{
		seedClients: map[string]ctrlruntimeclient.Client{},
		now:         func() time.Time { return now },
		newRevision: func() kubermaticv1.AcceleratorAccountingRevision {
			revisionNumber++
			return kubermaticv1.AcceleratorAccountingRevision(fmt.Sprintf("revision-%d", revisionNumber))
		},
	}
	resourceQuota := genResourceQuota(rqName, kubermaticv1.ResourceDetails{})
	activateResourceQuota(resourceQuota)
	resourceQuota.Spec.Quota.Accelerators = acceleratorQuota("4")

	first, _ := r.globalAcceleratorAccounting(resourceQuota, nil, nil)
	resourceQuota.Status.GlobalAcceleratorAccounting = first
	if first.ObservedAccountingRevision != "revision-1" {
		t.Fatalf("unexpected initial revision %q", first.ObservedAccountingRevision)
	}

	resourceQuota.Spec.Quota.CPU = quantityPointer("100")
	scalarOnly, _ := r.globalAcceleratorAccounting(resourceQuota, nil, nil)
	if scalarOnly.ObservedAccountingRevision != first.ObservedAccountingRevision || revisionNumber != 1 {
		t.Fatalf("scalar-only change issued a new revision: %#v", scalarOnly)
	}

	resourceQuota.Spec.Quota.Accelerators = acceleratorQuota("8")
	second, _ := r.globalAcceleratorAccounting(resourceQuota, nil, nil)
	resourceQuota.Status.GlobalAcceleratorAccounting = second
	if second.ObservedAccountingRevision != "revision-2" {
		t.Fatalf("unexpected second revision %q", second.ObservedAccountingRevision)
	}

	resourceQuota.Spec.Quota.Accelerators = acceleratorQuota("4")
	third, _ := r.globalAcceleratorAccounting(resourceQuota, nil, nil)
	if third.ObservedAccountingRevision != "revision-3" {
		t.Fatalf("A -> B -> A reused a revision: %q", third.ObservedAccountingRevision)
	}
}

func TestReconcileBlocksOnStaleAndMissingSeedReports(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	digest := kubermaticv1.AcceleratorQuotaDigestFor(nil)
	revision := kubermaticv1.AcceleratorAccountingRevision("revision-1")
	masterResourceQuota := genResourceQuota(rqName, kubermaticv1.ResourceDetails{})
	activateResourceQuota(masterResourceQuota)
	masterResourceQuota.Status.GlobalAcceleratorAccounting = &kubermaticv1.ResourceQuotaGlobalAcceleratorAccountingStatus{
		ObservedAccountingRevision: revision,
		ObservedQuotaDigest:        digest,
	}
	stale := genResourceQuota(rqName, kubermaticv1.ResourceDetails{})
	stale.Status.LocalAcceleratorAccounting = readyLocalAccounting(revision, digest, now.Add(-resources.AcceleratorAccountingHeartbeatTimeout-time.Second))

	masterClient := fake.NewClientBuilder().WithObjects(masterResourceQuota).Build()
	r := &reconciler{
		log:          kubermaticlog.Logger,
		recorder:     &events.FakeRecorder{},
		masterClient: masterClient,
		seedClients: map[string]ctrlruntimeclient.Client{
			"missing": fake.NewClientBuilder().Build(),
			"stale":   fake.NewClientBuilder().WithObjects(stale).Build(),
		},
		now: func() time.Time { return now },
	}

	request := reconcile.Request{NamespacedName: types.NamespacedName{Name: rqName}}
	result, err := r.Reconcile(ctx, request)
	if err != nil {
		t.Fatalf("reconciling failed: %v", err)
	}
	if result.RequeueAfter != resources.AcceleratorAccountingHeartbeatInterval {
		t.Fatalf("unexpected requeue: %v", result.RequeueAfter)
	}
	got := &kubermaticv1.ResourceQuota{}
	if err := masterClient.Get(ctx, request.NamespacedName, got); err != nil {
		t.Fatalf("getting ResourceQuota: %v", err)
	}
	status := got.Status.GlobalAcceleratorAccounting
	if status == nil || status.Ready || status.ActivationPhase != kubermaticv1.AcceleratorAccountingPhaseBlocked {
		t.Fatalf("expected blocked global accounting, got %#v", status)
	}
	if len(status.Blockers) != 2 ||
		status.Blockers[0].Type != kubermaticv1.AcceleratorAccountingBlockerTypeUnreachableSeed ||
		status.Blockers[1].Type != kubermaticv1.AcceleratorAccountingBlockerTypeStaleHeartbeat {
		t.Fatalf("unexpected blockers: %#v", status.Blockers)
	}
}

func TestGlobalAccountingKeepsNewClusterStateActivating(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	digest := kubermaticv1.AcceleratorQuotaDigestFor(nil)
	revision := kubermaticv1.AcceleratorAccountingRevision("revision-1")
	masterResourceQuota := genResourceQuota(rqName, kubermaticv1.ResourceDetails{})
	activateResourceQuota(masterResourceQuota)
	masterResourceQuota.Status.GlobalAcceleratorAccounting = &kubermaticv1.ResourceQuotaGlobalAcceleratorAccountingStatus{
		ObservedAccountingRevision: revision,
		ObservedQuotaDigest:        digest,
	}
	seedResourceQuota := genResourceQuota(rqName, kubermaticv1.ResourceDetails{})
	seedResourceQuota.Status.LocalAcceleratorAccounting = &kubermaticv1.ResourceQuotaLocalAcceleratorAccountingStatus{
		ObservedAccountingRevision: revision,
		ObservedQuotaDigest:        digest,
		Blockers: []kubermaticv1.AcceleratorAccountingBlocker{{
			Type:        kubermaticv1.AcceleratorAccountingBlockerTypeNewCluster,
			ClusterName: "cluster-a",
		}},
	}

	r := &reconciler{
		seedClients: map[string]ctrlruntimeclient.Client{"seed-a": nil},
		now:         func() time.Time { return now },
	}
	status, requeueAfter := r.globalAcceleratorAccounting(
		masterResourceQuota,
		map[string]*kubermaticv1.ResourceQuota{"seed-a": seedResourceQuota},
		nil,
	)

	if status.Ready || status.ActivationPhase != kubermaticv1.AcceleratorAccountingPhaseActivating {
		t.Fatalf("expected activating global accounting, got %#v", status)
	}
	if len(status.Blockers) != 1 || status.Blockers[0].Type != kubermaticv1.AcceleratorAccountingBlockerTypeNewCluster {
		t.Fatalf("expected only the NewCluster blocker, got %#v", status.Blockers)
	}
	if requeueAfter != resources.AcceleratorAccountingHeartbeatInterval {
		t.Fatalf("RequeueAfter = %v, want %v", requeueAfter, resources.AcceleratorAccountingHeartbeatInterval)
	}
}

func TestReconcileSeedReadErrorPreservesGlobalUsage(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	digest := kubermaticv1.AcceleratorQuotaDigestFor(nil)
	masterResourceQuota := genResourceQuota(rqName, kubermaticv1.ResourceDetails{})
	activateResourceQuota(masterResourceQuota)
	masterResourceQuota.Status.GlobalUsage = *genResourceDetails("9", "11G", "13G")
	masterResourceQuota.Status.GlobalUsage.Accelerators = acceleratorQuota("2")
	masterResourceQuota.Status.GlobalAcceleratorAccounting = &kubermaticv1.ResourceQuotaGlobalAcceleratorAccountingStatus{
		ObservedAccountingRevision: "revision-1",
		ObservedQuotaDigest:        digest,
	}
	expectedUsage := masterResourceQuota.Status.GlobalUsage.DeepCopy()

	masterClient := fake.NewClientBuilder().WithObjects(masterResourceQuota).Build()
	unreachableSeedClient := fake.NewClientBuilder().WithInterceptorFuncs(interceptor.Funcs{
		Get: func(context.Context, ctrlruntimeclient.WithWatch, ctrlruntimeclient.ObjectKey, ctrlruntimeclient.Object, ...ctrlruntimeclient.GetOption) error {
			return errors.New("seed unavailable")
		},
	}).Build()
	r := &reconciler{
		log:          kubermaticlog.Logger,
		recorder:     &events.FakeRecorder{},
		masterClient: masterClient,
		seedClients:  map[string]ctrlruntimeclient.Client{"unreachable": unreachableSeedClient},
		now:          func() time.Time { return now },
	}

	request := reconcile.Request{NamespacedName: types.NamespacedName{Name: rqName}}
	if _, err := r.Reconcile(ctx, request); err == nil {
		t.Fatal("expected Seed read error")
	}
	got := &kubermaticv1.ResourceQuota{}
	if err := masterClient.Get(ctx, request.NamespacedName, got); err != nil {
		t.Fatalf("getting ResourceQuota: %v", err)
	}
	if !diff.SemanticallyEqual(*expectedUsage, got.Status.GlobalUsage) {
		t.Fatalf("Seed read error changed the last complete global usage:\n%v", diff.ObjectDiff(*expectedUsage, got.Status.GlobalUsage))
	}
	status := got.Status.GlobalAcceleratorAccounting
	if status == nil || len(status.Blockers) != 1 || status.Blockers[0].Type != kubermaticv1.AcceleratorAccountingBlockerTypeUnreachableSeed {
		t.Fatalf("expected unreachable-Seed blocker, got %#v", status)
	}
}

func genResourceQuota(name string, localUsage kubermaticv1.ResourceDetails) *kubermaticv1.ResourceQuota {
	rq := &kubermaticv1.ResourceQuota{}
	rq.Name = name
	rq.Spec = kubermaticv1.ResourceQuotaSpec{
		Subject: kubermaticv1.Subject{
			Name: "project1",
			Kind: "project",
		},
	}

	rq.Status.LocalUsage = localUsage
	return rq
}

func genResourceDetails(cpu, mem, storage string) *kubermaticv1.ResourceDetails {
	return kubermaticv1.NewResourceDetails(resource.MustParse(cpu), resource.MustParse(mem), resource.MustParse(storage))
}

func acceleratorQuota(count string) []kubermaticv1.AcceleratorQuota {
	return []kubermaticv1.AcceleratorQuota{{
		Provider: accelerator.ProviderKubeVirt,
		Resources: map[corev1.ResourceName]resource.Quantity{
			corev1.ResourceName("nvidia.com/GH100_H200_NVL"): resource.MustParse(count),
		},
	}}
}

func activateResourceQuota(resourceQuota *kubermaticv1.ResourceQuota) {
	resourceQuota.Annotations = map[string]string{
		resources.AcceleratorAccountingEnabledAnnotation: resources.AcceleratorAccountingEnabledAnnotationValue,
	}
}

func readyLocalAccounting(revision kubermaticv1.AcceleratorAccountingRevision, digest kubermaticv1.AcceleratorQuotaDigest, observedAt time.Time) *kubermaticv1.ResourceQuotaLocalAcceleratorAccountingStatus {
	return &kubermaticv1.ResourceQuotaLocalAcceleratorAccountingStatus{
		ObservedAccountingRevision: revision,
		ObservedQuotaDigest:        digest,
		ObservedAt:                 metav1.NewTime(observedAt),
		Ready:                      true,
	}
}

func quantityPointer(value string) *resource.Quantity {
	quantity := resource.MustParse(value)
	return &quantity
}
