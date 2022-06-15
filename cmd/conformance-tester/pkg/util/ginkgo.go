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

package util

import (
	"bufio"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/onsi/ginkgo/reporters"
	"go.uber.org/zap"

	"k8s.io/apimachinery/pkg/util/wait"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type GinkgoResult struct {
	Logfile  string
	Report   *reporters.JUnitTestSuite
	Duration time.Duration
}

const (
	argSeparator = ` \
    `
)

type GinkgoRun struct {
	Name       string
	Cmd        *exec.Cmd
	ReportsDir string
	Timeout    time.Duration
}

func (r *GinkgoRun) Run(ctx context.Context, parentLog *zap.SugaredLogger, client ctrlruntimeclient.Client) (*GinkgoResult, error) {
	log := parentLog.With("reports-dir", r.ReportsDir)

	timedCtx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	// We're clearing up the temp dir on every run
	if err := os.RemoveAll(r.ReportsDir); err != nil {
		log.Errorw("Failed to remove temporary reports directory", zap.Error(err))
	}
	if err := os.MkdirAll(r.ReportsDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create temporary reports directory: %w", err)
	}

	// Make sure we write to a file instead of a byte buffer as the logs are pretty big
	file, err := os.CreateTemp("/tmp", r.Name+"-log")
	if err != nil {
		return nil, fmt.Errorf("failed to open logfile: %w", err)
	}
	defer file.Close()
	log = log.With("ginkgo-log", file.Name())

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	started := time.Now()

	// Copy the command as we cannot execute a command twice
	cmd := exec.CommandContext(timedCtx, "")
	cmd.Path = r.Cmd.Path
	cmd.Args = r.Cmd.Args
	cmd.Env = r.Cmd.Env
	cmd.Dir = r.Cmd.Dir
	cmd.ExtraFiles = r.Cmd.ExtraFiles
	if _, err := writer.Write([]byte(strings.Join(cmd.Args, argSeparator))); err != nil {
		return nil, fmt.Errorf("failed to write command to log: %w", err)
	}

	log.Infof("Starting Ginkgo run '%s'...", r.Name)

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

		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			log.Debugf("Ginkgo exited with a non-zero return code %d: %v", exitErr.ExitCode(), exitErr)
		} else {
			return nil, fmt.Errorf("ginkgo failed to start: %T %w", err, err)
		}
	}

	log.Debug("Ginkgo run completed, collecting reports...")

	// When running ginkgo in parallel, each ginkgo worker creates a own report, thus we must combine them
	combinedReport, err := collectReports(r.Name, r.ReportsDir)
	if err != nil {
		return nil, err
	}

	// If we have no junit files, we cannot return a valid report
	if len(combinedReport.TestCases) == 0 {
		return nil, errors.New("Ginkgo report is empty, it seems no tests where executed")
	}

	combinedReport.Time = time.Since(started).Seconds()

	log.Infof("Ginkgo run '%s' took %s", r.Name, time.Since(started))
	return &GinkgoResult{
		Logfile:  file.Name(),
		Report:   combinedReport,
		Duration: time.Since(started),
	}, nil
}

func collectReports(name, reportsDir string) (*reporters.JUnitTestSuite, error) {
	files, err := os.ReadDir(reportsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to list files in reportsDir '%s': %w", reportsDir, err)
	}

	resultSuite := &reporters.JUnitTestSuite{Name: name}

	var individualReportFiles []string
	for _, f := range files {
		if f.IsDir() {
			continue
		}

		if !strings.HasPrefix(f.Name(), "junit_") || !strings.HasSuffix(f.Name(), ".xml") {
			continue
		}

		absName := path.Join(reportsDir, f.Name())
		individualReportFiles = append(individualReportFiles, absName)

		b, err := os.ReadFile(absName)
		if err != nil {
			return nil, fmt.Errorf("failed to read file '%s': %w", absName, err)
		}

		suite := &reporters.JUnitTestSuite{}
		if err := xml.Unmarshal(b, suite); err != nil {
			return nil, fmt.Errorf("failed to unmarshal report file '%s': %w", absName, err)
		}

		AppendReport(resultSuite, suite)
	}

	for _, f := range individualReportFiles {
		if err := os.Remove(f); err != nil {
			return nil, fmt.Errorf("failed to remove report file: %w", err)
		}
	}

	return resultSuite, nil
}

// AppendReport appends a reporters.JUnitTestSuite to another.
// During that process, test cases are deduplicated (identified via their name).
// Successful or failing test cases always take precedence over skipped test cases.
func AppendReport(report, toAppend *reporters.JUnitTestSuite) {
	report.Errors += toAppend.Errors

	for _, testCase := range toAppend.TestCases {
		for i, existingCase := range report.TestCases {
			if testCase.Name == existingCase.Name && testCase.ClassName == existingCase.ClassName {
				if testCase.Skipped == nil && existingCase.Skipped != nil {
					// override existing test case
					report.TestCases[i] = testCase
					if testCase.FailureMessage != nil {
						report.Failures += 1
					}
				}

				// skip over adding this test case
				continue
			}

			report.TestCases = append(report.TestCases, testCase)
			if testCase.FailureMessage != nil {
				report.Failures += 1
			}
			report.Tests += 1
		}
	}
}
