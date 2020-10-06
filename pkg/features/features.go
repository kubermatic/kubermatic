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
	// OIDCKubeCfgEndpoint if enabled exposes an HTTP endpoint for generating kubeconfig for a cluster that will contain OIDC tokens
	OIDCKubeCfgEndpoint = "OIDCKubeCfgEndpoint"

	// PrometheusEndpoint if enabled exposes cluster's metrics HTTP endpoint
	PrometheusEndpoint = "PrometheusEndpoint"

	// OpenIDAuthPlugin if enabled configures the flags on the API server to use
	// OAuth2 identity providers.
	OpenIDAuthPlugin = "OpenIDAuthPlugin"

	// VerticalPodAutoscaler if enabled the cluster-controller will enable the
	// VerticalPodAutoscaler for all control plane components
	VerticalPodAutoscaler = "VerticalPodAutoscaler"

	// EtcdDataCorruptionChecks if enabled etcd will be started with
	// --experimental-initial-corrupt-check=true +
	// --experimental-corrupt-check-time=10m
	EtcdDataCorruptionChecks = "EtcdDataCorruptionChecks"

	// EtcdLauncher if enabled will apply the cluster level etcd-launcher feature flag on all clusters,
	// unless it's explicitly disabled at the cluster level
	EtcdLauncher = "EtcdLauncher"
)

// FeatureGate is map of key=value pairs that enables/disables various features.
type FeatureGate map[string]bool

// NewFeatures takes comma separated key=value pairs for features
// and returns a FeatureGate.
func NewFeatures(rawFeatures string) (FeatureGate, error) {
	fGate := FeatureGate{}
	for _, s := range strings.Split(rawFeatures, ",") {
		if len(s) == 0 {
			continue
		}

		arr := strings.SplitN(s, "=", 2)
		if len(arr) != 2 {
			return nil, fmt.Errorf("missing bool value for feature gate key = %s", s)
		}

		k := strings.TrimSpace(arr[0])
		v := strings.TrimSpace(arr[1])

		boolValue, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("invalid value %v for feature gate key = %s, use true|false instead", v, k)
		}
		fGate[k] = boolValue
	}

	return fGate, nil
}

// Enabled returns true if the feature gate value of a particular feature is true.
func (f FeatureGate) Enabled(feature string) bool {
	if value, ok := f[feature]; ok {
		return value
	}

	return false
}
