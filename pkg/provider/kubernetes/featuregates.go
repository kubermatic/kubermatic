package kubernetes

import "k8c.io/kubermatic/v2/pkg/features"

type featureGatesProvider features.FeatureGate

func NewFeatureGatesProvider(featureGates features.FeatureGate) featureGatesProvider {
    return featureGatesProvider(featureGates)
}

func (fg featureGatesProvider) GetFeatureGates() (features.FeatureGate, error) {
	return features.FeatureGate(fg), nil
}
