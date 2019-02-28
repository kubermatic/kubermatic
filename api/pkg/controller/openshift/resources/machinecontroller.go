package resources

import (
	"context"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroller"

	appsv1 "k8s.io/api/apps/v1"
)

const machineControllerImage = "quay.io/kubermatic/machine-controller-private:ff4cdb756d8710cddccd87325bf66f0217ceac7c-dirty"

func MachineController(_ context.Context, osData openshiftData) (string, resources.DeploymentCreator) {
	creator := machinecontroller.DeploymentCreator(osData)
	creator = deploymentImageAddingWrapper(creator, "machine-controller", machineControllerImage)
	return resources.MachineControllerDeploymentName, creator
}

func MachineControllerWebhook(_ context.Context, osData openshiftData) (string, resources.DeploymentCreator) {

	creator := machinecontroller.WebhookDeploymentCreator(osData)
	creator = deploymentImageAddingWrapper(creator, "machine-controller", machineControllerImage)
	return resources.MachineControllerWebhookDeploymentName, creator
}

func deploymentImageAddingWrapper(creator resources.DeploymentCreator, containerName, image string) resources.DeploymentCreator {

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
