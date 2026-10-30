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
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"

	"k8s.io/apimachinery/pkg/api/resource"
)

// The queries of the metering tool v1.4.1 (pkg/datasource/prometheus.go in
// k8c.io/metering). The checks run them themselves to compute the values a
// report must contain. A tool bump that changes how values are computed
// makes the value checks fail; update these queries together with the
// review of the change.
const (
	clusterInfoQuery    = `last_over_time(kubermatic_cluster_info{name="%s",phase="Running"}[$range])`
	clusterOwnerQuery   = `last_over_time(kubermatic_cluster_owner{cluster_name="%s"}[$range])`
	clusterCreatedQuery = `last_over_time(kubermatic_cluster_created{cluster="%s"}[$range])`
	clusterDeletedQuery = `last_over_time(kubermatic_cluster_deleted{cluster="%s"}[$range])`
	clusterLabelsQuery  = `last_over_time(kubermatic_cluster_labels{name="%s"}[$range])`
	projectInfoQuery    = `last_over_time(kubermatic_project_info{name="%s"}[$range])`
	projectLabelsQuery  = `last_over_time(kubermatic_project_labels{name="%s"}[$range])`

	averageKubeNodes              = `avg_over_time(sum(kube_node_info{cluster="%s"} or on() vector(0))[$range:])`
	averageNodeCPUUsage           = `sum(rate(node_cpu_usage_seconds_total{cluster="%s"}[$range]))`
	averageMachineCPUCores        = `avg_over_time((sum(machine_cpu_cores{cluster="%s"} or on() vector(0)))[$range:])`
	averageMachineMemoryBytes     = `avg_over_time((sum(machine_memory_bytes{cluster="%s"} or on() vector(0)))[$range:])`
	averageNodeUsedMemoryBytes    = `avg_over_time((sum(node_memory_working_set_bytes{cluster="%s"} or on() vector(0)))[$range:])`
	averageKubeletVolumeUsedBytes = `avg_over_time((sum by (cluster) (kubelet_volume_stats_used_bytes{cluster="%s", persistentvolumeclaim!=""}))[$range:])`

	averageContainerCPUUsage        = `sum(rate(container_cpu_usage_seconds_total{cluster="%s",namespace="%s",container=~".+",image=~".+"}[$range]))`
	averageContainerUsedMemoryBytes = `avg_over_time((sum(container_memory_working_set_bytes{cluster="%s",namespace="%s",container=~".+",image=~".+"} or on() vector(0)))[$range:])`
)

// Report columns of the metering tool v1.4.1. The tool treats the column
// order as its API: columns may be appended, but never moved or removed,
// because consumers read the files by position.
var (
	clusterReportColumns = []string{
		"project-name",
		"project-id",
		"project-labels",
		"cluster-name",
		"cluster-id",
		"cluster-labels",
		"average-cluster-machines",
		"average-available-cpu-millicores",
		"average-used-cpu-millicores",
		"average-available-memory-bytes",
		"average-used-memory-bytes",
		"created-at",
		"deleted-at",
		"cloud-provider",
		"datacenter",
		"cluster-owner",
		"average-persistent-storage-usage-bytes",
	}

	namespaceReportColumns = []string{
		"project-name",
		"project-id",
		"project-labels",
		"cluster-name",
		"cluster-id",
		"cluster-labels",
		"namespace-name",
		"average-used-cpu-millicores",
		"average-used-memory-bytes",
		"cluster-owner",
	}
)

// labelsFilteredFromReports are the metric labels the tool leaves out of the
// label columns.
var labelsFilteredFromReports = []string{"__name__", "instance", "job", "name", "label_project_id", "label_is_credential_preset"}

// window is the time range of a report. The tool evaluates every query at
// end; usage queries run once per full day of the window, each over 24h.
type window struct {
	start, end time.Time
}

func (w window) rangeExpr() string {
	return fmt.Sprintf("%dm", int(w.end.Sub(w.start).Minutes()))
}

func (w window) chunks() int {
	chunks := int(w.end.Sub(w.start) / (24 * time.Hour))
	return max(chunks, 1)
}

// toolQuerier runs queries the way the metering tool does.
type toolQuerier struct {
	prom promv1.API
	w    window
}

func (q toolQuerier) query(ctx context.Context, query string, at time.Time, rangeExpr string) (model.Vector, error) {
	return queryVector(ctx, q.prom, strings.ReplaceAll(query, "$range", rangeExpr), at)
}

