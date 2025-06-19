//go:build ee

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
	CertManagerRegistryPrefix = "quay.io/jetstack/cert-manager-"
	Flux2RegistryPrefix       = "ghcr.io/fluxcd/"
	MetalLBRegistryPrefix     = "quay.io/metallb/"
)

// Function signature for generating Helm values block
type ValuesGenerator func(app *appskubermaticv1.ApplicationInstallation, overwriteRegistry string) map[string]interface{}

// Map of functions to generate Helm values for different applications
var ValuesGenerators = map[string]ValuesGenerator{
	"argocd":              generateArgoCDValues,
	"cert-manager":        generateCertManagerValues,
	"falco":               generateFalcoValues,
	"flux2":               generateFluxValues,
	"k8sgpt-operator":     generateK8sGPTValues,
	"kube-vip":            generateKubeVIPValues,
	"metallb":             generateMetalLBValues,
	"nginx":               generateIngressNGINXValues,
	"nvidia-gpu-operator": generateNVIDIAGPUOperatorValues,
	"trivy":               generateTrivyAppValues,
	"trivy-operator":      generateTrivyOperatorValues,
	"aikit":               generateAiKitValues,
}

func generateArgoCDValues(app *appskubermaticv1.ApplicationInstallation, overwriteRegistry string) map[string]interface{} {
	globalValues := map[string]any{
		"image": map[string]any{
			"repository": registry.Must(registry.RewriteImage("quay.io/argoproj/argocd", overwriteRegistry)),
		},
	}
	dexValues := map[string]any{
		"image": map[string]any{
			"repository": registry.Must(registry.RewriteImage("ghcr.io/dexidp/dex", overwriteRegistry)),
		},
	}
	redisValues := map[string]any{
		"image": map[string]any{
			"repository": registry.Must(registry.RewriteImage("ecr-public.aws.com/docker/library/redis", overwriteRegistry)),
		},
		"exporter": map[string]any{
			"image": map[string]any{
				"repository": registry.Must(registry.RewriteImage("ghcr.io/oliver006/redis_exporter", overwriteRegistry)),
			},
		},
	}
	serverValues := map[string]any{
		"extensions": map[string]any{
			"image": map[string]any{
				"repository": registry.Must(registry.RewriteImage("quay.io/argoprojlabs/argocd-extension-installer", overwriteRegistry)),
			},
		},
	}
	values := map[string]any{
		"global": globalValues,
		"dex":    dexValues,
		"redis":  redisValues,
		"server": serverValues,
	}
	return values
}

func generateCertManagerValues(app *appskubermaticv1.ApplicationInstallation, overwriteRegistry string) map[string]interface{} {
	imageValues := map[string]any{
		"repository": registry.Must(registry.RewriteImage(CertManagerRegistryPrefix+"controller", overwriteRegistry)),
	}
	webhookValues := map[string]any{
		"image": map[string]any{
			"repository": registry.Must(registry.RewriteImage(CertManagerRegistryPrefix+"webhook", overwriteRegistry)),
		},
	}
	cainjectorValues := map[string]any{
		"image": map[string]any{
			"repository": registry.Must(registry.RewriteImage(CertManagerRegistryPrefix+"cainjector", overwriteRegistry)),
		},
	}
	acmesolverValues := map[string]any{
		"image": map[string]any{
			"repository": registry.Must(registry.RewriteImage(CertManagerRegistryPrefix+"acmesolver", overwriteRegistry)),
		},
	}
	startupapicheckValues := map[string]any{
		"image": map[string]any{
			"repository": registry.Must(registry.RewriteImage(CertManagerRegistryPrefix+"startupapicheck", overwriteRegistry)),
		},
	}
	values := map[string]any{
		"image":           imageValues,
		"webhook":         webhookValues,
		"cainjector":      cainjectorValues,
		"acmesolver":      acmesolverValues,
		"startupapicheck": startupapicheckValues,
	}
	return values
}

func generateFalcoValues(app *appskubermaticv1.ApplicationInstallation, overwriteRegistry string) map[string]interface{} {
	imageValues := map[string]any{
		"registry": overwriteRegistry,
	}
	registryValues := map[string]any{
		"image": map[string]any{
			"registry": overwriteRegistry,
		},
	}
	driverValues := map[string]any{
		"loader": map[string]any{
			"initContainer": registryValues,
		},
	}
	cliValues := registryValues

	values := map[string]any{
		"image":    imageValues,
		"driver":   driverValues,
		"falcoctl": cliValues,
	}
	return values
}

func generateFluxValues(app *appskubermaticv1.ApplicationInstallation, overwriteRegistry string) map[string]interface{} {
	cliValues := map[string]any{
		"image": registry.Must(registry.RewriteImage(Flux2RegistryPrefix+"flux-cli", overwriteRegistry)),
	}
	helmCtrlValues := map[string]any{
		"image": registry.Must(registry.RewriteImage(Flux2RegistryPrefix+"helm-controller", overwriteRegistry)),
	}
	imgAutomationValues := map[string]any{
		"image": registry.Must(registry.RewriteImage(Flux2RegistryPrefix+"image-automation-controller", overwriteRegistry)),
	}
	imgReflectionValues := map[string]any{
		"image": registry.Must(registry.RewriteImage(Flux2RegistryPrefix+"image-reflector-controller", overwriteRegistry)),
	}
	kustomizeValues := map[string]any{
		"image": registry.Must(registry.RewriteImage(Flux2RegistryPrefix+"kustomize-controller", overwriteRegistry)),
	}
	notificationValues := map[string]any{
		"image": registry.Must(registry.RewriteImage(Flux2RegistryPrefix+"notification-controller", overwriteRegistry)),
	}
	sourceValues := map[string]any{
		"image": registry.Must(registry.RewriteImage(Flux2RegistryPrefix+"source-controller", overwriteRegistry)),
	}
	values := map[string]any{
		"cli":                       cliValues,
		"helmController":            helmCtrlValues,
		"imageAutomationController": imgAutomationValues,
		"imageReflectionController": imgReflectionValues,
		"kustomizeController":       kustomizeValues,
		"notificationController":    notificationValues,
		"sourceController":          sourceValues,
	}
	return values
}

