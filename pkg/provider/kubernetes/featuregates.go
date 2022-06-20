/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package kubernetes

import (
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/provider"
)

type featureGatesProvider features.FeatureGate

// NewFeatureGatesProvider returns a new provider for feature gates.
func NewFeatureGatesProvider(featureGates features.FeatureGate) provider.FeatureGatesProvider {
	return featureGatesProvider(featureGates)
}

// GetFeatureGates returns feature gates.
func (fg featureGatesProvider) GetFeatureGates() (apiv2.FeatureGates, error) {
	var f apiv2.FeatureGates

	if v, ok := fg[features.KonnectivityService]; ok {
		f.KonnectivityService = &v
	}
	if v, ok := fg[features.OIDCKubeCfgEndpoint]; ok {
		f.OIDCKubeCfgEndpoint = &v
	}
	if v, ok := fg[features.OperatingSystemManager]; ok {
		f.OperatingSystemManager = &v
	}

	return f, nil
}
