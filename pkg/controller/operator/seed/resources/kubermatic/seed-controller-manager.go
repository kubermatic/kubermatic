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

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
	operatorresources "k8c.io/kubermatic/v3/pkg/controller/operator/seed/resources"
	"k8c.io/kubermatic/v3/pkg/resources"
	"k8c.io/kubermatic/v3/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func seedControllerManagerPodLabels() map[string]string {
	return map[string]string{
		operatorresources.NameLabel: operatorresources.SeedControllerManagerDeploymentName,
	}
}

func SeedControllerManagerDeploymentReconciler(workerName string, versions kubermatic.Versions, config *kubermaticv1.KubermaticConfiguration) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return operatorresources.SeedControllerManagerDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Spec.Replicas = config.Spec.ControllerManager.Replicas
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
				"-worker-count=4",
				fmt.Sprintf("-ca-bundle=/opt/ca-bundle/%s", resources.CABundleConfigMapKey),
				fmt.Sprintf("-namespace=%s", config.Namespace),
				fmt.Sprintf("-external-url=%s", config.Spec.Ingress.Domain),
				fmt.Sprintf("-etcd-disk-size=%s", config.Spec.UserCluster.EtcdVolumeSize),
				fmt.Sprintf("-feature-gates=%s", operatorresources.StringifyFeatureGates(config)),
				fmt.Sprintf("-worker-name=%s", workerName),
				fmt.Sprintf("-kubermatic-image=%s", config.Spec.UserCluster.KubermaticDockerRepository),
				fmt.Sprintf("-dnatcontroller-image=%s", config.Spec.UserCluster.DNATControllerDockerRepository),
				fmt.Sprintf("-etcd-launcher-image=%s", config.Spec.UserCluster.EtcdLauncherDockerRepository),
				fmt.Sprintf("-overwrite-registry=%s", config.Spec.UserCluster.OverwriteRegistry),
				fmt.Sprintf("-max-parallel-reconcile=%d", config.Spec.ControllerManager.MaximumParallelReconciles),
				fmt.Sprintf("-pprof-listen-address=%s", *config.Spec.ControllerManager.PProfEndpoint),
			}

			if config.Spec.ImagePullSecret != "" {
				args = append(args, fmt.Sprintf("-docker-pull-config-json-file=/opt/docker/%s", corev1.DockerConfigJsonKey))
			}

			if mla := config.Spec.UserCluster.MLA; mla != nil && mla.Enabled {
				args = append(args, "-enable-user-cluster-mla")
			}

			if config.Spec.ControllerManager.DebugLog {
				args = append(args, "-v=4", "-log-debug=true")
			} else {
				args = append(args, "-v=2")
			}

			if uCfg := config.Spec.UserCluster; uCfg != nil {
				if mcCfg := uCfg.MachineController; mcCfg != nil {
					if mcCfg.ImageTag != "" {
						args = append(args, fmt.Sprintf("-machine-controller-image-tag=%s", mcCfg.ImageTag))
					}
					if mcCfg.ImageRepository != "" {
						args = append(args, fmt.Sprintf("-machine-controller-image-repository=%s", mcCfg.ImageRepository))
					}
				}
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
					Name: "ca-bundle",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: config.Spec.CABundle.Name,
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
					Name:      "ca-bundle",
					MountPath: "/opt/ca-bundle/",
					ReadOnly:  true,
				},
			}

			if config.Spec.ImagePullSecret != "" {
				volumes = append(volumes, corev1.Volume{
					Name: "dockercfg",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: operatorresources.DockercfgSecretName,
						},
					},
				})
				volumeMounts = append(volumeMounts, corev1.VolumeMount{
					Name:      "dockercfg",
					MountPath: "/opt/docker/",
					ReadOnly:  true,
				})
			}

			if auth := config.Spec.Auth; auth != nil {
				args = append(args,
					fmt.Sprintf("-oidc-issuer-url=%s", auth.TokenIssuer),
					fmt.Sprintf("-oidc-issuer-client-id=%s", auth.IssuerClientID),
					fmt.Sprintf("-oidc-issuer-client-secret=%s", auth.IssuerClientSecret),
				)
			}

			d.Spec.Template.Spec.Volumes = volumes
			d.Spec.Template.Spec.InitContainers = []corev1.Container{
				createAddonsInitContainer(config.Spec.UserCluster.Addons, sharedAddonVolume, versions.Kubermatic),
			}
			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "controller-manager",
					Image:   config.Spec.ControllerManager.DockerRepository + ":" + versions.Kubermatic,
					Command: []string{"seed-controller-manager"},
					Args:    args,
					Env:     operatorresources.KubermaticProxyEnvironmentVars(config.Spec.Proxy),
					Ports: []corev1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: 8085,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: volumeMounts,
				},
			}

			if res := config.Spec.ControllerManager.Resources; res != nil {
				d.Spec.Template.Spec.Containers[0].Resources = *res
			}

			return d, nil
		}
	}
}

func createAddonsInitContainer(cfg *kubermaticv1.KubermaticAddonsConfiguration, addonVolume string, version string) corev1.Container {
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

func getAddonDockerTag(cfg *kubermaticv1.KubermaticAddonsConfiguration, version string) string {
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
			min := intstr.FromInt(1)
			if cfg.Spec.ControllerManager.Replicas != nil && *cfg.Spec.ControllerManager.Replicas < 2 {
				min = intstr.FromInt(0)
			}

			pdb.Spec.MinAvailable = &min
			pdb.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: seedControllerManagerPodLabels(),
			}

			return pdb, nil
		}
	}
}