// generateK8sGPTValues generates Helm values for k8sgpt
func generateK8sGPTValues(app *appskubermaticv1.ApplicationInstallation, overwriteRegistry string) map[string]interface{} {
	// Define image repositories for each component
	kubeRbacProxyValues := map[string]interface{}{
		"image": map[string]interface{}{
			"repository": registry.Must(registry.RewriteImage("quay.io/brancz/kube-rbac-proxy", overwriteRegistry)),
		},
	}
	managerValues := map[string]interface{}{
		"image": map[string]interface{}{
			"repository": registry.Must(registry.RewriteImage("ghcr.io/k8sgpt-ai/k8sgpt-operator", overwriteRegistry)),
		},
	}

	// Build the final values structure
	values := map[string]interface{}{
		"controllerManager": map[string]interface{}{
			"kubeRbacProxy": kubeRbacProxyValues,
			"manager":       managerValues,
		},
	}

	return values
}

// generateKubeVIPValues generates Helm values for kube-vip
func generateKubeVIPValues(app *appskubermaticv1.ApplicationInstallation, overwriteRegistry string) map[string]interface{} {
	// Define image repository for kube-vip
	imageValues := map[string]interface{}{
		"repository": registry.Must(registry.RewriteImage("ghcr.io/kube-vip/kube-vip", overwriteRegistry)),
	}

	// Build the final values structure
	values := map[string]interface{}{
		"image": imageValues,
	}

	return values
}

func generateMetalLBValues(app *appskubermaticv1.ApplicationInstallation, overwriteRegistry string) map[string]interface{} {
	// Define image repositories for each component
	controllerValues := map[string]interface{}{
		"image": map[string]interface{}{
			"repository": registry.Must(registry.RewriteImage(MetalLBRegistryPrefix+"controller", overwriteRegistry)),
		},
	}
	speakerValues := map[string]interface{}{
		"image": map[string]interface{}{
			"repository": registry.Must(registry.RewriteImage(MetalLBRegistryPrefix+"speaker", overwriteRegistry)),
		},
		"frr": map[string]interface{}{
			"image": map[string]interface{}{
				"repository": registry.Must(registry.RewriteImage("quay.io/frrouting/frr", overwriteRegistry)),
			},
		},
	}

	// Build the final values structure
	values := map[string]interface{}{
		"controller": controllerValues,
		"speaker":    speakerValues,
	}

	return values
}

// generateIngressNGINXValues generates Helm values for ingress-nginx
func generateIngressNGINXValues(app *appskubermaticv1.ApplicationInstallation, overwriteRegistry string) map[string]interface{} {
	// Build the final values structure
	values := map[string]any{
		"global": map[string]any{
			"image": map[string]any{
				"registry": overwriteRegistry,
			},
		},
	}

	return values
}

func generateNVIDIAGPUOperatorValues(app *appskubermaticv1.ApplicationInstallation, overwriteRegistry string) map[string]interface{} {
	// Build the final values structure
	repositoryValues := map[string]any{
		"repository": overwriteRegistry,
	}
	values := map[string]any{
		"nodeStatusExporter": repositoryValues,
		"operator":           repositoryValues,
		"validator":          repositoryValues,
	}

	return values
}

func generateTrivyOperatorValues(app *appskubermaticv1.ApplicationInstallation, overwriteRegistry string) map[string]interface{} {
	// Build the final values structure
	values := map[string]interface{}{
		"image": map[string]interface{}{
			"registry": overwriteRegistry,
		},
		"trivy": map[string]interface{}{
			"image": map[string]interface{}{
				"registry": overwriteRegistry,
			},
		},
	}
	return values
}

func generateTrivyAppValues(app *appskubermaticv1.ApplicationInstallation, overwriteRegistry string) map[string]interface{} {
	// Build the final values structure
	values := map[string]interface{}{
		"image": map[string]interface{}{
			"registry": overwriteRegistry,
		},
	}

	return values
}

func generateAiKitValues(app *appskubermaticv1.ApplicationInstallation, overwriteRegistry string) map[string]interface{} {
	// Build the final values structure
	values := map[string]any{
		"image": map[string]any{
			"repository": registry.Must(registry.RewriteImage("ghcr.io/sozercan/llama3", overwriteRegistry)),
		},
		"postInstall": map[string]any{
			"labelNamespace": map[string]any{
				"image": map[string]any{
					"repository": registry.Must(registry.RewriteImage("registry.k8s.io/kubectl", overwriteRegistry)),
				},
			},
		},
	}

	return values
}
