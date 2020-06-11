package kubermatic

import (
	"fmt"
	"strings"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/pkg/crd/operator/v1alpha1"
)

func featureGates(cfg *operatorv1alpha1.KubermaticConfiguration) string {
	features := make([]string, 0)
	for _, feature := range cfg.Spec.FeatureGates.List() {
		features = append(features, fmt.Sprintf("%s=true", feature))
	}

	return strings.Join(features, ",")
}
