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
	"fmt"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// csvReportRetention is the retention of the suite's CSV report, which the
	// retention checks expect as a bucket lifecycle rule.
	csvReportRetention = 2

	// suiteReportSchedule is the schedule of the suite's reports. The checks
	// start the CronJobs themselves, so it only has to be valid.
	suiteReportSchedule = "0 1 * * *"

	// readyTimeout covers the PVC provisioning and the image pulls of a fresh
	// metering Prometheus.
	readyTimeout = 10 * time.Minute
)

// SetupOptions are used when EnableMetering has to turn metering on.
type SetupOptions struct {
	StorageClassName string
	StorageSize      string
}

// EnableMetering turns metering on in the Seed when it is off, adds the
// suite's report configurations and waits until the metering Prometheus is
// ready and the report CronJobs exist. Storage settings of a Seed that already
// has metering on are left alone: changing them needs manual steps.
//
// The returned function reverts the Seed to its previous state.
func EnableMetering(ctx context.Context, in Input, opts SetupOptions) (func(context.Context) error, error) {
	seed := &kubermaticv1.Seed{ObjectMeta: metav1.ObjectMeta{Namespace: in.SeedNamespace, Name: in.SeedName}}

	revert, err := patchWithRevert(ctx, in.SeedClient, seed, func() {
		if seed.Spec.Metering == nil {
			seed.Spec.Metering = &kubermaticv1.MeteringConfiguration{}
		}

		metering := seed.Spec.Metering
		if !metering.Enabled {
			metering.Enabled = true
			if metering.StorageClassName == "" {
				metering.StorageClassName = opts.StorageClassName
			}
			if metering.StorageSize == "" {
				metering.StorageSize = opts.StorageSize
			}
		}

		if metering.ReportConfigurations == nil {
			metering.ReportConfigurations = map[string]kubermaticv1.MeteringReportConfiguration{}
		}

		metering.ReportConfigurations[in.CSVReport()] = kubermaticv1.MeteringReportConfiguration{
			Schedule:  suiteReportSchedule,
			Interval:  1,
			Retention: ptr.To[uint32](csvReportRetention),
			Types:     []string{"cluster", "namespace"},
			Format:    kubermaticv1.MeteringReportFormatCSV,
		}
		metering.ReportConfigurations[in.JSONReport()] = kubermaticv1.MeteringReportConfiguration{
			Schedule: suiteReportSchedule,
			Interval: 1,
			Types:    []string{"cluster", "namespace"},
			Format:   kubermaticv1.MeteringReportFormatJSON,
		}
	})
	if err != nil {
		return nil, fmt.Errorf("failed to enable metering: %w", err)
	}

	if err := waitForMetering(ctx, in, in.CSVReport(), in.JSONReport()); err != nil {
		return revert, err
	}

	return revert, nil
}

// suiteReportConfig returns the report configuration the suite uses for
// temporary reports.
func suiteReportConfig() kubermaticv1.MeteringReportConfiguration {
	return kubermaticv1.MeteringReportConfiguration{
		Schedule: suiteReportSchedule,
		Interval: 1,
		Types:    []string{"cluster"},
		Format:   kubermaticv1.MeteringReportFormatCSV,
	}
}

// waitForMetering waits until the metering Prometheus is ready and the
// CronJobs of the given reports exist.
func waitForMetering(ctx context.Context, in Input, reports ...string) error {
	return wait.PollImmediate(ctx, 5*time.Second, readyTimeout, func(ctx context.Context) (error, error) {
		sts := &appsv1.StatefulSet{}
		if err := in.SeedClient.Get(ctx, types.NamespacedName{Namespace: in.SeedNamespace, Name: prometheusName}, sts); err != nil {
			return fmt.Errorf("StatefulSet %s: %w", prometheusName, err), nil
		}
		if sts.Status.ReadyReplicas < 1 || sts.Status.ObservedGeneration < sts.Generation {
			return fmt.Errorf("StatefulSet %s is not ready", prometheusName), nil
		}

		for _, report := range reports {
			cronJob := &batchv1.CronJob{}
			if err := in.SeedClient.Get(ctx, types.NamespacedName{Namespace: in.SeedNamespace, Name: cronJobPrefix + report}, cronJob); err != nil {
				return fmt.Errorf("CronJob for report %s: %w", report, err), nil
			}
		}

		return nil, nil
	})
}

// The workload gives the user cluster a namespace with known usage and a
// PVC with known content, so the reports have rows to compare.
const (
	workloadNamespace = "metering-e2e"
	workloadName      = "metering-e2e-probe"
	workloadImage     = "docker.io/library/busybox:1.36"

	// workloadDataMiB is written to the PVC; the storage column has to show at
	// least this much.
	workloadDataMiB = 64
)

// StartWorkload deploys the probe workload into the user cluster. It is
// idempotent. Start it early, so its usage is scraped and federated before
// the report checks run.
func StartWorkload(ctx context.Context, in Input) error {
	if err := in.requireCluster(); err != nil {
		return err
	}

	objects := []ctrlruntimeclient.Object{
		&corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: workloadNamespace}},
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Namespace: workloadNamespace, Name: workloadName},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("1Gi")},
				},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Namespace: workloadNamespace, Name: workloadName},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name:  "probe",
					Image: workloadImage,
					// write the data once, then keep a little CPU busy so the
					// namespace has non-zero CPU usage
					Command: []string{"sh", "-c", fmt.Sprintf(
						"dd if=/dev/urandom of=/data/fill bs=1M count=%d && sync && while true; do i=0; while [ $i -lt 20000 ]; do i=$((i+1)); done; sleep 1; done",
						workloadDataMiB,
					)},
					VolumeMounts: []corev1.VolumeMount{{Name: "data", MountPath: "/data"}},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("10m"),
							corev1.ResourceMemory: resource.MustParse("16Mi"),
						},
					},
				}},
				Volumes: []corev1.Volume{{
					Name: "data",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: workloadName},
					},
				}},
			},
		},
	}

	for _, object := range objects {
		if err := in.UserClient.Create(ctx, object); err != nil && !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("failed to create %T %s: %w", object, object.GetName(), err)
		}
	}

	return wait.PollImmediate(ctx, 5*time.Second, readyTimeout, func(ctx context.Context) (error, error) {
		pod := &corev1.Pod{}
		if err := in.UserClient.Get(ctx, types.NamespacedName{Namespace: workloadNamespace, Name: workloadName}, pod); err != nil {
			return err, nil
		}
		if pod.Status.Phase != corev1.PodRunning {
			return fmt.Errorf("probe pod is %s", pod.Status.Phase), nil
		}

		return nil, nil
	})
}

// CleanupWorkload deletes the probe workload namespace.
func CleanupWorkload(ctx context.Context, in Input) error {
	if err := in.requireCluster(); err != nil {
		return err
	}

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: workloadNamespace}}
	if err := in.UserClient.Delete(ctx, ns); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete namespace %s: %w", workloadNamespace, err)
	}

	return nil
}
