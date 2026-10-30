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
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	promv1 "github.com/prometheus/client_golang/api/prometheus/v1"

	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/rand"
)

const (
	csvContentType  = "text/csv"
	jsonContentType = "text/jsonl"

	// ingestDelay is how far the end of a test report lies in the past, so
	// every sample up to the end has been scraped and federated before the
	// tool and the checks read it.
	ingestDelay = 3 * time.Minute

	// subqueryStep is the evaluation interval of the metering Prometheus,
	// which the tool's subqueries use as step.
	subqueryStep = 5 * time.Minute

	// testWindow is the length of the test reports. It only bounds the
	// metadata queries and the namespace list; usage is always read over a
	// day, as in the tool.
	testWindow = time.Hour
)

// reportSet is one run of the metering tool over a test window, in both
// formats and for both report types.
type reportSet struct {
	window        window
	clusterCSV    [][]string
	namespaceCSV  [][]string
	clusterJSON   [][]byte
	namespaceJSON [][]byte
}

var (
	reportSetsLock sync.Mutex
	reportSets     = map[string]*reportSet{}
)

// testReports runs the metering tool once per process and report prefix over
// a test window that ends shortly before now, and returns the parsed
// reports. The value checks share the run, because each tool run takes about
// a minute.
func testReports(ctx context.Context, in Input) (*reportSet, error) {
	reportSetsLock.Lock()
	defer reportSetsLock.Unlock()

	if set, ok := reportSets[in.ReportPrefix]; ok {
		return set, nil
	}

	// the probe workload has to be in the metering Prometheus before the
	// window ends, or its namespace and volume are missing from the reports
	if err := waitForWorkloadData(ctx, in); err != nil {
		return nil, fmt.Errorf("usage metrics are incomplete, cannot generate meaningful reports: %w", err)
	}

	// the memory and storage averages are subqueries with the 5m evaluation
	// interval as step, aligned to the epoch, so the window has to end on a
	// step that lies after the workload's first samples
	end := time.Now().UTC().Add(time.Minute).Truncate(subqueryStep).Add(subqueryStep)
	w := window{start: end.Add(-testWindow), end: end}

	// wait until the samples up to the end are ingested
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(time.Until(end) + ingestDelay):
	}

	base, err := toolArgs(ctx, in, in.CSVReport())
	if err != nil {
		return nil, err
	}

	seedName := in.SeedName
	set := &reportSet{window: w}

	// every Ginkgo process runs the tool on its own, so each run needs its own
	// directory, or a run for another window could replace the files in between
	dir := fmt.Sprintf("%s/%s-%s", in.adhocDir(), w.end.Format("20060102T1504"), rand.String(5))

	s3, err := meteringBucket(ctx, in)
	if err != nil {
		return nil, err
	}
	defer s3.stop()

	for _, format := range []string{"csv", "json"} {
		args := append(slices.Clone(base),
			"--output-dir="+dir,
			"--output-format="+format,
			"--start="+w.start.Format(time.RFC3339),
			"--end="+w.end.Format(time.RFC3339),
			"cluster", "namespace",
		)

		if _, err := runReportJob(ctx, in, in.CSVReport(), args); err != nil {
			return nil, fmt.Errorf("%s report run failed: %w", format, err)
		}

		for _, reportType := range []string{"cluster", "namespace"} {
			object, err := s3.get(ctx, reportKey(dir, seedName, reportType, w.start, w.end, format))
			if err != nil {
				return nil, err
			}

			switch {
			case format == "csv" && reportType == "cluster":
				set.clusterCSV, err = parseCSV(object.body)
			case format == "csv":
				set.namespaceCSV, err = parseCSV(object.body)
			case reportType == "cluster":
				set.clusterJSON, err = parseJSONLines(object.body)
			default:
				set.namespaceJSON, err = parseJSONLines(object.body)
			}
			if err != nil {
				return nil, fmt.Errorf("%s: %w", object.key, err)
			}
		}
	}

	reportSets[in.ReportPrefix] = set

	return set, nil
}

