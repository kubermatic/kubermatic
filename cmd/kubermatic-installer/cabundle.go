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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
	goyaml "gopkg.in/yaml.v3"

	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/util/yamled"
)

// CABundleValuesFile is where "deploy" records the CA certificates that were given
// via --ca-bundle / --additional-ca-bundle, so that administrators end up with a Helm
// values file they can inspect, diff and commit to a GitOps repository.
const CABundleValuesFile = "ca-bundle.values.yaml"

const (
	// caBundleChartPath is the kubermatic-operator chart's location relative to the charts directory.
	caBundleChartPath = "kubermatic-operator"
	// caBundleFileName is the CA bundle that ships with the kubermatic-operator chart.
	caBundleFileName = "static/ca-bundle.pem"
	// caBundleExtraDir is the chart directory in which administrators can drop additional
	// PEM files; everything in here is appended to the rendered CA bundle.
	caBundleExtraDir = "static/extra-ca"
)

// caBundleValuesPath and caBundleAdditionalValuesPath address the Helm values that the
// kubermatic-operator chart reads to assemble its CA bundle.
var (
	caBundleValuesPath           = yamled.Path{"caBundle", "certificates"}
	caBundleAdditionalValuesPath = yamled.Path{"caBundle", "additionalCertificates"}
)

// shippedCABundleFile returns the path of the CA bundle that ships with the chart.
func shippedCABundleFile(chartsDirectory string) string {
	return filepath.Join(chartsDirectory, caBundleChartPath, caBundleFileName)
}

// extraCABundleFiles returns the PEM files an administrator dropped into the chart's
// static/extra-ca directory, sorted by path so that the result is deterministic and
// matches the order in which the Helm chart renders them.
func extraCABundleFiles(chartsDirectory string) ([]string, error) {
	pattern := filepath.Join(chartsDirectory, caBundleChartPath, caBundleExtraDir, "*.pem")

	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to scan %s: %w", filepath.Dir(pattern), err)
	}

	sort.Strings(files)

	return files, nil
}

// readCertificateFiles reads and validates the given PEM files and concatenates them in
// the given order. Every file must contain at least one parseable certificate, so that a
// typo or a DER-encoded file is rejected here instead of much later, when the operator
// fails to reconcile the resulting ConfigMap.
func readCertificateFiles(filenames []string) (string, error) {
	var builder strings.Builder

	for _, filename := range filenames {
		bundle, err := certificates.NewCABundleFromFile(filename)
		if err != nil {
			return "", fmt.Errorf("invalid CA bundle %s: %w", filename, err)
		}

		pem := strings.TrimRight(bundle.String(), "\n")
		if pem == "" {
			continue
		}

		if builder.Len() > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(pem)
		builder.WriteString("\n")
	}

	return builder.String(), nil
}

// localCABundle assembles the CA bundle exactly like the kubermatic-operator chart does for
// everything that is available on disk, i.e. the bundle shipped with the chart plus all PEM
// files in the chart's static/extra-ca directory. Certificates that are only configured via
// Helm values are not visible here; commands that need those accept an explicit --ca-bundle.
func localCABundle(chartsDirectory string) (*certificates.CABundle, error) {
	files := []string{shippedCABundleFile(chartsDirectory)}

	extra, err := extraCABundleFiles(chartsDirectory)
	if err != nil {
		return nil, err
	}
	files = append(files, extra...)

	pem, err := readCertificateFiles(files)
	if err != nil {
		return nil, err
	}

	return certificates.NewCABundleFromBytes([]byte(pem))
}

// findDuplicateDropIn returns the path of the file in the chart's static/extra-ca directory
// that holds the same certificates as the given PEM, if any. Certificates that are already
// dropped into the chart do not need to be configured via Helm values as well.
func findDuplicateDropIn(chartsDirectory string, pem string) (string, error) {
	needle := normalizeCertificates(pem)
	if needle == "" {
		return "", nil
	}

	files, err := extraCABundleFiles(chartsDirectory)
	if err != nil {
		return "", err
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to read %s: %w", file, err)
		}

		if haystack := normalizeCertificates(string(content)); haystack != "" && strings.Contains(haystack, needle) {
			return file, nil
		}
	}

	return "", nil
}

// normalizeCertificates strips comments, blank lines and trailing whitespace from a PEM
// blob, so that two files holding the same certificates compare equal even when their
// human-readable headers differ.
func normalizeCertificates(pem string) string {
	var lines []string

	inCertificate := false
	for _, line := range strings.Split(pem, "\n") {
		line = strings.TrimSpace(line)

		switch {
		case strings.HasPrefix(line, "-----BEGIN "):
			inCertificate = true
		case strings.HasPrefix(line, "-----END "):
			lines = append(lines, line)
			inCertificate = false
			continue
		case !inCertificate:
			continue
		}

		if line != "" {
			lines = append(lines, line)
		}
	}

	return strings.Join(lines, "\n")
}

