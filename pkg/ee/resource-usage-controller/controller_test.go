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

package resourceusagecontroller

import (
	"context"
	"encoding/json"
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
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	"k8c.io/machine-controller/sdk/providerconfig"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/events"
	clocktesting "k8s.io/utils/clock/testing"
	ctrlruntimeevent "sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReconcile(t *testing.T) {
	testCases := []struct {
		name                  string
		cluster               *kubermaticv1.Cluster
		machines              []*clusterv1alpha1.Machine
		expectedResourceUsage *kubermaticv1.ResourceDetails
	}{
		{
			name:     "scenario 1: calculate resource usage from one machine",
			cluster:  generator.GenDefaultCluster(),
			machines: []*clusterv1alpha1.Machine{genFakeMachine("m1", "5", "5G", "10G")},
			expectedResourceUsage: &kubermaticv1.ResourceDetails{
				CPU:     getQuantity("5"),
				Memory:  getQuantity("5G"),
				Storage: getQuantity("10G"),
			},
		},
		{
			name: "scenario 2: set proper resource usage",
			cluster: func() *kubermaticv1.Cluster {
				c := generator.GenDefaultCluster()
				c.Status.ResourceUsage = kubermaticv1.NewResourceDetails(resource.MustParse("2"), resource.MustParse("1G"), resource.MustParse("2G"))
				return c
			}(),
			machines: []*clusterv1alpha1.Machine{genFakeMachine("m1", "5", "5G", "10G")},
			expectedResourceUsage: &kubermaticv1.ResourceDetails{
				CPU:     getQuantity("5"),
				Memory:  getQuantity("5G"),
				Storage: getQuantity("10G"),
			},
		},
		{
			name:    "scenario 3: calculate proper resource usage from 2 machines",
			cluster: generator.GenDefaultCluster(),
			machines: []*clusterv1alpha1.Machine{
				genFakeMachine("m1", "5", "5G", "10G"),
				genFakeMachine("m2", "2", "3G", "5G")},
			expectedResourceUsage: &kubermaticv1.ResourceDetails{
				CPU:     getQuantity("7"),
				Memory:  getQuantity("8G"),
				Storage: getQuantity("15G"),
			},
		},
		{
			name: "scenario 4: set zero usage if no machines",
			cluster: func() *kubermaticv1.Cluster {
				c := generator.GenDefaultCluster()
				c.Status.ResourceUsage = kubermaticv1.NewResourceDetails(resource.MustParse("2"), resource.MustParse("1G"), resource.MustParse("2G"))
				return c
			}(),
			expectedResourceUsage: &kubermaticv1.ResourceDetails{
				CPU:     getQuantity("0"),
				Memory:  getQuantity("0"),
				Storage: getQuantity("0"),
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			scheme := fake.NewScheme()
			utilruntime.Must(clusterv1alpha1.AddToScheme(scheme))

			seedClientBuilder := fake.NewClientBuilder().WithScheme(scheme)
			seedClientBuilder.WithObjects(tc.cluster)

			userClientBuilder := fake.NewClientBuilder().WithScheme(scheme)
			for _, m := range tc.machines {
				userClientBuilder.WithObjects(m)
			}

			seedClient := seedClientBuilder.Build()
			userClient := userClientBuilder.Build()

			r := reconciler{
				log:         kubermaticlog.Logger,
				seedClient:  seedClient,
				userClient:  userClient,
				clusterName: tc.cluster.Name,
				caBundle:    nil,
				recorder:    &events.FakeRecorder{},
				clusterIsPaused: func(c context.Context) (bool, error) {
					return false, nil
				},
			}

			request := reconcile.Request{NamespacedName: types.NamespacedName{Name: tc.cluster.Name}}
			if _, err := r.Reconcile(ctx, request); err != nil {
				t.Fatalf("reconciling failed: %v", err)
			}

			// check Cluster
			cluster := &kubermaticv1.Cluster{}
			err := seedClient.Get(ctx, types.NamespacedName{Name: tc.cluster.Name}, cluster)
			if err != nil {
				t.Fatalf("failed to get cluster: %v", err)
			}

			if !diff.SemanticallyEqual(tc.expectedResourceUsage, cluster.Status.ResourceUsage) {
				t.Fatalf("Objects differ:\n%v", diff.ObjectDiff(tc.expectedResourceUsage, cluster.Status.ResourceUsage))
			}
			if cluster.Status.AcceleratorAccounting != nil {
				t.Fatalf("inactive cluster accelerator accounting status = %#v, want nil", cluster.Status.AcceleratorAccounting)
			}
			if len(cluster.Status.ResourceUsage.Accelerators) != 0 {
				t.Fatalf("inactive cluster accelerator usage = %#v, want empty", cluster.Status.ResourceUsage.Accelerators)
			}
		})
	}
}

