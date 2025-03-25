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

package cloudcontroller

import (
	"errors"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/vpnsidecar"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	ccmContainerName           = "cloud-controller-manager"
	openvpnClientContainerName = "openvpn-client"
)

type ccmDeploymentReconcilerFunc func(*resources.TemplateData) reconciling.NamedDeploymentReconcilerFactory

func getCCMDeploymentReconciler(spec *kubermaticv1.ClusterSpec) ccmDeploymentReconcilerFunc {
	if spec != nil {
		switch {
		case spec.Cloud.AWS != nil:
			return awsDeploymentReconciler

		case spec.Cloud.Azure != nil:
			return azureDeploymentReconciler

		case spec.Cloud.Openstack != nil:
			return openStackDeploymentReconciler

		case spec.Cloud.Hetzner != nil:
			return hetznerDeploymentReconciler

		case spec.Cloud.GCP != nil:
			return gcpDeploymentReconciler

		case spec.Cloud.Anexia != nil:
			return anexiaDeploymentReconciler

		case spec.Cloud.VSphere != nil:
			return vsphereDeploymentReconciler

		case spec.Cloud.Kubevirt != nil:
			return kubevirtDeploymentReconciler

		case spec.Cloud.Digitalocean != nil:
			return digitalOceanDeploymentReconciler
		}
	}

	return nil
}

func HasCCM(spec *kubermaticv1.ClusterSpec) bool {
	return getCCMDeploymentReconciler(spec) != nil
}

// DeploymentReconciler returns the function to create and update the external cloud provider deployment.
func DeploymentReconciler(data *resources.TemplateData) reconciling.NamedDeploymentReconcilerFactory {
	deployer := getCCMDeploymentReconciler(&data.Cluster().Spec)

	if deployer != nil {
		creatorGetter := deployer(data)

		return func() (name string, create reconciling.DeploymentReconciler) {
			name, creator := creatorGetter()

			return name, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
				dep.Spec.Template.Spec.InitContainers = []corev1.Container{}

				modified, err := creator(dep)
				if err != nil {
					return nil, err
				}

				baseLabels := resources.BaseAppLabels(name, nil)
				kubernetes.EnsureLabels(modified, baseLabels)
				kubernetes.EnsureAnnotations(&modified.Spec.Template, map[string]string{
					resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
				})

				modified.Spec.Selector = &metav1.LabelSelector{
					MatchLabels: baseLabels,
				}

				containerNames := sets.New(ccmContainerName)

				if !data.IsKonnectivityEnabled() {
					// inject the openVPN sidecar container
					openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, openvpnClientContainerName)
					if err != nil {
						return nil, fmt.Errorf("failed to get openvpn sidecar: %w", err)
					}
					modified.Spec.Template.Spec.Containers = append(modified.Spec.Template.Spec.Containers, *openvpnSidecar)

					containerNames.Insert(openvpnSidecar.Name)
				}

				modified.Spec.Template, err = apiserver.IsRunningWrapper(data, modified.Spec.Template, containerNames)
				if err != nil {
					return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %w", err)
				}

				return modified, nil
			}
		}
	}

	return func() (name string, create reconciling.DeploymentReconciler) {
		return "unsupported", func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			return nil, errors.New("unsupported external cloud controller")
		}
	}
}
