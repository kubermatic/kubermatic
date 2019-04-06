package resources

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const machineControllerImage = "quay.io/kubermatic/machine-controller-private:8936af2a674563e8350ee2084546d40b71c665ea-dirty"

func MachineController(osData openshiftData) reconciling.NamedDeploymentCreatorGetter {
	name, creator := machinecontroller.DeploymentCreator(osData)()
	creator = deploymentImageAddingWrapper(creator, "machine-controller", machineControllerImage)
	return func() (string, reconciling.DeploymentCreator) {
		return name, func(in *appsv1.Deployment) (*appsv1.Deployment, error) {
			d, err := creator(in)
			if err != nil {
				return nil, err
			}
			d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, corev1.Volume{Name: "userdata-plugins",
				VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}})
			d.Spec.Template.Spec.InitContainers = append(d.Spec.Template.Spec.InitContainers, corev1.Container{
				Name:  "copy-userdata-plugin",
				Image: osData.ImageRegistry(resources.RegistryQuay) + "/kubermatic/api:" + resources.KUBERMATICCOMMIT,
				Command: []string{
					"/bin/sh",
					"-c",
					"set -e && cp /usr/local/bin/userdata-openshift /target/machine-controller-userdata-centos",
				},
				VolumeMounts: []corev1.VolumeMount{{Name: "userdata-plugins", MountPath: "/target"}},
			})
			for idx := range d.Spec.Template.Spec.Containers {
				if d.Spec.Template.Spec.Containers[idx].Name != "machine-controller" {
					continue
				}
				d.Spec.Template.Spec.Containers[idx].VolumeMounts = append(d.Spec.Template.Spec.Containers[idx].VolumeMounts, corev1.VolumeMount{
					Name: "userdata-plugins", MountPath: "/userdata-plugins"})
				d.Spec.Template.Spec.Containers[idx].Env = append(d.Spec.Template.Spec.Containers[idx].Env, corev1.EnvVar{Name: "USERDATA_PLUGIN_DIR", Value: "/userdata-plugins"})
			}
			return d, nil
		}
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