// waitForWorkloadData waits until the metering Prometheus has enough of the
// probe workload's data for non-zero report values: a few CPU samples for
// rate() and the written data on the PVC.
func waitForWorkloadData(ctx context.Context, in Input) error {
	if err := CheckUsageMetricContract(ctx, in); err != nil {
		return err
	}

	prom, err := meteringPrometheus(in)
	if err != nil {
		return err
	}

	queries := []string{
		fmt.Sprintf(`count_over_time(container_cpu_usage_seconds_total{cluster=%q,namespace=%q,container=~".+",image=~".+"}[10m]) >= 3`, in.Cluster.Name, workloadNamespace),
		fmt.Sprintf(`kubelet_volume_stats_used_bytes{cluster=%q,namespace=%q,persistentvolumeclaim=%q} >= %d`, in.Cluster.Name, workloadNamespace, workloadName, workloadDataMiB<<20),
	}

	return wait.PollImmediate(ctx, 15*time.Second, contractTimeout, func(ctx context.Context) (error, error) {
		for _, query := range queries {
			vector, err := queryVector(ctx, prom, query, time.Now())
			if err != nil {
				return err, nil
			}
			if len(vector) == 0 {
				return fmt.Errorf("not enough data yet for %s", query), nil
			}
		}

		return nil, nil
	})
}

// CheckScheduledReports runs each of the suite's report CronJobs once and
// verifies that both report types land in the bucket under the documented
// key, with the content type of the format and a valid file. The scheduled
// window ends at midnight UTC, so the rows depend on what ran yesterday and
// are not checked.
func CheckScheduledReports(ctx context.Context, in Input) error {
	s3, err := meteringBucket(ctx, in)
	if err != nil {
		return err
	}
	defer s3.stop()

	_, metering, err := in.meteringConfig(ctx)
	if err != nil {
		return err
	}

	var errs []error
	for _, report := range []string{in.CSVReport(), in.JSONReport()} {
		config, ok := metering.ReportConfigurations[report]
		if !ok {
			errs = append(errs, fmt.Errorf("the Seed has no report %s", report))
			continue
		}

		started, err := runReportJob(ctx, in, report, nil)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		format := string(config.Format)
		contentType := csvContentType
		if format == "json" {
			contentType = jsonContentType
		}

		for _, reportType := range config.Types {
			object, err := scheduledReport(ctx, s3, in, report, reportType, format, started, int(config.Interval))
			if err != nil {
				errs = append(errs, fmt.Errorf("report %s did not write the expected object: %w", report, err))
				continue
			}
			key := object.key

			// a Job that wrote nothing would still find an older object
			if object.lastModified.Before(started.Add(-time.Minute)) {
				errs = append(errs, fmt.Errorf("%s was last written at %s, before the report run at %s", key, object.lastModified, started))
			}

			if object.contentType != contentType {
				errs = append(errs, fmt.Errorf("%s has content type %q, expected %q", key, object.contentType, contentType))
			}

			if format == "json" {
				if _, err := parseJSONLines(object.body); err != nil {
					errs = append(errs, fmt.Errorf("%s: %w", key, err))
				}
				continue
			}

			rows, err := parseCSV(object.body)
			if err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", key, err))
				continue
			}

			errs = append(errs, checkHeader(key, rows, columnsOf(reportType)))
		}
	}

	return errors.Join(errs...)
}

// scheduledReport downloads the report a scheduled run wrote. The tool takes
// the date from its own clock, which can already be the next day when the Job
// started just before midnight UTC.
func scheduledReport(ctx context.Context, s3 *s3Bucket, in Input, report, reportType, format string, started time.Time, days int) (*s3Object, error) {
	end := started.Truncate(24 * time.Hour)

	object, err := s3.get(ctx, reportKey(report, in.SeedName, reportType, end.AddDate(0, 0, -days), end, format))
	if err == nil {
		return object, nil
	}

	next := end.AddDate(0, 0, 1)
	if object, nextErr := s3.get(ctx, reportKey(report, in.SeedName, reportType, next.AddDate(0, 0, -days), next, format)); nextErr == nil {
		return object, nil
	}

	return nil, err
}