// latest returns the sample with the greatest timestamp, the first one when
// they are equal, as the tool's consumeLatest does.
func (q toolQuerier) latest(ctx context.Context, query string) (*model.Sample, error) {
	vector, err := q.query(ctx, query, q.w.end, q.w.rangeExpr())
	if err != nil {
		return nil, err
	}

	var latest *model.Sample
	for _, sample := range vector {
		if latest == nil || sample.Timestamp > latest.Timestamp {
			latest = sample
		}
	}

	return latest, nil
}

func (q toolQuerier) sum(ctx context.Context, query string, at time.Time, rangeExpr string, scale float64) (uint64, error) {
	vector, err := q.query(ctx, query, at, rangeExpr)
	if err != nil {
		return 0, err
	}

	var total uint64
	for _, sample := range vector {
		total += uint64(float64(sample.Value) * scale)
	}

	return total, nil
}

// dailyAverage sums the query over every full day of the window, going back
// from the end, and divides by the number of days with integer division.
func (q toolQuerier) dailyAverage(ctx context.Context, query string, scale float64) (uint64, error) {
	chunks := q.w.chunks()
	at := q.w.end

	var total uint64
	for range chunks {
		value, err := q.sum(ctx, query, at, "1440m", scale)
		if err != nil {
			return 0, err
		}
		total += value
		at = at.Add(-24 * time.Hour)
	}

	return total / uint64(chunks), nil
}

// clusterRecord holds the columns both report types share.
type clusterRecord struct {
	projectName, projectID, projectLabels string
	clusterName, clusterID, clusterLabels string
	cloudProvider, datacenter, owner      string
	created, deleted                      string
	machines                              string
}

// expectedClusterRecord computes the shared columns of a cluster. ok is
// false when the tool would leave the cluster out, because it was not
// Running with a display name in the window.
func (q toolQuerier) expectedClusterRecord(ctx context.Context, clusterID string) (record clusterRecord, ok bool, err error) {
	record.clusterID = clusterID

	info, err := q.latest(ctx, fmt.Sprintf(clusterInfoQuery, clusterID))
	if err != nil {
		return record, false, err
	}
	if info == nil || info.Metric["display_name"] == "" {
		return record, false, nil
	}

	record.clusterName = string(info.Metric["display_name"])
	record.projectID = string(info.Metric["project"])
	record.cloudProvider = string(info.Metric["cloud_provider"])
	record.datacenter = string(info.Metric["datacenter"])

	if owner, err := q.latest(ctx, fmt.Sprintf(clusterOwnerQuery, clusterID)); err != nil {
		return record, false, err
	} else if owner != nil {
		record.owner = string(owner.Metric["user"])
	}

	if project, err := q.latest(ctx, fmt.Sprintf(projectInfoQuery, record.projectID)); err != nil {
		return record, false, err
	} else if project != nil {
		record.projectName = string(project.Metric["display_name"])
	}

	if created, err := q.latest(ctx, fmt.Sprintf(clusterCreatedQuery, clusterID)); err != nil {
		return record, false, err
	} else if created != nil {
		record.created = formatUnix(float64(created.Value))
	}

	if deleted, err := q.latest(ctx, fmt.Sprintf(clusterDeletedQuery, clusterID)); err != nil {
		return record, false, err
	} else if deleted != nil {
		record.deleted = formatUnix(float64(deleted.Value))
	}

	if labels, err := q.latest(ctx, fmt.Sprintf(clusterLabelsQuery, clusterID)); err != nil {
		return record, false, err
	} else if labels != nil {
		record.clusterLabels = flattenLabels(labels.Metric)
	}

	if labels, err := q.latest(ctx, fmt.Sprintf(projectLabelsQuery, record.projectID)); err != nil {
		return record, false, err
	} else if labels != nil {
		record.projectLabels = flattenLabels(labels.Metric)
	}

	nodes, err := q.query(ctx, fmt.Sprintf(averageKubeNodes, clusterID), q.w.end, q.w.rangeExpr())
	if err != nil {
		return record, false, err
	}
	var machines float64
	for _, sample := range nodes {
		machines += float64(sample.Value)
	}
	record.machines = strconv.FormatFloat(machines, 'f', 2, 64)

	return record, true, nil
}

