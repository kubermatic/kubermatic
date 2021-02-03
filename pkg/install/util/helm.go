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

package util

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"k8c.io/kubermatic/v2/pkg/install/helm"
	"k8c.io/kubermatic/v2/pkg/util/yamled"
)

// CheckHelmRelease tries to find an existing Helm release in the cluster
// and returns it. If the release is still in a pending state, the function
// will wait a bit for it to settle down. If the release status is failed
// for whatever reason, the release is purged to allow clean re-installs.
func CheckHelmRelease(ctx context.Context, log logrus.FieldLogger, helmClient helm.Client, namespace string, releaseName string) (*helm.Release, error) {
	// find possible pre-existing release
	log.WithField("name", releaseName).WithField("namespace", namespace).Debug("Checking for release…")

	release, err := helmClient.GetRelease(namespace, releaseName)
	if err != nil {
		return nil, fmt.Errorf("failed to check for an existing release: %v", err)
	}

	// release exists already, check if it's valid
	if release != nil {
		log.WithFields(logrus.Fields{
			"version": release.Version,
			"status":  release.Status,
		}).Debug("Existing release found.")

		// Sometimes installations can fail because prerequisites were not setup properly,
		// like a missing storage class. In this case, we want to allow the user to just
		// run the installer again and pick up where they left. Unfortunately Helm does not
		// support "upgrade --install" on failed installations: https://github.com/helm/helm/issues/3353
		// To work around this, we check the release status and purge it manually if it's failed.
		if statusRequiresPurge(release.Status) {
			log.Warn("Uninstalling defunct release before a clean installation is attempted…")

			if err := helmClient.UninstallRelease(namespace, releaseName); err != nil {
				return release, fmt.Errorf("failed to uninstall release %s: %v", releaseName, err)
			}

			release = nil
			log.Info("Release has been uninstalled.")
		}

		// Now we have either a stable release or nothing at all.
	}

	return release, nil
}

func DeployHelmChart(ctx context.Context, log logrus.FieldLogger, helmClient helm.Client, chart *helm.Chart, namespace string, releaseName string, values *yamled.Document, atomic bool, force bool, release *helm.Release) error {
	switch {
	case release == nil:
		log.Debug("Installing…")

	case release.Version.GreaterThan(chart.Version):
		log.Infof("Downgrading release from %s to %s…", release.Version, chart.Version)

	case release.Version.LessThan(chart.Version):
		log.Infof("Updating release from %s to %s…", release.Version, chart.Version)

	case force:
		log.Info("Re-installing because --force is set…")

	default:
		// check if the provided Helm values differ from the ones used to install the
		// existing release; if there are changes, perform an update
		appliedValues, err := helmClient.GetValues(namespace, releaseName)
		if err != nil {
			return fmt.Errorf("failed to retrieve Helm values used for release: %v", err)
		}

		if appliedValues.Equal(values) {
			log.Info("Release is up-to-date, nothing to do. Set --force to re-install anyway.")
			return nil
		}

		log.Info("Re-installing because values have been changed…")
	}

	helmValues, err := dumpHelmValues(values)
	if helmValues != "" {
		defer os.Remove(helmValues)
	}
	if err != nil {
		return err
	}

	flags := []string{}
	if atomic {
		flags = append(flags, "--atomic")
	}

	if err := helmClient.InstallChart(namespace, releaseName, chart.Directory, helmValues, nil, flags); err != nil {
		return fmt.Errorf("failed to install: %v", err)
	}

	return nil
}

// purging pending releases is intended with Helm 3, see
// https://github.com/helm/helm/issues/5595#issuecomment-634186584 for more information
func statusRequiresPurge(status helm.ReleaseStatus) bool {
	return status == helm.ReleaseStatusFailed ||
		status == helm.ReleaseStatusPendingInstall ||
		status == helm.ReleaseStatusPendingRollback ||
		status == helm.ReleaseStatusPendingUpgrade
}

func dumpHelmValues(values *yamled.Document) (string, error) {
	f, err := ioutil.TempFile("", "helmvalues.*")
	if err != nil {
		return "", err
	}
	defer f.Close()

	err = yaml.NewEncoder(f).Encode(values)
	if err != nil {
		err = fmt.Errorf("failed to write Helm values to file: %v", err)
	}

	return f.Name(), err
}
