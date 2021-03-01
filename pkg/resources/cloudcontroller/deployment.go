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
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	ccmContainerName           = "cloud-controller-manager"
	openvpnClientContainerName = "openvpn-client"
)

// DeploymentCreator returns the function to create and update the external cloud provider deployment.
func DeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	var creatorGetter reconciling.NamedDeploymentCreatorGetter

	switch {
	case data.Cluster().Spec.Cloud.Openstack != nil:
		creatorGetter = openStackDeploymentCreator(data)

	case data.Cluster().Spec.Cloud.Hetzner != nil:
		creatorGetter = hetznerDeploymentCreator(data)
	}

	if creatorGetter != nil {
		return func() (name string, create reconciling.DeploymentCreator) {
			name, creator := creatorGetter()

			return name, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
				dep.Spec.Template.Spec.InitContainers = []corev1.Container{}

				modified, err := creator(dep)
				if err != nil {
					return nil, err
				}

				containerNames := sets.NewString(ccmContainerName, openvpnClientContainerName)

				wrappedPodSpec, err := apiserver.IsRunningWrapper(data, modified.Spec.Template.Spec, containerNames)
				if err != nil {
					return nil, fmt.Errorf("failed to add apiserver.IsRunningWrapper: %v", err)
				}
				modified.Spec.Template.Spec = *wrappedPodSpec

				return modified, nil
			}
		}
	}

	return func() (name string, create reconciling.DeploymentCreator) {
		return "unsupported", func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			return nil, errors.New("unsupported external cloud controller")
		}
	}
}
