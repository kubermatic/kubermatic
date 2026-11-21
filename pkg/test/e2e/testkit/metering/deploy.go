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
	"slices"
	"strings"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// federationJobs are the scrape jobs of the metering Prometheus that federate
// from every user cluster Prometheus, see
// pkg/ee/metering/prometheus/configmap.go.
var federationJobs = []string{
	"cluster_kubelet_volume_stats",
	"cluster_node_cpu_usage_seconds_total",
	"cluster_machine_cpu_cores",
	"cluster_machine_memory_bytes",
	"cluster_node_memory_working_set_bytes",
	"cluster_container_cpu_usage_seconds_total",
	"cluster_container_memory_working_set_bytes",
	"machine_controller_machines_total",
	"kube_node_info",
	"container_cpu_usage_seconds_total",
	"container_memory_working_set_bytes",
}

const seedControllerManagerJob = "seed-controller-manager"

// CheckPrometheusDeployment verifies the objects of the metering Prometheus:
// six objects named metering-prometheus, Seed owner references where KKP sets
// them, the retention flag and a bound PVC with the configured class and
// size.
func CheckPrometheusDeployment(ctx context.Context, in Input) error {
	seed, metering, err := in.meteringConfig(ctx)
	if err != nil {
		return err
	}

	key := types.NamespacedName{Namespace: in.SeedNamespace, Name: prometheusName}
	clusterKey := types.NamespacedName{Name: prometheusName}

	sts := &appsv1.StatefulSet{}
	objects := []struct {
		obj         ctrlruntimeclient.Object
		key         types.NamespacedName
		seedIsOwner bool
	}{
		{&corev1.ServiceAccount{}, key, true},
		{&corev1.ConfigMap{}, key, true},
		{sts, key, true},
		{&corev1.Service{}, key, false},
		{&rbacv1.ClusterRole{}, clusterKey, false},
		{&rbacv1.ClusterRoleBinding{}, clusterKey, false},
	}

	var errs []error
	for _, o := range objects {
		if err := in.SeedClient.Get(ctx, o.key, o.obj); err != nil {
			errs = append(errs, fmt.Errorf("%T %s: %w", o.obj, prometheusName, err))
			continue
		}

		if value := o.obj.GetLabels()[nameLabel]; value != prometheusName {
			errs = append(errs, fmt.Errorf("%T %s: label %s is %q, expected %q", o.obj, prometheusName, nameLabel, value, prometheusName))
		}

		if o.seedIsOwner && !ownedBySeed(o.obj, seed) {
			errs = append(errs, fmt.Errorf("%T %s has no owner reference to Seed %s", o.obj, prometheusName, seed.Name))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}

	if sts.Status.ReadyReplicas < 1 {
		errs = append(errs, fmt.Errorf("StatefulSet %s has no ready replica", prometheusName))
	}

	retentionDays := metering.RetentionDays
	if retentionDays <= 0 {
		retentionDays = defaultRetentionDays
	}
	retentionFlag := fmt.Sprintf("--storage.tsdb.retention.time=%dd", retentionDays)
	if !containerHasArg(sts.Spec.Template.Spec.Containers, retentionFlag) {
		errs = append(errs, fmt.Errorf("StatefulSet %s does not run Prometheus with %s", prometheusName, retentionFlag))
	}

	errs = append(errs, checkPrometheusPVC(ctx, in, metering))

	return errors.Join(errs...)
}

func checkPrometheusPVC(ctx context.Context, in Input, metering *kubermaticv1.MeteringConfiguration) error {
	pvc := &corev1.PersistentVolumeClaim{}
	if err := in.SeedClient.Get(ctx, types.NamespacedName{Namespace: in.SeedNamespace, Name: prometheusPVCName}, pvc); err != nil {
		return fmt.Errorf("PVC %s: %w", prometheusPVCName, err)
	}

	var errs []error
	if pvc.Status.Phase != corev1.ClaimBound {
		errs = append(errs, fmt.Errorf("PVC %s is %s, expected Bound", prometheusPVCName, pvc.Status.Phase))
	}

	if class := ptrValue(pvc.Spec.StorageClassName); class != metering.StorageClassName {
		errs = append(errs, fmt.Errorf("PVC %s has storage class %q, the Seed configures %q", prometheusPVCName, class, metering.StorageClassName))
	}

	size := metering.StorageSize
	if size == "" {
		size = defaultStorageSize
	}

	expected, err := resource.ParseQuantity(size)
	if err != nil {
		return fmt.Errorf("Seed has an invalid storage size %q: %w", size, err)
	}

	if requested := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; requested.Cmp(expected) != 0 {
		errs = append(errs, fmt.Errorf("PVC %s requests %s, the Seed configures %s", prometheusPVCName, requested.String(), size))
	}

	return errors.Join(errs...)
}

// CheckReportCronJobs verifies that every report configuration of the Seed
// has a CronJob with the schedule, the arguments, the S3 environment and the
// CA bundle KKP documents, and that there is no CronJob for a report that is
// not configured.
func CheckReportCronJobs(ctx context.Context, in Input) error {
	seed, metering, err := in.meteringConfig(ctx)
	if err != nil {
		return err
	}

	caBundle, err := caBundleName(ctx, in)
	if err != nil {
		return err
	}

	cronJobs := &batchv1.CronJobList{}
	if err := in.SeedClient.List(ctx, cronJobs, ctrlruntimeclient.InNamespace(in.SeedNamespace), ctrlruntimeclient.MatchingLabels{componentLabel: componentValue}); err != nil {
		return fmt.Errorf("failed to list metering CronJobs: %w", err)
	}

	byName := map[string]*batchv1.CronJob{}
	for i := range cronJobs.Items {
		byName[cronJobs.Items[i].Name] = &cronJobs.Items[i]
	}

	var errs []error
	for report, config := range metering.ReportConfigurations {
		name := cronJobPrefix + report

		cronJob, ok := byName[name]
		if !ok {
			errs = append(errs, fmt.Errorf("report %s has no CronJob %s", report, name))
			continue
		}
		delete(byName, name)

		errs = append(errs, checkCronJob(cronJob, seed, report, config, caBundle))
	}

	for name := range byName {
		errs = append(errs, fmt.Errorf("CronJob %s has no report configuration in the Seed", name))
	}

	return errors.Join(errs...)
}

func checkCronJob(cronJob *batchv1.CronJob, seed *kubermaticv1.Seed, report string, config kubermaticv1.MeteringReportConfiguration, caBundle string) error {
	var errs []error

	if cronJob.Labels[nameLabel] != report {
		errs = append(errs, fmt.Errorf("CronJob %s: label %s is %q, expected %q", cronJob.Name, nameLabel, cronJob.Labels[nameLabel], report))
	}

	if cronJob.Spec.Schedule != config.Schedule {
		errs = append(errs, fmt.Errorf("CronJob %s: schedule %q, the Seed configures %q", cronJob.Name, cronJob.Spec.Schedule, config.Schedule))
	}

	if !ownedBySeed(cronJob, seed) {
		errs = append(errs, fmt.Errorf("CronJob %s has no owner reference to Seed %s", cronJob.Name, seed.Name))
	}

	podSpec := cronJob.Spec.JobTemplate.Spec.Template.Spec
	if len(podSpec.Containers) != 1 {
		return errors.Join(append(errs, fmt.Errorf("CronJob %s has %d containers, expected 1", cronJob.Name, len(podSpec.Containers)))...)
	}
	container := podSpec.Containers[0]

	if !strings.Contains(container.Image, "/kubermatic/metering:") {
		errs = append(errs, fmt.Errorf("CronJob %s runs image %s, expected the KKP metering image", cronJob.Name, container.Image))
	}

	if expected := expectedReportArgs(seed, report, config); !slices.Equal(container.Args, expected) {
		errs = append(errs, fmt.Errorf("CronJob %s has args %q, expected %q", cronJob.Name, container.Args, expected))
	}

	envKeys := map[string]string{
		"S3_ENDPOINT":       s3EndpointKey,
		"S3_BUCKET":         s3BucketKey,
		"ACCESS_KEY_ID":     s3AccessKeyKey,
		"SECRET_ACCESS_KEY": s3SecretKeyKey,
	}
	for _, env := range container.Env {
		key, ok := envKeys[env.Name]
		if !ok {
			continue
		}

		ref := env.ValueFrom
		if ref == nil || ref.SecretKeyRef == nil || ref.SecretKeyRef.Name != s3SecretName || ref.SecretKeyRef.Key != key {
			errs = append(errs, fmt.Errorf("CronJob %s: env %s does not come from Secret %s key %s", cronJob.Name, env.Name, s3SecretName, key))
		}
		delete(envKeys, env.Name)
	}
	for name := range envKeys {
		errs = append(errs, fmt.Errorf("CronJob %s has no env %s", cronJob.Name, name))
	}

	var caMounted bool
	for _, volume := range podSpec.Volumes {
		if volume.ConfigMap != nil && volume.ConfigMap.Name == caBundle {
			caMounted = true
		}
	}
	if !caMounted {
		errs = append(errs, fmt.Errorf("CronJob %s does not mount the CA bundle ConfigMap %s", cronJob.Name, caBundle))
	}

	return errors.Join(errs...)
}

// expectedReportArgs builds the metering tool arguments as documented for
// the report CronJobs, see pkg/ee/metering/cronjob.go.
func expectedReportArgs(seed *kubermaticv1.Seed, report string, config kubermaticv1.MeteringReportConfiguration) []string {
	args := []string{
		"--ca-bundle=/opt/ca-bundle/ca-bundle.pem",
		fmt.Sprintf("--prometheus-api=http://%s.%s.svc", prometheusName, seed.Namespace),
		"--output-dir=" + report,
		"--output-prefix=" + seed.Name,
	}

	if config.Format != "" {
		args = append(args, "--output-format="+string(config.Format))
	}

	if config.Monthly {
		args = append(args, "--last-month")
	} else {
		args = append(args, fmt.Sprintf("--last-number-of-days=%d", config.Interval))
	}

	return append(args, config.Types...)
}

// caBundleName returns the name of the CA bundle ConfigMap from the
// KubermaticConfiguration in the seed namespace.
func caBundleName(ctx context.Context, in Input) (string, error) {
	configs := &kubermaticv1.KubermaticConfigurationList{}
	if err := in.SeedClient.List(ctx, configs, ctrlruntimeclient.InNamespace(in.SeedNamespace)); err != nil {
		return "", fmt.Errorf("failed to list KubermaticConfigurations: %w", err)
	}

	if len(configs.Items) != 1 {
		return "", fmt.Errorf("expected one KubermaticConfiguration in %s, found %d", in.SeedNamespace, len(configs.Items))
	}

	if name := configs.Items[0].Spec.CABundle.Name; name != "" {
		return name, nil
	}

	return "ca-bundle", nil
}

// CheckPrometheusTargets verifies that the metering Prometheus scrapes the
// seed-controller-manager and federates every usage job from the user
// cluster's Prometheus.
func CheckPrometheusTargets(ctx context.Context, in Input) error {
	if err := in.requireCluster(); err != nil {
		return err
	}

	prom, err := meteringPrometheus(in)
	if err != nil {
		return err
	}

	namespace := in.Cluster.Status.NamespaceName

	// a new user cluster shows up with the next service discovery refresh
	return wait.PollImmediate(ctx, 10*time.Second, 3*time.Minute, func(ctx context.Context) (error, error) {
		targets, err := prom.Targets(ctx)
		if err != nil {
			return fmt.Errorf("failed to get targets: %w", err), nil
		}

		up := map[string]int{}
		var down []string
		for _, target := range targets.Active {
			job := string(target.Labels["job"])
			if job != seedControllerManagerJob && target.DiscoveredLabels["__meta_kubernetes_namespace"] != namespace {
				continue
			}

			if target.Health == "up" {
				up[job]++
			} else {
				down = append(down, fmt.Sprintf("%s %s: %s", job, target.ScrapeURL, target.LastError))
			}
		}

		var missing []string
		for _, job := range append([]string{seedControllerManagerJob}, federationJobs...) {
			if up[job] == 0 {
				missing = append(missing, job)
			}
		}

		if len(missing) > 0 {
			return fmt.Errorf("no healthy target for jobs %v (unhealthy: %v)", missing, down), nil
		}

		return nil, nil
	})
}

func ownedBySeed(obj metav1.Object, seed *kubermaticv1.Seed) bool {
	for _, ref := range obj.GetOwnerReferences() {
		if ref.Kind == "Seed" && ref.Name == seed.Name {
			return true
		}
	}

	return false
}

func containerHasArg(containers []corev1.Container, arg string) bool {
	for _, container := range containers {
		if slices.Contains(container.Args, arg) {
			return true
		}
	}

	return false
}

func ptrValue(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}
