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

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"

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

func MasterControllerManagerDeploymentCreator(cfg *kubermaticv1.KubermaticConfiguration, workerName string, versions kubermatic.Versions) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return common.MasterControllerManagerDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Spec.Replicas = cfg.Spec.MasterController.Replicas
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: masterControllerManagerPodLabels(),
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
				"-worker-count=20",
				fmt.Sprintf("-namespace=%s", cfg.Namespace),
				fmt.Sprintf("-pprof-listen-address=%s", *cfg.Spec.MasterController.PProfEndpoint),
				fmt.Sprintf("-feature-gates=%s", common.StringifyFeatureGates(cfg)),
			}

			if cfg.Spec.MasterController.DebugLog {
				args = append(args, "-v=4", "-log-debug=true")
			} else {
				args = append(args, "-v=2")
			}

			if workerName != "" {
				args = append(args, fmt.Sprintf("-worker-name=%s", workerName))
			}

			// since #8556, this Deployment doesn't use the webhook-cert Secret
			// anymore, but if we don't explicitly set the volumes to empty,
			// the volume will never be removed from the Deployment;
			// this line can be removed in KKP 2.22+
			d.Spec.Template.Spec.Volumes = []corev1.Volume{}

			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "controller-manager",
					Image:   cfg.Spec.MasterController.DockerRepository + ":" + versions.Kubermatic,
					Command: []string{"master-controller-manager"},
					Args:    args,
					Env:     common.ProxyEnvironmentVars(cfg),
					Ports: []corev1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: 8085,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Resources: cfg.Spec.MasterController.Resources,
				},
			}

			return d, nil
		}
	}
}

func MasterControllerManagerPDBCreator(cfg *kubermaticv1.KubermaticConfiguration) reconciling.NamedPodDisruptionBudgetCreatorGetter {
	name := "kubermatic-master-controller-manager"

	return func() (string, reconciling.PodDisruptionBudgetCreator) {
		return name, func(pdb *policyv1.PodDisruptionBudget) (*policyv1.PodDisruptionBudget, error) {
			// To prevent the PDB from blocking node rotations, we accept
			// 0 minAvailable if the replica count is only 1.
			// NB: The cfg is defaulted, so Replicas==nil cannot happen.
			min := intstr.FromInt(1)
			if cfg.Spec.MasterController.Replicas != nil && *cfg.Spec.MasterController.Replicas < 2 {
				min = intstr.FromInt(0)
			}

			pdb.Spec.MinAvailable = &min
			pdb.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: masterControllerManagerPodLabels(),
			}

			return pdb, nil
		}
	}
}
