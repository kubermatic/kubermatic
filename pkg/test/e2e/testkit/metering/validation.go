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
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/util/wait"

	batchv1 "k8s.io/api/batch/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// validReport is a report configuration the webhook and the CRD accept. The
// cases change one field of it.
func validReport() map[string]any {
	return map[string]any{
		"schedule": "0 1 * * *",
		"interval": 1,
		"type":     []string{"cluster"},
	}
}

func withField(field string, value any) map[string]any {
	report := validReport()
	report[field] = value

	return report
}

// meteringPatch is a JSON merge patch for spec.metering. It is sent as raw
// JSON, so zero values reach the API server instead of being dropped by
// omitempty.
func meteringPatch(metering map[string]any) []byte {
	patch, _ := json.Marshal(map[string]any{"spec": map[string]any{"metering": metering}})
	return patch
}

func reportPatch(name string, report map[string]any) []byte {
	return meteringPatch(map[string]any{"reports": map[string]any{name: report}})
}

func (in Input) seedRef() *kubermaticv1.Seed {
	return &kubermaticv1.Seed{ObjectMeta: metav1.ObjectMeta{Namespace: in.SeedNamespace, Name: in.SeedName}}
}

// dryRunPatch sends the patch to the Seed with dry run, so the webhook and
// the CRD validation run but nothing is stored.
func dryRunPatch(ctx context.Context, in Input, patch []byte) error {
	return in.SeedClient.Patch(ctx, in.seedRef(), ctrlruntimeclient.RawPatch(types.MergePatchType, patch), ctrlruntimeclient.DryRunAll)
}

// rejectionCase is an invalid configuration and a part of the message the
// rejection must carry, so a rejection for an unrelated reason does not count.
type rejectionCase struct {
	patch   []byte
	message string
}

// checkRejected sends every case as a dry run and verifies it is rejected for
// the expected reason. A valid configuration is sent first: if that is
// rejected too, the Seed fails validation for a reason unrelated to metering
// and the cases prove nothing.
func checkRejected(ctx context.Context, in Input, cases map[string]rejectionCase) error {
	if err := dryRunPatch(ctx, in, reportPatch(in.ReportPrefix+"-v", validReport())); err != nil {
		return fmt.Errorf("a valid report configuration is rejected, so the Seed cannot be validated: %w", err)
	}

	var errs []error
	for name, c := range cases {
		err := dryRunPatch(ctx, in, c.patch)
		switch {
		case err == nil:
			errs = append(errs, fmt.Errorf("%s was accepted", name))
		case !strings.Contains(err.Error(), c.message):
			errs = append(errs, fmt.Errorf("%s was rejected for another reason, expected %q: %w", name, c.message, err))
		}
	}

	return errors.Join(errs...)
}

// CheckCRDRejectsInvalidConfig verifies that the Seed CRD schema rejects
// report intervals and retentions below one day and unknown formats.
func CheckCRDRejectsInvalidConfig(ctx context.Context, in Input) error {
	if _, _, err := in.meteringConfig(ctx); err != nil {
		return err
	}

	name := in.ReportPrefix + "-v"

	return checkRejected(ctx, in, map[string]rejectionCase{
		"interval 0":     {reportPatch(name, withField("interval", 0)), "interval"},
		"retention 0":    {reportPatch(name, withField("retention", 0)), "retention"},
		"unknown format": {reportPatch(name, withField("format", "xml")), "format"},
	})
}

// CheckWebhookRejectsInvalidConfig verifies that the Seed webhook rejects the
// metering configurations pkg/validation/metering.go forbids: report names
// that are no RFC 1035 label, invalid cron schedules, unknown report types, a
// negative Prometheus retention and an invalid storage size.
func CheckWebhookRejectsInvalidConfig(ctx context.Context, in Input) error {
	if _, _, err := in.meteringConfig(ctx); err != nil {
		return err
	}

	name := in.ReportPrefix + "-v"

	return checkRejected(ctx, in, map[string]rejectionCase{
		"name with a leading digit":     {reportPatch("1"+in.ReportPrefix, validReport()), "rfc1035"},
		"name with a dot":               {reportPatch(in.ReportPrefix+".a", validReport()), "rfc1035"},
		"name with upper case":          {reportPatch("Daily"+in.ReportPrefix, validReport()), "rfc1035"},
		"invalid cron schedule":         {reportPatch(name, withField("schedule", "every day")), "invalid cron expression"},
		"unknown report type":           {reportPatch(name, withField("type", []string{"pods"})), "invalid report type"},
		"upper case report type":        {reportPatch(name, withField("type", []string{"Cluster"})), "invalid report type"},
		"negative Prometheus retention": {meteringPatch(map[string]any{"retentionDays": -1}), "invalid retentionDays"},
		"invalid storage size":          {meteringPatch(map[string]any{"storageSize": "a lot"}), "invalid storageSize"},
	})
}

// CheckAcceptsValidConfig verifies that the Seed accepts the documented
// report options: a cron descriptor, the JSON format, monthly reports and a
// report without retention.
func CheckAcceptsValidConfig(ctx context.Context, in Input) error {
	if _, _, err := in.meteringConfig(ctx); err != nil {
		return err
	}

	name := in.ReportPrefix + "-v"
	cases := map[string][]byte{
		"cron descriptor":   reportPatch(name, withField("schedule", "@daily")),
		"JSON format":       reportPatch(name, withField("format", "json")),
		"monthly report":    reportPatch(name, withField("monthly", true)),
		"retention":         reportPatch(name, withField("retention", 30)),
		"both report types": reportPatch(name, withField("type", []string{"cluster", "namespace"})),
	}

	var errs []error
	for caseName, patch := range cases {
		if err := dryRunPatch(ctx, in, patch); err != nil {
			errs = append(errs, fmt.Errorf("%s was rejected: %w", caseName, err))
		}
	}

	return errors.Join(errs...)
}

