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

package seed

import (
	"context"
	"fmt"
	"testing"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"

	"k8s.io/api/batch/v1beta1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
)

func TestBasicEEReconciling(t *testing.T) {
	testSeeds := map[string]*kubermaticv1.Seed{
		"seed-with-metering-config": {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "seed-with-metering-config",
				Namespace: "kubermatic",
			},
			Spec: kubermaticv1.SeedSpec{
				Metering: &kubermaticv1.MeteringConfiguration{
					Enabled:          true,
					StorageSize:      "10Gi",
					StorageClassName: "test",
					ReportConfigurations: map[string]*kubermaticv1.MeteringReportConfiguration{
						"weekly-test": {
							Schedule: "0 1 * * 6",
							Interval: 7,
						},
					},
				},
			},
		},
	}

	k8cConfig := kubermaticv1.KubermaticConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "kubermatic",
		},
		Spec: kubermaticv1.KubermaticConfigurationSpec{
			Ingress: kubermaticv1.KubermaticIngressConfiguration{
				Domain: "example.com",
			},
		},
	}

	type testcase struct {
		name            string
		seedToReconcile string
		configuration   *kubermaticv1.KubermaticConfiguration
		seedsOnMaster   []string
		syncedSeeds     sets.String // seeds where the seed-sync-controller copied the Seed CR over already
		assertion       func(test *testcase, reconciler *Reconciler) error
	}

	tests := []testcase{
		{
			name:            "when given metering configuration",
			seedToReconcile: "seed-with-metering-config",
			configuration:   &k8cConfig,
			seedsOnMaster:   []string{"seed-with-metering-config"},
			syncedSeeds:     sets.NewString("seed-with-metering-config"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				ctx := context.Background()

				if err := reconciler.reconcile(ctx, reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %w", err)
				}

				seedClient := reconciler.seedClients[test.seedToReconcile]

				cronJob := v1beta1.CronJob{}
				if err := seedClient.Get(ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      "weekly-test",
				}, &cronJob); err != nil {
					return fmt.Errorf("failed to find reporting cronjob: %w", err)
				}
				return nil
			},
		},
		{
			name:            "when removing metering configuration report",
			seedToReconcile: "seed-with-metering-config",
			configuration:   &k8cConfig,
			seedsOnMaster:   []string{"seed-with-metering-config"},
			syncedSeeds:     sets.NewString("seed-with-metering-config"),
			assertion: func(test *testcase, reconciler *Reconciler) error {
				ctx := context.Background()

				// reconciling to the initial state
				if err := reconciler.reconcile(ctx, reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %w", err)
				}

				seedClient := reconciler.seedClients[test.seedToReconcile]

				// asserting that reporting cron job exists
				cronJob := v1beta1.CronJob{}
				must(t, seedClient.Get(ctx, types.NamespacedName{Namespace: "kubermatic", Name: "weekly-test"}, &cronJob))

				seed := &kubermaticv1.Seed{}
				must(t, seedClient.Get(ctx, types.NamespacedName{Namespace: "kubermatic", Name: test.seedToReconcile}, seed))

				// removing reports from metering configuration
				seed.Spec.Metering.ReportConfigurations = map[string]*kubermaticv1.MeteringReportConfiguration{}
				must(t, seedClient.Update(ctx, seed))

				// letting the controller clean up
				if err := reconciler.reconcile(ctx, reconciler.log, test.seedToReconcile); err != nil {
					return fmt.Errorf("reconciliation failed: %w", err)
				}

				// asserting that reporting cron job is gone
				if err := seedClient.Get(ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      "weekly-test",
				}, &cronJob); err != nil {
					if kerrors.IsNotFound(err) {
						return nil
					}
				}

				return fmt.Errorf("failed to remove an orpahned reporting cron job")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reconciler := createTestReconciler(testSeeds, test.configuration, test.seedsOnMaster, test.syncedSeeds)

			if err := test.assertion(&test, reconciler); err != nil {
				t.Fatalf("Failure: %v", err)
			}
		})
	}
}
