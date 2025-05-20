/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package cilium

import (
	"fmt"
	"maps"
	"strings"

	appskubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/registry"

	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/yaml"
)

const (
	ciliumHelmChartName = "cilium"
	ciliumImageRegistry = "quay.io/cilium/"
)

// ApplicationDefinitionReconciler creates Cilium ApplicationDefinition managed by KKP to be used
// for installing Cilium CNI into KKP user clusters.
func ApplicationDefinitionReconciler(config *kubermaticv1.KubermaticConfiguration) reconciling.NamedApplicationDefinitionReconcilerFactory {
	return func() (string, reconciling.ApplicationDefinitionReconciler) {
		return kubermaticv1.CNIPluginTypeCilium.String(), func(app *appskubermaticv1.ApplicationDefinition) (*appskubermaticv1.ApplicationDefinition, error) {
			app.Labels = map[string]string{
				appskubermaticv1.ApplicationManagedByLabel: appskubermaticv1.ApplicationManagedByKKPValue,
				appskubermaticv1.ApplicationTypeLabel:      appskubermaticv1.ApplicationTypeCNIValue,
			}

			app.Spec.Description = "Cilium CNI - eBPF-based Networking, Security, and Observability"
			app.Spec.Method = appskubermaticv1.HelmTemplateMethod

			var credentials *appskubermaticv1.HelmCredentials
			if config.Spec.UserCluster.SystemApplications.HelmRegistryConfigFile != nil {
				credentials = &appskubermaticv1.HelmCredentials{
					RegistryConfigFile: config.Spec.UserCluster.SystemApplications.HelmRegistryConfigFile,
				}
			}

			app.Spec.Versions = []appskubermaticv1.ApplicationVersion{
				// NOTE: When introducing a new version, make sure it is:
				//  - introduced in pkg/cni/version.go with the version string exactly matching the Spec.Versions.Version here
				//  - Helm chart is mirrored in Kubermatic OCI registry, use the script cilium-mirror-chart.sh
				{
					Version: "1.13.0",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName:    ciliumHelmChartName,
								ChartVersion: "1.13.0",
								URL:          defaultHelmRepository(&config.Spec.UserCluster.SystemApplications),
								Credentials:  credentials,
							},
						},
					},
				},
				{
					Version: "1.13.3",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName:    ciliumHelmChartName,
								ChartVersion: "1.13.3",
								URL:          defaultHelmRepository(&config.Spec.UserCluster.SystemApplications),
								Credentials:  credentials,
							},
						},
					},
				},
				{
					Version: "1.13.4",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName:    ciliumHelmChartName,
								ChartVersion: "1.13.4",
								URL:          defaultHelmRepository(&config.Spec.UserCluster.SystemApplications),
								Credentials:  credentials,
							},
						},
					},
				},
				{
					Version: "1.13.6",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName:    ciliumHelmChartName,
								ChartVersion: "1.13.6",
								URL:          defaultHelmRepository(&config.Spec.UserCluster.SystemApplications),
								Credentials:  credentials,
							},
						},
					},
				},
				{
					Version: "1.13.7",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName:    ciliumHelmChartName,
								ChartVersion: "1.13.7",
								URL:          defaultHelmRepository(&config.Spec.UserCluster.SystemApplications),
								Credentials:  credentials,
							},
						},
					},
				},
				{
					Version: "1.13.8",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName:    ciliumHelmChartName,
								ChartVersion: "1.13.8",
								URL:          defaultHelmRepository(&config.Spec.UserCluster.SystemApplications),
								Credentials:  credentials,
							},
						},
					},
				},
				{
					Version: "1.13.14",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName:    ciliumHelmChartName,
								ChartVersion: "1.13.14",
								URL:          defaultHelmRepository(&config.Spec.UserCluster.SystemApplications),
								Credentials:  credentials,
							},
						},
					},
				},
				{
					Version: "1.14.1",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName:    ciliumHelmChartName,
								ChartVersion: "1.14.1",
								URL:          defaultHelmRepository(&config.Spec.UserCluster.SystemApplications),
								Credentials:  credentials,
							},
						},
					},
				},
				{
					Version: "1.14.2",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName:    ciliumHelmChartName,
								ChartVersion: "1.14.2",
								URL:          defaultHelmRepository(&config.Spec.UserCluster.SystemApplications),
								Credentials:  credentials,
							},
						},
					},
				},
				{
					Version: "1.14.3",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName:    ciliumHelmChartName,
								ChartVersion: "1.14.3",
								URL:          defaultHelmRepository(&config.Spec.UserCluster.SystemApplications),
								Credentials:  credentials,
							},
						},
					},
				},
				{
					Version: "1.14.9",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName:    ciliumHelmChartName,
								ChartVersion: "1.14.9",
								URL:          defaultHelmRepository(&config.Spec.UserCluster.SystemApplications),
								Credentials:  credentials,
							},
						},
					},
				},
				{
					Version: "1.14.16",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName:    ciliumHelmChartName,
								ChartVersion: "1.14.16",
								URL:          defaultHelmRepository(&config.Spec.UserCluster.SystemApplications),
								Credentials:  credentials,
							},
						},
					},
				},
				{
					Version: "1.15.3",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName:    ciliumHelmChartName,
								ChartVersion: "1.15.3",
								URL:          defaultHelmRepository(&config.Spec.UserCluster.SystemApplications),
								Credentials:  credentials,
							},
						},
					},
				},
				{
					Version: "1.15.10",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName:    ciliumHelmChartName,
								ChartVersion: "1.15.10",
								URL:          defaultHelmRepository(&config.Spec.UserCluster.SystemApplications),
								Credentials:  credentials,
							},
						},
					},
				},
				{
					Version: "1.15.16",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName:    ciliumHelmChartName,
								ChartVersion: "1.15.16",
								URL:          defaultHelmRepository(&config.Spec.UserCluster.SystemApplications),
								Credentials:  credentials,
							},
						},
					},
				},
				{
					Version: "1.16.6",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName:    ciliumHelmChartName,
								ChartVersion: "1.16.6",
								URL:          defaultHelmRepository(&config.Spec.UserCluster.SystemApplications),
								Credentials:  credentials,
							},
						},
					},
				},
				{
					Version: "1.16.9",
					Template: appskubermaticv1.ApplicationTemplate{
						Source: appskubermaticv1.ApplicationSource{
							Helm: &appskubermaticv1.HelmSource{
								ChartName:    ciliumHelmChartName,
								ChartVersion: "1.16.9",
								URL:          defaultHelmRepository(&config.Spec.UserCluster.SystemApplications),
								Credentials:  credentials,
							},
						},
					},
				},
			}

			// to make it compatible with older cilium controller versions, convert any DefaultValues into DefaultValuesBlock
			if err := convertDefaultValuesToDefaultValuesBlock(app); err != nil {
				return app, fmt.Errorf("failed to convert DefaultValues into DefaultValuesBlock: %w", err)
			}

			// we want to allow overriding the default values, so reconcile them only if nil
			if app.Spec.DefaultValuesBlock == "" || app.Spec.DefaultValuesBlock == "{}" {
				if err := setDefaultValues(app); err != nil {
					return app, err
				}
				return app, nil
			}

			defaultValues, err := app.Spec.GetParsedDefaultValues()
			if err != nil {
				return app, fmt.Errorf("failed to unmarshal CNI values: %w", err)
			}

			if defaultValues != nil {
				sanitizeValues(defaultValues)
				rawValues, err := yaml.Marshal(defaultValues)
				if err != nil {
					return app, fmt.Errorf("failed to marshal CNI values: %w", err)
				}

				app.Spec.DefaultValuesBlock = string(rawValues)
			}

			return app, nil
		}
	}
}

