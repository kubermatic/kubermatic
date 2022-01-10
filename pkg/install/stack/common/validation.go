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

	"github.com/Masterminds/semver/v3"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/controller/operator/defaults"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/install/stack"
)

func ValidateAllUserClustersAreCompatible(ctx context.Context, seed *kubermaticv1.Seed, opt *stack.DeployOptions) []error {
	var errs []error

	// we need the actual, effective versioning configuration, which most users will
	// probably not override
	defaulted, err := defaults.DefaultConfiguration(opt.KubermaticConfiguration, zap.NewNop().Sugar())
	if err != nil {
		return append(errs, fmt.Errorf("failed to apply default values to the KubermaticConfiguration: %w", err))
	}

	// create client into seed
	seedClient, err := opt.SeedClientGetter(seed)
	if err != nil {
		return append(errs, fmt.Errorf("failed to create client for Seed cluster %q: %w", seed.Name, err))
	}

	// list all userclusters
	clusters := kubermaticv1.ClusterList{}
	if err := seedClient.List(ctx, &clusters); err != nil {
		return append(errs, fmt.Errorf("failed to list user clusters on Seed %q: %w", seed.Name, err))
	}

	configuredVersions := defaulted.Spec.Versions.Kubernetes
	upgradeConstraints := []*semver.Constraints{}

	// do not parse and check the validity of constraints for each usercluster, but just once
	for i, update := range configuredVersions.Updates {
		// only consider automated updates, otherwise we might accept an unsupported
		// cluster that is never manually updated
		if update.Automatic == nil || !*update.Automatic {
			continue
		}

		from, err := semver.NewConstraint(update.From)
		if err != nil {
			errs = append(errs, fmt.Errorf("`from` constraint %q for update rule %d is invalid: %w", update.From, i, err))
			continue
		}

		upgradeConstraints = append(upgradeConstraints, from)
	}

	if len(errs) > 0 {
		return errs
	}

	// check that each cluster still matches the configured versions
	for _, cluster := range clusters.Items {
		clusterVersion := cluster.Spec.Version.Semver()
		validVersion := false

		// is this version still straight up supported?
		for _, configured := range configuredVersions.Versions {
			if configured.Equal(clusterVersion) {
				validVersion = true
				break
			}
		}

		if validVersion {
			continue
		}

		// is an upgrade path defined from the current version to something else?
		for _, update := range upgradeConstraints {
			if update.Check(clusterVersion) {
				validVersion = true
				break
			}
		}

		if !validVersion {
			errs = append(errs, fmt.Errorf("cluster %s (version %s) on Seed %s would not be supported anymore", cluster.Name, clusterVersion, seed.Name))
		}
	}

	return errs
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
