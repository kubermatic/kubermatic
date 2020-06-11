package seed

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/kubernetes"
	kubermaticlog "github.com/kubermatic/kubermatic/api/pkg/log"

	certmanagerv1alpha2 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func init() {
	utilruntime.Must(kubermaticv1.AddToScheme(scheme.Scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme.Scheme))
	utilruntime.Must(certmanagerv1alpha2.AddToScheme(scheme.Scheme))
}

func must(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Failed: %v", err)
	}
}

func TestBasicReconciling(t *testing.T) {
	now := metav1.NewTime(time.Now())

	allSeeds := map[string]*kubermaticv1.Seed{
		"europe": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "europe",
				Namespace: "kubermatic",
			},
		},
		"asia": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "asia",
				Namespace: "kubermatic",
			},
		},
		"goner": {
			ObjectMeta: metav1.ObjectMeta{
				Name:              "goner",
				Namespace:         "kubermatic",
				DeletionTimestamp: &now,
			},
		},
		"other": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "other",
				Namespace: "kube-system",
			},
		},
	}

	type testcase struct {
		name            string
		seedToReconcile string
		configuration   *operatorv1alpha1.KubermaticConfiguration
		seedsOnMaster   []string
		syncedSeeds     sets.String // seeds where the seed-sync-controller copied the Seed CR over already
		assertion       func(test *testcase, reconciler *Reconciler) error
	}

	tests := []testcase{
		{
			name:            "finalizer is set on Seed copy",
			seedToReconcile: "europe",
			configuration: &operatorv1alpha1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "kubermatic",
				},
				Spec: operatorv1alpha1.KubermaticConfigurationSpec{
					Ingress: operatorv1alpha1.KubermaticIngressConfiguration{
						Domain: "example.com",
					},
				},
			},
			seedsOnMaster: []string{"europe"},
			syncedSeeds:   sets.NewString("europe"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				if err := reconciler.reconcile(reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %v", err)
				}

				seed := kubermaticv1.Seed{}
				if err := reconciler.seedClients["europe"].Get(reconciler.ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      "europe",
				}, &seed); err != nil {
					return fmt.Errorf("failed to retrieve Seed: %v", err)
				}

				if !kubernetes.HasFinalizer(&seed, common.CleanupFinalizer) {
					return fmt.Errorf("Seed copy in seed cluster does not have cleanup finalizer %q", common.CleanupFinalizer)
				}

				return nil
			},
		},

		{
			name:            "finalizer triggers cleanup",
			seedToReconcile: "goner",
			configuration: &operatorv1alpha1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "kubermatic",
				},
				Spec: operatorv1alpha1.KubermaticConfigurationSpec{
					Ingress: operatorv1alpha1.KubermaticIngressConfiguration{
						Domain: "example.com",
					},
				},
			},
			seedsOnMaster: []string{"goner"},
			syncedSeeds:   sets.NewString("goner"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				if err := reconciler.reconcile(reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %v", err)
				}

				seed := kubermaticv1.Seed{}
				if err := reconciler.seedClients["goner"].Get(reconciler.ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      "goner",
				}, &seed); err != nil {
					return fmt.Errorf("failed to retrieve Seed: %v", err)
				}

				if kubernetes.HasFinalizer(&seed, common.CleanupFinalizer) {
					return fmt.Errorf("Seed copy in seed cluster should not have cleanup finalizer %q anymore", common.CleanupFinalizer)
				}

				return nil
			},
		},

		{
			name:            "all cluster-wide resources are cleaned up when deleting a seed",
			seedToReconcile: "europe",
			configuration: &operatorv1alpha1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "kubermatic",
				},
				Spec: operatorv1alpha1.KubermaticConfigurationSpec{
					Ingress: operatorv1alpha1.KubermaticIngressConfiguration{
						Domain: "example.com",
					},
				},
			},
			seedsOnMaster: []string{"europe"},
			syncedSeeds:   sets.NewString("europe"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				if err := reconciler.reconcile(reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %v", err)
				}

				seedClient := reconciler.seedClients["europe"]

				// assert that cluster-wide resources exist
				crbs := rbacv1.ClusterRoleBindingList{}
				must(t, seedClient.List(reconciler.ctx, &crbs))
				if len(crbs.Items) == 0 {
					return errors.New("Seed should have ClusterRoleBindings, but has none")
				}

				hooks := admissionregistrationv1beta1.ValidatingWebhookConfigurationList{}
				must(t, seedClient.List(reconciler.ctx, &hooks))
				if len(hooks.Items) == 0 {
					return errors.New("Seed should have ValidatingWebhookConfigurations, but has none")
				}

				// and now delete the Seed on the seed cluster
				seedName := types.NamespacedName{Namespace: "kubermatic", Name: "europe"}

				seed := &kubermaticv1.Seed{}
				must(t, seedClient.Get(reconciler.ctx, seedName, seed))
				seed.DeletionTimestamp = &now
				must(t, seedClient.Update(reconciler.ctx, seed))

				// let the controller clean up
				if err := reconciler.reconcile(reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %v", err)
				}

				// all global resources should be gone
				crbs = rbacv1.ClusterRoleBindingList{}
				must(t, seedClient.List(reconciler.ctx, &crbs))
				if length := len(crbs.Items); length > 0 {
					return fmt.Errorf("Seed should have no ClusterRoleBindings left over, but has %d", length)
				}

				hooks = admissionregistrationv1beta1.ValidatingWebhookConfigurationList{}
				must(t, seedClient.List(reconciler.ctx, &hooks))
				if length := len(hooks.Items); length > 0 {
					return fmt.Errorf("Seed should have no ValidatingWebhookConfigurations left over, but has %d", length)
				}

				return nil
			},
		},

		{
			name:            "seeds in other namespaces are ignored",
			seedToReconcile: "other",
			configuration: &operatorv1alpha1.KubermaticConfiguration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "kubermatic",
				},
				Spec: operatorv1alpha1.KubermaticConfigurationSpec{
					Ingress: operatorv1alpha1.KubermaticIngressConfiguration{
						Domain: "example.com",
					},
				},
			},
			seedsOnMaster: []string{"other"},
			syncedSeeds:   sets.NewString("other"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				// The controller should never attempt to reconcile the Seed, so removing the
				// seed client should not hurt it.
				reconciler.seedClients = map[string]ctrlruntimeclient.Client{}

				if err := reconciler.reconcile(reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %v", err)
				}

				return nil
			},
		},

		{
			name:            "seeds without KubermaticConfiguration are ignored",
			seedToReconcile: "europe",
			configuration:   nil,
			seedsOnMaster:   []string{"europe"},
			syncedSeeds:     sets.NewString("europe"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				if err := reconciler.reconcile(reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %v", err)
				}

				return nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reconciler := createTestReconciler(allSeeds, test.configuration, test.seedsOnMaster, test.syncedSeeds)

			if err := test.assertion(&test, reconciler); err != nil {
				t.Fatalf("Failure: %v", err)
			}
		})
	}
}