// CheckReportFormat verifies the format consumers depend on: the CSV columns
// of both report types in their documented order, and every field of the
// JSON reports.
func CheckReportFormat(ctx context.Context, in Input) error {
	if err := in.requireCluster(); err != nil {
		return err
	}

	set, err := testReports(ctx, in)
	if err != nil {
		return err
	}

	errs := []error{
		checkHeader("cluster CSV report", set.clusterCSV, clusterReportColumns),
		checkHeader("namespace CSV report", set.namespaceCSV, namespaceReportColumns),
	}

	clusterRow, err := findJSONRow(set.clusterJSON, in.Cluster.Name, "")
	if err != nil {
		errs = append(errs, fmt.Errorf("cluster JSON report: %w", err))
	} else {
		errs = append(errs, checkJSONFields("cluster JSON report", clusterRow, clusterJSONFields))
	}

	namespaceRow, err := findJSONRow(set.namespaceJSON, in.Cluster.Name, workloadNamespace)
	if err != nil {
		errs = append(errs, fmt.Errorf("namespace JSON report: %w", err))
	} else {
		errs = append(errs, checkJSONFields("namespace JSON report", namespaceRow, namespaceJSONFields))
	}

	return errors.Join(errs...)
}

// CheckClusterReportValues verifies the user cluster's row in the cluster
// report: the metadata columns against the cluster and project objects, and
// every column against the value the documented queries give for the same
// window.
func CheckClusterReportValues(ctx context.Context, in Input) error {
	if err := in.requireCluster(); err != nil {
		return err
	}

	set, err := testReports(ctx, in)
	if err != nil {
		return err
	}

	row, err := findRow(set.clusterCSV, 4, in.Cluster.Name)
	if err != nil {
		return fmt.Errorf("cluster CSV report: %w", err)
	}

	prom, err := meteringPrometheus(in)
	if err != nil {
		return err
	}

	expected, ok, err := toolQuerier{prom: prom, w: set.window}.expectedClusterRow(ctx, in.Cluster.Name)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("the metering Prometheus has no Running sample of the cluster in the report window, but the report has a row")
	}

	errs := []error{compareRow("cluster report", clusterReportColumns, row, expected)}

	project := in.Project
	cluster := in.Cluster
	for column, value := range map[string]string{
		"project-name": project.Spec.Name,
		"project-id":   project.Name,
		"cluster-name": cluster.Spec.HumanReadableName,
		"datacenter":   cluster.Spec.Cloud.DatacenterName,
		"created-at":   cluster.CreationTimestamp.UTC().Format(time.RFC3339),
		"deleted-at":   "",
	} {
		if actual := row[slices.Index(clusterReportColumns, column)]; actual != value {
			errs = append(errs, fmt.Errorf("cluster report: %s is %q, the cluster objects say %q", column, actual, value))
		}
	}

	errs = append(errs, checkUsagePlausible(ctx, in, row))

	return errors.Join(errs...)
}

