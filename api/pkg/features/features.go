package features

import (
	"fmt"
	"strconv"
	"strings"
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
	if enabled, ok := f[feature]; ok {
		return enabled
	}

	return false
}
