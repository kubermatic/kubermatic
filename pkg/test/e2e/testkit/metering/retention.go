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

	"github.com/minio/minio-go/v7"

	"k8c.io/kubermatic/v2/pkg/util/wait"

	"k8s.io/utils/ptr"
)

// retentionTimeout covers the Seed reconcile that sets the lifecycle rules.
const retentionTimeout = 3 * time.Minute

// expiryRule is the part of a bucket lifecycle rule metering sets.
type expiryRule struct {
	prefix string
	days   int
}

// lifecycleRules returns the enabled expiration rules of the metering
// bucket by ID.
func lifecycleRules(ctx context.Context, in Input) (map[string]expiryRule, error) {
	s3, err := meteringBucket(ctx, in)
	if err != nil {
		return nil, err
	}
	defer s3.stop()

	config, err := s3.client.GetBucketLifecycle(ctx, s3.bucket)
	if err != nil {
		if minio.ToErrorResponse(err).Code == "NoSuchLifecycleConfiguration" {
			return map[string]expiryRule{}, nil
		}

		return nil, fmt.Errorf("failed to get the bucket lifecycle: %w", err)
	}

	rules := map[string]expiryRule{}
	for _, rule := range config.Rules {
		if rule.Status != "Enabled" {
			continue
		}

		prefix := rule.RuleFilter.Prefix
		if prefix == "" {
			prefix = rule.Prefix
		}

		rules[rule.ID] = expiryRule{prefix: prefix, days: int(rule.Expiration.Days)}
	}

	return rules, nil
}

// waitForRules waits until the rules of the given reports match: a report
// with days > 0 needs a rule with that expiry and its name as prefix, a
// report with days 0 must have no rule.
func waitForRules(ctx context.Context, in Input, expected map[string]int) error {
	var last map[string]expiryRule

	err := wait.PollImmediate(ctx, 5*time.Second, retentionTimeout, func(ctx context.Context) (error, error) {
		rules, err := lifecycleRules(ctx, in)
		if err != nil {
			return err, nil
		}
		last = rules

		for report, days := range expected {
			rule, ok := rules[report]

			switch {
			case days == 0 && ok:
				return fmt.Errorf("report %s has no retention but still a lifecycle rule", report), nil
			case days > 0 && !ok:
				return fmt.Errorf("report %s has no lifecycle rule", report), nil
			case days > 0 && (rule.days != days || rule.prefix != report):
				return fmt.Errorf("report %s has rule %+v, expected %d days for prefix %s", report, rule, days, report), nil
			}
		}

		return nil, nil
	})
	if err != nil {
		return fmt.Errorf("%w (bucket rules: %+v)", err, last)
	}

	return nil
}

// CheckRetentionRules verifies that a report retention becomes a bucket
// lifecycle rule for the report's prefix, and that a report without
// retention gets none.
func CheckRetentionRules(ctx context.Context, in Input) error {
	return waitForRules(ctx, in, map[string]int{
		in.CSVReport():  csvReportRetention,
		in.JSONReport(): 0,
	})
}

// setRetention changes the retention of the suite's reports; days 0 removes
// it. It returns a function that restores the previous values.
func setRetention(ctx context.Context, in Input, days map[string]uint32) (func(context.Context) error, error) {
	seed := in.seedRef()

	return patchWithRevert(ctx, in.SeedClient, seed, func() {
		for report, value := range days {
			config := seed.Spec.Metering.ReportConfigurations[report]
			if value == 0 {
				config.Retention = nil
			} else {
				config.Retention = ptr.To(value)
			}
			seed.Spec.Metering.ReportConfigurations[report] = config
		}
	})
}

// CheckRetentionUpdate verifies that adding or changing the retention of one
// report updates only that report's lifecycle rule, and that removing it
// while another report keeps one removes the rule.
func CheckRetentionUpdate(ctx context.Context, in Input) error {
	steps := []struct {
		name     string
		change   map[string]uint32
		expected map[string]int
	}{
		{"add a retention to the JSON report", map[string]uint32{in.JSONReport(): 3}, map[string]int{in.CSVReport(): csvReportRetention, in.JSONReport(): 3}},
		{"change the retention of the CSV report", map[string]uint32{in.CSVReport(): 4}, map[string]int{in.CSVReport(): 4, in.JSONReport(): 3}},
		{"remove the retention of the JSON report", map[string]uint32{in.JSONReport(): 0}, map[string]int{in.CSVReport(): 4, in.JSONReport(): 0}},
	}

	var reverts []func(context.Context) error
	defer func() {
		ctx, cancel := revertContext()
		defer cancel()

		for i := len(reverts) - 1; i >= 0; i-- {
			if err := reverts[i](ctx); err != nil {
				in.Log.Warnw("Failed to restore the report retention", "error", err)
			}
		}
	}()

	for _, step := range steps {
		revert, err := setRetention(ctx, in, step.change)
		if err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
		reverts = append(reverts, revert)

		if err := waitForRules(ctx, in, step.expected); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}

	return nil
}

// CheckRetentionRemoval verifies that removing the last report retention
// also removes its lifecycle rule. It needs a Seed where only the suite's
// CSV report has a retention.
func CheckRetentionRemoval(ctx context.Context, in Input) error {
	_, metering, err := in.meteringConfig(ctx)
	if err != nil {
		return err
	}

	for name, config := range metering.ReportConfigurations {
		if name != in.CSVReport() && config.Retention != nil {
			return fmt.Errorf("report %s also has a retention; this check needs the suite's CSV report to be the only one", name)
		}
	}

	revert, err := setRetention(ctx, in, map[string]uint32{in.CSVReport(): 0})
	if err != nil {
		return err
	}

	removed := waitForRules(ctx, in, map[string]int{in.CSVReport(): 0})

	revertCtx, cancel := revertContext()
	defer cancel()

	return errors.Join(removed, revert(revertCtx), waitForRules(ctx, in, map[string]int{in.CSVReport(): csvReportRetention}))
}
