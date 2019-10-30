package kubermatic

import (
	"fmt"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/controller/operator/common"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

func seedControllerManagerPodLabels() map[string]string {
	return map[string]string{
		nameLabel: "kubermatic-seed-controller-manager",
	}
}

func SeedControllerManagerDeploymentCreator(workerName string, cfg *operatorv1alpha1.KubermaticConfiguration, seed *kubermaticv1.Seed) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return seedControllerManagerDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Spec.Replicas = pointer.Int32Ptr(2)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: seedControllerManagerPodLabels(),
			}

			d.Spec.Template.Labels = d.Spec.Selector.MatchLabels
			d.Spec.Template.Annotations = map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "8085",
				"fluentbit.io/parser":  "glog",
			}

			d.Spec.Template.Spec.ServiceAccountName = serviceAccountName
			d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{
					Name: common.DockercfgSecretName,
				},
			}

			args := []string{
				"-dynamic-datacenters=true",
				"-internal-address=0.0.0.0:8085",
				"-kubernetes-addons-path=/opt/addons/kubernetes",
				"-openshift-addons-path=/opt/addons/openshift",
				"-backup-container=/opt/backup/store-container.yaml",
				"-cleanup-container=/opt/backup/cleanup-container.yaml",
				"-docker-pull-config-json-file=/opt/docker/.dockerconfigjson",
				"-seed-admissionwebhook-cert-file=/opt/seed-webhook-serving-cert/serverCert.pem",
				"-seed-admissionwebhook-key-file=/opt/seed-webhook-serving-cert/serverKey.pem",
				fmt.Sprintf("-namespace=%s", cfg.Namespace),
				fmt.Sprintf("-external-url=%s", cfg.Spec.Domain),
				fmt.Sprintf("-datacenter-name=%s", seed.Name),
				fmt.Sprintf("-etcd-disk-size=%s", "5Gi"), // TODO: add to KubermaticConfiguration
				fmt.Sprintf("-feature-gates=%s", common.StringifyFeatureGates(cfg)),
				fmt.Sprintf("-nodeport-range=%s", cfg.Spec.SeedController.NodePortRange),
				fmt.Sprintf("-worker-count=%d", 4),
				fmt.Sprintf("-worker-name=%s", workerName),
				fmt.Sprintf("-kubermatic-image=%s", cfg.Spec.SeedController.KubermaticImage),
				fmt.Sprintf("-dnatcontoller-image=%s", "TODO"),
				fmt.Sprintf("-kubernetes-addons-list=%s", strings.Join(cfg.Spec.SeedController.Addons.Kubernetes.Default, ",")),
				fmt.Sprintf("-openshift-addons-list=%s", strings.Join(cfg.Spec.SeedController.Addons.Openshift.Default, ",")),
				fmt.Sprintf("-overwrite-registry=%s", cfg.Spec.SeedController.OverwriteRegistry),
				fmt.Sprintf("-apiserver-default-replicas=%d", 2),
				fmt.Sprintf("-controller-manager-default-replicas=%d", 1),
				fmt.Sprintf("-scheduler-default-replicas=%d", 1),
				fmt.Sprintf("-max-parallel-reconcile=%d", 10),
				fmt.Sprintf("-apiserver-reconciling-disabled-by-default=%v", false),

				// {{- if .Values.kubermatic.clusterNamespacePrometheus.disableDefaultRules }}
				// - -in-cluster-prometheus-disable-default-rules=true
				// {{- end }}
				// {{- if .Values.kubermatic.clusterNamespacePrometheus.rules }}
				// - -in-cluster-prometheus-rules-file=/opt/incluster-prometheus-rules/_customrules.yaml
				// {{- end }}
				// {{- if .Values.kubermatic.clusterNamespacePrometheus.disableDefaultScrapingConfigs }}
				// - -in-cluster-prometheus-disable-default-scraping-configs=true
				// {{- end }}
				// {{- if .Values.kubermatic.clusterNamespacePrometheus.scrapingConfigs }}
				// - -in-cluster-prometheus-scraping-configs-file=/opt/incluster-prometheus-scraping-configs/_custom-scraping-configs.yaml
				// {{- end }}
				// {{- if .Values.kubermatic.monitoringScrapeAnnotationPrefix }}
				// - -monitoring-scrape-annotation-prefix={{ .Values.kubermatic.monitoringScrapeAnnotationPrefix }}
				// {{- end }}
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
								Name: common.BackupContainersConfigMapName,
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

			if len(cfg.Spec.MasterFiles) > 0 {
				args = append(
					args,
					"-versions=/opt/master-files/versions.yaml",
					"-updates=/opt/master-files/updates.yaml",
					"-master-resources=/opt/master-files",
				)

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
			}

			if cfg.Spec.FeatureGates.Has(openIDAuthFeatureFlag) {
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

			d.Spec.Template.Spec.Volumes = volumes
			d.Spec.Template.Spec.InitContainers = []corev1.Container{
				copyKubernetesAddonsContainer(cfg, sharedAddonVolume),
				copyOpenshiftAddonsContainer(cfg, sharedAddonVolume),
			}
			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "controller-manager",
					Image:   cfg.Spec.SeedController.Image,
					Command: []string{"kubermatic-controller-manager"},
					Args:    args,
					Ports: []corev1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: 8085,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: volumeMounts,
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("512Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("500m"),
							corev1.ResourceMemory: resource.MustParse("1Gi"),
						},
					},
				},
			}

			return d, nil
		}
	}
}

func copyKubernetesAddonsContainer(cfg *operatorv1alpha1.KubermaticConfiguration, addonVolume string) corev1.Container {
	return corev1.Container{
		Name:    "copy-addons-kubernetes",
		Image:   cfg.Spec.SeedController.Addons.Kubernetes.Image,
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

func copyOpenshiftAddonsContainer(cfg *operatorv1alpha1.KubermaticConfiguration, addonVolume string) corev1.Container {
	return corev1.Container{
		Name:    "copy-addons-openshift",
		Image:   cfg.Spec.SeedController.Addons.Openshift.Image,
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
