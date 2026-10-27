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

	"github.com/onsi/ginkgo/v2"

	mlatestkit "k8c.io/kubermatic/v2/pkg/test/e2e/testkit/mla"
)

// Scope tells how far the effects of a check reach. Suites that share a seed
// with other workloads (for example conformance-ee) filter on it.
type Scope string

const (
	// ScopeCluster checks only touch objects of the cluster under test.
	ScopeCluster Scope = "cluster"
	// ScopeSeedWide checks change objects that affect every MLA cluster on
	// the seed.
	ScopeSeedWide Scope = "seed-wide"
	// ScopeDisruptive checks turn MLA off, for the cluster and for its Seed,
	// so every MLA cluster on the seed is affected and nothing can run after
	// them.
	ScopeDisruptive Scope = "disruptive"
)

// Mode tells how a check may be scheduled next to the other checks on the
// shared cluster.
type Mode string

const (
	// ModeParallel checks only create uniquely named objects and can run next
	// to each other.
	ModeParallel Mode = "parallel"
	// ModeSerial checks change config that is shared per cluster or restart
	// shared components, so they run alone.
	ModeSerial Mode = "serial"
	// ModeLast checks run alone and after every other check.
	ModeLast Mode = "last"
)

// lastPriority makes a ModeLast check start after all other checks, also when
// the suite runs in a single process where Serial does not reorder specs.
const lastPriority = -1000

// Feature is one customer-facing MLA behaviour and the check that proves it.
// I is the input type of its check: mlatestkit.Input for user cluster checks,
// mlatestkit.SeedInput for seed checks.
type Feature[I any] struct {
	// ID is a short stable name for the check, used in labels and filters.
	ID    string
	Name  string
	Scope Scope
	Mode  Mode
	// NeedsNodes is set when the check needs worker nodes in the user
	// cluster, because the MLA agents or probe pods have to run there.
	NeedsNodes bool
	// NeedsUserCluster is set on seed checks that need at least one MLA user
	// cluster on the seed, for example because they look for its metrics.
	NeedsUserCluster bool
	// KnownIssue describes a confirmed product bug that makes the check fail.
	// Such checks get the known-issue label, which CI skips until the bug is
	// fixed; remove the field together with the fix.
	KnownIssue string
	// Doc is the documentation page or ticket the feature comes from.
	Doc   string
	Check func(ctx context.Context, in I) error
}

const (
	userGuideDoc  = "kkp-docs: tutorials-howtos/monitoring-logging-alerting/user-cluster/user-guide"
	adminGuideDoc = "kkp-docs: tutorials-howtos/monitoring-logging-alerting/user-cluster/admin-guide"
	archDoc       = "kkp-docs: architecture/monitoring-logging-alerting/user-cluster"

	seedInstallDoc   = "kkp-docs: tutorials-howtos/monitoring-logging-alerting/master-seed/installation"
	seedCustomizeDoc = "kkp-docs: tutorials-howtos/monitoring-logging-alerting/master-seed/customization"
)