// applyCABundleFlags injects the certificates given via --ca-bundle and --additional-ca-bundle
// into the Helm values, so that the kubermatic-operator chart renders them into its ca-bundle
// ConfigMap. Values configured in a Helm values file are overwritten, as the flags are the more
// specific configuration.
//
// The resulting values are additionally written to CABundleValuesFile. That file is only a
// record of what was installed; the installation itself uses the values returned here, so a
// failed write is logged and otherwise ignored.
func applyCABundleFlags(logger *logrus.Logger, helmValues *yamled.Document, chartsDirectory string, caBundle string, additionalCABundles []string) error {
	recorded := map[string]string{}

	set := func(path yamled.Path, filenames []string, flag string) error {
		if len(filenames) == 0 {
			return nil
		}

		pem, err := readCertificateFiles(filenames)
		if err != nil {
			return err
		}

		if helmValues.Has(path) {
			logger.Warnf("Both %s and the Helm value %s are configured, using %s.", flag, path.String(), flag)
		}

		if duplicate, err := findDuplicateDropIn(chartsDirectory, pem); err != nil {
			logger.Debugf("Failed to check for duplicate certificates: %v", err)
		} else if duplicate != "" {
			logger.Warnf("Certificates given via %s are also provided by %s and will end up in the CA bundle twice.", flag, duplicate)
		}

		if !helmValues.Set(path, pem) {
			return fmt.Errorf("failed to set Helm value %s", path.String())
		}

		recorded[path.End().(string)] = pem

		return nil
	}

	var replacement []string
	if caBundle != "" {
		replacement = []string{caBundle}
	}

	if err := set(caBundleValuesPath, replacement, "--ca-bundle"); err != nil {
		return err
	}

	if err := set(caBundleAdditionalValuesPath, additionalCABundles, "--additional-ca-bundle"); err != nil {
		return err
	}

	recordCABundleValues(logger, recorded)

	return nil
}

// recordCABundleValues writes the given certificates to CABundleValuesFile. Failing to do so
// is not fatal, as the file is a convenience for the administrator and not an input to the
// installation.
func recordCABundleValues(logger *logrus.Logger, certificates map[string]string) {
	if len(certificates) == 0 {
		return
	}

	// A failure here is not fatal: the certificates have already been applied to the Helm
	// values that drive this installation, so say so explicitly. Otherwise a read-only
	// working directory looks like the CA configuration itself failed.
	encoded, err := encodeCABundleValues(certificates)
	if err != nil {
		logger.Warnf("Could not encode %s: %v.", CABundleValuesFile, err)
		logger.Warn("The CA certificates were still applied to this installation; only the copy for your records is missing.")

		return
	}

	if err := os.WriteFile(CABundleValuesFile, []byte(encoded), 0644); err != nil {
		logger.Warnf("Could not write %s in the current directory: %v.", CABundleValuesFile, err)
		logger.Warn("The CA certificates were still applied to this installation; only the copy for your records is missing.")

		return
	}

	logger.Infof("Wrote the configured CA certificates to %s.", CABundleValuesFile)
}

// encodeCABundleValues renders the given certificates as a Helm values document. Every PEM is
// written as a literal block scalar, so that the resulting file stays readable and can be
// diffed line by line.
func encodeCABundleValues(certificates map[string]string) (string, error) {
	caBundle := &goyaml.Node{Kind: goyaml.MappingNode}

	// keep the keys in the order in which the chart applies them
	for _, key := range []string{"certificates", "additionalCertificates"} {
		pem, exists := certificates[key]
		if !exists {
			continue
		}

		caBundle.Content = append(caBundle.Content,
			&goyaml.Node{Kind: goyaml.ScalarNode, Tag: "!!str", Value: key},
			&goyaml.Node{Kind: goyaml.ScalarNode, Tag: "!!str", Style: goyaml.LiteralStyle, Value: pem},
		)
	}

	root := &goyaml.Node{
		Kind: goyaml.MappingNode,
		Content: []*goyaml.Node{
			{Kind: goyaml.ScalarNode, Tag: "!!str", Value: "caBundle"},
			caBundle,
		},
	}

	encoded, err := goyaml.Marshal(root)
	if err != nil {
		return "", fmt.Errorf("failed to encode Helm values: %w", err)
	}

	return string(encoded), nil
}
