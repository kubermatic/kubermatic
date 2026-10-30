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

package metering

import (
	"context"
	"errors"
	"fmt"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// setReports adds report configurations and removes the ones mapped to nil.
func setReports(ctx context.Context, in Input, reports map[string]*kubermaticv1.MeteringReportConfiguration) error {
	seed := in.seedRef()

	_, err := patchWithRevert(ctx, in.SeedClient, seed, func() {
		for name, config := range reports {
			if config == nil {
				delete(seed.Spec.Metering.ReportConfigurations, name)
			} else {
				seed.Spec.Metering.ReportConfigurations[name] = *config
			}
		}
	})

	return err
}

// waitForCronJob waits until the report's CronJob exists, or is gone when
// exists is false.
func waitForCronJob(ctx context.Context, in Input, report string, exists bool) error {
	return wait.PollImmediate(ctx, 2*time.Second, 2*time.Minute, func(ctx context.Context) (error, error) {
		cronJob := &batchv1.CronJob{}
		err := in.SeedClient.Get(ctx, types.NamespacedName{Namespace: in.SeedNamespace, Name: cronJobPrefix + report}, cronJob)

		switch {
		case exists && err != nil:
			return fmt.Errorf("CronJob of report %s: %w", report, err), nil
		case !exists && err == nil && cronJob.DeletionTimestamp == nil:
			return fmt.Errorf("CronJob of report %s still exists", report), nil
		case !exists && err != nil && !apierrors.IsNotFound(err):
			return err, nil
		}

		return nil, nil
	})
}

// CheckReportConfigLifecycle verifies the documented behaviour of report
// configurations: adding one creates its CronJob, renaming one replaces the
// CronJob, and deleting one removes the CronJob but keeps the reports it
// wrote in the bucket.
func CheckReportConfigLifecycle(ctx context.Context, in Input) error {
	if _, _, err := in.meteringConfig(ctx); err != nil {
		return err
	}

	first := in.ReportPrefix + "-tmp"
	renamed := in.ReportPrefix + "-tmp2"
	config := suiteReportConfig()

	defer func() {
		ctx, cancel := revertContext()
		defer cancel()

		if err := setReports(ctx, in, map[string]*kubermaticv1.MeteringReportConfiguration{first: nil, renamed: nil}); err != nil {
			in.Log.Warnw("Failed to remove the temporary reports", "error", err)
		}
	}()

	if err := setReports(ctx, in, map[string]*kubermaticv1.MeteringReportConfiguration{first: &config}); err != nil {
		return err
	}
	if err := waitForCronJob(ctx, in, first, true); err != nil {
		return fmt.Errorf("adding a report: %w", err)
	}

	started, err := runReportJob(ctx, in, first, nil)
	if err != nil {
		return err
	}

	if err := setReports(ctx, in, map[string]*kubermaticv1.MeteringReportConfiguration{first: nil, renamed: &config}); err != nil {
		return err
	}

	var errs []error
	if err := waitForCronJob(ctx, in, renamed, true); err != nil {
		errs = append(errs, fmt.Errorf("renaming a report: %w", err))
	}
	if err := waitForCronJob(ctx, in, first, false); err != nil {
		errs = append(errs, fmt.Errorf("renaming a report: %w", err))
	}

	if err := setReports(ctx, in, map[string]*kubermaticv1.MeteringReportConfiguration{renamed: nil}); err != nil {
		return errors.Join(append(errs, err)...)
	}
	if err := waitForCronJob(ctx, in, renamed, false); err != nil {
		errs = append(errs, fmt.Errorf("deleting a report: %w", err))
	}

	s3, err := meteringBucket(ctx, in)
	if err != nil {
		return errors.Join(append(errs, err)...)
	}
	defer s3.stop()

	if _, err := scheduledReport(ctx, s3, in, first, "cluster", "csv", started, 1); err != nil {
		errs = append(errs, fmt.Errorf("the report written before the rename is gone: %w", err))
	}

	return errors.Join(errs...)
}

// CheckDisableAndReenable verifies that turning metering off removes the
// metering Prometheus and the report CronJobs but keeps the Prometheus PVC
// and the S3 Secret, and that turning it on again brings everything back
// with the data collected before.
func CheckDisableAndReenable(ctx context.Context, in Input) error {
	if _, _, err := in.meteringConfig(ctx); err != nil {
		return err
	}

	disabledAt := time.Now()

	seed := in.seedRef()
	revert, err := patchWithRevert(ctx, in.SeedClient, seed, func() {
		seed.Spec.Metering.Enabled = false
	})
	if err != nil {
		return err
	}

	// always turn metering on again, also when the checks below fail
	reverted := false
	defer func() {
		if reverted {
			return
		}

		ctx, cancel := revertContext()
		defer cancel()

		if err := revert(ctx); err != nil {
			in.Log.Warnw("Failed to turn metering on again", "error", err)
		}
	}()

	var errs []error
	if err := waitForUndeploy(ctx, in); err != nil {
		errs = append(errs, err)
	}

	for _, kept := range []ctrlruntimeclient.Object{&corev1.PersistentVolumeClaim{}, &corev1.Secret{}} {
		name := prometheusPVCName
		if _, ok := kept.(*corev1.Secret); ok {
			name = s3SecretName
		}

		if err := in.SeedClient.Get(ctx, types.NamespacedName{Namespace: in.SeedNamespace, Name: name}, kept); err != nil {
			errs = append(errs, fmt.Errorf("%T %s is gone after disabling metering: %w", kept, name, err))
		}
	}

	revertCtx, cancel := revertContext()
	defer cancel()

	if err := revert(revertCtx); err != nil {
		return errors.Join(append(errs, err)...)
	}
	reverted = true

	if err := waitForMetering(ctx, in, in.CSVReport(), in.JSONReport()); err != nil {
		return errors.Join(append(errs, fmt.Errorf("after enabling metering again: %w", err))...)
	}

	errs = append(errs, checkDataKept(ctx, in, disabledAt))

	return errors.Join(errs...)
}

// waitForUndeploy waits until the metering Prometheus objects and the report
// CronJobs are gone.
func waitForUndeploy(ctx context.Context, in Input) error {
	return wait.PollImmediate(ctx, 2*time.Second, 3*time.Minute, func(ctx context.Context) (error, error) {
		key := types.NamespacedName{Namespace: in.SeedNamespace, Name: prometheusName}
		clusterKey := types.NamespacedName{Name: prometheusName}

		var left []string
		for _, o := range []struct {
			obj ctrlruntimeclient.Object
			key types.NamespacedName
		}{
			{&corev1.ServiceAccount{}, key},
			{&corev1.ConfigMap{}, key},
			{&appsv1.StatefulSet{}, key},
			{&corev1.Service{}, key},
			{&rbacv1.ClusterRole{}, clusterKey},
			{&rbacv1.ClusterRoleBinding{}, clusterKey},
		} {
			err := in.SeedClient.Get(ctx, o.key, o.obj)
			if err == nil {
				left = append(left, fmt.Sprintf("%T", o.obj))
			} else if !apierrors.IsNotFound(err) {
				return err, nil
			}
		}

		cronJobs := &batchv1.CronJobList{}
		if err := in.SeedClient.List(ctx, cronJobs, ctrlruntimeclient.InNamespace(in.SeedNamespace), ctrlruntimeclient.MatchingLabels{componentLabel: componentValue}); err != nil {
			return err, nil
		}
		for _, cronJob := range cronJobs.Items {
			left = append(left, "CronJob "+cronJob.Name)
		}

		if len(left) > 0 {
			return fmt.Errorf("metering objects left after disabling: %v", left), nil
		}

		return nil, nil
	})
}

// checkDataKept verifies that the metering Prometheus still has the samples
// it collected before metering was turned off. It reads the scrape health of
// the seed-controller-manager, which exists on every seed, also without user
// clusters.
func checkDataKept(ctx context.Context, in Input, before time.Time) error {
	prom, err := meteringPrometheus(in)
	if err != nil {
		return err
	}

	query := `count(up{job="seed-controller-manager"})`

	return wait.PollImmediate(ctx, 10*time.Second, 3*time.Minute, func(ctx context.Context) (error, error) {
		vector, err := queryVector(ctx, prom, query, before.Add(-time.Minute))
		if err != nil {
			return err, nil
		}

		if len(vector) == 0 || vector[0].Value == 0 {
			return fmt.Errorf("the metering Prometheus has no samples from before it was turned off"), nil
		}

		return nil, nil
	})
}