// UserClusterFeatures are the user cluster MLA checks in registration order.
var UserClusterFeatures = []Feature[mlatestkit.Input]{
	{ID: "U8", Name: "creates a Grafana org named after the project and links it", Scope: ScopeCluster, Mode: ModeParallel, Doc: userGuideDoc, Check: mlatestkit.CheckGrafanaOrg},
	{ID: "U9", Name: "provisions the KKP admin into Grafana", Scope: ScopeCluster, Mode: ModeParallel, Doc: userGuideDoc, Check: mlatestkit.CheckGrafanaUser},
	{ID: "U9-viewer", Name: "gives project viewers the Grafana Viewer role and removes them again", Scope: ScopeCluster, Mode: ModeParallel, Doc: userGuideDoc, Check: mlatestkit.CheckGrafanaViewer},
	{ID: "U10", Name: "creates healthy Prometheus, Loki and Alertmanager datasources", Scope: ScopeCluster, Mode: ModeParallel, Doc: userGuideDoc, Check: mlatestkit.CheckDatasources},
	{ID: "U11", Name: "syncs the predefined dashboards into the project org", Scope: ScopeCluster, Mode: ModeParallel, Doc: adminGuideDoc, Check: mlatestkit.CheckCanaryDashboard},
	{ID: "U11-custom", Name: "syncs a custom dashboard ConfigMap into the project org", Scope: ScopeSeedWide, Mode: ModeParallel, Doc: adminGuideDoc, Check: mlatestkit.CheckCustomDashboard},
	{ID: "U13-logs", Name: "creates, updates and deletes a logs RuleGroup in the Loki ruler", Scope: ScopeCluster, Mode: ModeParallel, Doc: userGuideDoc, Check: mlatestkit.CheckLogsRuleGroup},
	{ID: "U13-metrics", Name: "creates, updates and deletes a metrics RuleGroup in the Cortex ruler", Scope: ScopeCluster, Mode: ModeParallel, Doc: userGuideDoc, Check: mlatestkit.CheckMetricsRuleGroup},
	{ID: "U14", Name: "syncs an admin default RuleGroup into the cluster", Scope: ScopeSeedWide, Mode: ModeParallel, Doc: adminGuideDoc, Check: mlatestkit.CheckDefaultRuleGroup},
	{ID: "U3", Name: "ships pod metrics to Cortex", Scope: ScopeCluster, Mode: ModeParallel, NeedsNodes: true, Doc: archDoc, Check: mlatestkit.CheckMetricsRoundTrip},
	{ID: "U3-contract", Name: "ships every metric the MLA dashboards use", Scope: ScopeCluster, Mode: ModeParallel, NeedsNodes: true, Doc: userGuideDoc, Check: mlatestkit.CheckUserClusterMetricContract},
	{ID: "U4", Name: "ships pod logs to Loki", Scope: ScopeCluster, Mode: ModeParallel, NeedsNodes: true, Doc: archDoc, Check: mlatestkit.CheckLogsRoundTrip},
	{ID: "U4-gateway", Name: "accepts mTLS writes and rejects clients without a cert", Scope: ScopeCluster, Mode: ModeParallel, Doc: archDoc, Check: mlatestkit.CheckGatewayWritePath},
	{ID: "U5", Name: "ignores a client-supplied tenant header", Scope: ScopeCluster, Mode: ModeParallel, Doc: archDoc, Check: mlatestkit.CheckTenantHeaderSpoofing},
	{ID: "U6", Name: "scrapes pods with the path and scrape-slow annotations", Scope: ScopeCluster, Mode: ModeParallel, NeedsNodes: true, Doc: userGuideDoc, Check: mlatestkit.CheckScrapeAnnotations},
	{ID: "U7", Name: "scrapes custom scrape configs and survives an invalid one", Scope: ScopeCluster, Mode: ModeSerial, NeedsNodes: true, Doc: userGuideDoc, Check: mlatestkit.CheckCustomScrapeConfig},
	{ID: "U12", Name: "delivers a metrics alert to the configured receiver", Scope: ScopeCluster, Mode: ModeSerial, Doc: userGuideDoc, Check: mlatestkit.CheckMetricsAlertDelivery},
	{ID: "U12-logs", Name: "delivers a logs alert to the configured receiver", Scope: ScopeCluster, Mode: ModeSerial, NeedsNodes: true, Doc: userGuideDoc, Check: mlatestkit.CheckLogsAlertDelivery},
	{ID: "U15", Name: "applies and enforces monitoring and logging rate limits", Scope: ScopeCluster, Mode: ModeSerial, Doc: adminGuideDoc, Check: mlatestkit.CheckRateLimits},
	{ID: "U1", Name: "runs only the agent of the enabled MLA part", Scope: ScopeCluster, Mode: ModeSerial, NeedsNodes: true, Doc: userGuideDoc, Check: mlatestkit.CheckMLAToggles},
	{ID: "U17", Name: "applies agent resources and replicas from the cluster spec", Scope: ScopeCluster, Mode: ModeSerial, NeedsNodes: true, Doc: userGuideDoc, Check: mlatestkit.CheckAgentSettings},
	{ID: "U19", Name: "cleans up agents, gateway and Grafana when MLA is disabled", Scope: ScopeDisruptive, Mode: ModeLast, Doc: adminGuideDoc, Check: mlatestkit.DisableMLAAndCheckCleanup},
}

