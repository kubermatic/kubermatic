/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package tests

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/onsi/ginkgo/reporters"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/metrics"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/scenarios"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/util"
	"k8c.io/kubermatic/v2/pkg/util/wait"
	"k8c.io/machine-controller/sdk/providerconfig"

	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func TestKubernetesConformance(
	ctx context.Context,
	log *zap.SugaredLogger,
	opts *types.Options,
	scenario scenarios.Scenario,
	cluster *kubermaticv1.Cluster,
	userClusterClient ctrlruntimeclient.Client,
	kubeconfigFilename string,
	cloudConfigFilename string,
	report *reporters.JUnitTestSuite,
) error {
	if !opts.Tests.Has(types.ConformanceTests) {
		log.Info("Kubernetes conformance tests disabled, skipping.")
		return nil
	}

	ginkgoRuns, err := getGinkgoRuns(ctx, opts, scenario, kubeconfigFilename, cloudConfigFilename, cluster)
	if err != nil {
		return fmt.Errorf("failed to get Ginkgo runs: %w", err)
	}

	// Since we retry failed conformance tests up to 3 times, we want to ensure that the
	// cluster is as clean as possible and so we remove stray webhooks and namespaces.
	// In order not to have to maintain a list of default namespaces/webhooks, we scan the
	// cluster now and whatever exists now is deemed to be "default".
	namespaces, webhooks, err := getDefaultClusterContents(ctx, userClusterClient)
	if err != nil {
		return fmt.Errorf("failed to determine default cluster contents: %w", err)
	}

	failures := false

	// Run the ginkgo tests
	for _, run := range ginkgoRuns {
		if err := util.JUnitWrapper(fmt.Sprintf("[Ginkgo] Run ginkgo %q tests", run.Name), report, func() error {
			ginkgoRes, err := runGinkgoRunWithRetries(ctx, log, opts, scenario, run, userClusterClient, namespaces, webhooks)
			if ginkgoRes != nil {
				// We append the report from Ginkgo to our scenario wide report
				util.AppendReport(report, ginkgoRes.Report)
			}

			return err
		}); err != nil {
			log.Errorf("Ginkgo scenario '%s' failed, giving up retrying: %v", err)
			failures = true
		}
	}

	if failures {
		return errors.New("ginkgo run(s) failed")
	}

	return nil
}