// setDefaultValues sets the default values for the application.
func setDefaultValues(app *appskubermaticv1.ApplicationDefinition) error {
	defaultValues := map[string]any{
		"operator": map[string]any{
			"replicas": 1,
		},
		"envoy": map[string]any{
			"enabled": false,
		},
		"hubble": map[string]any{
			"relay": map[string]any{
				"enabled": true,
			},
			"ui": map[string]any{
				"enabled": true,
			},
			"tls": map[string]any{
				"auto": map[string]any{
					"method": "cronJob",
				},
			},
		},
	}
	rawValues, err := yaml.Marshal(defaultValues)
	if err != nil {
		return fmt.Errorf("failed to marshal default CNI values: %w", err)
	}
	app.Spec.DefaultValuesBlock = string(rawValues)
	return nil
}

func sanitizeValues(values map[string]any) {
	// If not specified, set envoy.enabled to false
	// https://github.com/cilium/cilium/commit/471f19a16593e1e9342c31bf3e26e5383737cb0a
	if envoy, ok := values["envoy"].(map[string]any); ok {
		if _, ok := envoy["enabled"]; !ok {
			envoy["enabled"] = false
		}
	} else {
		values["envoy"] = map[string]any{
			"enabled": false,
		}
	}
}

// GetAppInstallOverrideValues returns Helm values to be enforced on the cluster's ApplicationInstallation
// of the Cilium CNI managed by KKP.
func GetAppInstallOverrideValues(cluster *kubermaticv1.Cluster, overwriteRegistry string) map[string]any {
	podSecurityContext := map[string]any{
		"seccompProfile": map[string]any{
			"type": "RuntimeDefault",
		},
	}
	defaultValues := map[string]any{
		"podSecurityContext": podSecurityContext,
	}
	values := map[string]any{
		"podSecurityContext": podSecurityContext,
	}
	valuesEnvoy := map[string]any{
		"podSecurityContext": podSecurityContext,
	}

	valuesOperator := map[string]any{
		"securityContext": map[string]any{
			"seccompProfile": map[string]any{
				"type": "RuntimeDefault",
			},
		},
		"podSecurityContext": podSecurityContext,
	}
	valuesCni := map[string]any{
		// we run Cilium as non-exclusive CNI to allow for Multus use-cases
		"exclusive": false,
	}

	valuesCertGen := maps.Clone(defaultValues)
	valuesRelay := maps.Clone(defaultValues)
	valuesFrontend := map[string]any{}
	valuesBackend := map[string]any{}

	if cluster.Spec.ClusterNetwork.ProxyMode == resources.EBPFProxyMode {
		values["kubeProxyReplacement"] = "true"
		values["k8sServiceHost"] = cluster.Status.Address.ExternalName
		values["k8sServicePort"] = cluster.Status.Address.Port

		nodePortRange := cluster.Spec.ComponentsOverride.Apiserver.NodePortRange
		if nodePortRange != "" && nodePortRange != resources.DefaultNodePortRange {
			values["nodePort"] = map[string]any{
				"range": strings.ReplaceAll(nodePortRange, "-", ","),
			}
		}
	} else {
		values["kubeProxyReplacement"] = "false"
		values["sessionAffinity"] = true
		valuesCni["chainingMode"] = "portmap"
	}

	ipamOperator := map[string]any{
		"clusterPoolIPv4PodCIDRList": cluster.Spec.ClusterNetwork.Pods.GetIPv4CIDRs(),
	}

	if cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv4 != nil {
		ipamOperator["clusterPoolIPv4MaskSize"] = *cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv4
	}

	if cluster.IsDualStack() {
		values["ipv6"] = map[string]any{"enabled": true}
		ipamOperator["clusterPoolIPv6PodCIDRList"] = cluster.Spec.ClusterNetwork.Pods.GetIPv6CIDRs()
		if cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv6 != nil {
			ipamOperator["clusterPoolIPv6MaskSize"] = *cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv6
		}
	}

	values["ipam"] = map[string]any{"operator": ipamOperator}

	// Override images if registry override is configured
	if overwriteRegistry != "" {
		values["image"] = map[string]any{
			"repository": registry.Must(registry.RewriteImage(ciliumImageRegistry+"cilium", overwriteRegistry)),
			"useDigest":  false,
		}
		valuesOperator["image"] = map[string]any{
			"repository": registry.Must(registry.RewriteImage(ciliumImageRegistry+"operator", overwriteRegistry)),
			"useDigest":  false,
		}
		valuesRelay["image"] = map[string]any{
			"repository": registry.Must(registry.RewriteImage(ciliumImageRegistry+"hubble-relay", overwriteRegistry)),
			"useDigest":  false,
		}
		valuesBackend["image"] = map[string]any{
			"repository": registry.Must(registry.RewriteImage(ciliumImageRegistry+"hubble-ui-backend", overwriteRegistry)),
			"useDigest":  false,
		}
		valuesFrontend["image"] = map[string]any{
			"repository": registry.Must(registry.RewriteImage(ciliumImageRegistry+"hubble-ui", overwriteRegistry)),
			"useDigest":  false,
		}
		valuesCertGen["image"] = map[string]any{
			"repository": registry.Must(registry.RewriteImage(ciliumImageRegistry+"certgen", overwriteRegistry)),
			"useDigest":  false,
		}
		valuesEnvoy["image"] = map[string]any{
			"repository": registry.Must(registry.RewriteImage(ciliumImageRegistry+"cilium-envoy", overwriteRegistry)),
			"useDigest":  false,
		}
	}

	uiSecContext := maps.Clone(podSecurityContext)
	uiSecContext["enabled"] = true

	values["cni"] = valuesCni
	values["envoy"] = valuesEnvoy
	values["operator"] = valuesOperator
	values["certgen"] = valuesCertGen
	values["hubble"] = map[string]any{
		"relay": valuesRelay,
		"ui": map[string]any{
			"securityContext": uiSecContext,
			"frontend":        valuesFrontend,
			"backend":         valuesBackend,
		},
	}

	return values
}

