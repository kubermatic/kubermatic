package kubermatic

import (
	"fmt"
	"strings"

	operatorv1alpha1 "github.com/kubermatic/kubermatic/api/pkg/crd/operator/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func seedControllerManagerPodLabels() map[string]string {
	return map[string]string{
		nameLabel:    "seed-controller-manager",
		versionLabel: "v1",
	}
}

func SeedControllerManagerDeploymentCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return seedControllerManagerDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			specLabels := seedControllerManagerPodLabels()

			d.Spec.Replicas = i32ptr(2)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: specLabels,
			}

			d.Spec.Template.Labels = specLabels
			d.Spec.Template.Annotations = map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "8085",
				"fluentbit.io/parser":  "glog",

				// TODO: add checksums for kubeconfig, datacenters etc. to trigger redeployments
			}

			d.Spec.Template.Spec.ServiceAccountName = serviceAccountName
			d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{
					Name: dockercfgSecretName,
				},
			}

			args := []string{
				"-v=2",
				"-logtostderr",
				"-internal-address=0.0.0.0:8085",
				fmt.Sprintf("-external-url=%s", cfg.Spec.Domain),
				fmt.Sprintf("-datacenter-name=%s", "{{ .Values.kubermatic.controller.datacenterName }}"),
				fmt.Sprintf("-etcd-disk-size=%s", "{{ .Values.kubermatic.etcd.diskSize }}"),
				fmt.Sprintf("-kubernetes-addons-list=%s", strings.Join(cfg.Spec.SeedController.Addons.Kubernetes.Default, ",")),
				fmt.Sprintf("-openshift-addons-list=%s", strings.Join(cfg.Spec.SeedController.Addons.Openshift.Default, ",")),
				fmt.Sprintf("-overwrite-registry=%s", cfg.Spec.SeedController.OverwriteRegistry),
				fmt.Sprintf("-nodeport-range=%s", cfg.Spec.SeedController.NodePortRange),
				fmt.Sprintf("-feature-gates=%s", featureGates(cfg)),
				"-kubernetes-addons-path=/opt/addons/kubernetes",
				"-openshift-addons-path=/opt/addons/openshift",
				"-backup-container=/opt/backup/store-container.yaml",
				"-cleanup-container=/opt/backup/cleanup-container.yaml",
				"-docker-pull-config-json-file=/opt/docker/.dockerconfigjson",

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

			if cfg.Spec.SeedController.KubermaticImage != "" {
				args = append(args, fmt.Sprintf("-kubermatic-image=%s", cfg.Spec.SeedController.KubermaticImage))
			}

			volumes := []corev1.Volume{
				{
					Name: "addons",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "backup-containers",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							DefaultMode:          i32ptr(420),
							LocalObjectReference: corev1.LocalObjectReference{Name: backupContainersConfigMapName},
						},
					},
				},
				{
					Name: "dockercfg",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							DefaultMode: i32ptr(420),
							SecretName:  dockercfgSecretName,
						},
					},
				},
			}

			volumeMounts := []corev1.VolumeMount{
				{
					MountPath: "/opt/addons/",
					Name:      "addons",
					ReadOnly:  true,
				},
				{
					MountPath: "/opt/backup/",
					Name:      "backup-containers",
					ReadOnly:  true,
				},
				{
					MountPath: "/opt/docker/",
					Name:      "dockercfg",
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
							DefaultMode: i32ptr(420),
							SecretName:  masterFilesSecretName,
						},
					},
				})

				volumeMounts = append(volumeMounts, corev1.VolumeMount{
					MountPath: "/opt/master-files/",
					Name:      "master-files",
					ReadOnly:  true,
				})
			}

			if cfg.Spec.FeatureGates["OpenIDAuthPlugin"] {
				args = append(
					args,
					"-oidc-ca-file=/opt/dex-ca/caBundle.pem",
					fmt.Sprintf("-oidc-issuer-url=%s", cfg.Spec.Auth.TokenIssuer),
					fmt.Sprintf("-oidc-issuer-client-id=%s", cfg.Spec.Auth.IssuerClientID),
					fmt.Sprintf("-oidc-issuer-client-secret=%s", cfg.Spec.Auth.IssuerClientSecret),
				)

				volumes = append(volumes, corev1.Volume{
					Name: "dex-ca",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							DefaultMode: i32ptr(420),
							SecretName:  dexCASecretName,
						},
					},
				})

				volumeMounts = append(volumeMounts, corev1.VolumeMount{
					MountPath: "/opt/dex-ca/",
					Name:      "dex-ca",
					ReadOnly:  true,
				})
			}

			if cfg.Spec.Datacenters != "" {
				args = append(args, "-datacenters=/opt/datacenters/datacenters.yaml")

				volumes = append(volumes, corev1.Volume{
					Name: "datacenters",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							DefaultMode: i32ptr(420),
							SecretName:  datacentersSecretName,
						},
					},
				})

				volumeMounts = append(volumeMounts, corev1.VolumeMount{
					MountPath: "/opt/datacenters/",
					Name:      "datacenters",
					ReadOnly:  true,
				})
			}

			// TODO: Add in-cluster prometheus stuff

			d.Spec.Template.Spec.Volumes = volumes
			d.Spec.Template.Spec.InitContainers = []corev1.Container{
				seedControllerManagerCopyKubernetesAddonsContainer(cfg.Spec.SeedController.Addons.Kubernetes),
				seedControllerManagerCopyOpenshiftAddonsContainer(cfg.Spec.SeedController.Addons.Openshift),
			}

			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            "controller-manager",
					Image:           dockerImage(cfg.Spec.SeedController.Image),
					ImagePullPolicy: cfg.Spec.SeedController.Image.PullPolicy,
					Command:         []string{"kubermatic-controller-manager"},
					Args:            args,
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
					TerminationMessagePolicy: corev1.TerminationMessageReadFile,
					TerminationMessagePath:   "/dev/termination-log",
				},
			}

			return d, nil
		}
	}
}

