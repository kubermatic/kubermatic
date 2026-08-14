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

package seedcontroller

import (
	"context"
	"testing"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/machine/accelerator"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/test/diff"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/events"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const rqName = "resourceQuota"
const projectID = "project1"

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name          string
		requestName   string
		resourceQuota *kubermaticv1.ResourceQuota
		seedClient    ctrlruntimeclient.Client
		expectedUsage kubermaticv1.ResourceDetails
	}{
		{
			name:          "scenario 1: calculate rq local usage",
			requestName:   rqName,
			resourceQuota: genResourceQuota(rqName),
			seedClient: fake.
				NewClientBuilder().
				WithObjects(genResourceQuota(rqName),
					genCluster("c1", projectID, "2", "5G", "10G"),
					genCluster("c2", projectID, "5", "2G", "8G"),
					genCluster("notSameProjectCluster", "impostor", "3", "3G", "3G")).
				Build(),
			expectedUsage: *genResourceDetails("7", "7G", "18G"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			r := &reconciler{
				log:        kubermaticlog.Logger,
				recorder:   &events.FakeRecorder{},
				seedClient: tc.seedClient,
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.requestName}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			rq := &kubermaticv1.ResourceQuota{}
			err := tc.seedClient.Get(ctx, request.NamespacedName, rq)

			if err != nil {
				t.Fatalf("failed to get resource quota: %v", err)
			}

			if !diff.SemanticallyEqual(tc.expectedUsage, rq.Status.LocalUsage) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedUsage, rq.Status.LocalUsage))
			}
		})
	}
}