func TestReconcileActiveAcceleratorAccountingWithoutMachines(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.August, 14, 10, 0, 0, 0, time.UTC)
	cluster := generator.GenDefaultCluster()
	cluster.Spec.Cloud.Kubevirt = &kubermaticv1.KubevirtCloudSpec{}
	resourceQuota := activeAcceleratorAccountingResourceQuota(cluster.Labels[kubermaticv1.ProjectIDLabelKey], nil)

	scheme := fake.NewScheme()
	utilruntime.Must(clusterv1alpha1.AddToScheme(scheme))
	seedClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(cluster, resourceQuota).Build()
	userClient := fake.NewClientBuilder().WithScheme(scheme).Build()

	r := reconciler{
		log:                   kubermaticlog.Logger,
		seedClient:            seedClient,
		userClient:            userClient,
		clusterName:           cluster.Name,
		recorder:              &events.FakeRecorder{},
		controllerVersion:     "v2.99.0-test",
		acceleratorAccounting: true,
		clock:                 clocktesting.NewFakeClock(now),
		clusterIsPaused: func(context.Context) (bool, error) {
			return false, nil
		},
	}

	result, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{Name: cluster.Name}})
	if err != nil {
		t.Fatalf("reconciling failed: %v", err)
	}
	if result.RequeueAfter != resources.AcceleratorAccountingHeartbeatInterval {
		t.Fatalf("RequeueAfter = %v, want %v", result.RequeueAfter, resources.AcceleratorAccountingHeartbeatInterval)
	}

	machineResult, err := r.Reconcile(ctx, reconcile.Request{NamespacedName: types.NamespacedName{
		Namespace: metav1.NamespaceSystem,
		Name:      "machine-a",
	}})
	if err != nil {
		t.Fatalf("reconciling Machine event failed: %v", err)
	}
	if machineResult.RequeueAfter != 0 {
		t.Fatalf("Machine event RequeueAfter = %v, want no periodic requeue", machineResult.RequeueAfter)
	}

	got := &kubermaticv1.Cluster{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: cluster.Name}, got); err != nil {
		t.Fatalf("failed to get cluster: %v", err)
	}
	if got.Status.AcceleratorAccounting == nil {
		t.Fatal("accelerator accounting status is nil")
	}
	if !got.Status.AcceleratorAccounting.Ready {
		t.Fatalf("accelerator accounting status = %#v, want ready", got.Status.AcceleratorAccounting)
	}
	if !got.Status.AcceleratorAccounting.ObservedAt.Time.Equal(now) {
		t.Fatalf("ObservedAt = %v, want %v", got.Status.AcceleratorAccounting.ObservedAt.Time, now)
	}
	if len(got.Status.ResourceUsage.Accelerators) != 0 {
		t.Fatalf("accelerator usage = %#v, want empty", got.Status.ResourceUsage.Accelerators)
	}
}

