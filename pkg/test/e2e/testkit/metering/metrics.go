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

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticv1helper "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1/helper"
	"k8c.io/kubermatic/v2/pkg/util/wait"
)

// usageMetric is a series the metering tool reads from the metering
// Prometheus. The selector is added to the cluster matcher and follows the
// tool's own queries.
type usageMetric struct {
	name     string
	selector string
}

// usageContract lists every usage series the metering tool v1.4.1 queries
// (pkg/datasource/prometheus.go in k8c.io/metering). The per-cluster
// Prometheus keep/drop filters and the metering federation jobs must let all
// of them through, or report columns silently turn to zero. The container and
// volume entries are narrowed to the probe workload, so the check proves the
// namespace and storage columns have data.
func usageContract() []usageMetric {
	workload := fmt.Sprintf(`namespace=%q,container=~".+",image=~".+"`, workloadNamespace)

	return []usageMetric{
		{name: "kube_node_info"},
		{name: "node_cpu_usage_seconds_total"},
		{name: "machine_cpu_cores"},
		{name: "machine_memory_bytes"},
		{name: "node_memory_working_set_bytes"},
		{name: "container_cpu_usage_seconds_total", selector: workload},
		{name: "container_memory_working_set_bytes", selector: workload},
		{name: "kubelet_volume_stats_used_bytes", selector: fmt.Sprintf(`namespace=%q,persistentvolumeclaim=%q`, workloadNamespace, workloadName)},
	}
}

// contractTimeout covers a fresh probe workload: kubelet volume stats, the
// user cluster Prometheus scrape and the federation scrape.
const contractTimeout = 8 * time.Minute

// CheckClusterMetadataMetrics verifies the kubermatic_* metrics the metering
// tool uses for the project and cluster columns, and that their labels match
// the cluster and project objects.
func CheckClusterMetadataMetrics(ctx context.Context, in Input) error {
	if err := in.requireCluster(); err != nil {
		return err
	}

	prom, err := meteringPrometheus(in)
	if err != nil {
		return err
	}

	cluster := in.Cluster
	project := in.Project

	return wait.PollImmediate(ctx, 15*time.Second, 3*time.Minute, func(ctx context.Context) (error, error) {
		var errs []error

		info, err := singleSeries(ctx, prom, fmt.Sprintf(`kubermatic_cluster_info{name=%q}`, cluster.Name))
		if err != nil {
			errs = append(errs, err)
		} else {
			errs = append(errs,
				expectLabel(info, "display_name", cluster.Spec.HumanReadableName),
				expectLabel(info, "project", project.Name),
				expectLabel(info, "phase", string(kubermaticv1.ClusterRunning)),
				expectLabel(info, "datacenter", cluster.Spec.Cloud.DatacenterName),
			)
			if provider, err := kubermaticv1helper.ClusterCloudProviderName(cluster.Spec.Cloud); err != nil {
				errs = append(errs, err)
			} else {
				errs = append(errs, expectLabel(info, "cloud_provider", provider))
			}
		}

		// the owner comes from status.userEmail, which clusters created
		// without the dashboard (for example by the e2e jig) do not have
		owner, err := singleSeries(ctx, prom, fmt.Sprintf(`kubermatic_cluster_owner{cluster_name=%q}`, cluster.Name))
		if err != nil {
			errs = append(errs, err)
		} else if hasOwner := owner.Metric["user"] != ""; hasOwner != (cluster.Status.UserEmail != "") {
			errs = append(errs, fmt.Errorf("kubermatic_cluster_owner has user %q, the cluster has owner email %q", owner.Metric["user"], cluster.Status.UserEmail))
		}

		created, err := singleSeries(ctx, prom, fmt.Sprintf(`kubermatic_cluster_created{cluster=%q}`, cluster.Name))
		if err != nil {
			errs = append(errs, err)
		} else if int64(created.Value) != cluster.CreationTimestamp.Unix() {
			errs = append(errs, fmt.Errorf("kubermatic_cluster_created is %d, the cluster was created at %d", int64(created.Value), cluster.CreationTimestamp.Unix()))
		}

		if _, err := singleSeries(ctx, prom, fmt.Sprintf(`kubermatic_cluster_labels{name=%q}`, cluster.Name)); err != nil {
			errs = append(errs, err)
		}

		projectInfo, err := singleSeries(ctx, prom, fmt.Sprintf(`kubermatic_project_info{name=%q}`, project.Name))
		if err != nil {
			errs = append(errs, err)
		} else {
			errs = append(errs, expectLabel(projectInfo, "display_name", project.Spec.Name))
		}

		if _, err := singleSeries(ctx, prom, fmt.Sprintf(`kubermatic_project_labels{name=%q}`, project.Name)); err != nil {
			errs = append(errs, err)
		}

		deleted, err := queryVector(ctx, prom, fmt.Sprintf(`kubermatic_cluster_deleted{cluster=%q}`, cluster.Name), time.Now())
		if err != nil {
			errs = append(errs, err)
		} else if len(deleted) > 0 {
			errs = append(errs, errors.New("kubermatic_cluster_deleted exists for a cluster that is not being deleted"))
		}

		return errors.Join(errs...), nil
	})
}