// ValidateValuesUpdate validates the update operation on provided Cilium Helm values.
func ValidateValuesUpdate(newValues, oldValues map[string]any, fieldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	// Validate immutability of specific top-level value subtrees, managed solely by KKP
	exclusions := []exclusion{
		{
			fullPath:  "cni.chainingMode",
			pathParts: []string{"cni", "chainingMode"},
		},
		{
			fullPath:  "ipam.operator.clusterPoolIPv4MaskSize",
			pathParts: []string{"ipam", "operator", "clusterPoolIPv4MaskSize"},
		},
		{
			fullPath:  "ipam.operator.clusterPoolIPv6MaskSize",
			pathParts: []string{"ipam", "operator", "clusterPoolIPv6MaskSize"},
		},
	}
	allErrs = append(allErrs, validateImmutableValues(newValues, oldValues, fieldPath, []string{
		"cni",
		"ipam",
		"ipv6",
	}, exclusions)...)

	// Validate that mandatory top-level values are present
	allErrs = append(allErrs, validateMandatoryValues(newValues, fieldPath, []string{
		"kubeProxyReplacement", // can be changed if switching KKP proxy mode, but must be present
	})...)

	// Validate config for strict kubeProxyReplacement (ebpf "proxy mode")
	if newValues["kubeProxyReplacement"] == "strict" {
		// Validate mandatory values for kube-proxy-replacement
		allErrs = append(allErrs, validateMandatoryValues(newValues, fieldPath, []string{
			"k8sServiceHost",
			"k8sServicePort",
		})...)
	}

	// Validate immutability of operator.securityContext
	operPath := fieldPath.Child("operator")
	newOperator, ok := newValues["operator"].(map[string]any)
	if !ok {
		allErrs = append(allErrs, field.Invalid(operPath, newValues["operator"], "value missing or incorrect"))
	}
	oldOperator, ok := oldValues["operator"].(map[string]any)
	if !ok {
		allErrs = append(allErrs, field.Invalid(operPath, oldValues["operator"], "value missing or incorrect"))
	}
	allErrs = append(allErrs, validateImmutableValues(newOperator, oldOperator, operPath, []string{
		"securityContext",
	}, []exclusion{})...)

	return allErrs
}

