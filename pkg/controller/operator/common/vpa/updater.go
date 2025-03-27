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

package vpa

import (
	"fmt"
	"strconv"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/version/kubermatic"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

const (
	UpdaterName = "vpa-updater"
	updaterPort = 8942
)

func UpdaterServiceAccountReconciler() reconciling.NamedServiceAccountReconcilerFactory {
	return func() (string, reconciling.ServiceAccountReconciler) {
		return UpdaterName, func(sa *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			return sa, nil
		}
	}
}

// Sourced from
// https://github.com/kubernetes/autoscaler/blob/vertical-pod-autoscaler-1.0.0/vertical-pod-autoscaler/deploy/updater-deployment.yaml

func UpdaterDeploymentReconciler(cfg *kubermaticv1.KubermaticConfiguration, versions kubermatic.Versions) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return UpdaterName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Spec.Replicas = ptr.To[int32](1)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: appPodLabels(UpdaterName),
			}

			d.Spec.Template.Labels = d.Spec.Selector.MatchLabels
			d.Spec.Template.Annotations = map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/port":   strconv.Itoa(updaterPort),
				"fluentbit.io/parser":  "glog",
			}

			d.Spec.Template.Spec.ServiceAccountName = UpdaterName
			d.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				RunAsNonRoot: ptr.To(true),
				RunAsUser:    ptr.To[int64](65534),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
			}
			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "updater",
					Image:   cfg.Spec.VerticalPodAutoscaler.Updater.DockerRepository + ":" + Version,
					Command: []string{"/updater"},
					Args: []string{
						fmt.Sprintf("--address=:%d", updaterPort),
						"--evict-after-oom-threshold=30m",
						"--updater-interval=10m",
						"--logtostderr",
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: updaterPort,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					Resources: cfg.Spec.VerticalPodAutoscaler.Updater.Resources,
				},
			}

			return d, nil
		}
	}
}

func UpdaterPDBReconciler() reconciling.NamedPodDisruptionBudgetReconcilerFactory {
	maxUnavailable := intstr.FromInt(1)
	return func() (string, reconciling.PodDisruptionBudgetReconciler) {
		return UpdaterName, func(pdb *policyv1.PodDisruptionBudget) (*policyv1.PodDisruptionBudget, error) {
			pdb.Spec.MaxUnavailable = &maxUnavailable
			pdb.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: appPodLabels(UpdaterName),
			}
			return pdb, nil
		}
	}
}
