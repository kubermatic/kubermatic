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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func masterControllerManagerPodLabels() map[string]string {
	return map[string]string{
		common.NameLabel: common.MasterControllerManagerDeploymentName,
	}
}

func MasterControllerManagerDeploymentReconciler(cfg *kubermaticv1.KubermaticConfiguration, workerName string, versions kubermatic.Versions) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return common.MasterControllerManagerDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			labels := masterControllerManagerPodLabels()

			d.Spec.Replicas = cfg.Spec.MasterController.Replicas
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels,
			}

			kubernetes.EnsureLabels(&d.Spec.Template, labels)
			kubernetes.EnsureAnnotations(&d.Spec.Template, map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   "8085",
				"fluentbit.io/parser":  "json_iso",
			})

			d.Spec.Template.Spec.ServiceAccountName = serviceAccountName

			if len(cfg.Spec.MasterController.NodeSelector) > 0 {
				d.Spec.Template.Spec.NodeSelector = cfg.Spec.MasterController.NodeSelector
			}

			if len(cfg.Spec.MasterController.Tolerations) > 0 {
				d.Spec.Template.Spec.Tolerations = cfg.Spec.MasterController.Tolerations
			}

			if cfg.Spec.MasterController.Affinity.NodeAffinity != nil ||
				cfg.Spec.MasterController.Affinity.PodAffinity != nil ||
				cfg.Spec.MasterController.Affinity.PodAntiAffinity != nil {
				d.Spec.Template.Spec.Affinity = &cfg.Spec.MasterController.Affinity
			}

			args := []string{
				"-logtostderr",
				"-internal-address=0.0.0.0:8085",
				"-worker-count=20",
				fmt.Sprintf("-namespace=%s", cfg.Namespace),
				fmt.Sprintf("-pprof-listen-address=%s", *cfg.Spec.MasterController.PProfEndpoint),
				fmt.Sprintf("-feature-gates=%s", common.StringifyFeatureGates(cfg)),
				fmt.Sprintf("-overwrite-registry=%s", cfg.Spec.UserCluster.OverwriteRegistry),
			}

			if cfg.Spec.MasterController.DebugLog {
				args = append(args, "-v=4", "-log-debug=true")
			} else {
				args = append(args, "-v=2")
			}

			if workerName != "" {
				args = append(args, fmt.Sprintf("-worker-name=%s", workerName))
			}

			d.Spec.Template.Spec.SecurityContext = &common.PodSecurityContext
			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "controller-manager",
					Image:   cfg.Spec.MasterController.DockerRepository + ":" + versions.KubermaticContainerTag,
					Command: []string{"master-controller-manager"},
					Args:    args,
					Env:     common.KubermaticProxyEnvironmentVars(&cfg.Spec.Proxy),
					Ports: []corev1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: 8085,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Resources:       cfg.Spec.MasterController.Resources,
					SecurityContext: &common.ContainerSecurityContext,
				},
			}

			return d, nil
		}
	}
}

func MasterControllerManagerPDBReconciler(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedPodDisruptionBudgetReconcilerFactory {
	name := "kubermatic-master-controller-manager"

	return func() (string, reconciling.PodDisruptionBudgetReconciler) {
		return name, func(pdb *policyv1.PodDisruptionBudget) (*policyv1.PodDisruptionBudget, error) {
			// To prevent the PDB from blocking node rotations, we accept
			// 0 minAvailable if the replica count is only 1.
			// NB: The cfg is defaulted, so Replicas==nil cannot happen.
			minReplicas := intstr.FromInt(1)
			if cfg.Spec.MasterController.Replicas != nil && *cfg.Spec.MasterController.Replicas < 2 {
				minReplicas = intstr.FromInt(0)
			}

			pdb.Spec.MinAvailable = &minReplicas
			pdb.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: masterControllerManagerPodLabels(),
			}

			return pdb, nil
		}
	}
}