func TestPausedReconcileKeepsActiveAccountingHeartbeatScheduled(t *testing.T) {
	tests := []struct {
		name                  string
		acceleratorAccounting bool
		request               reconcile.Request
		expectedRequeue       time.Duration
	}{
		{
			name:                  "active accounting remains scheduled",
			acceleratorAccounting: true,
			request:               reconcile.Request{NamespacedName: types.NamespacedName{Name: "cluster-a"}},
			expectedRequeue:       resources.AcceleratorAccountingHeartbeatInterval,
		},
		{
			name:                  "active Machine request remains event driven",
			acceleratorAccounting: true,
			request: reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: metav1.NamespaceSystem,
				Name:      "machine-a",
			}},
		},
		{
			name:    "inactive accounting keeps existing pause behavior",
			request: reconcile.Request{NamespacedName: types.NamespacedName{Name: "cluster-a"}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := reconciler{
				clusterName:           "cluster-a",
				acceleratorAccounting: tc.acceleratorAccounting,
				clusterIsPaused: func(context.Context) (bool, error) {
					return true, nil
				},
			}
			result, err := r.Reconcile(context.Background(), tc.request)
			if err != nil {
				t.Fatalf("reconciling paused cluster failed: %v", err)
			}
			if result.RequeueAfter != tc.expectedRequeue {
				t.Fatalf("RequeueAfter = %v, want %v", result.RequeueAfter, tc.expectedRequeue)
			}
		})
	}
}

func TestAcceleratorUsageAndStatus(t *testing.T) {
	now := time.Date(2026, time.August, 14, 11, 0, 0, 0, time.UTC)
	resourceQuota := activeAcceleratorAccountingResourceQuota("project-a", []kubermaticv1.AcceleratorQuota{{
		Provider: accelerator.ProviderKubeVirt,
		Resources: corev1.ResourceList{
			"nvidia.com/H200": resource.MustParse("8"),
		},
	}})

	validOne := providerMachine(t, "valid-one", providerconfig.CloudProviderKubeVirt)
	setFootprint(t, validOne, accelerator.NewKubeVirtFootprint(corev1.ResourceList{
		"nvidia.com/H200": resource.MustParse("1"),
	}))
	validTwo := providerMachine(t, "valid-two", providerconfig.CloudProviderKubeVirt)
	setFootprint(t, validTwo, accelerator.NewKubeVirtFootprint(corev1.ResourceList{
		"nvidia.com/H200":  resource.MustParse("2"),
		"example.com/fpga": resource.MustParse("1"),
	}))
	empty := providerMachine(t, "empty", providerconfig.CloudProviderKubeVirt)
	setFootprint(t, empty, accelerator.NewKubeVirtFootprint(nil))
	legacy := providerMachine(t, "legacy", providerconfig.CloudProviderKubeVirt)
	malformed := providerMachine(t, "malformed", providerconfig.CloudProviderKubeVirt)
	malformed.Annotations = map[string]string{accelerator.FootprintAnnotationKey: `{`}
	unsupported := providerMachine(t, "unsupported", providerconfig.CloudProviderKubeVirt)
	unsupported.Annotations = map[string]string{accelerator.FootprintAnnotationKey: `{"schemaVersion":"v2","provider":"kubevirt","resources":{}}`}
	providerMismatch := providerMachine(t, "provider-mismatch", providerconfig.CloudProviderAWS)
	setFootprint(t, providerMismatch, accelerator.NewKubeVirtFootprint(corev1.ResourceList{
		"nvidia.com/H200": resource.MustParse("100"),
	}))
	irrelevant := providerMachine(t, "irrelevant", providerconfig.CloudProviderAWS)

	machines := &clusterv1alpha1.MachineList{Items: []clusterv1alpha1.Machine{
		*validOne, *validTwo, *empty, *legacy, *malformed, *unsupported, *providerMismatch, *irrelevant,
	}}
	r := reconciler{
		controllerVersion: "v2.99.0-test",
		clock:             clocktesting.NewFakeClock(now),
	}

	usage, status := r.acceleratorUsageAndStatus("cluster-a", resourceQuota, machines)
	if len(usage) != 1 || usage[0].Provider != accelerator.ProviderKubeVirt {
		t.Fatalf("usage = %#v, want one KubeVirt provider bucket", usage)
	}
	h200 := usage[0].Resources["nvidia.com/H200"]
	if h200.Cmp(resource.MustParse("3")) != 0 {
		t.Fatalf("H200 usage = %s, want 3", h200.String())
	}
	fpga := usage[0].Resources["example.com/fpga"]
	if fpga.Cmp(resource.MustParse("1")) != 0 {
		t.Fatalf("FPGA usage = %s, want 1", fpga.String())
	}
	if status.Ready {
		t.Fatalf("status = %#v, want blocked", status)
	}
	if status.MachinesWithoutFootprint != 1 {
		t.Fatalf("MachinesWithoutFootprint = %d, want 1", status.MachinesWithoutFootprint)
	}
	if status.MachinesWithInvalidFootprint != 3 {
		t.Fatalf("MachinesWithInvalidFootprint = %d, want 3", status.MachinesWithInvalidFootprint)
	}
	if status.FootprintSchemaVersion != accelerator.SchemaVersionV1Alpha1 || status.ControllerVersion != "v2.99.0-test" {
		t.Fatalf("capability fields = %#v", status)
	}
	if !status.ObservedAt.Time.Equal(now) {
		t.Fatalf("ObservedAt = %v, want %v", status.ObservedAt.Time, now)
	}

	expectedBlockers := map[kubermaticv1.AcceleratorAccountingBlockerType]int32{
		kubermaticv1.AcceleratorAccountingBlockerTypeLegacyMachines:             1,
		kubermaticv1.AcceleratorAccountingBlockerTypeUnsupportedFootprintSchema: 1,
		kubermaticv1.AcceleratorAccountingBlockerTypeInvalidFootprints:          2,
	}
	if len(status.Blockers) != len(expectedBlockers) {
		t.Fatalf("blockers = %#v, want %d blockers", status.Blockers, len(expectedBlockers))
	}
	for _, blocker := range status.Blockers {
		if expectedCount, exists := expectedBlockers[blocker.Type]; !exists || blocker.Count != expectedCount || blocker.ClusterName != "cluster-a" {
			t.Errorf("unexpected blocker %#v", blocker)
		}
	}
}

