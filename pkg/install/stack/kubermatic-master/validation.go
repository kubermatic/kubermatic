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

package kubermaticmaster

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	k8csemver "k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/install/stack"
	"k8c.io/kubermatic/v2/pkg/install/util"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/edition"
	"k8c.io/kubermatic/v2/pkg/util/yamled"

	corev1 "k8s.io/api/core/v1"
)

func (*MasterStack) ValidateState(ctx context.Context, opt stack.DeployOptions) []error {
	var errs []error

	// validation can only happen if KKP was already installed, otherwise the resource types
	// won't even be known by the kube-apiserver
	crdsExists, err := util.HasAllReadyCRDs(ctx, opt.KubeClient, []string{
		"clusters.kubermatic.k8c.io",
		"seeds.kubermatic.k8c.io",
	})
	if err != nil {
		return append(errs, fmt.Errorf("failed to check for CRDs: %w", err))
	}

	if !crdsExists {
		return nil // nothing to do
	}

	// Ensure that no KKP upgrade was skipped.
	kkpMinorVersion := semverlib.MustParse(opt.Versions.GitVersion).Minor()
	minMinorRequired := kkpMinorVersion - 1

	// The configured KubermaticConfiguration might be a static YAML file,
	// which would not have a status set at all. To ensure that we always
	// get the currently live config, we fetch it from the cluster. This
	// dynamically fetched config is only relevant to the version check, all
	// other validations are supposed to be based on the given config.
	config, err := kubernetes.GetRawKubermaticConfiguration(ctx, opt.KubeClient, KubermaticOperatorNamespace)
	if err != nil && !errors.Is(err, provider.ErrNoKubermaticConfigurationFound) {
		return append(errs, fmt.Errorf("failed to fetch KubermaticConfiguration: %w", err))
	}

	var currentVersion string
	if config != nil {
		currentVersion = config.Status.KubermaticVersion
	}

	if currentVersion != "" {
		currentSemver, err := semverlib.NewVersion(currentVersion)
		if err != nil {
			return append(errs, fmt.Errorf("failed to parse existing KKP version %q: %w", currentVersion, err))
		}

		if currentSemver.Minor() < minMinorRequired {
			return append(errs, fmt.Errorf("existing installation is on version %s and must be updated to KKP 2.%d first (sequentially to all minor releases in-between)", currentVersion, kkpMinorVersion))
		}
	}

	// If a KubermaticConfiguration exists, check its status to compare editions.
	if config != nil && config.Status.KubermaticEdition != "" {
		currentEdition, err := edition.FromString(config.Status.KubermaticEdition)
		if err != nil {
			return append(errs, fmt.Errorf("failed to validate KKP edition: %w", err))
		}

		installerEdition := opt.Versions.KubermaticEdition

		if currentEdition != installerEdition {
			if opt.AllowEditionChange {
				opt.Logger.Warnf("This installation will change KKP to the %s.", installerEdition)
			} else {
				return append(errs, fmt.Errorf("existing installation uses the %s, refusing to change to %s (if this is intended, please add the --allow-edition-change flag)", currentEdition, installerEdition))
			}
		}
	}

	// we need the actual, effective versioning configuration, which most users will
	// probably not override
	defaulted, err := defaulting.DefaultConfiguration(opt.KubermaticConfiguration, zap.NewNop().Sugar())
	if err != nil {
		return append(errs, fmt.Errorf("failed to apply default values to the KubermaticConfiguration: %w", err))
	}

	allSeeds, err := opt.SeedsGetter()
	if err != nil {
		return append(errs, fmt.Errorf("failed to list Seeds: %w", err))
	}

	upgradeConstraints, constraintErrs := getAutoUpdateConstraints(defaulted)
	if len(constraintErrs) > 0 {
		return constraintErrs
	}

	for seedName, seed := range allSeeds {
		seedLog := opt.Logger.WithField("seed", seedName)

		if opt.SkipSeedValidation.Has(seedName) {
			seedLog.Info("Seed validation was skipped.")
			continue
		}

		seedLog.Info("Checking seed cluster…")

		// ensure seeds are also up-to-date before we continue
		seedVersion := seed.Status.Versions.Kubermatic
		if seedVersion != "" {
			seedSemver, err := semverlib.NewVersion(seedVersion)
			if err != nil {
				errs = append(errs, fmt.Errorf("Seed cluster %q version %q is invalid: %w", seedName, seedVersion, err))
				continue
			}

			if seedSemver.Minor() < minMinorRequired {
				errs = append(errs, fmt.Errorf("Seed cluster %q is on version %s and must be updated first", seedName, seedVersion))
				continue
			}
		}

		// if the operator has the chance to reconcile the seed...
		if conditions := seed.Status.Conditions; conditions[kubermaticv1.SeedConditionKubeconfigValid].Status == corev1.ConditionTrue {
			// ... it should be healthy
			if conditions[kubermaticv1.SeedConditionResourcesReconciled].Status != corev1.ConditionTrue {
				errs = append(errs, fmt.Errorf("Seed cluster %q is not healthy, please verify the operator logs or events on the Seed object", seedName))
				continue
			}
		}

		// create client into seed
		seedClient, err := opt.SeedClientGetter(seed)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to create client for Seed cluster %q: %w", seedName, err))
			continue
		}

		// list all userclusters
		clusters := kubermaticv1.ClusterList{}
		if err := seedClient.List(ctx, &clusters); err != nil {
			errs = append(errs, fmt.Errorf("failed to list user clusters on Seed %q: %w", seedName, err))
			continue
		}

		// check that each cluster still matches the configured versions
		for _, cluster := range clusters.Items {
			clusterVersion := cluster.Spec.Version

			if !clusterVersionIsConfigured(clusterVersion, defaulted, upgradeConstraints) {
				errs = append(errs, fmt.Errorf("cluster %s (version %s) on Seed %s would not be supported anymore", cluster.Name, clusterVersion, seedName))
			}

			// we effectively don't support docker in KKP 2.22; Kubernetes 1.23 clusters that are
			// still running it need to be upgraded before proceeding with the KKP upgrade.
			if cluster.Spec.ContainerRuntime == "docker" {
				errs = append(errs, fmt.Errorf("cluster %s on Seed %s is running 'docker' as container runtime; please upgrade it to 'containerd' before proceeding", cluster.Name, seedName))
			}
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

func (*MasterStack) ValidateConfiguration(config *kubermaticv1.KubermaticConfiguration, helmValues *yamled.Document, opt stack.DeployOptions, logger logrus.FieldLogger) (*kubermaticv1.KubermaticConfiguration, *yamled.Document, []error) {
	kubermaticFailures := validateKubermaticConfiguration(config)
	for idx, e := range kubermaticFailures {
		kubermaticFailures[idx] = prefixError("KubermaticConfiguration: ", e)
	}

	helmFailures := validateHelmValues(config, helmValues, logger)
	for idx, e := range helmFailures {
		helmFailures[idx] = prefixError("Helm values: ", e)
	}

	return config, helmValues, append(kubermaticFailures, helmFailures...)
}

func validateKubermaticConfiguration(config *kubermaticv1.KubermaticConfiguration) []error {
	failures := []error{}

	if config.Namespace != KubermaticOperatorNamespace {
		failures = append(failures, errors.New("the namespace must be \"kubermatic\""))
	}

	if config.Spec.Ingress.Domain == "" {
		failures = append(failures, errors.New("spec.ingress.domain cannot be left empty"))
	}

	// only validate auth-related keys if we are not setting up a headless system
	if !config.Spec.FeatureGates[features.HeadlessInstallation] {
		failures = validateRandomSecret(config, config.Spec.Auth.ServiceAccountKey, "spec.auth.serviceAccountKey", failures)

		if err := validateServiceAccountKey(config.Spec.Auth.ServiceAccountKey); err != nil {
			failures = append(failures, fmt.Errorf("spec.auth.serviceAccountKey is invalid: %w", err))
		}

		if config.Spec.FeatureGates[features.OIDCKubeCfgEndpoint] {
			failures = validateRandomSecret(config, config.Spec.Auth.IssuerClientSecret, "spec.auth.issuerClientSecret", failures)
			failures = validateRandomSecret(config, config.Spec.Auth.IssuerCookieKey, "spec.auth.issuerCookieKey", failures)
		}
	}

	return failures
}

func validateServiceAccountKey(privateKey string) error {
	if len(privateKey) == 0 {
		return errors.New("the signing key cannot be empty")
	}
	if len(privateKey) < 32 {
		return errors.New("the signing key is too short, use 32 bytes or longer")
	}
	return nil
}

func validateRandomSecret(config *kubermaticv1.KubermaticConfiguration, value string, path string, failures []error) []error {
	if value == "" {
		secret, err := randomString()
		if err == nil {
			failures = append(failures, fmt.Errorf("%s must be a non-empty secret, for example: %s", path, secret))
		} else {
			failures = append(failures, fmt.Errorf("%s must be a non-empty secret", path))
		}
	}

	return failures
}

type dexClient struct {
	ID string `yaml:"id"`
}

func validateHelmValues(config *kubermaticv1.KubermaticConfiguration, helmValues *yamled.Document, logger logrus.FieldLogger) []error {
	if helmValues.IsEmpty() {
		return []error{fmt.Errorf("No Helm Values file was provided, or the file was empty; installation cannot proceed. Please use the flag --helm-values=<valuesfile.yaml>")}
	}

	failures := []error{}

	path := yamled.Path{"kubermaticOperator", "imagePullSecret"}
	if value, _ := helmValues.GetString(path); value == "" {
		logger.Warnf("Helm values: %s is empty, setting to spec.imagePullSecret from KubermaticConfiguration", path.String())
		helmValues.Set(path, config.Spec.ImagePullSecret)
	}

	defaultedConfig, err := defaulting.DefaultConfiguration(config, zap.NewNop().Sugar())
	if err != nil {
		failures = append(failures, fmt.Errorf("failed to process KubermaticConfiguration: %w", err))
		return failures // must stop here, without defaulting the clientID check can be misleading
	}

	if !config.Spec.FeatureGates[features.HeadlessInstallation] {
		domainPath := yamled.Path{"dex", "ingress", "hosts", 0, "host"}
		clientPath := yamled.Path{"dex", "config", "staticClients"}
		connectorsPath := yamled.Path{"dex", "config", "connectors"}
		staticPasswordsPath := yamled.Path{"dex", "config", "staticPasswords"}

		if domain, _ := helmValues.GetString(domainPath); domain == "" {
			logger.WithField("domain", config.Spec.Ingress.Domain).Warnf("Helm values: %s is empty, setting to spec.ingress.domain from KubermaticConfiguration", domainPath.String())
			helmValues.Set(domainPath, config.Spec.Ingress.Domain)
		}

		clientID := defaultedConfig.Spec.Auth.ClientID
		hasDexIssues := false
		clients := []dexClient{}

		if err := helmValues.DecodeAtPath(clientPath, &clients); err != nil {
			hasDexIssues = true
			logger.Warn("Helm values: There are no Dex/OAuth clients configured.")
		} else {
			hasMatchingClient := false

			for _, client := range clients {
				if client.ID == clientID {
					hasMatchingClient = true
					break
				}
			}

			if !hasMatchingClient {
				hasDexIssues = true
				logger.Warnf("Helm values: The Dex configuration does not contain a `%s` client to allow logins to the Kubermatic dashboard.", clientID)
			}
		}

		connectors, _ := helmValues.GetArray(connectorsPath)
		staticPasswords, _ := helmValues.GetArray(staticPasswordsPath)

		if len(connectors) == 0 && len(staticPasswords) == 0 {
			hasDexIssues = true
			logger.Warn("Helm values: There are no connectors or static passwords configured for Dex.")
		}

		if len(staticPasswords) > 0 {
			if passwordDBEnabled, _ := helmValues.GetBool(yamled.Path{"dex", "config", "enablePasswordDB"}); !passwordDBEnabled {
				hasDexIssues = true
				logger.Warnf("Static passwords are defined but 'dex.config.enablePasswordDB' is not set to true. Password authentication will NOT work until you set it to true.")
			}
		}

		if hasDexIssues {
			logger.Warnf("If you intend to use Dex, please refer to the example configuration to define a `%s` client and connectors.", clientID)
		}
	}

	return failures
}

func prefixError(prefix string, e error) error {
	return fmt.Errorf("%s%w", prefix, e)
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

// isPublicIP validates whether ip provided is public.
func isPublicIP(ipAddress string) bool {
	ipAddr := net.ParseIP(ipAddress)
	return ipAddr != nil && !ipAddr.IsPrivate()
}