// expectedClusterRow computes the cluster report row of a cluster. ok is
// false when the tool leaves the cluster out.
func (q toolQuerier) expectedClusterRow(ctx context.Context, clusterID string) ([]string, bool, error) {
	record, ok, err := q.expectedClusterRecord(ctx, clusterID)
	if err != nil || !ok {
		return nil, ok, err
	}

	usage := map[string]uint64{}
	for name, query := range map[string]struct {
		query string
		scale float64
	}{
		"usedCPU":         {averageNodeCPUUsage, 1e9},
		"availableCPU":    {averageMachineCPUCores, 1e9},
		"usedMemory":      {averageNodeUsedMemoryBytes, 1},
		"availableMemory": {averageMachineMemoryBytes, 1},
		"storage":         {averageKubeletVolumeUsedBytes, 1},
	} {
		value, err := q.dailyAverage(ctx, fmt.Sprintf(query.query, clusterID), query.scale)
		if err != nil {
			return nil, false, err
		}
		usage[name] = value
	}

	return []string{
		record.projectName,
		record.projectID,
		record.projectLabels,
		record.clusterName,
		record.clusterID,
		record.clusterLabels,
		record.machines,
		milliCores(usage["availableCPU"]),
		milliCores(usage["usedCPU"]),
		bytesValue(usage["availableMemory"]),
		bytesValue(usage["usedMemory"]),
		record.created,
		record.deleted,
		record.cloudProvider,
		record.datacenter,
		record.owner,
		strconv.FormatUint(usage["storage"], 10),
	}, true, nil
}

// expectedNamespaceRows computes the namespace report rows of a cluster,
// sorted by namespace.
func (q toolQuerier) expectedNamespaceRows(ctx context.Context, clusterID string) ([][]string, error) {
	record, ok, err := q.expectedClusterRecord(ctx, clusterID)
	if err != nil || !ok {
		return nil, err
	}

	values, _, err := q.prom.LabelValues(ctx, "namespace", []string{fmt.Sprintf(`container_cpu_usage_seconds_total{cluster=%q}`, clusterID)}, q.w.start, q.w.end)
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	var rows [][]string
	for _, value := range values {
		namespace := strings.TrimPrefix(string(value), "cluster-")
		if namespace == clusterID || namespace == "" {
			continue
		}

		cpu, err := q.dailyAverage(ctx, fmt.Sprintf(averageContainerCPUUsage, clusterID, namespace), 1e9)
		if err != nil {
			return nil, err
		}

		memory, err := q.dailyAverage(ctx, fmt.Sprintf(averageContainerUsedMemoryBytes, clusterID, namespace), 1)
		if err != nil {
			return nil, err
		}

		rows = append(rows, []string{
			record.projectName,
			record.projectID,
			record.projectLabels,
			record.clusterName,
			record.clusterID,
			record.clusterLabels,
			namespace,
			milliCores(cpu),
			bytesValue(memory),
			record.owner,
		})
	}

	sortRows(rows, 6)

	return rows, nil
}

// milliCores formats nano cores as the tool does: as a resource.Quantity,
// whose MilliValue rounds up.
func milliCores(nanoCores uint64) string {
	return strconv.FormatInt(resource.NewScaledQuantity(int64(nanoCores), -9).MilliValue(), 10)
}

func bytesValue(value uint64) string {
	return strconv.FormatInt(resource.NewScaledQuantity(int64(value), 0).Value(), 10)
}

// formatUnix formats a Unix timestamp as RFC3339 in UTC; the tool runs in
// UTC.
func formatUnix(value float64) string {
	return time.Unix(int64(value), 0).UTC().Format(time.RFC3339)
}

// flattenLabels returns the sorted "key=value" list of the label columns.
func flattenLabels(metric model.Metric) string {
	var pairs []string
	for name, value := range metric {
		if slices.Contains(labelsFilteredFromReports, string(name)) {
			continue
		}
		pairs = append(pairs, fmt.Sprintf("%s=%s", name, value))
	}

	sort.Strings(pairs)

	return strings.Join(pairs, ",")
}

func flattenMap(labels map[string]string) string {
	metric := model.Metric{}
	for name, value := range labels {
		metric[model.LabelName(name)] = model.LabelValue(value)
	}

	return flattenLabels(metric)
}

func sortRows(rows [][]string, column int) {
	sort.Slice(rows, func(i, j int) bool {
		return rows[i][column] < rows[j][column]
	})
}
