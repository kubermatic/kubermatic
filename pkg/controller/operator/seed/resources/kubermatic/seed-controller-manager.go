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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/features"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func seedControllerManagerPodLabels() map[string]string {
	return map[string]string{
		common.NameLabel: common.SeedControllerManagerDeploymentName,
	}
}

func SeedControllerManagerDeploymentReconciler(workerName string, versions kubermatic.Versions, cfg *kubermaticv1.KubermaticConfiguration, seed *kubermaticv1.Seed) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return common.SeedControllerManagerDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			sharedAddonVolume := "addons"
			tempVolume := "temp"

			d.Spec.Replicas = cfg.Spec.SeedController.Replicas
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: seedControllerManagerPodLabels(),
			}

			kubernetes.EnsureLabels(&d.Spec.Template, d.Spec.Selector.MatchLabels)
			kubernetes.EnsureAnnotations(&d.Spec.Template, map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "8085",
				"fluentbit.io/parser":  "json_iso",
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: strings.Join([]string{sharedAddonVolume, tempVolume}, ","),
			})

			d.Spec.Template.Spec.ServiceAccountName = serviceAccountName

			if len(cfg.Spec.SeedController.NodeSelector) > 0 {
				d.Spec.Template.Spec.NodeSelector = cfg.Spec.SeedController.NodeSelector
			}

			if len(cfg.Spec.SeedController.Tolerations) > 0 {
				d.Spec.Template.Spec.Tolerations = cfg.Spec.SeedController.Tolerations
			}

			if cfg.Spec.SeedController.Affinity.NodeAffinity != nil ||
				cfg.Spec.SeedController.Affinity.PodAffinity != nil ||
				cfg.Spec.SeedController.Affinity.PodAntiAffinity != nil {
				d.Spec.Template.Spec.Affinity = &cfg.Spec.SeedController.Affinity
			}

			var disabledCollectors string
			if len(seed.Spec.DisabledCollectors) > 0 {
				disabledCollectors = join(seed.Spec.DisabledCollectors, ",")
			} else {
				disabledCollectors = join(cfg.Spec.SeedController.DisabledCollectors, ",")
			}

			args := []string{
				"-logtostderr",
				"-internal-address=0.0.0.0:8085",
				"-worker-count=4",
				fmt.Sprintf("-ca-bundle=/opt/ca-bundle/%s", resources.CABundleConfigMapKey),
				fmt.Sprintf("-namespace=%s", cfg.Namespace),
				fmt.Sprintf("-external-url=%s", cfg.Spec.Ingress.Domain),
				fmt.Sprintf("-seed-name=%s", seed.Name),
				fmt.Sprintf("-etcd-disk-size=%s", cfg.Spec.UserCluster.EtcdVolumeSize),
				fmt.Sprintf("-feature-gates=%s", common.StringifyFeatureGates(cfg)),
				fmt.Sprintf("-worker-name=%s", workerName),
				fmt.Sprintf("-kubermatic-image=%s", cfg.Spec.UserCluster.KubermaticDockerRepository),
				fmt.Sprintf("-dnatcontroller-image=%s", cfg.Spec.UserCluster.DNATControllerDockerRepository),
				fmt.Sprintf("-etcd-launcher-image=%s", cfg.Spec.UserCluster.EtcdLauncherDockerRepository),
				fmt.Sprintf("-overwrite-registry=%s", cfg.Spec.UserCluster.OverwriteRegistry),
				fmt.Sprintf("-max-parallel-reconcile=%d", cfg.Spec.SeedController.MaximumParallelReconciles),
				fmt.Sprintf("-pprof-listen-address=%s", *cfg.Spec.SeedController.PProfEndpoint),
				fmt.Sprintf("-disabled-collectors=%s", disabledCollectors),
			}

			if cfg.Spec.ImagePullSecret != "" {
				args = append(args, fmt.Sprintf("-docker-pull-config-json-file=/opt/docker/%s", corev1.DockerConfigJsonKey))
			}

			if seed.Spec.MLA != nil && seed.Spec.MLA.UserClusterMLAEnabled {
				args = append(args, "-enable-user-cluster-mla")
			}

			if cfg.Spec.SeedController.DebugLog {
				args = append(args, "-v=4", "-log-debug=true")
			} else {
				args = append(args, "-v=2")
			}

			if cfg.Spec.SeedController.BackupInterval.Duration > 0 {
				args = append(args, fmt.Sprintf("-backup-interval=%s", cfg.Spec.SeedController.BackupInterval.Duration))
			}

			if cfg.Spec.SeedController.BackupCount != nil {
				args = append(args, fmt.Sprintf("-backup-count=%d", *cfg.Spec.SeedController.BackupCount))
			}

			mcCfg := cfg.Spec.UserCluster.MachineController
			if mcCfg.ImageTag != "" {
				args = append(args, fmt.Sprintf("-machine-controller-image-tag=%s", mcCfg.ImageTag))
			}
			if mcCfg.ImageRepository != "" {
				args = append(args, fmt.Sprintf("-machine-controller-image-repository=%s", mcCfg.ImageRepository))
			}

			volumes := []corev1.Volume{
				{
					Name: sharedAddonVolume,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: tempVolume,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				},
				{
					Name: "ca-bundle",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: resources.CABundleConfigMapName,
							},
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
					Name:      tempVolume,
					MountPath: "/tmp/",
				},
				{
					Name:      "ca-bundle",
					MountPath: "/opt/ca-bundle/",
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

			configureSeedLevelOIDCProvider := seed.Spec.OIDCProviderConfiguration != nil

			if configureSeedLevelOIDCProvider {
				args = append(args,
					fmt.Sprintf("-oidc-issuer-url=%s", seed.Spec.OIDCProviderConfiguration.IssuerURL),
					fmt.Sprintf("-oidc-issuer-client-id=%s", seed.Spec.OIDCProviderConfiguration.IssuerClientID),
					fmt.Sprintf("-oidc-issuer-client-secret=%s", seed.Spec.OIDCProviderConfiguration.IssuerClientSecret),
				)
			}

			// Use settings from KubermaticConfiguration only if was not configured for on seed level before.
			if _, fgSet := cfg.Spec.FeatureGates[features.OpenIDAuthPlugin]; !configureSeedLevelOIDCProvider && fgSet {
				args = append(args,
					fmt.Sprintf("-oidc-issuer-url=%s", cfg.Spec.Auth.TokenIssuer),
					fmt.Sprintf("-oidc-issuer-client-id=%s", cfg.Spec.Auth.IssuerClientID),
					fmt.Sprintf("-oidc-issuer-client-secret=%s", cfg.Spec.Auth.IssuerClientSecret),
				)
			}

			d.Spec.Template.Spec.SecurityContext = &common.PodSecurityContext
			d.Spec.Template.Spec.Volumes = volumes
			d.Spec.Template.Spec.InitContainers = []corev1.Container{
				createAddonsInitContainer(cfg.Spec.UserCluster.Addons, sharedAddonVolume, versions.KubermaticContainerTag),
			}
			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "controller-manager",
					Image:   cfg.Spec.SeedController.DockerRepository + ":" + versions.KubermaticContainerTag,
					Command: []string{"seed-controller-manager"},
					Args:    args,
					Env:     common.SeedProxyEnvironmentVars(seed.Spec.ProxySettings),
					Ports: []corev1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: 8085,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts:    volumeMounts,
					Resources:       cfg.Spec.SeedController.Resources,
					SecurityContext: &common.ContainerSecurityContext,
				},
			}

			return d, nil
		}
	}
}