// reportTypes are the report types the metering tool knows. It matches them
// case-sensitively and ignores any other value.
var reportTypes = []string{"cluster", "namespace"}

// CheckDocumentedSeedExample verifies that the report configuration of the
// Seed example in the metering docs produces reports: the Seed accepts it and
// stores report types the metering tool knows. The docs list the types as
// "Cluster" and "Namespace", which the tool ignores, so such a report runs
// and writes nothing.
func CheckDocumentedSeedExample(ctx context.Context, in Input) error {
	if _, _, err := in.meteringConfig(ctx); err != nil {
		return err
	}

	name := in.ReportPrefix + "-docs"
	report := map[string]any{
		"interval": 7,
		"schedule": "0 1 * * 6",
		"type":     []string{"Cluster", "Namespace"},
	}

	seed := in.seedRef()
	if err := in.SeedClient.Patch(ctx, seed, ctrlruntimeclient.RawPatch(types.MergePatchType, reportPatch(name, report)), ctrlruntimeclient.DryRunAll); err != nil {
		return fmt.Errorf("the Seed example from the metering docs is rejected: %w", err)
	}

	var errs []error
	for _, reportType := range seed.Spec.Metering.ReportConfigurations[name].Types {
		if !slices.Contains(reportTypes, reportType) {
			errs = append(errs, fmt.Errorf("the Seed example from the metering docs stores report type %q, which the metering tool ignores, so the report stays empty", reportType))
		}
	}

	return errors.Join(errs...)
}

// CheckSeedWebhookRuns verifies that the Seed validating webhook is called
// for the Seed: a report type only the webhook checks must be rejected by the
// webhook.
func CheckSeedWebhookRuns(ctx context.Context, in Input) error {
	err := dryRunPatch(ctx, in, reportPatch(in.ReportPrefix+"-v", withField("type", []string{"pods"})))
	if err == nil {
		return errors.New("an unknown report type was accepted: the Seed validating webhook did not run for this Seed")
	}

	if !apierrors.IsForbidden(err) || !strings.Contains(err.Error(), "admission webhook") {
		return fmt.Errorf("expected a rejection by the Seed admission webhook: %w", err)
	}

	return nil
}

// CheckAcceptedConfigIsDeployed verifies that report configurations the
// Seed webhook accepts also get a CronJob: a report name long enough to
// push the CronJob name over its 52 character limit, and a schedule with a
// time zone, which CronJobs reject in the schedule field. Either the webhook
// rejects them or the operator deploys them; accepting them and failing in
// the reconcile breaks the reconcile of the other reports.
func CheckAcceptedConfigIsDeployed(ctx context.Context, in Input) error {
	if _, _, err := in.meteringConfig(ctx); err != nil {
		return err
	}

	// "metering-" plus 44 characters is one more than a CronJob name may have
	longName := in.ReportPrefix + "-" + strings.Repeat("x", 44-len(in.ReportPrefix)-1)

	cases := []struct {
		name   string
		report string
		config map[string]any
	}{
		{name: "44 character report name", report: longName, config: validReport()},
		{name: "schedule with CRON_TZ", report: in.ReportPrefix + "-tz", config: withField("schedule", "CRON_TZ=UTC 0 1 * * *")},
	}

	var errs []error
	for _, c := range cases {
		errs = append(errs, applyAndExpectCronJob(ctx, in, c.name, c.report, c.config))
	}

	return errors.Join(errs...)
}

func applyAndExpectCronJob(ctx context.Context, in Input, caseName, report string, config map[string]any) error {
	err := in.SeedClient.Patch(ctx, in.seedRef(), ctrlruntimeclient.RawPatch(types.MergePatchType, reportPatch(report, config)))
	if err != nil {
		if apierrors.IsInvalid(err) || apierrors.IsForbidden(err) || apierrors.IsBadRequest(err) {
			in.Log.Infow("Seed rejected the configuration", "case", caseName, "error", err)
			return nil
		}

		return fmt.Errorf("%s: unexpected error: %w", caseName, err)
	}

	defer func() {
		ctx, cancel := revertContext()
		defer cancel()

		if err := in.SeedClient.Patch(ctx, in.seedRef(), ctrlruntimeclient.RawPatch(types.MergePatchType, reportPatch(report, nil))); err != nil {
			in.Log.Warnw("Failed to remove the test report", "report", report, "error", err)
		}
	}()

	// look the CronJob up by its labels, so a fix that shortens long names
	// also passes
	err = wait.PollImmediate(ctx, 5*time.Second, 2*time.Minute, func(ctx context.Context) (error, error) {
		cronJobs := &batchv1.CronJobList{}
		if err := in.SeedClient.List(ctx, cronJobs, ctrlruntimeclient.InNamespace(in.SeedNamespace), ctrlruntimeclient.MatchingLabels{componentLabel: componentValue, nameLabel: report}); err != nil {
			return err, nil
		}

		if len(cronJobs.Items) == 0 {
			return errors.New("no CronJob yet"), nil
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("%s: the Seed accepted report %s, but it got no CronJob: %w", caseName, report, err)
	}

	return nil
}
