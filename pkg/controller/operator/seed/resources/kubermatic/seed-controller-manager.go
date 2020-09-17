/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package kubermatic

import (
	"fmt"
	"strings"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func seedControllerManagerPodLabels() map[string]string {
	return map[string]string{
		common.NameLabel: common.SeedControllerManagerDeploymentName,
	}
}

func SeedControllerManagerDeploymentCreator(workerName string, versions common.Versions, cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return common.SeedControllerManagerDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Spec.Replicas = cfg.Spec.SeedController.Replicas
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: seedControllerManagerPodLabels(),
			}

			d.Spec.Template.Labels = d.Spec.Selector.MatchLabels
			d.Spec.Template.Annotations = map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "8085",
				"fluentbit.io/parser":  "json_iso",
			}

			d.Spec.Template.Spec.ServiceAccountName = serviceAccountName

			args := []string{
				"-logtostderr",
				"-internal-address=0.0.0.0:8085",
				"-kubernetes-addons-path=/opt/addons/kubernetes",
				"-openshift-addons-path=/opt/addons/openshift",
				"-worker-count=4",
				fmt.Sprintf("-backup-container=/opt/backup/%s", storeContainerKey),
				fmt.Sprintf("-cleanup-container=/opt/backup/%s", cleanupContainerKey),
				fmt.Sprintf("-docker-pull-config-json-file=/opt/docker/%s", corev1.DockerConfigJsonKey),
				fmt.Sprintf("-seed-admissionwebhook-cert-file=/opt/seed-webhook-serving-cert/%s", resources.ServingCertSecretKey),
				fmt.Sprintf("-seed-admissionwebhook-key-file=/opt/seed-webhook-serving-cert/%s", resources.ServingCertKeySecretKey),
				fmt.Sprintf("-namespace=%s", cfg.Namespace),
				fmt.Sprintf("-external-url=%s", cfg.Spec.Ingress.Domain),
				fmt.Sprintf("-datacenter-name=%s", seed.Name),
				fmt.Sprintf("-etcd-disk-size=%s", cfg.Spec.UserCluster.EtcdVolumeSize),
				fmt.Sprintf("-feature-gates=%s", common.StringifyFeatureGates(cfg)),
				fmt.Sprintf("-nodeport-range=%s", cfg.Spec.UserCluster.NodePortRange),
				fmt.Sprintf("-worker-name=%s", workerName),
				fmt.Sprintf("-kubermatic-image=%s", cfg.Spec.UserCluster.KubermaticDockerRepository),
				fmt.Sprintf("-dnatcontroller-image=%s", cfg.Spec.UserCluster.DNATControllerDockerRepository),
				fmt.Sprintf("-overwrite-registry=%s", cfg.Spec.UserCluster.OverwriteRegistry),
				fmt.Sprintf("-apiserver-default-replicas=%d", *cfg.Spec.UserCluster.APIServerReplicas),
				fmt.Sprintf("-controller-manager-default-replicas=%d", 1),
				fmt.Sprintf("-scheduler-default-replicas=%d", 1),
				fmt.Sprintf("-max-parallel-reconcile=%d", 10),
				fmt.Sprintf("-apiserver-reconciling-disabled-by-default=%v", cfg.Spec.UserCluster.DisableAPIServerEndpointReconciling),
				fmt.Sprintf("-pprof-listen-address=%s", *cfg.Spec.SeedController.PProfEndpoint),
				fmt.Sprintf("-in-cluster-prometheus-disable-default-rules=%v", cfg.Spec.UserCluster.Monitoring.DisableDefaultRules),
				fmt.Sprintf("-in-cluster-prometheus-disable-default-scraping-configs=%v", cfg.Spec.UserCluster.Monitoring.DisableDefaultScrapingConfigs),
			}

			// Only EE does support dynamic-datacenters
			if versions.KubermaticEdition.IsEE() {
				args = append(args, "-dynamic-datacenters=true")
			}

			if cfg.Spec.UserCluster.Monitoring.ScrapeAnnotationPrefix != "" {
				args = append(args, fmt.Sprintf("-monitoring-scrape-annotation-prefix=%s", cfg.Spec.UserCluster.Monitoring.ScrapeAnnotationPrefix))
			}

			if cfg.Spec.SeedController.DebugLog {
				args = append(args, "-v=4", "-log-debug=true")
			} else {
				args = append(args, "-v=2")
			}

			sharedAddonVolume := "addons"
			volumes := []corev1.Volume{
				{
					Name: sharedAddonVolume,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "backup-container",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: backupContainersConfigMapName,
							},
						},
					},
				},
				{
					Name: "seed-webhook-serving-cert",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: common.SeedWebhookServingCertSecretName,
						},
					},
				},
			}

			volumeMounts := []corev1.VolumeMount{
				{
					Name:      sharedAddonVolume,
					MountPath: "/opt/addons/",
					ReadOnly:  true,
				},
				{
					Name:      "backup-container",
					MountPath: "/opt/backup/",
					ReadOnly:  true,
				},
				{
					Name:      "seed-webhook-serving-cert",
					MountPath: "/opt/seed-webhook-serving-cert/",
					ReadOnly:  true,
				},
			}

			if cfg.Spec.ImagePullSecret != "" {
				volumes = append(volumes, corev1.Volume{
					Name: "dockercfg",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: common.DockercfgSecretName,
						},
					},
				})
				volumeMounts = append(volumeMounts, corev1.VolumeMount{
					Name:      "dockercfg",
					MountPath: "/opt/docker/",
					ReadOnly:  true,
				})
			}

			args = append(
				args,
				"-versions=/opt/extra-files/versions.yaml",
				"-updates=/opt/extra-files/updates.yaml",
			)

			if cfg.Spec.UserCluster.Addons.Openshift.DefaultManifests != "" {
				args = append(args, "-openshift-addons-file=/opt/extra-files/"+common.OpenshiftAddonsFileName)
			} else {
				args = append(args, fmt.Sprintf("-openshift-addons-list=%s", strings.Join(cfg.Spec.UserCluster.Addons.Openshift.Default, ",")))
			}

			if cfg.Spec.UserCluster.Addons.Kubernetes.DefaultManifests != "" {
				args = append(args, "-kubernetes-addons-file=/opt/extra-files/"+common.KubernetesAddonsFileName)
			} else {
				args = append(args, fmt.Sprintf("-kubernetes-addons-list=%s", strings.Join(cfg.Spec.UserCluster.Addons.Kubernetes.Default, ",")))
			}

			volumes = append(volumes, corev1.Volume{
				Name: "extra-files",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: common.ExtraFilesSecretName,
					},
				},
			})

			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				MountPath: "/opt/extra-files/",
				Name:      "extra-files",
				ReadOnly:  true,
			})

			if cfg.Spec.FeatureGates.Has(features.OpenIDAuthPlugin) {
				args = append(args,
					fmt.Sprintf("-oidc-issuer-url=%s", cfg.Spec.Auth.TokenIssuer),
					fmt.Sprintf("-oidc-issuer-client-id=%s", cfg.Spec.Auth.IssuerClientID),
					fmt.Sprintf("-oidc-issuer-client-secret=%s", cfg.Spec.Auth.IssuerClientSecret),
				)

				if cfg.Spec.Auth.CABundle != "" {
					args = append(args, "-oidc-ca-file=/opt/dex-ca/caBundle.pem")

					volumes = append(volumes, corev1.Volume{
						Name: "dex-ca",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName: common.DexCASecretName,
							},
						},
					})

					volumeMounts = append(volumeMounts, corev1.VolumeMount{
						Name:      "dex-ca",
						MountPath: "/opt/dex-ca",
						ReadOnly:  true,
					})
				}
			}

			if len(cfg.Spec.UserCluster.Monitoring.CustomScrapingConfigs) > 0 {
				path := "/opt/" + clusterNamespacePrometheusScrapingConfigsConfigMapName
				args = append(args, fmt.Sprintf("-in-cluster-prometheus-scraping-configs-file=%s/%s", path, clusterNamespacePrometheusScrapingConfigsKey))

				volumes = append(volumes, corev1.Volume{
					Name: clusterNamespacePrometheusScrapingConfigsConfigMapName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: clusterNamespacePrometheusScrapingConfigsConfigMapName,
							},
						},
					},
				})

				volumeMounts = append(volumeMounts, corev1.VolumeMount{
					Name:      clusterNamespacePrometheusScrapingConfigsConfigMapName,
					MountPath: path,
					ReadOnly:  true,
				})
			}

			if len(cfg.Spec.UserCluster.Monitoring.CustomRules) > 0 {
				path := "/opt/" + clusterNamespacePrometheusRulesConfigMapName
				args = append(args, fmt.Sprintf("-in-cluster-prometheus-rules-file=%s/%s", path, clusterNamespacePrometheusRulesKey))

				volumes = append(volumes, corev1.Volume{
					Name: clusterNamespacePrometheusRulesConfigMapName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: clusterNamespacePrometheusRulesConfigMapName,
							},
						},
					},
				})

				volumeMounts = append(volumeMounts, corev1.VolumeMount{
					Name:      clusterNamespacePrometheusRulesConfigMapName,
					MountPath: path,
					ReadOnly:  true,
				})
			}

			d.Spec.Template.Spec.Volumes = volumes
			d.Spec.Template.Spec.InitContainers = []corev1.Container{
				createKubernetesAddonsInitContainer(cfg, sharedAddonVolume, versions.Kubermatic),
				createOpenshiftAddonsInitContainer(cfg, sharedAddonVolume, versions.Kubermatic),
			}
			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "controller-manager",
					Image:   cfg.Spec.SeedController.DockerRepository + ":" + versions.Kubermatic,
					Command: []string{"seed-controller-manager"},
					Args:    args,
					Env:     common.ProxyEnvironmentVars(cfg),
					Ports: []corev1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: 8085,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: volumeMounts,
					Resources:    cfg.Spec.SeedController.Resources,
				},
			}

			return d, nil
		}
	}
}

