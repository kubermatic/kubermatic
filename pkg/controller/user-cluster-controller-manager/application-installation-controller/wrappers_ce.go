//go:build !ee

/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package applicationinstallationcontroller

import (
	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
)

const (
	ClusterAutoscalerDefaultRepository = "registry.k8s.io/autoscaling/cluster-autoscaler"
)

// Function signature for generating Helm values block
type ValuesGenerator func(app *appskubermaticv1.ApplicationInstallation, overwriteRegistry string) map[string]interface{}

// Map of functions to generate Helm values for different applications
var ValuesGenerators = map[string]ValuesGenerator{
	"cluster-autoscaler": generateClusterAutoscalerValues,
}

func generateClusterAutoscalerValues(app *appskubermaticv1.ApplicationInstallation, overwriteRegistry string) map[string]interface{} {
	// Build the final values structure
	values := map[string]any{
		"image": map[string]any{
			"repository": registry.Must(registry.RewriteImage(ClusterAutoscalerDefaultRepository, overwriteRegistry)),
		},
	}

	return values
}