func TestReconcileAcceleratorAccounting(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	digest := kubermaticv1.AcceleratorQuotaDigestFor(nil)
	revision := kubermaticv1.AcceleratorAccountingRevision("revision-1")

	resourceQuota := genResourceQuota(rqName)
	resourceQuota.Annotations = map[string]string{
		resources.AcceleratorAccountingEnabledAnnotation: resources.AcceleratorAccountingEnabledAnnotationValue,
	}
	resourceQuota.Status.GlobalAcceleratorAccounting = &kubermaticv1.ResourceQuotaGlobalAcceleratorAccountingStatus{
		ObservedAccountingRevision: revision,
		ObservedQuotaDigest:        digest,
	}

	first := genCluster("c1", projectID, "2", "5G", "10G")
	first.Spec.Cloud.Kubevirt = &kubermaticv1.KubevirtCloudSpec{}
	first.Status.ResourceUsage.Accelerators = acceleratorUsage("1")
	first.Status.AcceleratorAccounting = readyClusterAccounting(revision, digest, now.Add(-time.Minute))
	second := genCluster("c2", projectID, "5", "2G", "8G")
	second.Spec.Cloud.Kubevirt = &kubermaticv1.KubevirtCloudSpec{}
	second.Status.ResourceUsage.Accelerators = acceleratorUsage("2")
	second.Status.AcceleratorAccounting = readyClusterAccounting(revision, digest, now.Add(-2*time.Minute))
	nonKubeVirt := genCluster("non-kubevirt", projectID, "1", "1G", "1G")
	nonKubeVirt.Status.ResourceUsage.Accelerators = acceleratorUsage("99")

	seedClient := fake.NewClientBuilder().WithObjects(resourceQuota, first, second, nonKubeVirt).Build()
	r := &reconciler{
		log:               kubermaticlog.Logger,
		recorder:          &events.FakeRecorder{},
		seedClient:        seedClient,
		controllerVersion: "v2.29.0",
		now:               func() time.Time { return now },
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
	if err := seedClient.Get(ctx, request.NamespacedName, got); err != nil {
		t.Fatalf("getting ResourceQuota: %v", err)
	}
	expectedUsage := genResourceDetails("8", "8G", "19G")
	expectedUsage.Accelerators = acceleratorUsage("3")
	if !diff.SemanticallyEqual(*expectedUsage, got.Status.LocalUsage) {
		t.Fatalf("local usage differs:\n%v", diff.ObjectDiff(*expectedUsage, got.Status.LocalUsage))
	}
	expectedAccounting := &kubermaticv1.ResourceQuotaLocalAcceleratorAccountingStatus{
		ObservedAccountingRevision: revision,
		ObservedQuotaDigest:        digest,
		ObservedAt:                 metav1.NewTime(now.Add(-2 * time.Minute)),
		Ready:                      true,
	}
	if !diff.SemanticallyEqual(expectedAccounting, got.Status.LocalAcceleratorAccounting) {
		t.Fatalf("local accounting differs:\n%v", diff.ObjectDiff(expectedAccounting, got.Status.LocalAcceleratorAccounting))
	}
}

func TestLocalAcceleratorAccountingReadiness(t *testing.T) {
	now := time.Date(2026, 8, 14, 12, 0, 0, 0, time.UTC)
	digest := kubermaticv1.AcceleratorQuotaDigestFor(nil)
	revision := kubermaticv1.AcceleratorAccountingRevision("revision-1")
	resourceQuota := genResourceQuota(rqName)
	resourceQuota.Annotations = map[string]string{
		resources.AcceleratorAccountingEnabledAnnotation: resources.AcceleratorAccountingEnabledAnnotationValue,
	}
	resourceQuota.Status.GlobalAcceleratorAccounting = &kubermaticv1.ResourceQuotaGlobalAcceleratorAccountingStatus{
		ObservedAccountingRevision: revision,
		ObservedQuotaDigest:        digest,
	}
	r := &reconciler{
		controllerVersion: "v2.29.0",
		now:               func() time.Time { return now },
	}

	t.Run("Seed without relevant clusters publishes an explicit ready heartbeat", func(t *testing.T) {
		status, requeueAfter := r.localAcceleratorAccounting(resourceQuota, nil)
		if !status.Ready || !status.ObservedAt.Equal(&metav1.Time{Time: now}) {
			t.Fatalf("expected an empty ready attestation at %v, got %#v", now, status)
		}
		if requeueAfter != resources.AcceleratorAccountingHeartbeatInterval {
			t.Fatalf("unexpected requeue: %v", requeueAfter)
		}
	})

	t.Run("stale cluster heartbeat blocks readiness", func(t *testing.T) {
		cluster := genCluster("stale", projectID, "0", "0", "0")
		cluster.Spec.Cloud.Kubevirt = &kubermaticv1.KubevirtCloudSpec{}
		cluster.Status.AcceleratorAccounting = readyClusterAccounting(revision, digest, now.Add(-resources.AcceleratorAccountingHeartbeatTimeout-time.Second))

		status, requeueAfter := r.localAcceleratorAccounting(resourceQuota, []kubermaticv1.Cluster{*cluster})
		if status.Ready {
			t.Fatal("expected stale accounting to be not ready")
		}
		if len(status.Blockers) != 1 || status.Blockers[0].Type != kubermaticv1.AcceleratorAccountingBlockerTypeStaleHeartbeat {
			t.Fatalf("expected stale-heartbeat blocker, got %#v", status.Blockers)
		}
		if requeueAfter != resources.AcceleratorAccountingHeartbeatInterval {
			t.Fatalf("unexpected requeue: %v", requeueAfter)
		}
	})

	t.Run("incompatible controller version blocks readiness", func(t *testing.T) {
		cluster := genCluster("old-controller", projectID, "0", "0", "0")
		cluster.Spec.Cloud.Kubevirt = &kubermaticv1.KubevirtCloudSpec{}
		cluster.Status.AcceleratorAccounting = readyClusterAccounting(revision, digest, now)
		cluster.Status.AcceleratorAccounting.ControllerVersion = "v2.28.0"

		status, _ := r.localAcceleratorAccounting(resourceQuota, []kubermaticv1.Cluster{*cluster})
		if status.Ready {
			t.Fatal("expected incompatible controller version to be not ready")
		}
		if len(status.Blockers) != 1 || status.Blockers[0].Type != kubermaticv1.AcceleratorAccountingBlockerTypeIncompatibleControllerVersion {
			t.Fatalf("expected incompatible-controller-version blocker, got %#v", status.Blockers)
		}
	})
}

func genResourceQuota(name string) *kubermaticv1.ResourceQuota {
	rq := &kubermaticv1.ResourceQuota{}
	rq.Name = name
	rq.Spec = kubermaticv1.ResourceQuotaSpec{
		Subject: kubermaticv1.Subject{
			Name: projectID,
			Kind: kubermaticv1.ProjectSubjectKind,
		},
	}

	return rq
}

func genResourceDetails(cpu, mem, storage string) *kubermaticv1.ResourceDetails {
	return kubermaticv1.NewResourceDetails(resource.MustParse(cpu), resource.MustParse(mem), resource.MustParse(storage))
}

func genCluster(name, projectID, cpu, mem, storage string) *kubermaticv1.Cluster {
	cluster := &kubermaticv1.Cluster{}
	cluster.Name = name
	cluster.Labels = map[string]string{kubermaticv1.ProjectIDLabelKey: projectID}
	cluster.Status.ResourceUsage = genResourceDetails(cpu, mem, storage)

	return cluster
}

func acceleratorUsage(count string) []kubermaticv1.AcceleratorQuota {
	return []kubermaticv1.AcceleratorQuota{{
		Provider: accelerator.ProviderKubeVirt,
		Resources: corev1.ResourceList{
			corev1.ResourceName("nvidia.com/GH100_H200_NVL"): resource.MustParse(count),
		},
	}}
}

func readyClusterAccounting(revision kubermaticv1.AcceleratorAccountingRevision, digest kubermaticv1.AcceleratorQuotaDigest, observedAt time.Time) *kubermaticv1.ClusterAcceleratorAccountingStatus {
	return &kubermaticv1.ClusterAcceleratorAccountingStatus{
		ObservedAccountingRevision: revision,
		ObservedQuotaDigest:        digest,
		FootprintSchemaVersion:     accelerator.SchemaVersionV1Alpha1,
		ControllerVersion:          "v2.29.0",
		ObservedAt:                 metav1.NewTime(observedAt),
		Ready:                      true,
	}
}