func TestAcceleratorAccountingRevisionAndDigestBlockers(t *testing.T) {
	testCases := []struct {
		name            string
		globalStatus    *kubermaticv1.ResourceQuotaGlobalAcceleratorAccountingStatus
		expectedBlocker kubermaticv1.AcceleratorAccountingBlockerType
	}{
		{
			name:            "missing master report",
			expectedBlocker: kubermaticv1.AcceleratorAccountingBlockerTypeMissingReport,
		},
		{
			name: "quota digest mismatch",
			globalStatus: &kubermaticv1.ResourceQuotaGlobalAcceleratorAccountingStatus{
				ObservedAccountingRevision: "revision-a",
				ObservedQuotaDigest:        "sha256:stale",
			},
			expectedBlocker: kubermaticv1.AcceleratorAccountingBlockerTypeQuotaDigestMismatch,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resourceQuota := activeAcceleratorAccountingResourceQuota("project-a", nil)
			resourceQuota.Status.GlobalAcceleratorAccounting = tc.globalStatus
			r := reconciler{clock: clocktesting.NewFakeClock(time.Now())}

			_, status := r.acceleratorUsageAndStatus("cluster-a", resourceQuota, &clusterv1alpha1.MachineList{})
			if status.Ready || len(status.Blockers) != 1 || status.Blockers[0].Type != tc.expectedBlocker {
				t.Fatalf("status = %#v, want only %q blocker", status, tc.expectedBlocker)
			}
		})
	}
}