// checkUsagePlausible compares the usage columns with the user cluster's
// nodes. The averages run over a day and count the time before the cluster
// existed as zero, so for a cluster younger than a day they can only be
// below the current capacity. For older clusters, nodes removed during the
// day can push the averages above it, so only the lower bounds apply.
func checkUsagePlausible(ctx context.Context, in Input, row []string) error {
	nodes := &corev1.NodeList{}
	if err := in.UserClient.List(ctx, nodes); err != nil {
		return fmt.Errorf("failed to list user cluster nodes: %w", err)
	}

	var cpu, memory resource.Quantity
	for _, node := range nodes.Items {
		cpu.Add(node.Status.Capacity[corev1.ResourceCPU])
		memory.Add(node.Status.Capacity[corev1.ResourceMemory])
	}

	value := func(column string) int64 {
		v, _ := strconv.ParseInt(row[slices.Index(clusterReportColumns, column)], 10, 64)
		return v
	}

	if len(nodes.Items) == 0 {
		return nil
	}

	var errs []error
	machines, _ := strconv.ParseFloat(row[slices.Index(clusterReportColumns, "average-cluster-machines")], 64)
	if machines <= 0 {
		errs = append(errs, fmt.Errorf("average-cluster-machines is %.2f with %d nodes", machines, len(nodes.Items)))
	}

	young := time.Since(in.Cluster.CreationTimestamp.Time) < 24*time.Hour

	for _, check := range []struct {
		column   string
		capacity int64
	}{
		{"average-available-cpu-millicores", cpu.MilliValue()},
		{"average-used-cpu-millicores", cpu.MilliValue()},
		{"average-available-memory-bytes", memory.Value()},
		{"average-used-memory-bytes", memory.Value()},
	} {
		// cadvisor and the kubelet may report the capacity slightly
		// differently, so allow 2%
		if v := value(check.column); v <= 0 || (young && v > check.capacity*102/100) {
			errs = append(errs, fmt.Errorf("%s is %d, the nodes have a capacity of %d", check.column, v, check.capacity))
		}
	}

	return errors.Join(errs...)
}

// CheckNamespaceReportValues verifies the user cluster's rows in the
// namespace report: the probe workload's namespace has usage, the control
// plane namespace is left out, and every row has the value the documented
// queries give for the same window.
func CheckNamespaceReportValues(ctx context.Context, in Input) error {
	if err := in.requireCluster(); err != nil {
		return err
	}

	set, err := testReports(ctx, in)
	if err != nil {
		return err
	}

	if len(set.namespaceCSV) == 0 {
		return errors.New("namespace CSV report is empty")
	}

	var rows [][]string
	for _, row := range set.namespaceCSV[1:] {
		if row[4] == in.Cluster.Name {
			rows = append(rows, row)
		}
	}
	sortRows(rows, 6)

	prom, err := meteringPrometheus(in)
	if err != nil {
		return err
	}

	expected, err := toolQuerier{prom: prom, w: set.window}.expectedNamespaceRows(ctx, in.Cluster.Name)
	if err != nil {
		return err
	}

	var errs []error

	// the expected list is read after the tool ran, so it can hold namespaces
	// that got their first series in between; every reported namespace must
	// be in it with the same values
	for _, row := range rows {
		want, err := findRow(expected, 6, row[6])
		if err != nil {
			errs = append(errs, fmt.Errorf("namespace report lists %s, which the documented queries do not return", row[6]))
			continue
		}

		errs = append(errs, compareRow("namespace report "+row[6], namespaceReportColumns, row, want))
	}

	for _, namespace := range []string{"kube-system", workloadNamespace} {
		row, err := findRow(rows, 6, namespace)
		if err != nil {
			errs = append(errs, fmt.Errorf("namespace report: %w", err))
			continue
		}

		for _, column := range []int{7, 8} {
			if row[column] == "0" {
				errs = append(errs, fmt.Errorf("namespace report: %s of %s is 0", namespaceReportColumns[column], namespace))
			}
		}
	}

	// the tool strips the "cluster-" prefix and skips the control plane
	// namespace by the cluster name
	if _, err := findRow(rows, 6, in.Cluster.Name); err == nil {
		errs = append(errs, fmt.Errorf("namespace report lists the control plane namespace of cluster %s", in.Cluster.Name))
	}

	return errors.Join(errs...)
}

