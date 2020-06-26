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
	nodelabelerapi "github.com/kubermatic/kubermatic/api/pkg/controller/user-cluster-controller-manager/node-labeler/api"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	DeploymentName = "container-linux-update-operator"
)

var (
	deploymentReplicas       int32 = 1
	deploymentMaxSurge             = intstr.FromInt(1)
	deploymentMaxUnavailable       = intstr.FromString("25%")
)

type GetImageRegistry func(reg string) string

func DeploymentCreator(getRegistry GetImageRegistry, updateWindow kubermaticv1.UpdateWindow) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return DeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			dep.Spec.Replicas = &deploymentReplicas

			dep.Spec.Strategy = appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDeployment{
					MaxSurge:       &deploymentMaxSurge,
					MaxUnavailable: &deploymentMaxUnavailable,
				},
			}

			labels := map[string]string{"app": "container-linux-update-operator"}
			dep.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
			dep.Spec.Template.ObjectMeta.Labels = labels
			dep.Spec.Template.Spec.ServiceAccountName = ServiceAccountName

			// The operator should only run on ContainerLinux nodes
			dep.Spec.Template.Spec.NodeSelector = map[string]string{nodelabelerapi.DistributionLabelKey: nodelabelerapi.ContainerLinuxLabelValue}

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
					Image:   getRegistry(resources.RegistryQuay) + "/coreos/container-linux-update-operator:v0.7.0",
					Command: []string{"/bin/update-operator"},
					Env:     env,
				},
			}

			return dep, nil
		}
	}
}
