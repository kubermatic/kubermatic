package cloudcontroller

import (
	"errors"

	"github.com/kubermatic/kubermatic/pkg/resources"
	"github.com/kubermatic/kubermatic/pkg/resources/reconciling"
	appsv1 "k8s.io/api/apps/v1"
)

// DeploymentCreator returns the function to create and update the external cloud provider deployment.
func DeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	if data.Cluster().Spec.Cloud.Openstack != nil {
		return openStackDeploymentCreator(data)
	}

	return func() (name string, create reconciling.DeploymentCreator) {
		return osName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			return nil, errors.New("unsupported external cloud controller")
		}
	}
}
