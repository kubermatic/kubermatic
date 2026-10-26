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

package mla

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"k8c.io/kubermatic/v2/pkg/util/wait"

	"sigs.k8s.io/yaml"
)

const (
	cortexQueryFrontendService = "cortex-query-frontend"
	cortexQueryFrontendPort    = 8080

	// tenantHeader selects the Cortex and Loki tenant, which is the cluster
	// name for MLA user clusters.
	tenantHeader = "X-Scope-OrgID"
)

// metricContractYAML lists the metric names the seed rules and the MLA
// dashboards use. hack/verify-mla-metric-contract.sh keeps it in sync with the
// charts.
//
//go:embed metric-contract.yaml
var metricContractYAML []byte

// MetricContract is the parsed metric-contract.yaml.
type MetricContract struct {
	Seed        []string `json:"seed"`
	UserCluster []string `json:"userCluster"`
	NotInCI     []struct {
		Name   string `json:"name"`
		Reason string `json:"reason"`
	} `json:"notInCI"`
}

// LoadMetricContract returns the embedded metric contract.
func LoadMetricContract() (*MetricContract, error) {
	contract := &MetricContract{}
	if err := yaml.Unmarshal(metricContractYAML, contract); err != nil {
		return nil, fmt.Errorf("failed to parse the metric contract: %w", err)
	}

	return contract, nil
}

// missingNames returns the expected names that are not in present.
func missingNames(expected, present []string) []string {
	have := make(map[string]bool, len(present))
	for _, name := range present {
		have[name] = true
	}

	var missing []string
	for _, name := range expected {
		if !have[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)

	return missing
}

// CheckSeedMetricContract verifies that every metric the seed rules and
// dashboards use, and that the contract lists under seed, exists in the seed
// Prometheus. Some of these names only exist once the seed runs an MLA user
// cluster.
func CheckSeedMetricContract(ctx context.Context, in SeedInput) error {
	contract, err := LoadMetricContract()
	if err != nil {
		return err
	}

	proxy, err := seedPrometheus(in)
	if err != nil {
		return err
	}

	err = wait.PollImmediate(ctx, 30*time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		var names []string
		if err := promGet(ctx, proxy, "/api/v1/label/__name__/values", &names); err != nil {
			return err, nil
		}

		if missing := missingNames(contract.Seed, names); len(missing) > 0 {
			return fmt.Errorf("%d of %d contract metrics missing in the seed Prometheus: %s", len(missing), len(contract.Seed), strings.Join(missing, ", ")), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("seed metric contract: %w", err)
	}

	return nil
}

// CheckUserClusterMetricContract verifies that every metric the contract lists
// under userCluster exists in Cortex for the cluster's tenant, which needs the
// MLA agents running on worker nodes.
func CheckUserClusterMetricContract(ctx context.Context, in Input) error {
	contract, err := LoadMetricContract()
	if err != nil {
		return err
	}

	proxy, err := newServiceProxy(in.SeedConfig, mlaNamespace, cortexQueryFrontendService, cortexQueryFrontendPort)
	if err != nil {
		return err
	}

	header := http.Header{tenantHeader: []string{in.Cluster.Name}}

	err = wait.PollImmediate(ctx, 30*time.Second, 8*time.Minute, func(ctx context.Context) (error, error) {
		now := time.Now()
		path := fmt.Sprintf("/prometheus/api/v1/label/__name__/values?start=%d&end=%d", now.Add(-time.Hour).Unix(), now.Unix())

		code, body, err := proxy.do(ctx, http.MethodGet, path, header, nil)
		if err != nil {
			return err, nil
		}
		if code != http.StatusOK {
			return fmt.Errorf("expected HTTP 200, got HTTP %d: %s", code, body), nil
		}

		names := []string{}
		response := promAPIResponse{}
		if err := json.Unmarshal(body, &response); err != nil {
			return fmt.Errorf("unable to decode response: %w", err), nil
		}
		if err := json.Unmarshal(response.Data, &names); err != nil {
			return fmt.Errorf("unable to decode label values: %w", err), nil
		}

		if missing := missingNames(contract.UserCluster, names); len(missing) > 0 {
			return fmt.Errorf("%d of %d contract metrics missing in Cortex for tenant %s: %s", len(missing), len(contract.UserCluster), in.Cluster.Name, strings.Join(missing, ", ")), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("user cluster metric contract: %w", err)
	}

	return nil
}