// CheckStorageReport verifies that the data on the probe workload's PVC
// shows up in the storage column of the cluster report.
func CheckStorageReport(ctx context.Context, in Input) error {
	if err := in.requireCluster(); err != nil {
		return err
	}

	set, err := testReports(ctx, in)
	if err != nil {
		return err
	}

	row, err := findRow(set.clusterCSV, 4, in.Cluster.Name)
	if err != nil {
		return fmt.Errorf("cluster CSV report: %w", err)
	}

	column := slices.Index(clusterReportColumns, "average-persistent-storage-usage-bytes")
	stored, err := strconv.ParseUint(row[column], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid storage column %q: %w", row[column], err)
	}

	// the average covers every subquery step where a PVC series existed,
	// including steps before the probe finished writing, so the lower bound
	// is the written data weighted by the steps that saw it
	prom, err := meteringPrometheus(in)
	if err != nil {
		return err
	}

	written := uint64(workloadDataMiB) << 20
	volumes := fmt.Sprintf(`sum by (cluster) (kubelet_volume_stats_used_bytes{cluster=%q, persistentvolumeclaim!=""})`, in.Cluster.Name)

	total, err := scalarAt(ctx, prom, fmt.Sprintf(`count_over_time((%s)[1440m:])`, volumes), set.window.end)
	if err != nil {
		return err
	}
	full, err := scalarAt(ctx, prom, fmt.Sprintf(`count_over_time((%s >= %d)[1440m:])`, volumes, written), set.window.end)
	if err != nil {
		return err
	}

	if full == 0 {
		return fmt.Errorf("no step in the report window saw the %d bytes the probe workload wrote", written)
	}

	if minimum := uint64(float64(written) * full / total); stored < minimum {
		return fmt.Errorf("storage column is %d bytes, expected at least %d (%d bytes in %.0f of %.0f steps)", stored, minimum, written, full, total)
	}

	return nil
}

// scalarAt runs a query that returns at most one sample and returns its
// value, or 0 when it returns nothing.
func scalarAt(ctx context.Context, prom promv1.API, query string, at time.Time) (float64, error) {
	vector, err := queryVector(ctx, prom, query, at)
	if err != nil {
		return 0, err
	}

	if len(vector) == 0 {
		return 0, nil
	}

	return float64(vector[0].Value), nil
}

// CheckCSVMatchesJSON verifies that the CSV and JSON reports of the same
// window carry the same values for the user cluster.
func CheckCSVMatchesJSON(ctx context.Context, in Input) error {
	if err := in.requireCluster(); err != nil {
		return err
	}

	set, err := testReports(ctx, in)
	if err != nil {
		return err
	}

	var errs []error

	csvRow, err := findRow(set.clusterCSV, 4, in.Cluster.Name)
	if err != nil {
		return fmt.Errorf("cluster CSV report: %w", err)
	}

	rawRow, err := findJSONRow(set.clusterJSON, in.Cluster.Name, "")
	if err != nil {
		return fmt.Errorf("cluster JSON report: %w", err)
	}

	jsonRow := clusterJSONRow{}
	if err := json.Unmarshal(rawRow, &jsonRow); err != nil {
		return fmt.Errorf("cluster JSON report: %w", err)
	}

	if jsonRow.AvgPersistentStorageUsage == nil {
		errs = append(errs, errors.New("cluster JSON report has no avgPersistentStorageUsage"))
		jsonRow.AvgPersistentStorageUsage = new(uint64)
	}

	errs = append(errs, compareRow("cluster JSON vs CSV", clusterReportColumns, jsonRow.csv(), csvRow))

	if len(set.namespaceCSV) == 0 {
		return errors.Join(append(errs, errors.New("namespace CSV report is empty"))...)
	}

	jsonNamespaces := map[string]bool{}
	for _, rawRow := range set.namespaceJSON {
		row := namespaceJSONRow{}
		if err := json.Unmarshal(rawRow, &row); err != nil {
			errs = append(errs, fmt.Errorf("namespace JSON report: %w", err))
			continue
		}

		if row.Cluster.Name != in.Cluster.Name {
			continue
		}

		var match []string
		for _, csvRow := range set.namespaceCSV {
			if csvRow[4] == row.Cluster.Name && csvRow[6] == row.Namespace {
				match = csvRow
			}
		}
		jsonNamespaces[row.Namespace] = true

		// the JSON run comes after the CSV run, and Prometheus lists
		// namespaces whose first series arrived in between; those have no
		// usage in the window yet
		if match == nil {
			if !row.ConsumedResources.Cpu().IsZero() || !row.ConsumedResources.Memory().IsZero() {
				errs = append(errs, fmt.Errorf("namespace %s has usage in the JSON report but is missing from the CSV report", row.Namespace))
			}
			continue
		}

		errs = append(errs, compareRow("namespace "+row.Namespace+" JSON vs CSV", namespaceReportColumns, row.csv(), match))
	}

	for _, csvRow := range set.namespaceCSV[1:] {
		if csvRow[4] == in.Cluster.Name && !jsonNamespaces[csvRow[6]] {
			errs = append(errs, fmt.Errorf("namespace %s is in the CSV report but not in the JSON report", csvRow[6]))
		}
	}

	return errors.Join(errs...)
}

