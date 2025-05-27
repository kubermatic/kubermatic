//go:build ee

/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package seed

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/ee/metering"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
)

func meteringCredsNotFound(err error) bool {
	return apierrors.IsNotFound(err) && strings.Contains(err.Error(), metering.SecretName)
}

func TestMeteringReconciling(t *testing.T) {
	now := metav1.NewTime(time.Now())
	allSeeds := getSeeds(now)

	type testcase struct {
		name            string
		seedToReconcile string
		configuration   *kubermaticv1.KubermaticConfiguration
		seedsOnMaster   []string
		syncedSeeds     sets.Set[string] // seeds where the seed-sync-controller copied the Seed CR over already
		assertion       func(t *testing.T, test *testcase, reconciler *Reconciler) error
	}

	tests := []testcase{
		{
			name:            "when given metering configuration",
			seedToReconcile: "seed-with-metering-config",
			configuration:   &k8cConfig,
			seedsOnMaster:   []string{"seed-with-metering-config"},
			syncedSeeds:     sets.New("seed-with-metering-config"),
			assertion: func(t *testing.T, test *testcase, reconciler *Reconciler) error {
				ctx := context.Background()

				if err := reconciler.reconcile(ctx, reconciler.log, test.seedToReconcile); err != nil {
					// ignore missing secret to avoid http call to S3
					if !meteringCredsNotFound(err) {
						return fmt.Errorf("reconciliation failed: %w", err)
					}
				}

				seedClient := reconciler.seedClients[test.seedToReconcile]

				cronJob := batchv1.CronJob{}
				err := seedClient.Get(ctx, types.NamespacedName{Namespace: "kubermatic", Name: "metering-weekly-test"}, &cronJob)
				if err != nil {
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
			syncedSeeds:     sets.New("seed-with-metering-config"),
			assertion: func(t *testing.T, test *testcase, reconciler *Reconciler) error {
				ctx := context.Background()

				// reconciling to the initial state
				if err := reconciler.reconcile(ctx, reconciler.log, test.seedToReconcile); err != nil && !meteringCredsNotFound(err) {
					return fmt.Errorf("reconciliation failed: %w", err)
				}

				seedClient := reconciler.seedClients[test.seedToReconcile]

				// asserting that reporting cron job exists
				cronJob := batchv1.CronJob{}
				must(t, seedClient.Get(ctx, types.NamespacedName{Namespace: "kubermatic", Name: "metering-weekly-test"}, &cronJob))

				seed := &kubermaticv1.Seed{}
				must(t, seedClient.Get(ctx, types.NamespacedName{Namespace: "kubermatic", Name: test.seedToReconcile}, seed))

				// removing reports from metering configuration
				seed.Spec.Metering.ReportConfigurations = map[string]kubermaticv1.MeteringReportConfiguration{}
				must(t, seedClient.Update(ctx, seed))

				// letting the controller clean up
				if err := reconciler.reconcile(ctx, reconciler.log, test.seedToReconcile); err != nil && !meteringCredsNotFound(err) {
					return fmt.Errorf("reconciliation failed: %w", err)
				}

				// asserting that reporting cron job is gone
				if err := seedClient.Get(ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      "weekly-test",
				}, &cronJob); err != nil {
					if apierrors.IsNotFound(err) {
						return nil
					}
				}

				return fmt.Errorf("failed to remove an orphaned reporting cron job")
			},
		},

		{
			name:            "when disabling metering configuration",
			seedToReconcile: "seed-with-metering-config",
			configuration:   &k8cConfig,
			seedsOnMaster:   []string{"seed-with-metering-config"},
			syncedSeeds:     sets.New("seed-with-metering-config"),
			assertion: func(t *testing.T, test *testcase, reconciler *Reconciler) error {
				ctx := context.Background()

				// reconciling to the initial state
				if err := reconciler.reconcile(ctx, reconciler.log, test.seedToReconcile); err != nil && !meteringCredsNotFound(err) {
					return fmt.Errorf("reconciliation failed: %w", err)
				}

				seedClient := reconciler.seedClients[test.seedToReconcile]

				// asserting that reporting cron job exists
				cronJob := batchv1.CronJob{}
				must(t, seedClient.Get(ctx, types.NamespacedName{Namespace: "kubermatic", Name: "metering-weekly-test"}, &cronJob))

				// asserting that metering statefulSet exists
				statefulSet := appsv1.StatefulSet{}
				must(t, seedClient.Get(ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      "metering-prometheus",
				}, &statefulSet))

				seed := &kubermaticv1.Seed{}
				must(t, seedClient.Get(ctx, types.NamespacedName{Namespace: "kubermatic", Name: test.seedToReconcile}, seed))

				// removing reports from metering configuration
				seed.Spec.Metering.Enabled = false
				must(t, seedClient.Update(ctx, seed))

				// letting the controller clean up
				if err := reconciler.reconcile(ctx, reconciler.log, test.seedToReconcile); err != nil && !meteringCredsNotFound(err) {
					return fmt.Errorf("reconciliation failed: %w", err)
				}

				// asserting that reporting cron job is gone
				if err := seedClient.Get(ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      "weekly-test",
				}, &cronJob); err != nil {
					if !apierrors.IsNotFound(err) {
						return fmt.Errorf("failed to remove reporting cron jobs")
					}
				}

				if err := seedClient.Get(ctx, types.NamespacedName{
					Namespace: "kubermatic",
					Name:      "kubermatic-metering",
				}, &statefulSet); err != nil {
					if !apierrors.IsNotFound(err) {
						return fmt.Errorf("failed to remove metering statefulSet")
					}
				}

				return nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reconciler := createTestReconciler(allSeeds, test.configuration, test.seedsOnMaster, test.syncedSeeds)

			if err := test.assertion(t, &test, reconciler); err != nil {
				t.Fatalf("Failure: %v", err)
			}
		})
	}
}
