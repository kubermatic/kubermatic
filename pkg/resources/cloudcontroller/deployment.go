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

	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/apiserver"
	"k8c.io/kubermatic/v2/pkg/resources/vpnsidecar"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	ccmContainerName           = "cloud-controller-manager"
	openvpnClientContainerName = "openvpn-client"
)

// DeploymentReconciler returns the function to create and update the external cloud provider deployment.
func DeploymentReconciler(data *resources.TemplateData) reconciling.NamedDeploymentReconcilerFactory {
	var creatorGetter reconciling.NamedDeploymentReconcilerFactory

	switch {
	case data.Cluster().Spec.Cloud.AWS != nil:
		creatorGetter = awsDeploymentReconciler(data)

	case data.Cluster().Spec.Cloud.Azure != nil:
		creatorGetter = azureDeploymentReconciler(data)

	case data.Cluster().Spec.Cloud.Openstack != nil:
		creatorGetter = openStackDeploymentReconciler(data)

	case data.Cluster().Spec.Cloud.Hetzner != nil:
		creatorGetter = hetznerDeploymentReconciler(data)

	case data.Cluster().Spec.Cloud.Anexia != nil:
		creatorGetter = anexiaDeploymentReconciler(data)

	case data.Cluster().Spec.Cloud.VSphere != nil:
		creatorGetter = vsphereDeploymentReconciler(data)

	case data.Cluster().Spec.Cloud.Kubevirt != nil:
		creatorGetter = kubevirtDeploymentReconciler(data)

	case data.Cluster().Spec.Cloud.Digitalocean != nil:
		creatorGetter = digitalOceanDeploymentReconciler(data)
	}

	if creatorGetter != nil {
		return func() (name string, create reconciling.DeploymentReconciler) {
			name, creator := creatorGetter()

			return name, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
				dep.Spec.Template.Spec.InitContainers = []corev1.Container{}

				modified, err := creator(dep)
				if err != nil {
					return nil, err
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

				wrappedPodSpec, err := apiserver.IsRunningWrapper(data, modified.Spec.Template.Spec, containerNames)
				if err != nil {
					return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %w", err)
				}
				modified.Spec.Template.Spec = *wrappedPodSpec

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