// CheckUsageMetricContract verifies that every usage series in usageContract
// exists in the metering Prometheus for the user cluster.
func CheckUsageMetricContract(ctx context.Context, in Input) error {
	if err := in.requireCluster(); err != nil {
		return err
	}

	prom, err := meteringPrometheus(in)
	if err != nil {
		return err
	}

	return wait.PollImmediate(ctx, 30*time.Second, contractTimeout, func(ctx context.Context) (error, error) {
		var missing []string

		for _, metric := range usageContract() {
			selector := fmt.Sprintf(`cluster=%q`, in.Cluster.Name)
			if metric.selector != "" {
				selector += "," + metric.selector
			}
			query := fmt.Sprintf("%s{%s}", metric.name, selector)

			vector, err := queryVector(ctx, prom, query, time.Now())
			if err != nil {
				return err, nil
			}
			if len(vector) == 0 {
				missing = append(missing, query)
			}
		}

		if len(missing) > 0 {
			return fmt.Errorf("the metering Prometheus has no series for %v", missing), nil
		}

		return nil, nil
	})
}

// CheckNoDuplicateContainerSeries verifies that every container has one
// series per container metric, although two federation jobs pull the
// container metrics. Duplicates would inflate the namespace report.
func CheckNoDuplicateContainerSeries(ctx context.Context, in Input) error {
	if err := in.requireCluster(); err != nil {
		return err
	}

	prom, err := meteringPrometheus(in)
	if err != nil {
		return err
	}

	// without any container series the duplicate query below is empty and
	// would pass, so first make sure the series are there
	if err := CheckUsageMetricContract(ctx, in); err != nil {
		return err
	}

	var errs []error
	for _, metric := range []string{"container_cpu_usage_seconds_total", "container_memory_working_set_bytes"} {
		query := fmt.Sprintf(`count by (namespace, pod, container, id, cpu) (%s{cluster=%q,container=~".+",image=~".+"}) > 1`, metric, in.Cluster.Name)

		vector, err := queryVector(ctx, prom, query, time.Now())
		if err != nil {
			errs = append(errs, err)
			continue
		}

		for _, sample := range vector {
			errs = append(errs, fmt.Errorf("%s has %v series for %s", metric, sample.Value, sample.Metric))
		}
	}

	return errors.Join(errs...)
}

// CheckLabelValueOrder verifies that project labels keep their values in
// kubermatic_project_labels when sanitizing the keys changes their sort
// order: "metering-e2e-x" sorts before "metering-e2e1", but
// "label_metering_e2e_x" sorts after "label_metering_e2e1". Cluster labels
// go through the same code, but they are also copied to the user cluster's
// nodes and stay there, so the check uses the project.
func CheckLabelValueOrder(ctx context.Context, in Input) error {
	if err := in.requireCluster(); err != nil {
		return err
	}

	prom, err := meteringPrometheus(in)
	if err != nil {
		return err
	}

	project := &kubermaticv1.Project{}
	project.Name = in.Project.Name

	revert, err := patchWithRevert(ctx, in.SeedClient, project, func() {
		if project.Labels == nil {
			project.Labels = map[string]string{}
		}
		project.Labels["metering-e2e-x"] = "dash"
		project.Labels["metering-e2e1"] = "digit"
	})
	if err != nil {
		return err
	}
	defer func() {
		ctx, cancel := revertContext()
		defer cancel()

		if err := revert(ctx); err != nil {
			in.Log.Warnw("Failed to remove the test labels from the project", "error", err)
		}
	}()

	var last model.Metric
	err = wait.PollImmediate(ctx, 15*time.Second, 3*time.Minute, func(ctx context.Context) (error, error) {
		series, err := singleSeries(ctx, prom, fmt.Sprintf(`kubermatic_project_labels{name=%q}`, in.Project.Name))
		if err != nil {
			return err, nil
		}
		last = series.Metric

		// with the bug, this label carries the other value, which still
		// ends the wait
		if series.Metric["label_metering_e2e_x"] == "" {
			return errors.New("the test labels are not scraped yet"), nil
		}

		return nil, nil
	})
	if err != nil {
		return err
	}

	return errors.Join(
		expectMetricLabel(last, "label_metering_e2e_x", "dash"),
		expectMetricLabel(last, "label_metering_e2e1", "digit"),
	)
}

// singleSeries runs an instant query that must return at least one series
// and returns the first.
func singleSeries(ctx context.Context, prom promv1.API, query string) (*model.Sample, error) {
	vector, err := queryVector(ctx, prom, query, time.Now())
	if err != nil {
		return nil, err
	}

	if len(vector) == 0 {
		return nil, fmt.Errorf("no series for %s", query)
	}

	return vector[0], nil
}

func expectLabel(sample *model.Sample, name, expected string) error {
	return expectMetricLabel(sample.Metric, name, expected)
}

func expectMetricLabel(metric model.Metric, name, expected string) error {
	if value := string(metric[model.LabelName(name)]); value != expected {
		return fmt.Errorf("%s: label %s is %q, expected %q", metric[model.MetricNameLabel], name, value, expected)
	}

	return nil
}