// JSON report rows of metering v1.4.1.
type jsonObject struct {
	Name              string            `json:"name"`
	HumanReadableName string            `json:"humanReadableName"`
	Labels            map[string]string `json:"labels"`
	Owner             string            `json:"owner"`
}

type clusterJSONRow struct {
	Project                   jsonObject          `json:"project"`
	Cluster                   jsonObject          `json:"cluster"`
	CloudProvider             string              `json:"cloudProvider"`
	Datacenter                string              `json:"datacenter"`
	Created                   time.Time           `json:"created"`
	Deleted                   time.Time           `json:"deleted"`
	AvgClusterNodes           float64             `json:"avgClusterNodes"`
	AvgAvailableResources     corev1.ResourceList `json:"avgAvailableResources"`
	AvgConsumedResources      corev1.ResourceList `json:"avgConsumedResources"`
	AvgPersistentStorageUsage *uint64             `json:"avgPersistentStorageUsage"`
}

// csv returns the row in the column order of the CSV report.
func (r clusterJSONRow) csv() []string {
	return []string{
		r.Project.HumanReadableName,
		r.Project.Name,
		flattenMap(r.Project.Labels),
		r.Cluster.HumanReadableName,
		r.Cluster.Name,
		flattenMap(r.Cluster.Labels),
		strconv.FormatFloat(r.AvgClusterNodes, 'f', 2, 64),
		strconv.FormatInt(r.AvgAvailableResources.Cpu().MilliValue(), 10),
		strconv.FormatInt(r.AvgConsumedResources.Cpu().MilliValue(), 10),
		strconv.FormatInt(r.AvgAvailableResources.Memory().Value(), 10),
		strconv.FormatInt(r.AvgConsumedResources.Memory().Value(), 10),
		formatTime(r.Created),
		formatTime(r.Deleted),
		r.CloudProvider,
		r.Datacenter,
		r.Cluster.Owner,
		strconv.FormatUint(*r.AvgPersistentStorageUsage, 10),
	}
}

type namespaceJSONRow struct {
	Project           jsonObject          `json:"project"`
	Cluster           jsonObject          `json:"cluster"`
	Namespace         string              `json:"namespace"`
	ConsumedResources corev1.ResourceList `json:"consumedResources"`
}

func (r namespaceJSONRow) csv() []string {
	return []string{
		r.Project.HumanReadableName,
		r.Project.Name,
		flattenMap(r.Project.Labels),
		r.Cluster.HumanReadableName,
		r.Cluster.Name,
		flattenMap(r.Cluster.Labels),
		r.Namespace,
		strconv.FormatInt(r.ConsumedResources.Cpu().MilliValue(), 10),
		strconv.FormatInt(r.ConsumedResources.Memory().Value(), 10),
		r.Cluster.Owner,
	}
}

// formatTime formats a time like the CSV report: empty for the zero time.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}

	return t.UTC().Format(time.RFC3339)
}

// JSON fields of metering v1.4.1 that consumers depend on, as paths into one
// row. Fields may be added, but never renamed or removed.
var (
	objectFields   = []string{"name", "humanReadableName", "labels", "owner"}
	resourceFields = []string{"cpu", "memory"}

	clusterJSONFields = append(append(append(append(
		prefixed("project", objectFields),
		prefixed("cluster", objectFields)...),
		prefixed("avgAvailableResources", resourceFields)...),
		prefixed("avgConsumedResources", resourceFields)...),
		"cloudProvider", "datacenter", "created", "deleted", "avgClusterNodes", "avgPersistentStorageUsage",
	)

	namespaceJSONFields = append(append(append(
		prefixed("project", objectFields),
		prefixed("cluster", objectFields)...),
		prefixed("consumedResources", resourceFields)...),
		"namespace",
	)
)

