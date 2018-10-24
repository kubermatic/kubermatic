package main

import (
	"fmt"
	"strconv"
	"strings"
)

const (
	// PrometheusEndpoint if enabled exposes cluster's metrics HTTP endpoint
	PrometheusEndpoint = "PrometheusEndpoint"
)

type featureGate map[string]bool

func newFeatures(rawFeatures string) (featureGate, error) {
	fGate := featureGate{}
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

func (f featureGate) enabled(feature string) bool {
	if enabled, ok := f[feature]; ok {
		return enabled
	}

	return false
}
