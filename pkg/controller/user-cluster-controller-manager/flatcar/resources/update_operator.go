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

package resources

import (
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	OperatorDeploymentName = "flatcar-linux-update-operator"
	operatorVersion        = "v0.9.0"
)

var (
	deploymentReplicas       int32 = 1
	deploymentMaxSurge             = intstr.FromInt(1)
	deploymentMaxUnavailable       = intstr.FromString("25%")
)

func OperatorDeploymentReconciler(imageRewriter registry.ImageRewriter, updateWindow kubermaticv1.UpdateWindow) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return OperatorDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Spec.Replicas = &deploymentReplicas

			dep.Spec.Strategy = appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       &deploymentMaxSurge,
					MaxUnavailable: &deploymentMaxUnavailable,
				},
			}

			// We broke compatibility with upstream in #5875 and instead of performing a migration,
			// we simply keep the changed labels.
			labels := map[string]string{"app.kubernetes.io/name": OperatorDeploymentName}

			dep.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
			dep.Spec.Template.Labels = labels
			dep.Spec.Template.Spec.ServiceAccountName = operatorServiceAccountName

			env := []corev1.EnvVar{
				{
					Name: "POD_NAMESPACE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							APIVersion: "v1",
							FieldPath:  "metadata.namespace",
						},
					},
				},
			}

			if updateWindow.Start != "" {
				env = append(env, corev1.EnvVar{
					Name:  "UPDATE_OPERATOR_REBOOT_WINDOW_START",
					Value: updateWindow.Start,
				})
			}

			if updateWindow.Length != "" {
				env = append(env, corev1.EnvVar{
					Name:  "UPDATE_OPERATOR_REBOOT_WINDOW_LENGTH",
					Value: updateWindow.Length,
				})
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    "update-operator",
					Image:   operatorImage(imageRewriter),
					Command: []string{"/bin/update-operator"},
					Env:     env,
				},
			}

			return dep, nil
		}
	}
}

func operatorImage(imageRewriter registry.ImageRewriter) string {
	return registry.Must(imageRewriter("ghcr.io/flatcar/flatcar-linux-update-operator:" + operatorVersion))
}
