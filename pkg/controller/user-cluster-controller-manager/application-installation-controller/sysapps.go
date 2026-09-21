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
	"strings"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
)

const (
	ClusterAutoscalerDefaultRepository = "registry.k8s.io/autoscaling/cluster-autoscaler"
	AgentGatewayControllerImage        = "cr.agentgateway.dev/controller"
	AgentGatewayProxyImage             = "cr.agentgateway.dev/agentgateway"
)

// Function signature for generating Helm values block.
type ValuesGenerator func(app *appskubermaticv1.ApplicationInstallation, overwriteRegistry string) map[string]interface{}

// Map of functions to generate Helm values for system applications.
var SystemAppsValuesGenerators = map[string]ValuesGenerator{
	"agentgateway":       generateAgentGatewayValues,
	"cluster-autoscaler": generateClusterAutoscalerValues,
}

func generateClusterAutoscalerValues(app *appskubermaticv1.ApplicationInstallation, overwriteRegistry string) map[string]interface{} {
	values := map[string]any{
		"image": map[string]any{
			"repository": registry.Must(registry.RewriteImage(ClusterAutoscalerDefaultRepository, overwriteRegistry)),
		},
	}

	return values
}

func generateAgentGatewayValues(app *appskubermaticv1.ApplicationInstallation, overwriteRegistry string) map[string]interface{} {
	controllerRegistry, controllerRepository := splitRegistryRepository(registry.Must(registry.RewriteImage(AgentGatewayControllerImage, overwriteRegistry)))
	proxyRegistry, proxyRepository := splitRegistryRepository(registry.Must(registry.RewriteImage(AgentGatewayProxyImage, overwriteRegistry)))

	values := map[string]any{
		"controller": map[string]any{
			"image": map[string]any{
				"registry":   controllerRegistry,
				"repository": controllerRepository,
			},
		},
		"proxy": map[string]any{
			"image": map[string]any{
				"registry":   proxyRegistry,
				"repository": proxyRepository,
			},
		},
	}

	return values
}

func splitRegistryRepository(image string) (string, string) {
	imageRegistry, imageRepository, ok := strings.Cut(image, "/")
	if !ok {
		return "", image
	}
	return imageRegistry, imageRepository
}