func prefixed(prefix string, fields []string) []string {
	var paths []string
	for _, field := range fields {
		paths = append(paths, prefix+"."+field)
	}

	return paths
}

// checkJSONFields verifies that every path exists in the JSON object.
func checkJSONFields(name string, raw []byte, paths []string) error {
	object := map[string]any{}
	if err := json.Unmarshal(raw, &object); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}

	var missing []string
	for _, path := range paths {
		var current any = object
		for _, part := range strings.Split(path, ".") {
			fields, ok := current.(map[string]any)
			if !ok {
				current = nil
				break
			}

			value, ok := fields[part]
			if !ok {
				current = nil
				break
			}

			// labels may be null, which still counts as present
			if value == nil {
				value = struct{}{}
			}
			current = value
		}

		if current == nil {
			missing = append(missing, path)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("%s misses the fields %v", name, missing)
	}

	return nil
}

// checkHeader verifies that the report starts with the documented columns
// in their order. Extra columns at the end are allowed.
func checkHeader(name string, rows [][]string, columns []string) error {
	if len(rows) == 0 {
		return fmt.Errorf("%s has no header", name)
	}

	header := rows[0]
	if len(header) < len(columns) || !slices.Equal(header[:len(columns)], columns) {
		return fmt.Errorf("%s has the columns %v, consumers expect them to start with %v", name, header, columns)
	}

	return nil
}

func columnsOf(reportType string) []string {
	if reportType == "namespace" {
		return namespaceReportColumns
	}

	return clusterReportColumns
}

// compareRow compares the first len(columns) values of two rows column by
// column.
func compareRow(name string, columns, actual, expected []string) error {
	var errs []error
	for i, column := range columns {
		var a, e string
		if i < len(actual) {
			a = actual[i]
		}
		if i < len(expected) {
			e = expected[i]
		}

		if a != e {
			errs = append(errs, fmt.Errorf("%s: %s is %q, expected %q", name, column, a, e))
		}
	}

	return errors.Join(errs...)
}

// findRow returns the data row whose column has the value.
func findRow(rows [][]string, column int, value string) ([]string, error) {
	for i, row := range rows {
		// skip the CSV header
		if i == 0 && len(rows[0]) > 0 && rows[0][0] == clusterReportColumns[0] {
			continue
		}

		if column < len(row) && row[column] == value {
			return row, nil
		}
	}

	return nil, fmt.Errorf("no row with %q", value)
}

// findJSONRow returns the JSON row of the cluster, and of the namespace when
// it is not empty.
func findJSONRow(rows [][]byte, clusterID, namespace string) ([]byte, error) {
	for _, raw := range rows {
		row := namespaceJSONRow{}
		if err := json.Unmarshal(raw, &row); err != nil {
			return nil, err
		}

		if row.Cluster.Name == clusterID && (namespace == "" || row.Namespace == namespace) {
			return raw, nil
		}
	}

	return nil, fmt.Errorf("no row for cluster %s %s", clusterID, namespace)
}

func parseCSV(body []byte) ([][]string, error) {
	rows, err := csv.NewReader(bytes.NewReader(body)).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("invalid CSV: %w", err)
	}

	return rows, nil
}

// parseJSONLines splits a JSON Lines file and checks that every line is a
// JSON object.
func parseJSONLines(body []byte) ([][]byte, error) {
	var rows [][]byte

	scanner := bufio.NewScanner(bytes.NewReader(body))
	scanner.Buffer(make([]byte, 1<<20), 16<<20)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		if !json.Valid(line) || line[0] != '{' {
			return nil, fmt.Errorf("line %d is not a JSON object", len(rows)+1)
		}

		rows = append(rows, bytes.Clone(line))
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return rows, nil
}
