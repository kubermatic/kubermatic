/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/onsi/ginkgo/reporters"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	argSeparator = ` \
    `
)

type ginkgoResult struct {
	logfile  string
	report   *reporters.JUnitTestSuite
	duration time.Duration
}

// executeGinkgoRunWithRetries executes the passed GinkgoRun and retries if it failed hard(Failed to execute the Ginkgo binary for example)
// Or if the JUnit report from Ginkgo contains failed tests.
// Only if Ginkgo failed hard, an error will be returned. If some tests still failed after retrying the run, the report will reflect that.
func (r *testRunner) executeGinkgoRunWithRetries(ctx context.Context, log *zap.SugaredLogger, scenario testScenario, run *ginkgoRun, client ctrlruntimeclient.Client) (ginkgoRes *ginkgoResult, err error) {
	const maxAttempts = 3

	attempts := 1
	defer func() {
		ginkgoAttemptsMetric.With(prometheus.Labels{
			"scenario": scenario.Name(),
			"run":      run.name,
		}).Set(float64(attempts))
		updateMetrics(log)
	}()

	for attempts = 1; attempts <= maxAttempts; attempts++ {
		ginkgoRes, err = r.executeGinkgoRun(ctx, log, run, client)

		if ginkgoRes != nil {
			ginkgoRuntimeMetric.With(prometheus.Labels{
				"scenario": scenario.Name(),
				"run":      run.name,
				"attempt":  strconv.Itoa(attempts),
			}).Set(ginkgoRes.duration.Seconds())
			updateMetrics(log)
		}

		if err != nil {
			// Something critical happened and we don't have a valid result
			log.Errorf("Failed to execute the Ginkgo run '%s': %v", run.name, err)
			continue
		}

		if ginkgoRes.report.Errors > 0 || ginkgoRes.report.Failures > 0 {
			msg := fmt.Sprintf("Ginkgo run '%s' had failed tests.", run.name)
			if attempts < maxAttempts {
				msg = fmt.Sprintf("%s. Retrying...", msg)
			}
			log.Info(msg)
			if r.printGinkoLogs {
				if err := printFileUnbuffered(ginkgoRes.logfile); err != nil {
					log.Infof("Error printing ginkgo logfile: %v", err)
				}
				log.Info("Successfully printed logfile")
			}
			continue
		}

		// Ginkgo run successfully and no test failed
		return ginkgoRes, err
	}

	return ginkgoRes, err
}

type ginkgoRun struct {
	name       string
	cmd        *exec.Cmd
	reportsDir string
	timeout    time.Duration
}

