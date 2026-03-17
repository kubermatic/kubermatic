/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

// Package gateway provides utilities for working with Gateway API resources.
package gateway

import (
	"slices"
	"strings"

	gatewayapiv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	// CoreListenerHTTP is the name of the HTTP listener managed by kubermatic-operator.
	CoreListenerHTTP gatewayapiv1.SectionName = "http"
	// CoreListenerHTTPS is the name of the HTTPS listener managed by kubermatic-operator.
	CoreListenerHTTPS gatewayapiv1.SectionName = "https"
)

// CoreListenerNames defines the listener names managed by kubermatic-operator.
// These are always replaced with the operator's desired state.
var CoreListenerNames = map[gatewayapiv1.SectionName]struct{}{
	CoreListenerHTTP:  {},
	CoreListenerHTTPS: {},
}

// MergeListeners combines core listeners with preserved non-core listeners from existing.
// Core listeners (http, https) come from the operator's desired state.
// Non-core listeners (hostname-based) are preserved from existing state.
// Returns sorted slice for deterministic comparison.
func MergeListeners(core, existing []gatewayapiv1.Listener) []gatewayapiv1.Listener {
	result := make([]gatewayapiv1.Listener, 0, len(core)+len(existing))
	result = append(result, core...)

	for _, l := range existing {
		if _, isCoreListener := CoreListenerNames[l.Name]; !isCoreListener {
			result = append(result, l)
		}
	}

	SortListenersByName(result)
	return result
}

// SortListenersByName sorts listeners alphabetically by name for deterministic ordering.
func SortListenersByName(listeners []gatewayapiv1.Listener) {
	slices.SortFunc(listeners, func(a, b gatewayapiv1.Listener) int {
		return strings.Compare(string(a.Name), string(b.Name))
	})
}
