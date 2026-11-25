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

	"github.com/onsi/ginkgo/v2"

	meteringtestkit "k8c.io/kubermatic/v2/pkg/test/e2e/testkit/metering"
)

// Scope tells how far the effects of a check reach. Suites that share a seed
// with other workloads filter on it.
type Scope string

const (
	// ScopeSeed checks only read metering objects or create uniquely named
	// ones.
	ScopeSeed Scope = "seed"
	// ScopeSeedWide checks change the Seed's metering configuration, the
	// bucket lifecycle or the cluster under test.
	ScopeSeedWide Scope = "seed-wide"
	// ScopeDisruptive checks turn metering off for the whole seed, so
	// nothing can run after them.
	ScopeDisruptive Scope = "disruptive"
)

// Mode tells how a check may be scheduled next to the other checks.
type Mode string

const (
	// ModeParallel checks can run next to each other.
	ModeParallel Mode = "parallel"
	// ModeSerial checks change shared config, so they run alone.
	ModeSerial Mode = "serial"
	// ModeLast checks run alone and after every other check.
	ModeLast Mode = "last"
)

// lastPriority makes a ModeLast check start after all other checks, also when
// the suite runs in a single process where Serial does not reorder specs.
const lastPriority = -1000

// Feature is one customer-facing metering behaviour and the check that
// proves it.
type Feature struct {
	// ID is a short stable name for the check, used in labels and filters.
	ID    string
	Name  string
	Scope Scope
	Mode  Mode
	// NeedsUserCluster is set when the check reads the usage of a user
	// cluster, so the suite input must name one.
	NeedsUserCluster bool
	// KnownIssue describes a confirmed product bug that makes the check fail.
	// Such checks get the known-issue label, which CI skips until the bug is
	// fixed; remove the field together with the fix.
	KnownIssue string
	// Doc is the documentation page or code the feature comes from.
	Doc   string
	Check func(ctx context.Context, in meteringtestkit.Input) error
}

// seedWebhookIssue explains why the Seed webhook checks fail: the webhook only
// selects namespaces labelled app.kubernetes.io/name=<KKP namespace>, and
// nothing sets that label. Confirmed on the Dev environment on 2026-10-20.
const seedWebhookIssue = "the Seed webhook selects namespaces labelled app.kubernetes.io/name=<KKP namespace>, which nothing sets, so it never runs"

const (
	meteringDoc  = "kkp-docs: tutorials-howtos/metering"
	seedRefDoc   = "kkp-docs: references/crds seed.spec.metering"
	operatorCode = "kubermatic: pkg/ee/metering"
	toolCode     = "metering: pkg/report, pkg/datasource (v1.4.1)"
)

