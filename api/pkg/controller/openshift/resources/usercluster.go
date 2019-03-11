package resources

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/usercluster"

	appsv1 "k8s.io/api/apps/v1"
)

func UserClusterController(osData openshiftData) resources.NamedDeploymentCreatorGetter {

	name, creator := usercluster.DeploymentCreator(osData)()
	return func() (string, resources.DeploymentCreator) {
		return name, addContainerArg(creator, "usercluster-controller", "-openshift", "true")
	}
}

func addContainerArg(creator resources.DeploymentCreator, containerName string, arg ...string) resources.DeploymentCreator {
	return func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
		deployment, err := creator(d)
		if err != nil {
			return nil, err
		}

		var wasFound bool
		for idx := range deployment.Spec.Template.Spec.Containers {
			if deployment.Spec.Template.Spec.Containers[idx].Name == containerName {
				wasFound = true
				deployment.Spec.Template.Spec.Containers[idx].Args = append(deployment.Spec.Template.Spec.Containers[idx].Args, arg...)
			}
		}
		if !wasFound {
			return nil, fmt.Errorf("container %s was not found", containerName)
		}
		return deployment, nil
	}
}