func createTestReconciler(allSeeds map[string]*kubermaticv1.Seed, cfg *operatorv1alpha1.KubermaticConfiguration, seeds []string, syncedSeeds sets.String) *Reconciler {
	masterObjects := []runtime.Object{}
	if cfg != nil {
		masterObjects = append(masterObjects, cfg)
	}

	masterSeeds := map[string]*kubermaticv1.Seed{} // makes the seedsGetter implementation easier
	seedObjects := map[string][]runtime.Object{}
	seedClients := map[string]ctrlruntimeclient.Client{}
	seedRecorders := map[string]record.EventRecorder{}

	for _, seedName := range seeds {
		masterSeed := allSeeds[seedName].DeepCopy()

		masterObjects = append(masterObjects, masterSeed)

		// the seedsGetter is only returning seeds in its given namespace, so we have to replicate the
		// behaviour here for the dummy seeds getter
		if masterSeed.Namespace == "kubermatic" {
			masterSeeds[seedName] = masterSeed
		}

		seedObjects[seedName] = []runtime.Object{}
		if syncedSeeds.Has(seedName) {
			// make sure to put a copy of the Seed CR into the seed "cluster"
			seedObjects[seedName] = append(seedObjects[seedName], masterSeed.DeepCopy())
		}

		seedClients[seedName] = ctrlruntimefakeclient.NewFakeClient(seedObjects[seedName]...)
		seedRecorders[seedName] = record.NewFakeRecorder(999)
	}

	masterClient := ctrlruntimefakeclient.NewFakeClient(masterObjects...)
	masterRecorder := record.NewFakeRecorder(999)

	versions := common.NewDefaultVersions()
	versions.Kubermatic = "latest"
	versions.UI = "latest"

	seedsGetter := func() (map[string]*kubermaticv1.Seed, error) {
		return masterSeeds, nil
	}

	return &Reconciler{
		ctx:            context.Background(),
		log:            kubermaticlog.New(true, kubermaticlog.FormatConsole).Sugar(),
		scheme:         scheme.Scheme,
		namespace:      "kubermatic",
		masterClient:   masterClient,
		masterRecorder: masterRecorder,
		seedClients:    seedClients,
		seedRecorders:  seedRecorders,
		seedsGetter:    seedsGetter,
		versions:       versions,
	}
}
