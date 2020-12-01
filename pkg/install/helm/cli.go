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
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
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

func (c *cli) InstallChart(namespace string, releaseName string, chartDirectory string, valuesFile string, values map[string]string, flags []string) error {
	command := []string{
		"upgrade",
		"--install",
		"--values", valuesFile,
		"--timeout", c.timeout.String(),
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
		return nil, fmt.Errorf("failed to list releases: %v", err)
	}

	releases := []Release{}
	if err := json.NewDecoder(bytes.NewReader(output)).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to parse Helm output: %v", err)
	}

	for idx, release := range releases {
		nameParts := strings.Split(release.Chart, "-")
		tail := nameParts[len(nameParts)-1]

		version, err := semver.NewVersion(tail)
		if err == nil {
			releases[idx].Version = version
		}

		releases[idx].Chart = strings.Join(nameParts[:len(nameParts)-1], "-")
	}

	return releases, nil
}

func (c *cli) UninstallRelease(namespace string, name string) error {
	_, err := c.run(namespace, "uninstall", name)

	return err
}

func (c *cli) RenderChart(namespace string, releaseName string, chartDirectory string, valuesFile string, values map[string]string) ([]byte, error) {
	command := []string{"template"}

	if valuesFile != "" {
		command = append(command, "--values", valuesFile)
	}

	command = append(command, valuesToFlags(values)...)
	command = append(command, releaseName, chartDirectory)

	return c.run(namespace, command...)
}

func (c *cli) Version() (*semver.Version, error) {
	// add --client to gracefully handle Helm 2 (Helm 3 ignores the flag, thankfully);
	// Helm 2 will output "<no value>", whereas Helm 3 would outright reject the
	// Helm-2-style templating string "{{ .Client.SemVer }}"
	output, err := c.run("", "version", "--client", "--template", "{{ .Version }}")
	if err != nil {
		return nil, err
	}

	out := strings.TrimSpace(string(output))
	if out == "<no value>" {
		out = "v2.99.99"
	}

	return semver.NewVersion(out)
}

func (c *cli) run(namespace string, args ...string) ([]byte, error) {
	globalArgs := []string{}

	if c.kubeContext != "" {
		globalArgs = append(globalArgs, "--kube-context", c.kubeContext)
	}

	if namespace != "" {
		globalArgs = append(globalArgs, "--namespace", namespace)
	}

	cmd := exec.Command(c.binary, append(globalArgs, args...)...)
	cmd.Env = append(cmd.Env, "KUBECONFIG="+c.kubeconfig)

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
