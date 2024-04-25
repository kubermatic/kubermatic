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
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/exp/maps"

	appskubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/apps.kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/resources/registry"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

const (
	ciliumHelmChartName = "cilium"
	ciliumImageRegistry = "quay.io/cilium/"
	ociPrefix           = "oci://"
)

func toOciUrl(s string) string {
	if strings.HasPrefix(s, ociPrefix) {
		return s
	}
	return ociPrefix + s
}

// ApplicationDefinitionReconciler creates Cilium ApplicationDefinition managed by KKP to be used
// for installing Cilium CNI into KKP usr clusters.
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
								URL:          toOciUrl(config.Spec.UserCluster.SystemApplications.HelmRepository),
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
								URL:          toOciUrl(config.Spec.UserCluster.SystemApplications.HelmRepository),
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
								URL:          toOciUrl(config.Spec.UserCluster.SystemApplications.HelmRepository),
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
								URL:          toOciUrl(config.Spec.UserCluster.SystemApplications.HelmRepository),
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
								URL:          toOciUrl(config.Spec.UserCluster.SystemApplications.HelmRepository),
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
								URL:          toOciUrl(config.Spec.UserCluster.SystemApplications.HelmRepository),
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
								URL:          toOciUrl(config.Spec.UserCluster.SystemApplications.HelmRepository),
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
								URL:          toOciUrl(config.Spec.UserCluster.SystemApplications.HelmRepository),
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
								URL:          toOciUrl(config.Spec.UserCluster.SystemApplications.HelmRepository),
								Credentials:  credentials,
							},
						},
					},
				},
			}

			// we want to allow overriding the default values, so reconcile them only if nil
			if app.Spec.DefaultValues == nil {
				defaultValues := map[string]any{
					"operator": map[string]any{
						"replicas": 1,
					},
					"hubble": map[string]any{
						"relay": map[string]any{
							"enabled": true,
						},
						"ui": map[string]any{
							"enabled": true,
						},
						// cronJob TLS cert gen method needs to be used for backward compatibility with older KKP
						"tls": map[string]any{
							"auto": map[string]any{
								"method": "cronJob",
							},
						},
					},
				}
				rawValues, err := json.Marshal(defaultValues)
				if err != nil {
					return app, fmt.Errorf("failed to marshall default CNI values: %w", err)
				}
				app.Spec.DefaultValues = &runtime.RawExtension{Raw: rawValues}
			}
			return app, nil
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
		values["kubeProxyReplacement"] = "strict"
		values["k8sServiceHost"] = cluster.Status.Address.ExternalName
		values["k8sServicePort"] = cluster.Status.Address.Port

		nodePortRange := cluster.Spec.ComponentsOverride.Apiserver.NodePortRange
		if nodePortRange != "" && nodePortRange != resources.DefaultNodePortRange {
			values["nodePort"] = map[string]any{
				"range": strings.ReplaceAll(nodePortRange, "-", ","),
			}
		}
	} else {
		values["kubeProxyReplacement"] = "disabled"
		values["sessionAffinity"] = true
		valuesCni["chainingMode"] = "portmap"
	}

	ipamOperator := map[string]any{
		"clusterPoolIPv4PodCIDRList": cluster.Spec.ClusterNetwork.Pods.GetIPv4CIDRs(),
		"clusterPoolIPv4MaskSize":    fmt.Sprintf("%d", *cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv4),
	}
	if cluster.IsDualStack() {
		values["ipv6"] = map[string]any{"enabled": true}
		ipamOperator["clusterPoolIPv6PodCIDRList"] = cluster.Spec.ClusterNetwork.Pods.GetIPv6CIDRs()
		ipamOperator["clusterPoolIPv6MaskSize"] = fmt.Sprintf("%d", *cluster.Spec.ClusterNetwork.NodeCIDRMaskSizeIPv6)
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
	}

	values["cni"] = valuesCni
	values["operator"] = valuesOperator
	values["certgen"] = valuesCertGen
	values["hubble"] = map[string]any{
		"relay": valuesRelay,
		"ui": map[string]any{
			"securityContext": podSecurityContext,
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
	allErrs = append(allErrs, validateImmutableValues(newValues, oldValues, fieldPath, []string{
		"cni",
		"ipam",
		"ipv6",
	}, []string{
		"cni.chainingMode",
		"ipam.operator.clusterPoolIPv4PodCIDR",
	})...)

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
	}, []string{})...)

	return allErrs
}

func validateImmutableValues(newValues, oldValues map[string]any, fieldPath *field.Path, immutableValues []string, excludedKeys []string) field.ErrorList {
	allErrs := field.ErrorList{}
	allowedValues := map[string]bool{}

	for _, v := range immutableValues {
		for _, exclusion := range excludedKeys {
			if strings.HasPrefix(exclusion, v) {
				if excludedKeyExists(newValues, strings.Split(exclusion, ".")...) {
					allowedValues[v] = true
				}
				if excludedKeyExists(oldValues, strings.Split(exclusion, ".")...) {
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