func seedControllerManagerCopyKubernetesAddonsContainer(cfg operatorv1alpha1.KubermaticAddonConfiguration) corev1.Container {
	return corev1.Container{
		Name:            "copy-addons-kubernetes",
		Image:           dockerImage(cfg.Image),
		ImagePullPolicy: cfg.Image.PullPolicy,
		Command:         []string{"/bin/sh"},
		Args:            []string{"-c", "mkdir -p /opt/addons/kubernetes && cp -r /addons/* /opt/addons/kubernetes"},
		VolumeMounts: []corev1.VolumeMount{
			{
				MountPath: "/opt/addons",
				Name:      "addons",
			},
		},
	}
}

func seedControllerManagerCopyOpenshiftAddonsContainer(cfg operatorv1alpha1.KubermaticAddonConfiguration) corev1.Container {
	return corev1.Container{
		Name:            "copy-addons-openshift",
		Image:           dockerImage(cfg.Image),
		ImagePullPolicy: cfg.Image.PullPolicy,
		Command:         []string{"/bin/sh"},
		Args:            []string{"-c", "mkdir -p /opt/addons/openshift && cp -r /addons/* /opt/addons/openshift"},
		VolumeMounts: []corev1.VolumeMount{
			{
				MountPath: "/opt/addons",
				Name:      "addons",
			},
		},
	}
}

func SeedControllerManagerPDBCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedPodDisruptionBudgetCreatorGetter {
	name := "kubermatic-seed-controller-manager-v1"

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

func SeedControllerManagerServiceCreator(cfg *operatorv1alpha1.KubermaticConfiguration) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return seedControllerManagerServiceName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Spec.Type = corev1.ServiceTypeNodePort
			s.Spec.Selector = seedControllerManagerPodLabels()

			s.Spec.Ports = mergeServicePort(s.Spec.Ports, corev1.ServicePort{
				Name:       "metrics",
				Port:       8085,
				TargetPort: intstr.FromInt(8085),
				Protocol:   corev1.ProtocolTCP,
			})

			return s, nil
		}
	}
}
