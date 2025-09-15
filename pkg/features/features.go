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

package features

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// OIDCKubeCfgEndpoint if enabled exposes an HTTP endpoint for generating kubeconfig for a cluster that will contain OIDC tokens.
	OIDCKubeCfgEndpoint = "OIDCKubeCfgEndpoint"

	// PrometheusEndpoint if enabled exposes cluster's metrics HTTP endpoint.
	PrometheusEndpoint = "PrometheusEndpoint"

	// OpenIDAuthPlugin if enabled configures the flags on the API server to use
	// OAuth2 identity providers.
	OpenIDAuthPlugin = "OpenIDAuthPlugin"

	// VerticalPodAutoscaler if enabled the cluster-controller will enable the
	// VerticalPodAutoscaler for all control plane components.
	VerticalPodAutoscaler = "VerticalPodAutoscaler"

	// EtcdDataCorruptionChecks if enabled etcd will be started with
	// --experimental-initial-corrupt-check=true +
	// --experimental-corrupt-check-time=10m.
	EtcdDataCorruptionChecks = "EtcdDataCorruptionChecks"

	// EtcdLauncher if enabled will apply the cluster level etcd-launcher feature flag on all clusters,
	// unless it's explicitly disabled at the cluster level.
	EtcdLauncher = "EtcdLauncher"

	// UserClusterMLA if enabled MonitoringLoggingAlerting stack will be deployed with corresponding controller.
	UserClusterMLA = "UserClusterMLA"

	// HeadlessInstallation feature makes the KKP installer not install nginx and Dex. This is useful to create
	// a KKP system without UI/API deployments, that will only be interacted with using kubectl or similar means.
	// Setting this feature flag will make KKP ignore any UI/API/Ingress configuration.
	// This feature is in preview and not yet ready for production.
	HeadlessInstallation = "HeadlessInstallation"

	// DevelopmentEnvironment feature enables additional controllers only useful in shared development clusters.
	// Currently this includes the kkp-cluster-stuck-controller, but additional tweaks might be added to this feature
	// gate in the future.
	// This feature perpetually in preview and never ready for production.
	DevelopmentEnvironment = "DevelopmentEnvironment"

	// DisableUserSSHKey if enabled disables the SSH key functionality in KKP. This will prevent users from managing SSH keys
	// and disable related controllers and components such as the userSSHKeySynchronizerFactoryCreator and
	// usersshkeyprojectownershipcontroller.
	DisableUserSSHKey = "DisableUserSSHKey"

	// ExternalApplicationCatalogManager enables the external application catalog manager.
	// It allows the new Application Catalog manager to work, and prevent the current controllers in KKP master to
	// reconcile ApplicationDefinitions of the default catalog.
	// Setting this feature flag to true will delegate the ApplicationDefinition reconciliation
	// responsibility to the new external (out-tree) application catalog controller manager.
	ExternalApplicationCatalogManager = "ExternalApplicationCatalogManager"

	// DynamicResourceAllocation if enabled, it lets Kubernetes allocate resources to your Pods with DRA.
	DynamicResourceAllocation = "DynamicResourceAllocation"
)

// FeatureGate is map of key=value pairs that enables/disables various features.
type FeatureGate map[string]bool

// NewFeatures takes comma separated key=value pairs for features
// and returns a FeatureGate.
func NewFeatures(rawFeatures string) (FeatureGate, error) {
	fGate := FeatureGate{}
	return fGate, fGate.Set(rawFeatures)
}

func (f FeatureGate) String() string {
	activeFeatures := make([]string, 0, len(f))
	for f, isActive := range f {
		if isActive {
			activeFeatures = append(activeFeatures, f)
		}
	}
	return strings.Join(activeFeatures, ", ")
}

func (f FeatureGate) Set(s string) error {
	for _, s := range strings.Split(s, ",") {
		if len(s) == 0 {
			continue
		}

		arr := strings.SplitN(s, "=", 2)
		if len(arr) != 2 {
			return fmt.Errorf("missing bool value for feature gate %s", s)
		}

		k := strings.TrimSpace(arr[0])
		v := strings.TrimSpace(arr[1])

		boolValue, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("invalid value %v for feature gate %s, use true|false instead", v, k)
		}
		f[k] = boolValue
	}
	return nil
}

// Enabled returns true if the feature gate value of a particular feature is true.
func (f FeatureGate) Enabled(feature string) bool {
	if value, ok := f[feature]; ok {
		return value
	}

	return false
}