func (r *testRunner) getGinkgoRuns(
	log *zap.SugaredLogger,
	scenario testScenario,
	kubeconfigFilename,
	cloudConfigFilename string,
	cluster *kubermaticv1.Cluster,
) ([]*ginkgoRun, error) {
	kubeconfigFilename = path.Clean(kubeconfigFilename)
	repoRoot := path.Clean(r.repoRoot)
	MajorMinor := fmt.Sprintf("%d.%d", cluster.Spec.Version.Major(), cluster.Spec.Version.Minor())

	nodeNumberTotal := int32(r.nodeCount)

	ginkgoSkipParallel := `\[Serial\]`
	if minor := cluster.Spec.Version.Minor(); minor >= 16 && minor <= 20 {
		// These require the nodes NodePort to be available from the tester, which is not the case for us.
		// TODO: Maybe add an option to allow the NodePorts in the SecurityGroup?
		ginkgoSkipParallel = strings.Join([]string{
			ginkgoSkipParallel,
			"Services should be able to change the type from ExternalName to NodePort",
			"Services should be able to create a functioning NodePort service",
			"Services should be able to switch session affinity for NodePort service",
			"Services should have session affinity timeout work for NodePort service",
			"Services should have session affinity work for NodePort service",
		}, "|")
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
			ginkgoSkip:    ginkgoSkipParallel,
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
	versionRoot := path.Join(repoRoot, MajorMinor)
	binRoot := path.Join(versionRoot, "/platforms/linux/amd64")
	var ginkgoRuns []*ginkgoRun
	for _, run := range runs {

		reportsDir := path.Join("/tmp", scenario.Name(), run.name)
		env := []string{
			// `kubectl diff` needs to find /usr/bin/diff
			fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
			fmt.Sprintf("HOME=%s", r.homeDir),
			fmt.Sprintf("AWS_SSH_KEY=%s", path.Join(r.homeDir, ".ssh", "google_compute_engine")),
			fmt.Sprintf("LOCAL_SSH_KEY=%s", path.Join(r.homeDir, ".ssh", "google_compute_engine")),
			fmt.Sprintf("KUBE_SSH_KEY=%s", path.Join(r.homeDir, ".ssh", "google_compute_engine")),
		}

		args := []string{
			"-progress",
			fmt.Sprintf("-nodes=%d", run.parallelTests),
			"-noColor=true",
			"-flakeAttempts=2",
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

		osSpec := scenario.OS()
		switch {
		case osSpec.Ubuntu != nil:
			args = append(args, "--node-os-distro=ubuntu")
			env = append(env, "KUBE_SSH_USER=ubuntu")
		case osSpec.Centos != nil:
			args = append(args, "--node-os-distro=centos")
			env = append(env, "KUBE_SSH_USER=centos")
		case osSpec.Flatcar != nil:
			args = append(args, "--node-os-distro=flatcar")
			env = append(env, "KUBE_SSH_USER=core")
		}

		cmd := exec.Command(path.Join(binRoot, "ginkgo"), args...)
		cmd.Env = env

		ginkgoRuns = append(ginkgoRuns, &ginkgoRun{
			name:       run.name,
			cmd:        cmd,
			reportsDir: reportsDir,
			timeout:    run.timeout,
		})
	}

	return ginkgoRuns, nil
}

func (r *testRunner) executeGinkgoRun(ctx context.Context, parentLog *zap.SugaredLogger, run *ginkgoRun, client ctrlruntimeclient.Client) (*ginkgoResult, error) {
	log := parentLog.With("reports-dir", run.reportsDir)

	if err := r.deleteAllNonDefaultNamespaces(ctx, log, client); err != nil {
		return nil, fmt.Errorf("failed to cleanup namespaces before the Ginkgo run: %v", err)
	}

	timedCtx, cancel := context.WithTimeout(ctx, run.timeout)
	defer cancel()

	// We're clearing up the temp dir on every run
	if err := os.RemoveAll(run.reportsDir); err != nil {
		log.Errorw("Failed to remove temporary reports directory", zap.Error(err))
	}
	if err := os.MkdirAll(run.reportsDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create temporary reports directory: %v", err)
	}

	// Make sure we write to a file instead of a byte buffer as the logs are pretty big
	logDirectory := r.containerLogsDirectory
	if len(logDirectory) == 0 {
		logDirectory = "/tmp"
	}

	file, err := ioutil.TempFile(logDirectory, "ginkgo-"+run.name+"*.log")
	if err != nil {
		return nil, fmt.Errorf("failed to open logfile: %v", err)
	}
	defer file.Close()
	log = log.With("ginkgo-log", file.Name())

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	started := time.Now()

	// Copy the command as we cannot execute a command twice
	cmd := exec.CommandContext(timedCtx, "")
	cmd.Path = run.cmd.Path
	cmd.Args = run.cmd.Args
	cmd.Env = run.cmd.Env
	cmd.Dir = run.cmd.Dir
	cmd.ExtraFiles = run.cmd.ExtraFiles
	if _, err := writer.Write([]byte(strings.Join(cmd.Args, argSeparator))); err != nil {
		return nil, fmt.Errorf("failed to write command to log: %v", err)
	}

	log.Infof("Starting Ginkgo run '%s'...", run.name)

	// Flush to disk so we can actually watch logs
	stopCh := make(chan struct{}, 1)
	defer close(stopCh)
	go wait.Until(func() {
		if err := writer.Flush(); err != nil {
			log.Warnw("Failed to flush log writer", zap.Error(err))
		}
		if err := file.Sync(); err != nil {
			log.Warnw("Failed to sync log file", zap.Error(err))
		}
	}, 1*time.Second, stopCh)

	cmd.Stdout = writer
	cmd.Stderr = writer

	if err := cmd.Run(); err != nil {
		// did the context's timeout kick in?
		if ctxErr := timedCtx.Err(); ctxErr != nil {
			return nil, ctxErr
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			log.Debugf("Ginkgo exited with a non-zero return code %d: %v", exitErr.ExitCode(), exitErr)
		} else {
			return nil, fmt.Errorf("ginkgo failed to start: %T %v", err, err)
		}
	}

	log.Debug("Ginkgo run completed, collecting reports...")

	// When running ginkgo in parallel, each ginkgo worker creates a own report, thus we must combine them
	combinedReport, err := collectReports(run.name, run.reportsDir)
	if err != nil {
		return nil, err
	}

	// If we have no junit files, we cannot return a valid report
	if len(combinedReport.TestCases) == 0 {
		return nil, errors.New("Ginkgo report is empty, it seems no tests where executed")
	}

	combinedReport.Time = time.Since(started).Seconds()

	log.Infof("Ginkgo run '%s' took %s", run.name, time.Since(started))
	return &ginkgoResult{
		logfile:  file.Name(),
		report:   combinedReport,
		duration: time.Since(started),
	}, nil
}