func createAddonsInitContainer(cfg kubermaticv1.KubermaticAddonsConfiguration, addonVolume string, version string) corev1.Container {
	return corev1.Container{
		Name:    "copy-addons",
		Image:   cfg.DockerRepository + ":" + getAddonDockerTag(cfg, version),
		Command: []string{"/bin/sh"},
		Args: []string{
			"-c",
			"mkdir -p /opt/addons && cp -r /addons/* /opt/addons",
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      addonVolume,
				MountPath: "/opt/addons/",
			},
		},
	}
}

func getAddonDockerTag(cfg kubermaticv1.KubermaticAddonsConfiguration, version string) string {
	if cfg.DockerTagSuffix != "" {
		version = fmt.Sprintf("%s-%s", version, cfg.DockerTagSuffix)
	}

	return version
}

func SeedControllerManagerPDBReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedPodDisruptionBudgetReconcilerFactory {
	name := "kubermatic-seed-controller-manager"

	return func() (string, reconciling.PodDisruptionBudgetReconciler) {
		return name, func(pdb *policyv1.PodDisruptionBudget) (*policyv1.PodDisruptionBudget, error) {
			// To prevent the PDB from blocking node rotations, we accept
			// 0 minAvailable if the replica count is only 1.
			// NB: The cfg is defaulted, so Replicas==nil cannot happen.
			minReplicas := intstr.FromInt(1)
			if cfg.Spec.SeedController.Replicas != nil && *cfg.Spec.SeedController.Replicas < 2 {
				minReplicas = intstr.FromInt(0)
			}

			pdb.Spec.MinAvailable = &minReplicas
			pdb.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: seedControllerManagerPodLabels(),
			}

			return pdb, nil
		}
	}
}

func join(collectors []kubermaticv1.MetricsCollector, sep string) string {
	names := make([]string, 0, len(collectors))
	for _, collector := range collectors {
		names = append(names, string(collector))
	}
	return strings.Join(names, sep)
}
