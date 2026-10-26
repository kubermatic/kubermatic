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
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/prometheus/common/model"
)

func TestMilliCoresRoundsUp(t *testing.T) {
	for nanoCores, expected := range map[uint64]string{
		0:          "0",
		1:          "1",
		1_000_000:  "1",
		1_000_001:  "2",
		1744000000: "1744",
	} {
		if actual := milliCores(nanoCores); actual != expected {
			t.Errorf("milliCores(%d) = %s, expected %s", nanoCores, actual, expected)
		}
	}
}

func TestFlattenLabels(t *testing.T) {
	metric := model.Metric{
		"__name__":                   "kubermatic_cluster_labels",
		"name":                       "abc",
		"instance":                   "x",
		"job":                        "seed-controller-manager",
		"label_project_id":           "p",
		"label_is_credential_preset": "true",
		"label_zone":                 "a",
		"label_env":                  "prod",
	}

	if actual, expected := flattenLabels(metric), "label_env=prod,label_zone=a"; actual != expected {
		t.Errorf("flattenLabels = %q, expected %q", actual, expected)
	}

	if actual := flattenMap(nil); actual != "" {
		t.Errorf("flattenMap(nil) = %q, expected empty", actual)
	}
}

func TestWindow(t *testing.T) {
	end := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)

	for duration, expected := range map[time.Duration]int{
		time.Hour:      1,
		36 * time.Hour: 1,
		72 * time.Hour: 3,
	} {
		w := window{start: end.Add(-duration), end: end}
		if actual := w.chunks(); actual != expected {
			t.Errorf("chunks of %s = %d, expected %d", duration, actual, expected)
		}
	}

	if actual := (window{start: end.Add(-90 * time.Second), end: end}).rangeExpr(); actual != "1m" {
		t.Errorf("rangeExpr truncates to whole minutes, got %s", actual)
	}
}

func TestCheckHeader(t *testing.T) {
	appended := append(slices.Clone(clusterReportColumns), "a-new-column")

	// metering v1.2.2, shipped with KKP v2.29, has no storage column
	v122 := clusterReportColumns[:len(clusterReportColumns)-1]

	reordered := slices.Clone(clusterReportColumns)
	reordered[0], reordered[1] = reordered[1], reordered[0]

	for name, tc := range map[string]struct {
		header []string
		valid  bool
	}{
		"current columns":  {clusterReportColumns, true},
		"appended column":  {appended, true},
		"missing column":   {v122, false},
		"reordered column": {reordered, false},
	} {
		err := checkHeader(name, [][]string{tc.header}, clusterReportColumns)
		if (err == nil) != tc.valid {
			t.Errorf("%s: valid = %v, error: %v", name, tc.valid, err)
		}
	}

	if err := checkHeader("empty", nil, clusterReportColumns); err == nil {
		t.Error("an empty report must fail")
	}
}

// clusterRowV141 is a cluster report row as metering v1.4.1 writes it, in
// both formats.
const clusterRowV141JSON = `{"project":{"name":"z4sm5ndqkv","humanReadableName":"bs","labels":{"label_metering_test":"on"},"owner":""},"cluster":{"name":"abc123","humanReadableName":"jolly-ptolemy","labels":null,"owner":"user-1"},"cloudProvider":"aws","datacenter":"aws-eu-central-1a","created":"2026-08-13T10:00:00Z","deleted":"0001-01-01T00:00:00Z","avgClusterNodes":0.87,"avgAvailableResources":{"cpu":"1744m","memory":"2760172256"},"avgConsumedResources":{"cpu":"212m","memory":"980Ki"},"avgPersistentStorageUsage":67108864}`

var clusterRowV141CSV = []string{
	"bs", "z4sm5ndqkv", "label_metering_test=on", "jolly-ptolemy", "abc123", "",
	"0.87", "1744", "212", "2760172256", "1003520",
	"2026-08-13T10:00:00Z", "", "aws", "aws-eu-central-1a", "user-1", "67108864",
}

func TestClusterJSONRowMatchesCSV(t *testing.T) {
	row := clusterJSONRow{}
	if err := json.Unmarshal([]byte(clusterRowV141JSON), &row); err != nil {
		t.Fatal(err)
	}

	if err := compareRow("json", clusterReportColumns, row.csv(), clusterRowV141CSV); err != nil {
		t.Error(err)
	}
}

func TestCheckJSONFields(t *testing.T) {
	if err := checkJSONFields("v1.4.1", []byte(clusterRowV141JSON), clusterJSONFields); err != nil {
		t.Errorf("a v1.4.1 row must pass: %v", err)
	}

	// metering v1.3.x and v1.4.0 dropped the storage field from JSON reports
	withoutStorage := map[string]any{}
	if err := json.Unmarshal([]byte(clusterRowV141JSON), &withoutStorage); err != nil {
		t.Fatal(err)
	}
	delete(withoutStorage, "avgPersistentStorageUsage")

	raw, _ := json.Marshal(withoutStorage)
	if err := checkJSONFields("v1.4.0", raw, clusterJSONFields); err == nil {
		t.Error("a row without avgPersistentStorageUsage must fail")
	}
}

func TestParseJSONLines(t *testing.T) {
	rows, err := parseJSONLines([]byte(clusterRowV141JSON + "\n" + clusterRowV141JSON + "\n"))
	if err != nil || len(rows) != 2 {
		t.Errorf("expected 2 rows, got %d: %v", len(rows), err)
	}

	if rows, err := parseJSONLines(nil); err != nil || len(rows) != 0 {
		t.Errorf("an empty report must parse to no rows, got %d: %v", len(rows), err)
	}

	if _, err := parseJSONLines([]byte("[1,2]\n")); err == nil {
		t.Error("a line that is not an object must fail")
	}
}

func TestReportKey(t *testing.T) {
	start := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC)

	expected := "weekly/kubermatic-cluster-report-2026-10-11-2026-10-18.csv"
	if actual := reportKey("weekly", "kubermatic", "cluster", start, end, "csv"); actual != expected {
		t.Errorf("reportKey = %s, expected %s", actual, expected)
	}
}

func TestServiceAddress(t *testing.T) {
	for host, expected := range map[string][2]string{
		"minio.minio.svc.cluster.local": {"minio", "minio"},
		"minio.minio.svc":               {"minio", "minio"},
		"s3.amazonaws.com":              {"", ""},
		"localhost":                     {"", ""},
	} {
		name, namespace, ok := serviceAddress(host)
		if [2]string{name, namespace} != expected || ok != (expected[0] != "") {
			t.Errorf("serviceAddress(%s) = %s, %s, %v", host, name, namespace, ok)
		}
	}
}