type exclusion struct {
	fullPath  string
	pathParts []string
}

func validateImmutableValues(newValues, oldValues map[string]any, fieldPath *field.Path, immutableValues []string, exclusions []exclusion) field.ErrorList {
	allErrs := field.ErrorList{}
	allowedValues := map[string]bool{}

	for _, v := range immutableValues {
		// Check if the value in `oldValues` is non-empty and missing in `newValues`
		if _, existsInOld := oldValues[v]; existsInOld {
			if _, existsInNew := newValues[v]; !existsInNew || newValues[v] == nil || fmt.Sprint(newValues[v]) == "" {
				allErrs = append(allErrs, field.Invalid(fieldPath.Child(v), newValues[v], "value is immutable"))
				continue
			}
		}
		for _, exclusion := range exclusions {
			if strings.HasPrefix(exclusion.fullPath, v) {
				if excludedKeyExists(newValues, exclusion.pathParts...) {
					allowedValues[v] = true
				}
				if excludedKeyExists(oldValues, exclusion.pathParts...) {
					allowedValues[v] = true
				}
			}
		}
		if fmt.Sprint(oldValues[v]) != fmt.Sprint(newValues[v]) && !allowedValues[v] {
			allErrs = append(allErrs, field.Invalid(fieldPath.Child(v), newValues[v], "value is immutable"))
		}
	}
	return allErrs
}

func validateMandatoryValues(newValues map[string]any, fieldPath *field.Path, mandatoryValues []string) field.ErrorList {
	allErrs := field.ErrorList{}
	for _, v := range mandatoryValues {
		if _, ok := newValues[v]; !ok {
			allErrs = append(allErrs, field.NotFound(fieldPath.Child(v), newValues[v]))
		}
	}
	return allErrs
}

func excludedKeyExists(values map[string]any, keys ...string) bool {
	if len(keys) == 0 || values == nil {
		return false
	}
	root := keys[0]
	value, ok := values[root]
	if !ok {
		return false
	}

	if len(keys) == 1 {
		return true
	}

	if innerMap, ok := value.(map[string]any); ok {
		return excludedKeyExists(innerMap, keys[1:]...)
	}

	return false
}

func convertDefaultValuesToDefaultValuesBlock(app *appskubermaticv1.ApplicationDefinition) error {
	if app.Spec.DefaultValues != nil {
		oldDefVals, err := yaml.JSONToYAML(app.Spec.DefaultValues.Raw)
		if err != nil {
			return err
		}
		app.Spec.DefaultValuesBlock = string(oldDefVals)
		app.Spec.DefaultValues = nil
	}
	return nil
}

// defaultHelmRepository returns the default Helm repository for the given system applications configuration.
// if the configuration contains a HelmRepository, it will be used as the helm repository.
// otherwise the default system applications helm repository will be used.
func defaultHelmRepository(conf *kubermaticv1.SystemApplicationsConfiguration) string {
	if conf == nil {
		return ""
	}

	if conf.HelmRepository != "" {
		return registry.ToOCIURL(conf.HelmRepository)
	}

	return defaulting.DefaultSystemApplicationsHelmRepository
}
