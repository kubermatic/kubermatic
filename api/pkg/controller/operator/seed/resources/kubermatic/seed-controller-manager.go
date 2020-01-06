package kubermatic

import (
	"fmt"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/features"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

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
			d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{
					Name: common.DockercfgSecretName,
				},
			}

			args := []string{
				"-logtostderr",
				"-dynamic-datacenters=true",
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
					Name: "dockercfg",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: common.DockercfgSecretName,
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
					Name:      "dockercfg",
					MountPath: "/opt/docker/",
					ReadOnly:  true,
				},
				{
					Name:      "seed-webhook-serving-cert",
					MountPath: "/opt/seed-webhook-serving-cert/",
					ReadOnly:  true,
				},
			}

			args = append(
				args,
				"-versions=/opt/master-files/versions.yaml",
				"-updates=/opt/master-files/updates.yaml",
			)

			if cfg.Spec.UserCluster.Addons.Openshift.DefaultManifests != "" {
				args = append(args, "-openshift-addons-file=/opt/master-files/"+common.OpenshiftAddonsFileName)
			} else {
				args = append(args, fmt.Sprintf("-openshift-addons-list=%s", strings.Join(cfg.Spec.UserCluster.Addons.Openshift.Default, ",")))
			}

			if cfg.Spec.UserCluster.Addons.Kubernetes.DefaultManifests != "" {
				args = append(args, "-kubernetes-addons-file=/opt/master-files/"+common.KubernetesAddonsFileName)
			} else {

				args = append(args, fmt.Sprintf("-kubernetes-addons-list=%s", strings.Join(cfg.Spec.UserCluster.Addons.Kubernetes.Default, ",")))
			}

			volumes = append(volumes, corev1.Volume{
				Name: "master-files",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: common.MasterFilesSecretName,
					},
				},
			})

			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				MountPath: "/opt/master-files/",
				Name:      "master-files",
				ReadOnly:  true,
			})

			if cfg.Spec.FeatureGates.Has(features.OpenIDAuthPlugin) {
				args = append(args,
					"-oidc-ca-file=/opt/dex-ca/caBundle.pem",
					fmt.Sprintf("-oidc-issuer-url=%s", cfg.Spec.Auth.TokenIssuer),
					fmt.Sprintf("-oidc-issuer-client-id=%s", cfg.Spec.Auth.IssuerClientID),
					fmt.Sprintf("-oidc-issuer-client-secret=%s", cfg.Spec.Auth.IssuerClientSecret),
				)

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
