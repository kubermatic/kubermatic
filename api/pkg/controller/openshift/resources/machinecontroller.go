package resources

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
)

const machineControllerImage = "quay.io/kubermatic/machine-controller-private:8936af2a674563e8350ee2084546d40b71c665ea-dirty"

func MachineController(osData openshiftData) reconciling.NamedDeploymentCreatorGetter {
	name, creator := machinecontroller.DeploymentCreator(osData)()
	return func() (string, reconciling.DeploymentCreator) {
		return name, deploymentImageAddingWrapper(creator, "machine-controller", machineControllerImage)
	}
}

func MachineControllerWebhook(osData openshiftData) reconciling.NamedDeploymentCreatorGetter {

	name, creator := machinecontroller.WebhookDeploymentCreator(osData)()
	return func() (string, reconciling.DeploymentCreator) {
		return name, deploymentImageAddingWrapper(creator, "machine-controller", machineControllerImage)
	}
}

func deploymentImageAddingWrapper(creator reconciling.DeploymentCreator, containerName, image string) reconciling.DeploymentCreator {

	return func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
		d, err := creator(d)
		if err != nil {
			return nil, err
		}

		var containerWasFound bool
		for idx := range d.Spec.Template.Spec.Containers {
			if d.Spec.Template.Spec.Containers[idx].Name == containerName {
				d.Spec.Template.Spec.Containers[idx].Image = image
				containerWasFound = true
				break
			}
		}

		if !containerWasFound {
			return nil, fmt.Errorf("couldn't find a container with name %s", containerName)
		}
		return d, nil
	}

}
