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
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/seed-controller-manager/mla"
	"k8c.io/kubermatic/v2/pkg/test/generator"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	logsRuleGroupName    = "test-rule"
	metricsRuleGroupName = "test-metric-rule"

	logsRuleGroup = `
name: test-rule
rules:
- alert: HighThroughputLogStreams
  expr: sum by(container)(rate({job=~"kube-system/.*"}[1m])) >= 50
  for: 2s
  labels:
    severity: critical
  annotations:
    summary: "log stream is a bit high"
    description: "log stream is a bit high"
`
)

// CheckLogsRuleGroup creates a logs RuleGroup and verifies it reaches the Loki
// ruler of the cluster's tenant.
func CheckLogsRuleGroup(ctx context.Context, in Input) error {
	proxy, err := newServiceProxy(in.SeedConfig, mlaNamespace, lokiRulerService, lokiRulerPort)
	if err != nil {
		return err
	}

	data := strings.Replace(logsRuleGroup, "name: "+logsRuleGroupName, "name: "+logsRuleGroupName+"-"+rand.String(6), 1)

	return checkRuleGroup(ctx, in, proxy, mla.LogRuleGroupConfigEndpoint, []byte(data), kubermaticv1.RuleGroupTypeLogs)
}

// CheckMetricsRuleGroup creates a metrics RuleGroup and verifies it reaches the
// Cortex ruler of the cluster's tenant.
func CheckMetricsRuleGroup(ctx context.Context, in Input) error {
	proxy, err := newServiceProxy(in.SeedConfig, mlaNamespace, cortexRulerService, cortexRulerPort)
	if err != nil {
		return err
	}

	data := generator.GenerateTestRuleGroupData(metricsRuleGroupName + "-" + rand.String(6))

	return checkRuleGroup(ctx, in, proxy, mla.MetricsRuleGroupConfigEndpoint, data, kubermaticv1.RuleGroupTypeMetrics)
}