func createKubernetesAddonsInitContainer(cfg *operatorv1alpha1.KubermaticConfiguration, addonVolume string, dockerTag string) corev1.Container {
	return corev1.Container{
		Name:    "copy-addons-kubernetes",
		Image:   cfg.Spec.UserCluster.Addons.Kubernetes.DockerRepository + ":" + dockerTag,
		Command: []string{"/bin/sh"},
		Args: []string{
			"-c",
			"mkdir -p /opt/addons/kubernetes && cp -r /addons/* /opt/addons/kubernetes",
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      addonVolume,
				MountPath: "/opt/addons/",
			},
		},
	}
}

func createOpenshiftAddonsInitContainer(cfg *operatorv1alpha1.KubermaticConfiguration, addonVolume string, dockerTag string) corev1.Container {
	return corev1.Container{
		Name:    "copy-addons-openshift",
		Image:   cfg.Spec.UserCluster.Addons.Openshift.DockerRepository + ":" + dockerTag,
		Command: []string{"/bin/sh"},
		Args: []string{
			"-c",
			"mkdir -p /opt/addons/openshift && cp -r /addons/* /opt/addons/openshift",
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      addonVolume,
				MountPath: "/opt/addons/",
			},
		},
	}
}

func SeedControllerManagerPDBCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedPodDisruptionBudgetCreatorGetter {
	name := "kubermatic-seed-controller-manager"

	return func() (string, reconciling.PodDisruptionBudgetCreator) {
		return name, func(pdb *policyv1beta1.PodDisruptionBudget) (*policyv1beta1.PodDisruptionBudget, error) {
			min := intstr.FromInt(1)

			pdb.Spec.MinAvailable = &min
			pdb.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: seedControllerManagerPodLabels(),
			}

			return pdb, nil
		}
	}
}
