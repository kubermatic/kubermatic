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
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	utilpointer "k8s.io/utils/pointer"
)

const (
	cloudCredentialOperatorDeploymentName = "cloud-credential-operator"
)

func CloudCredentialOperator(data openshiftData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return cloudCredentialOperatorDeploymentName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			image, err := cloudCredentialOperatorImage(data.Cluster().Spec.Version.String(), data.ImageRegistry(""))
			if err != nil {
				return nil, err
			}

			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: resources.BaseAppLabels(cloudCredentialOperatorDeploymentName, nil),
			}
			d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{Name: openshiftImagePullSecretName},
			}
			d.Spec.Template.Spec.AutomountServiceAccountToken = utilpointer.BoolPtr(false)
			d.Spec.Template.Spec.InitContainers = []corev1.Container{}
			d.Spec.Template.Spec.Containers = []corev1.Container{{
				Name:    cloudCredentialOperatorDeploymentName,
				Command: []string{"/root/manager", "--log-level", "debug"},
				Image:   image,
				Env: []corev1.EnvVar{
					{
						Name:  "RELEASE_VERSION",
						Value: data.Cluster().Spec.Version.String(),
					},
					{
						Name:  "KUBECONFIG",
						Value: "/etc/kubernetes/kubeconfig/kubeconfig",
					},
				},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      resources.InternalUserClusterAdminKubeconfigSecretName,
					MountPath: "/etc/kubernetes/kubeconfig",
				}},
			}}
			d.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					// TODO: Properly limit this instead of just using cluster-admin
					Name: resources.InternalUserClusterAdminKubeconfigSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.InternalUserClusterAdminKubeconfigSecretName,
						},
					},
				},
			}

			d.Spec.Template.Labels, err = data.GetPodTemplateLabels(cloudCredentialOperatorDeploymentName, d.Spec.Template.Spec.Volumes, nil)
			if err != nil {
				return nil, err
			}
			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, d.Spec.Template.Spec, sets.NewString(cloudCredentialOperatorDeploymentName), "CredentialsRequest,cloudcredential.openshift.io/v1")
			if err != nil {
				return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
			}
			d.Spec.Template.Spec = *wrappedPodSpec

			return d, nil
		}

	}
}