// Features are the metering checks in registration order.
var Features = []Feature{
	{ID: "M1", Name: "deploys the metering Prometheus with the configured storage and retention", Scope: ScopeSeed, Mode: ModeParallel, Doc: meteringDoc, Check: meteringtestkit.CheckPrometheusDeployment},
	{ID: "M2", Name: "creates one report CronJob per report configuration", Scope: ScopeSeed, Mode: ModeParallel, Doc: meteringDoc, Check: meteringtestkit.CheckReportCronJobs},
	{ID: "M3", Name: "scrapes the seed-controller-manager and federates the user cluster", Scope: ScopeSeed, Mode: ModeParallel, NeedsUserCluster: true, Doc: operatorCode, Check: meteringtestkit.CheckPrometheusTargets},

	{ID: "M10", Name: "rejects report intervals, retentions and formats the CRD forbids", Scope: ScopeSeed, Mode: ModeParallel, Doc: seedRefDoc, Check: meteringtestkit.CheckCRDRejectsInvalidConfig},
	{ID: "M10-webhook", Name: "rejects invalid report names, schedules, types and storage settings", Scope: ScopeSeed, Mode: ModeParallel, Doc: seedRefDoc, Check: meteringtestkit.CheckWebhookRejectsInvalidConfig,
		KnownIssue: seedWebhookIssue},
	{ID: "M11", Name: "accepts the documented report options", Scope: ScopeSeed, Mode: ModeParallel, Doc: seedRefDoc, Check: meteringtestkit.CheckAcceptsValidConfig},
	{ID: "M12", Name: "deploys every report configuration the Seed accepts", Scope: ScopeSeedWide, Mode: ModeSerial, Doc: operatorCode, Check: meteringtestkit.CheckAcceptedConfigIsDeployed,
		KnownIssue: "report names over 43 characters and CRON_TZ schedules pass validation, but CronJobs reject them (and " + seedWebhookIssue + ")"},
	{ID: "M16", Name: "produces reports from the Seed example in the metering docs", Scope: ScopeSeed, Mode: ModeParallel, Doc: meteringDoc, Check: meteringtestkit.CheckDocumentedSeedExample,
		KnownIssue: "the docs list the report types as Cluster and Namespace; the Seed stores them because its webhook does not run, and the metering tool ignores them, so the report stays empty"},
	{ID: "M17", Name: "runs the Seed validating webhook for the Seed", Scope: ScopeSeed, Mode: ModeParallel, Doc: "kubermatic: pkg/controller/operator/common/webhook.go", Check: meteringtestkit.CheckSeedWebhookRuns,
		KnownIssue: seedWebhookIssue},

	{ID: "M20", Name: "collects the cluster and project metadata of the user cluster", Scope: ScopeSeed, Mode: ModeParallel, NeedsUserCluster: true, Doc: operatorCode, Check: meteringtestkit.CheckClusterMetadataMetrics},
	{ID: "M21", Name: "federates every usage metric the reports read", Scope: ScopeSeed, Mode: ModeParallel, NeedsUserCluster: true, Doc: toolCode, Check: meteringtestkit.CheckUsageMetricContract},
	{ID: "M22", Name: "keeps one series per container despite two federation jobs", Scope: ScopeSeed, Mode: ModeParallel, NeedsUserCluster: true, Doc: operatorCode, Check: meteringtestkit.CheckNoDuplicateContainerSeries},
	{ID: "M24", Name: "keeps project label values when sanitizing changes their order", Scope: ScopeSeedWide, Mode: ModeSerial, NeedsUserCluster: true, Doc: "kubermatic: pkg/collectors", Check: meteringtestkit.CheckLabelValueOrder,
		KnownIssue: "kubermatic_cluster_labels and kubermatic_project_labels assign values in raw key order but names in sanitized order"},

	{ID: "M30", Name: "uploads the scheduled reports under the documented keys", Scope: ScopeSeed, Mode: ModeParallel, Doc: meteringDoc, Check: meteringtestkit.CheckScheduledReports},
	{ID: "M31", Name: "keeps the CSV columns and JSON fields consumers depend on", Scope: ScopeSeed, Mode: ModeParallel, NeedsUserCluster: true, Doc: meteringDoc, Check: meteringtestkit.CheckReportFormat},
	{ID: "M32", Name: "reports the user cluster's metadata and usage", Scope: ScopeSeed, Mode: ModeParallel, NeedsUserCluster: true, Doc: meteringDoc, Check: meteringtestkit.CheckClusterReportValues},
	{ID: "M33", Name: "reports the usage of the user cluster's namespaces", Scope: ScopeSeed, Mode: ModeParallel, NeedsUserCluster: true, Doc: meteringDoc, Check: meteringtestkit.CheckNamespaceReportValues},
	{ID: "M34", Name: "writes the same values to CSV and JSON reports", Scope: ScopeSeed, Mode: ModeParallel, NeedsUserCluster: true, Doc: seedRefDoc, Check: meteringtestkit.CheckCSVMatchesJSON},
	{ID: "M39", Name: "reports the data stored on persistent volumes", Scope: ScopeSeed, Mode: ModeParallel, NeedsUserCluster: true, Doc: toolCode, Check: meteringtestkit.CheckStorageReport},

	{ID: "M50", Name: "turns a report retention into a bucket lifecycle rule", Scope: ScopeSeed, Mode: ModeParallel, Doc: meteringDoc, Check: meteringtestkit.CheckRetentionRules},
	{ID: "M51", Name: "updates only the lifecycle rule of a changed report", Scope: ScopeSeedWide, Mode: ModeSerial, Doc: meteringDoc, Check: meteringtestkit.CheckRetentionUpdate},
	{ID: "M52", Name: "removes the lifecycle rule when the last retention is removed", Scope: ScopeSeedWide, Mode: ModeSerial, Doc: meteringDoc, Check: meteringtestkit.CheckRetentionRemoval,
		KnownIssue: "the operator skips the lifecycle update when no report has a retention, so the old rules stay"},

	{ID: "M60", Name: "adds, renames and deletes report configurations without losing reports", Scope: ScopeSeedWide, Mode: ModeSerial, Doc: meteringDoc, Check: meteringtestkit.CheckReportConfigLifecycle},
	{ID: "M61", Name: "keeps the collected data when metering is turned off and on", Scope: ScopeDisruptive, Mode: ModeLast, Doc: operatorCode, Check: meteringtestkit.CheckDisableAndReenable},
}

const (
	// needsUserClusterLabel marks checks that need a user cluster; skip them
	// with --label-filter='!needs-user-cluster'.
	needsUserClusterLabel = "needs-user-cluster"

	// knownIssueLabel marks checks that fail because of a confirmed product
	// bug; CI skips them with --label-filter='!known-issue'.
	knownIssueLabel = "known-issue"
)

// decorators turns the feature metadata into Ginkgo labels and ordering
// decorators.
func (f Feature) decorators() []any {
	labels := []string{"metering", "id:" + f.ID, "scope:" + string(f.Scope), "mode:" + string(f.Mode)}
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