func checkRuleGroup(ctx context.Context, in Input, proxy *serviceProxy, endpoint string, data []byte, kind kubermaticv1.RuleGroupType) error {
	ruleGroup, expected, err := createRuleGroupObject(ctx, in, data, kind)
	if err != nil {
		return err
	}
	defer deleteIgnoreNotFound(in, ruleGroup)

	path := fmt.Sprintf("%s/default/%s", endpoint, ruleGroup.Name)
	header := http.Header{mla.RuleGroupTenantHeaderName: []string{in.Cluster.Name}}

	if err := waitForRuleGroup(ctx, proxy, path, header, expected); err != nil {
		return fmt.Errorf("%s rule group %q not found in the ruler: %w", kind, ruleGroup.Name, err)
	}

	// an update of the RuleGroup must reach the ruler
	updated, updatedData, err := withExtraRuleLabel(expected)
	if err != nil {
		return err
	}

	oldRuleGroup := ruleGroup.DeepCopy()
	ruleGroup.Spec.Data = updatedData
	if err := in.SeedClient.Patch(ctx, ruleGroup, ctrlruntimeclient.MergeFrom(oldRuleGroup)); err != nil {
		return fmt.Errorf("failed to update the RuleGroup: %w", err)
	}

	if err := waitForRuleGroup(ctx, proxy, path, header, updated); err != nil {
		return fmt.Errorf("updated %s rule group %q not served by the ruler: %w", kind, ruleGroup.Name, err)
	}

	// deleting the RuleGroup must remove it from the ruler
	if err := in.SeedClient.Delete(ctx, ruleGroup); err != nil {
		return fmt.Errorf("failed to delete the RuleGroup: %w", err)
	}

	err = wait.PollImmediate(ctx, time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		code, body, err := proxy.do(ctx, http.MethodGet, path, header, nil)
		if err != nil {
			return err, nil
		}
		if code != http.StatusNotFound {
			return fmt.Errorf("ruler still serves the group (HTTP %d: %s)", code, body), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("deleted %s rule group %q still in the ruler: %w", kind, ruleGroup.Name, err)
	}

	return nil
}

func waitForRuleGroup(ctx context.Context, proxy *serviceProxy, path string, header http.Header, expected map[string]interface{}) error {
	return wait.PollImmediate(ctx, time.Second, 5*time.Minute, func(ctx context.Context) (error, error) {
		code, body, err := proxy.do(ctx, http.MethodGet, path, header, nil)
		if err != nil {
			return fmt.Errorf("unable to get rule group: %w", err), nil
		}

		if code != http.StatusOK {
			return fmt.Errorf("expected HTTP 200, got HTTP %d: %s", code, body), nil
		}

		actual := map[string]interface{}{}
		if err := yaml.Unmarshal(body, &actual); err != nil {
			return fmt.Errorf("unable to decode response body: %w", err), nil
		}

		if !reflect.DeepEqual(actual, expected) {
			return errors.New("response does not match the expected rule group"), nil
		}

		return nil, nil
	})
}

// withExtraRuleLabel returns a copy of the rule group with a label added to its
// first rule, as parsed data and as YAML.
func withExtraRuleLabel(group map[string]interface{}) (map[string]interface{}, []byte, error) {
	raw, err := yaml.Marshal(group)
	if err != nil {
		return nil, nil, err
	}

	updated := map[string]interface{}{}
	if err := yaml.Unmarshal(raw, &updated); err != nil {
		return nil, nil, err
	}

	rules, ok := updated["rules"].([]interface{})
	if !ok || len(rules) == 0 {
		return nil, nil, errors.New("rule group has no rules")
	}

	rule, ok := rules[0].(map[string]interface{})
	if !ok {
		return nil, nil, errors.New("unexpected rule format")
	}

	labels, _ := rule["labels"].(map[string]interface{})
	if labels == nil {
		labels = map[string]interface{}{}
	}
	labels["e2e_updated"] = "true"
	rule["labels"] = labels

	data, err := yaml.Marshal(updated)
	if err != nil {
		return nil, nil, err
	}

	return updated, data, nil
}

// createRuleGroupObject creates the RuleGroup in the cluster namespace, or
// keeps it if a previous run on the same cluster already created it, and
// returns it with its parsed data.
func createRuleGroupObject(ctx context.Context, in Input, data []byte, kind kubermaticv1.RuleGroupType) (*kubermaticv1.RuleGroup, map[string]interface{}, error) {
	expected := map[string]interface{}{}
	if err := yaml.Unmarshal(data, &expected); err != nil {
		return nil, nil, fmt.Errorf("unable to unmarshal expected rule group: %w", err)
	}

	name, ok := expected["name"].(string)
	if !ok || name == "" {
		return nil, nil, errors.New("rule group data has no name")
	}

	ruleGroup := &kubermaticv1.RuleGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: in.Cluster.Status.NamespaceName,
		},
		Spec: kubermaticv1.RuleGroupSpec{
			RuleGroupType: kind,
			Cluster: corev1.ObjectReference{
				Name: in.Cluster.Name,
			},
			Data: data,
		},
	}

	if err := in.SeedClient.Create(ctx, ruleGroup); err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, nil, fmt.Errorf("unable to create %s rule group: %w", kind, err)
	}

	return ruleGroup, expected, nil
}

// yamlUnmarshal decodes YAML the same way the rule group and Alertmanager
// checks compare served configs.
func yamlUnmarshal(data []byte, out any) error {
	return yaml.Unmarshal(data, out)
}

func overridesMatch(actual mla.TenantOverride, expected kubermaticv1.MonitoringRateLimitSettings) bool {
	equal := func(value *int32, want int32) bool {
		return value != nil && *value == want
	}

	return equal(actual.IngestionRate, expected.IngestionRate) &&
		equal(actual.IngestionBurstSize, expected.IngestionBurstSize) &&
		equal(actual.MaxSeriesPerMetric, expected.MaxSeriesPerMetric) &&
		equal(actual.MaxSeriesTotal, expected.MaxSeriesTotal) &&
		equal(actual.MaxSamplesPerQuery, expected.MaxSamplesPerQuery) &&
		equal(actual.MaxSeriesPerQuery, expected.MaxSeriesPerQuery)
}