// SeedFeatures are the seed MLA checks in registration order.
var SeedFeatures = []Feature[mlatestkit.SeedInput]{
	{ID: "S1", Name: "runs every seed MLA workload", Scope: ScopeCluster, Mode: ModeParallel, Doc: seedInstallDoc, Check: mlatestkit.CheckSeedWorkloads},
	{ID: "S2-targets", Name: "scrapes the seed targets", Scope: ScopeCluster, Mode: ModeParallel, Doc: seedInstallDoc, Check: mlatestkit.CheckSeedTargets},
	{ID: "S2-rules", Name: "evaluates the seed rules without errors", Scope: ScopeCluster, Mode: ModeParallel, Doc: seedCustomizeDoc, Check: mlatestkit.CheckSeedRules},
	{ID: "S2-labels", Name: "keeps the label contract of the seed rules", Scope: ScopeCluster, Mode: ModeParallel, Doc: seedCustomizeDoc, Check: mlatestkit.CheckSeedLabels},
	{ID: "S2-contract", Name: "has every metric the seed rules and dashboards use", Scope: ScopeCluster, Mode: ModeParallel, NeedsUserCluster: true, Doc: seedCustomizeDoc, Check: mlatestkit.CheckSeedMetricContract},
	{ID: "S5", Name: "routes an alert through Alertmanager to karma", Scope: ScopeSeedWide, Mode: ModeParallel, Doc: seedCustomizeDoc, Check: mlatestkit.CheckSeedAlertRouting},
	{ID: "S6", Name: "serves healthy datasources and the shipped dashboards in seed Grafana", Scope: ScopeCluster, Mode: ModeParallel, Doc: seedCustomizeDoc, Check: mlatestkit.CheckSeedGrafana},
	{ID: "S7", Name: "ships seed pod logs to Loki", Scope: ScopeCluster, Mode: ModeParallel, Doc: seedInstallDoc, Check: mlatestkit.CheckSeedLogs},
	{ID: "S7-labels", Name: "ships logs of a pod with as many labels as a user cluster apiserver to Loki", Scope: ScopeCluster, Mode: ModeParallel, Doc: seedInstallDoc, Check: mlatestkit.CheckSeedLogLabelLimit},
	{ID: "S7-controlplane", Name: "ships the logs of every user cluster control plane container to Loki", Scope: ScopeCluster, Mode: ModeParallel, NeedsUserCluster: true, Doc: seedInstallDoc, Check: mlatestkit.CheckControlPlaneLogs},
	{ID: "S7-discards", Name: "rejects no log line because of its labels", Scope: ScopeCluster, Mode: ModeParallel, Doc: seedInstallDoc, Check: mlatestkit.CheckSeedLogDiscards},
	{ID: "S8", Name: "probes through the blackbox exporter", Scope: ScopeCluster, Mode: ModeParallel, Doc: seedInstallDoc, Check: mlatestkit.CheckBlackboxExporter},
}

// needsNodesLabel marks checks that cannot pass on a user cluster without
// worker nodes; skip them with --label-filter='!needs-nodes'.
const needsNodesLabel = "needs-nodes"

// needsUserClusterLabel marks seed checks that need an MLA user cluster on the
// seed; skip them with --label-filter='!needs-user-cluster'.
const needsUserClusterLabel = "needs-user-cluster"

// knownIssueLabel marks checks that fail because of a confirmed product bug;
// CI skips them with --label-filter='!known-issue'.
const knownIssueLabel = "known-issue"

// decorators turns the feature metadata into Ginkgo labels and ordering
// decorators. Labels use Ginkgo label sets, so a suite can filter with for
// example --label-filter='scope: containsAny cluster'.
func (f Feature[I]) decorators() []any {
	labels := []string{"mla", "id:" + f.ID, "scope:" + string(f.Scope), "mode:" + string(f.Mode)}
	if f.NeedsNodes {
		labels = append(labels, needsNodesLabel)
	}
	if f.NeedsUserCluster {
		labels = append(labels, needsUserClusterLabel)
	}
	if f.KnownIssue != "" {
		labels = append(labels, knownIssueLabel)
	}

	args := []any{ginkgo.Label(labels...)}

	switch f.Mode {
	case ModeSerial:
		args = append(args, ginkgo.Serial)
	case ModeLast:
		args = append(args, ginkgo.Serial, ginkgo.SpecPriority(lastPriority))
	}

	return args
}