func TestResourceQuotaAccountingChangedPredicateIgnoresAggregateHeartbeats(t *testing.T) {
	oldQuota := activeAcceleratorAccountingResourceQuota("project-a", nil)
	newQuota := oldQuota.DeepCopy()
	newQuota.Status.GlobalAcceleratorAccounting.ObservedAt = metav1.NewTime(time.Now())
	newQuota.Status.GlobalAcceleratorAccounting.Ready = true
	newQuota.Status.GlobalAcceleratorAccounting.ActivationPhase = kubermaticv1.AcceleratorAccountingPhaseReady

	predicate := resourceQuotaAccountingChangedPredicate()
	if predicate.Update(ctrlruntimeevent.TypedUpdateEvent[*kubermaticv1.ResourceQuota]{ObjectOld: oldQuota, ObjectNew: newQuota}) {
		t.Fatal("aggregate-only global status change should not enqueue and create an accounting feedback loop")
	}

	newQuota.Status.GlobalAcceleratorAccounting.ObservedAccountingRevision = "revision-b"
	if !predicate.Update(ctrlruntimeevent.TypedUpdateEvent[*kubermaticv1.ResourceQuota]{ObjectOld: oldQuota, ObjectNew: newQuota}) {
		t.Fatal("accounting revision change must enqueue the cluster")
	}
}

func genFakeMachine(name, cpu, memory, storage string) *clusterv1alpha1.Machine {
	return generator.GenTestMachine(name,
		fmt.Sprintf(`{"cloudProvider":"fake", "cloudProviderSpec":{"cpu":"%s","memory":"%s","storage":"%s"}}`, cpu, memory, storage),
		nil, nil)
}

func providerMachine(t *testing.T, name string, provider providerconfig.CloudProvider) *clusterv1alpha1.Machine {
	t.Helper()

	cloudProviderSpec, err := json.Marshal(map[string]any{})
	if err != nil {
		t.Fatalf("failed to marshal cloud provider spec: %v", err)
	}
	providerSpec, err := json.Marshal(providerconfig.Config{
		CloudProvider:     provider,
		CloudProviderSpec: runtime.RawExtension{Raw: cloudProviderSpec},
	})
	if err != nil {
		t.Fatalf("failed to marshal provider spec: %v", err)
	}

	return &clusterv1alpha1.Machine{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: metav1.NamespaceSystem},
		Spec: clusterv1alpha1.MachineSpec{
			ProviderSpec: clusterv1alpha1.ProviderSpec{Value: &runtime.RawExtension{Raw: providerSpec}},
		},
	}
}

func setFootprint(t *testing.T, machine *clusterv1alpha1.Machine, footprint accelerator.Footprint) {
	t.Helper()

	encoded, err := accelerator.Encode(footprint)
	if err != nil {
		t.Fatalf("failed to encode footprint: %v", err)
	}
	machine.Annotations = map[string]string{accelerator.FootprintAnnotationKey: encoded}
}

func activeAcceleratorAccountingResourceQuota(projectID string, accelerators []kubermaticv1.AcceleratorQuota) *kubermaticv1.ResourceQuota {
	digest := kubermaticv1.AcceleratorQuotaDigestFor(accelerators)
	return &kubermaticv1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name: "quota-" + projectID,
			Labels: map[string]string{
				kubermaticv1.ResourceQuotaSubjectNameLabelKey: projectID,
				kubermaticv1.ResourceQuotaSubjectKindLabelKey: kubermaticv1.ProjectSubjectKind,
			},
			Annotations: map[string]string{
				resources.AcceleratorAccountingEnabledAnnotation: resources.AcceleratorAccountingEnabledAnnotationValue,
			},
		},
		Spec: kubermaticv1.ResourceQuotaSpec{
			Subject: kubermaticv1.Subject{Name: projectID, Kind: kubermaticv1.ProjectSubjectKind},
			Quota:   kubermaticv1.ResourceDetails{Accelerators: accelerators},
		},
		Status: kubermaticv1.ResourceQuotaStatus{
			GlobalAcceleratorAccounting: &kubermaticv1.ResourceQuotaGlobalAcceleratorAccountingStatus{
				ActivationPhase:            kubermaticv1.AcceleratorAccountingPhaseActivating,
				ObservedAccountingRevision: "revision-a",
				ObservedQuotaDigest:        digest,
			},
		},
	}
}

func getQuantity(q string) *resource.Quantity {
	res := resource.MustParse(q)
	return &res
}
