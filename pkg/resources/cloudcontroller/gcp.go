/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package cloudcontroller

import (
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

const GCPCCMDeploymentName = "gcp-cloud-controller-manager"

func gcpDeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (name string, create reconciling.DeploymentCreator) {
		return GCPCCMDeploymentName, func(deployment *appsv1.Deployment) (*appsv1.Deployment, error) {
			deployment.Labels = resources.BaseAppLabels(GCPCCMDeploymentName, nil)
			deployment.Spec.Replicas = resources.Int32(1)

			deployment.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(GCPCCMDeploymentName, nil),
			}

			podLabels, err := data.GetPodTemplateLabels(GCPCCMDeploymentName, deployment.Spec.Template.Spec.Volumes, nil)
			if err != nil {
				return nil, fmt.Errorf("unable to get pod template labels: %w", err)
			}

			deployment.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
			}

			deployment.Spec.Template.Spec.AutomountServiceAccountToken = pointer.Bool(false)
			deployment.Spec.Template.Spec.Volumes = getVolumes(data.IsKonnectivityEnabled(), false)
			deployment.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:  ccmContainerName,
					Image: data.ImageRegistry(resources.RegistryDocker) + "/opsdockerimage/gcp-controller-manager:1edadd08fb75221f975961642cfde871dba8fe90",
					Command: []string{
						"/gcp-controller-manager",
						"--kubeconfig=/etc/kubernetes/kubeconfig/kubeconfig",
					},
					Env:          getEnvVars(),
					VolumeMounts: getVolumeMounts(false),
				},
			}

			return deployment, nil
		}
	}
}