func getDefaultClusterContents(ctx context.Context, client ctrlruntimeclient.Client) (sets.Set[string], sets.Set[string], error) {
	namespaceList := &corev1.NamespaceList{}
	if err := client.List(ctx, namespaceList); err != nil {
		return nil, nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	namespaces := sets.New[string]()
	for _, ns := range namespaceList.Items {
		namespaces.Insert(ns.Name)
	}

	webhookList := &admissionregistrationv1.ValidatingWebhookConfigurationList{}
	if err := client.List(ctx, webhookList); err != nil {
		return nil, nil, fmt.Errorf("failed to list webhooks: %w", err)
	}

	webhooks := sets.New[string]()
	for _, webhook := range webhookList.Items {
		webhooks.Insert(webhook.Name)
	}

	return namespaces, webhooks, nil
}

// runGinkgoRunWithRetries executes the passed GinkgoRun and retries if it failed hard(Failed to execute the Ginkgo binary for example)
// Or if the JUnit report from Ginkgo contains failed tests.
// Only if Ginkgo failed hard, an error will be returned. If some tests still failed after retrying the run, the report will reflect that.
func runGinkgoRunWithRetries(
	ctx context.Context,
	log *zap.SugaredLogger,
	opts *types.Options,
	scenario scenarios.Scenario,
	run *util.GinkgoRun,
	client ctrlruntimeclient.Client,
	namespacesToKeep sets.Set[string],
	webhooksToKeep sets.Set[string],
) (ginkgoRes *util.GinkgoResult, err error) {
	const maxAttempts = 3

	attempts := 1
	defer func() {
		metrics.GinkgoAttemptsMetric.With(prometheus.Labels{
			"scenario": scenario.Name(),
			"run":      run.Name,
		}).Set(float64(attempts))
		metrics.UpdateMetrics(log)
	}()

	for attempts = 1; attempts <= maxAttempts; attempts++ {
		ginkgoRes, err = runGinkgo(ctx, log, opts, run, client, namespacesToKeep, webhooksToKeep)

		if ginkgoRes != nil {
			metrics.GinkgoRuntimeMetric.With(prometheus.Labels{
				"scenario": scenario.Name(),
				"run":      run.Name,
				"attempt":  strconv.Itoa(attempts),
			}).Set(ginkgoRes.Duration.Seconds())
			metrics.UpdateMetrics(log)
		}

		if err != nil {
			// Something critical happened and we don't have a valid result
			log.Errorf("Failed to execute the Ginkgo run '%s': %v", run.Name, err)
			continue
		}

		if ginkgoRes.Report.Errors > 0 || ginkgoRes.Report.Failures > 0 {
			msg := fmt.Sprintf("Ginkgo run '%s' had failed tests.", run.Name)
			if attempts < maxAttempts {
				msg = fmt.Sprintf("%s. Retrying...", msg)
			}
			log.Info(msg)
			continue
		}

		// Ginkgo run successfully and no test failed
		return ginkgoRes, err
	}

	return ginkgoRes, err
}

func runGinkgo(
	ctx context.Context,
	parentLog *zap.SugaredLogger,
	opts *types.Options,
	run *util.GinkgoRun,
	client ctrlruntimeclient.Client,
	namespacesToKeep sets.Set[string],
	webhooksToKeep sets.Set[string],
) (*util.GinkgoResult, error) {
	log := parentLog.With("reports-dir", run.ReportsDir)

	if err := cleanupBeforeGinkgo(ctx, log, opts, client, namespacesToKeep, webhooksToKeep); err != nil {
		return nil, fmt.Errorf("failed to cleanup before the Ginkgo run: %w", err)
	}

	return run.Run(ctx, log, client)
}

func getGinkgoRuns(
	ctx context.Context,
	opts *types.Options,
	scenario scenarios.Scenario,
	kubeconfigFilename,
	cloudConfigFilename string,
	cluster *kubermaticv1.Cluster,
) ([]*util.GinkgoRun, error) {
	kubeconfigFilename = path.Clean(kubeconfigFilename)
	repoRoot := path.Clean(opts.RepoRoot)

	nodeNumberTotal := int32(opts.NodeCount)

	ginkgoSkipParallel := []string{
		`\[Serial\]`,
		// These require the nodes NodePort to be available from the tester, which is not the case for us.
		// TODO: Maybe add an option to allow the NodePorts in the SecurityGroup?
		"Services should be able to change the type from ExternalName to NodePort",
		"Services should be able to change the type from NodePort to ExternalName",
		"Services should be able to change the type from ClusterIP to ExternalName",
		"Services should be able to create a functioning NodePort service",
		"Services should be able to switch session affinity for NodePort service",
		"Services should have session affinity timeout work for NodePort service",
		"Services should have session affinity work for NodePort service",
	}

	// Cilium does not support Kubernetes 1.29 conformance tests yet,
	// see https://github.com/cilium/cilium/issues/29913
	if cluster.Spec.CNIPlugin.Type == kubermaticv1.CNIPluginTypeCilium {
		ginkgoSkipParallel = append(ginkgoSkipParallel, "Services should serve endpoints on same port and different protocols")
	}

	runs := []struct {
		name          string
		ginkgoFocus   string
		ginkgoSkip    string
		parallelTests int
		timeout       time.Duration
	}{
		{
			name:          "parallel",
			ginkgoFocus:   `\[Conformance\]`,
			ginkgoSkip:    strings.Join(ginkgoSkipParallel, "|"),
			parallelTests: int(nodeNumberTotal) * 3,
			timeout:       60 * time.Minute,
		},
		{
			name:          "serial",
			ginkgoFocus:   `\[Serial\].*\[Conformance\]`,
			ginkgoSkip:    `should not cause race condition when used for configmap`,
			parallelTests: 1,
			timeout:       60 * time.Minute,
		},
	}
	versionRoot := path.Join(repoRoot, cluster.Spec.Version.MajorMinor())
	binRoot := path.Join(versionRoot, "/platforms/linux/amd64")
	var ginkgoRuns []*util.GinkgoRun
	for _, run := range runs {
		reportsDir := path.Join(opts.ReportsRoot, scenario.Name(), run.name)
		env := []string{
			// `kubectl diff` needs to find /usr/bin/diff
			fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
			fmt.Sprintf("HOME=%s", opts.HomeDir),
			fmt.Sprintf("AWS_SSH_KEY=%s", path.Join(opts.HomeDir, ".ssh", "google_compute_engine")),
			fmt.Sprintf("LOCAL_SSH_KEY=%s", path.Join(opts.HomeDir, ".ssh", "google_compute_engine")),
			fmt.Sprintf("KUBE_SSH_KEY=%s", path.Join(opts.HomeDir, ".ssh", "google_compute_engine")),
		}

		args := []string{
			"-progress",
			fmt.Sprintf("-nodes=%d", run.parallelTests),
			"-noColor=true",
			"-flakeAttempts=3",
			fmt.Sprintf(`-focus=%s`, run.ginkgoFocus),
			fmt.Sprintf(`-skip=%s`, run.ginkgoSkip),
			path.Join(binRoot, "e2e.test"),
			"--",
			"--disable-log-dump",
			fmt.Sprintf("--repo-root=%s", versionRoot),
			fmt.Sprintf("--report-dir=%s", reportsDir),
			fmt.Sprintf("--report-prefix=%s", run.name),
			fmt.Sprintf("--kubectl-path=%s", path.Join(binRoot, "kubectl")),
			fmt.Sprintf("--kubeconfig=%s", kubeconfigFilename),
			fmt.Sprintf("--num-nodes=%d", nodeNumberTotal),
			fmt.Sprintf("--cloud-config-file=%s", cloudConfigFilename),
		}

		args = append(args, "--provider=local")

		switch scenario.OperatingSystem() {
		case providerconfig.OperatingSystemUbuntu:
			args = append(args, "--node-os-distro=ubuntu")
			env = append(env, "KUBE_SSH_USER=ubuntu")
		case providerconfig.OperatingSystemFlatcar:
			args = append(args, "--node-os-distro=flatcar")
			env = append(env, "KUBE_SSH_USER=core")
		}

		cmd := exec.CommandContext(ctx, path.Join(binRoot, "ginkgo"), args...)
		cmd.Env = env

		ginkgoRuns = append(ginkgoRuns, &util.GinkgoRun{
			Name:       run.name,
			Cmd:        cmd,
			ReportsDir: reportsDir,
			Timeout:    run.timeout,
		})
	}

	return ginkgoRuns, nil
}

func cleanupBeforeGinkgo(
	ctx context.Context,
	log *zap.SugaredLogger,
	opts *types.Options,
	client ctrlruntimeclient.Client,
	namespacesToKeep sets.Set[string],
	webhooksToKeep sets.Set[string],
) error {
	log.Info("Removing non-default webhooks...")

	if err := wait.PollImmediate(ctx, opts.UserClusterPollInterval, opts.CustomTestTimeout, func(ctx context.Context) (transient error, terminal error) {
		webhookList := &admissionregistrationv1.ValidatingWebhookConfigurationList{}
		if err := client.List(ctx, webhookList); err != nil {
			return fmt.Errorf("failed to list webhooks: %w", err), nil
		}

		remaining := sets.New[string]()
		for _, webhook := range webhookList.Items {
			if webhooksToKeep.Has(webhook.Name) {
				continue
			}

			if webhook.DeletionTimestamp == nil {
				wlog := log.With("webhook", webhook.Name)

				if err := client.Delete(ctx, &webhook); err != nil {
					wlog.Errorw("Failed to delete webhook", zap.Error(err))
				} else {
					wlog.Debug("Deleted webhook.")
				}
			}

			remaining.Insert(webhook.Name)
		}

		if remaining.Len() == 0 {
			return nil, nil
		}

		return fmt.Errorf("could not delete all webhooks: %v", sets.List(remaining)), nil
	}); err != nil {
		return err
	}

	log.Info("Removing non-default namespaces...")

	// For these we do not wait for the deletion to be done, as it's enough to trigger
	// the deletion and have the resources disappear over time.

	namespaceList := &corev1.NamespaceList{}
	if err := client.List(ctx, namespaceList); err != nil {
		log.Errorw("Failed to delete namespaces", zap.Error(err))
		return nil
	}

	for _, namespace := range namespaceList.Items {
		if namespacesToKeep.Has(namespace.Name) {
			continue
		}

		// If it's not gone & the DeletionTimestamp is nil, delete it
		if namespace.DeletionTimestamp == nil {
			nslog := log.With("namespace", namespace.Name)

			if err := client.Delete(ctx, &namespace); err != nil {
				nslog.Errorw("Failed to delete namespace", zap.Error(err))
			} else {
				nslog.Debug("Deleted namespace.")
			}
		}
	}

	return nil
}
