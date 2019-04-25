package resources

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/machinecontroller"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func MachineController(osData openshiftData) reconciling.NamedDeploymentCreatorGetter {
	name, creator := machinecontroller.DeploymentCreator(osData)()

	// We do two things here:
	// * Add a kubermatic-api initcontainer that copies the openshift-userdata binary
	//   into a shared volume and configure the machine-controller via env var to use that for CentOS
	// * Append to the machine-controller cmd so it uses a service account token instead of bootstrap
	//   tokens
	return func() (string, reconciling.DeploymentCreator) {
		return name, func(in *appsv1.Deployment) (*appsv1.Deployment, error) {
			d, err := creator(in)
			if err != nil {
				return nil, err
			}
			d.Spec.Template.Spec.Volumes = append(d.Spec.Template.Spec.Volumes, corev1.Volume{Name: "userdata-plugins",
				VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}})
			d.Spec.Template.Spec.InitContainers = append(d.Spec.Template.Spec.InitContainers, corev1.Container{
				Name:            "copy-userdata-plugin",
				Image:           osData.ImageRegistry(resources.RegistryQuay) + "/kubermatic/api:" + resources.KUBERMATICCOMMIT,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command: []string{
					"/bin/sh",
					"-c",
					"set -e && cp /usr/local/bin/userdata-openshift /target/machine-controller-userdata-centos",
				},
				TerminationMessagePath:   corev1.TerminationMessagePathDefault,
				TerminationMessagePolicy: corev1.TerminationMessageReadFile,
				VolumeMounts:             []corev1.VolumeMount{{Name: "userdata-plugins", MountPath: "/target"}},
			})
			for idx := range d.Spec.Template.Spec.Containers {
				if d.Spec.Template.Spec.Containers[idx].Name != "machine-controller" {
					continue
				}
				d.Spec.Template.Spec.Containers[idx].Command = append(d.Spec.Template.Spec.Containers[idx].Command,
					"-bootstrap-token-service-account-name", "openshift-infra/node-bootstrapper")
				d.Spec.Template.Spec.Containers[idx].VolumeMounts = append(d.Spec.Template.Spec.Containers[idx].VolumeMounts, corev1.VolumeMount{
					Name: "userdata-plugins", MountPath: "/userdata-plugins"})
				d.Spec.Template.Spec.Containers[idx].Env = append(d.Spec.Template.Spec.Containers[idx].Env,
					corev1.EnvVar{Name: "MACHINE_CONTROLLER_USERDATA_PLUGIN_DIR", Value: "/userdata-plugins"})
			}
			return d, nil
		}
	}
}
