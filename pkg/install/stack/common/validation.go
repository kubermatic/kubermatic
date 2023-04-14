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

package common

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	semverlib "github.com/Masterminds/semver/v3"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	k8csemver "k8c.io/api/v3/pkg/semver"
	"k8c.io/kubermatic/v3/pkg/defaulting"
	"k8c.io/kubermatic/v3/pkg/install/stack"
)

func ValidateAllUserClustersAreCompatible(ctx context.Context, opt *stack.DeployOptions) []error {
	var errs []error

	// we need the actual, effective versioning configuration, which most users will
	// probably not override
	defaulted, err := defaulting.DefaultConfiguration(opt.KubermaticConfiguration, zap.NewNop().Sugar())
	if err != nil {
		return append(errs, fmt.Errorf("failed to apply default values to the KubermaticConfiguration: %w", err))
	}

	// list all userclusters
	clusters := kubermaticv1.ClusterList{}
	if err := opt.KubeClient.List(ctx, &clusters); err != nil {
		return append(errs, fmt.Errorf("failed to list user clusters: %w", err))
	}

	upgradeConstraints, constraintErrs := getAutoUpdateConstraints(defaulted)
	if len(constraintErrs) > 0 {
		return constraintErrs
	}

	// check that each cluster still matches the configured versions
	for _, cluster := range clusters.Items {
		clusterVersion := cluster.Spec.Version

		if !clusterVersionIsConfigured(clusterVersion, defaulted, upgradeConstraints) {
			errs = append(errs, fmt.Errorf("cluster %s (version %s) would not be supported anymore", cluster.Name, clusterVersion))
		}
	}

	return errs
}

func getAutoUpdateConstraints(defaultedConfig *kubermaticv1.KubermaticConfiguration) ([]*semverlib.Constraints, []error) {
	var errs []error

	upgradeConstraints := []*semverlib.Constraints{}

	for i, update := range defaultedConfig.Spec.Versions.Updates {
		// only consider automated updates, otherwise we might accept an unsupported
		// cluster that is never manually updated
		if update.Automatic == nil || !*update.Automatic {
			continue
		}

		from, err := semverlib.NewConstraint(update.From)
		if err != nil {
			errs = append(errs, fmt.Errorf("`from` constraint %q for update rule %d is invalid: %w", update.From, i, err))
			continue
		}

		upgradeConstraints = append(upgradeConstraints, from)
	}

	return upgradeConstraints, errs
}

func clusterVersionIsConfigured(version k8csemver.Semver, defaultedConfig *kubermaticv1.KubermaticConfiguration, constraints []*semverlib.Constraints) bool {
	// is this version still straight up supported?
	for _, configured := range defaultedConfig.Spec.Versions.Versions {
		if configured.Equal(&version) {
			return true
		}
	}

	sversion := version.Semver()

	// is an upgrade path defined from the current version to something else?
	for _, update := range constraints {
		if update.Check(sversion) {
			return true
		}
	}

	return false
}

func ValidateRandomSecret(value string, path string) error {
	if value == "" {
		secret, err := randomString()
		if err == nil {
			return fmt.Errorf("%s must be a non-empty secret, for example: %s", path, secret)
		}

		return fmt.Errorf("%s must be a non-empty secret", path)
	}

	return nil
}

func randomString() (string, error) {
	c := 32
	b := make([]byte, c)

	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.RawURLEncoding.EncodeToString(b), nil
}
