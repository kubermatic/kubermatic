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

package helm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"

	"k8c.io/kubermatic/v2/pkg/util/yamled"
)

type cli struct {
	binary      string
	kubeconfig  string
	kubeContext string
	timeout     time.Duration
	logger      logrus.FieldLogger
}

// NewCLI returns a new Client implementation that uses a local helm
// binary to perform chart installations.
func NewCLI(binary string, kubeconfig string, kubeContext string, timeout time.Duration, logger logrus.FieldLogger) (Client, error) {
	if timeout.Seconds() < 10 {
		return nil, errors.New("timeout must be >= 10 seconds")
	}

	return &cli{
		binary:      binary,
		kubeconfig:  kubeconfig,
		kubeContext: kubeContext,
		timeout:     timeout,
		logger:      logger,
	}, nil
}

func (c *cli) BuildChartDependencies(chartDirectory string, flags []string) (err error) {
	chart, err := LoadChart(chartDirectory)
	if err != nil {
		return err
	}

	for idx, dep := range chart.Dependencies {
		// Skip OCI repositories as they don't need to be added to helm repos
		if strings.HasPrefix(dep.Repository, "oci://") {
			c.logger.Debugf("Skipping OCI repository: %s", dep.Repository)
			continue
		}

		repoName := fmt.Sprintf("dep-%s-%d", chart.Name, idx)
		repoAddFlags := []string{
			"repo",
			"add",
			repoName,
			dep.Repository,
		}

		repoRemoveFlags := []string{
			"repo",
			"remove",
			repoName,
		}

		if _, err = c.run("default", repoAddFlags...); err != nil {
			return err
		}

		defer func() {
			_, removeErr := c.run("default", repoRemoveFlags...)
			if err != nil && removeErr != nil {
				err = fmt.Errorf("%w; error: clean up resources failed: cannot remove repository: %w", err, removeErr)
			}
			if err == nil && removeErr != nil {
				err = fmt.Errorf("error: clean up resources failed: cannot remove repository: %w", removeErr)
			}
		}()
	}

	command := []string{
		"dependency",
		"build",
	}

	command = append(command, chartDirectory)

	_, err = c.run("default", command...)

	return err
}

func (c *cli) InstallChart(namespace string, releaseName string, chartDirectory string, valuesFile string, values map[string]string, flags []string) error {
	command := []string{
		"upgrade",
		"--install",
		"--timeout", c.timeout.String(),
	}

	if valuesFile != "" {
		command = append(command, "--values", valuesFile)
	} else {
		command = append(command, "--reset-values")
	}

	command = append(command, valuesToFlags(values)...)
	command = append(command, flags...)
	command = append(command, releaseName, chartDirectory)

	_, err := c.run(namespace, command...)

	return err
}

func (c *cli) GetRelease(namespace string, name string) (*Release, error) {
	releases, err := c.ListReleases(namespace)
	if err != nil {
		return nil, err
	}

	for idx, r := range releases {
		if r.Namespace == namespace && r.Name == name {
			return &releases[idx], nil
		}
	}

	return nil, nil
}

func (c *cli) ListReleases(namespace string) ([]Release, error) {
	args := []string{"list", "--all", "-o", "json"}
	if namespace == "" {
		args = append(args, "--all-namespaces")
	}

	output, err := c.run(namespace, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list releases: %w", err)
	}

	releases := []Release{}
	if err := json.NewDecoder(bytes.NewReader(output)).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse Helm output: %w", err)
	}

	for idx, release := range releases {
		version, chart, err := guessChartName(release.Chart)
		if err != nil {
			return nil, fmt.Errorf("failed to determine version of release %s: %w", release.Name, err)
		}

		releases[idx].Chart = chart
		releases[idx].Version = version
	}

	return releases, nil
}

func guessChartName(fullChart string) (*semverlib.Version, string, error) {
	parts := strings.Split(fullChart, "-")
	if len(parts) == 1 {
		return nil, "", fmt.Errorf("%q is too short to be <chart>-<version>", fullChart)
	}

	chartName := parts[0]
	parts = parts[1:]

	for len(parts) > 0 {
		versionString := strings.Join(parts, "-")

		// have we found a valid version?
		version, err := semverlib.NewVersion(versionString)
		if err == nil {
			return version, chartName, nil
		}

		// not a valid version, treat the first part as part of the chart name
		chartName = fmt.Sprintf("%s-%s", chartName, parts[0])
		parts = parts[1:]
	}

	return nil, "", fmt.Errorf("cannot determine chart and release from %q", fullChart)
}

func (c *cli) UninstallRelease(namespace string, name string) error {
	_, err := c.run(namespace, "uninstall", name)

	return err
}

func (c *cli) RenderChart(namespace string, releaseName string, chartDirectory string, valuesFile string, values map[string]string, flags []string) ([]byte, error) {
	command := []string{"template"}

	if valuesFile != "" {
		command = append(command, "--values", valuesFile)
	}

	command = append(command, flags...)
	command = append(command, valuesToFlags(values)...)
	command = append(command, releaseName, chartDirectory)

	return c.run(namespace, command...)
}

func (c *cli) GetValues(namespace string, releaseName string) (*yamled.Document, error) {
	command := []string{"get", "values", releaseName, "-o", "yaml"}

	output, err := c.run(namespace, command...)
	if err != nil {
		return nil, err
	}

	return yamled.Load(bytes.NewReader(output))
}

func (c *cli) Version() (*semverlib.Version, error) {
	// Helm 2 will output "<no value>", whereas Helm 3 would outright reject the
	// Helm-2-style templating string "{{ .Client.SemVer }}"
	output, err := c.run("", "version", "--template", "{{ .Version }}")
	if err != nil {
		return nil, err
	}

	out := strings.TrimSpace(string(output))
	if out == "<no value>" {
		out = "v2.99.99"
	}

	return semverlib.NewVersion(out)
}

func (c *cli) run(namespace string, args ...string) ([]byte, error) {
	globalArgs := []string{}

	if c.kubeContext != "" {
		globalArgs = append(globalArgs, "--kube-context", c.kubeContext)
	}

	if namespace != "" {
		globalArgs = append(globalArgs, "--namespace", namespace)
	}

	cmd := exec.CommandContext(context.Background(), c.binary, append(globalArgs, args...)...)
	// "If Env contains duplicate environment keys, only the last
	// value in the slice for each duplicate key is used."
	// Source: https://pkg.go.dev/os/exec#Cmd.Env
	cmd.Env = append(os.Environ(), "KUBECONFIG="+c.kubeconfig)

	c.logger.Debugf("$ KUBECONFIG=%s %s", c.kubeconfig, strings.Join(cmd.Args, " "))

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd.Stdout = stdout
	cmd.Stderr = stderr

	err := cmd.Run()
	if err != nil {
		err = errors.New(strings.TrimSpace(stderr.String()))
	}

	return stdout.Bytes(), err
}

func valuesToFlags(values map[string]string) []string {
	set := make([]string, 0, len(values))

	for name, value := range values {
		set = append(set, fmt.Sprintf("%s=%s", name, value))
	}

	if len(set) > 0 {
		return []string{"--set", strings.Join(set, ",")}
	}

	return nil
}
