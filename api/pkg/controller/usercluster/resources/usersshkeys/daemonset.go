package usersshkeys

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	daemonSetName = "kubermatic/user-ssh-keys-agent"
)

var (
	daemonSetMaxUnavailable = intstr.FromInt(1)
	hostPathType            = corev1.HostPathUnset
)

type GetImageRegistry func(reg string) string

func DaemonSetCreator(getRegistry GetImageRegistry) reconciling.NamedDaemonSetCreatorGetter {
	return func() (string, reconciling.DaemonSetCreator) {
		return daemonSetName, func(ds *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
			ds.Spec.UpdateStrategy = appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &daemonSetMaxUnavailable,
				},
			}

			ds.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:    daemonSetName,
					Image:   getRegistry(resources.RegistryQuay) + "/" + daemonSetName,
					Command: []string{fmt.Sprintf("/bin/%v", daemonSetName)},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "root",
							MountPath: resources.AuthorizedKeysPath + "root/authorized_keys",
						},
						{
							Name:      "core",
							MountPath: resources.AuthorizedKeysPath + "core/authorized_keys",
						},
						{
							Name:      "centos",
							MountPath: resources.AuthorizedKeysPath + "centos/authorized_keys",
						},
						{
							Name:      "ubuntu",
							MountPath: resources.AuthorizedKeysPath + "ubuntu/authorized_keys",
						},
					},
				},
			}

			ds.Spec.Template.Spec.Tolerations = []corev1.Toleration{
				{
					Effect:   corev1.TaintEffectNoSchedule,
					Operator: corev1.TolerationOpExists,
				},
				{
					Effect:   corev1.TaintEffectNoExecute,
					Operator: corev1.TolerationOpExists,
				},
			}

			ds.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "root",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/root/.ssh/authorized_keys",
							Type: &hostPathType,
						},
					},
				},
				{
					Name: "core",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/core/.ssh/authorized_keys",
							Type: &hostPathType,
						},
					},
				},
				{
					Name: "centos",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/centos/.ssh/authorized_keys",
							Type: &hostPathType,
						},
					},
				},
				{
					Name: "ubuntu",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/ubuntu/.ssh/authorized_keys",
							Type: &hostPathType,
						},
					},
				},
			}

			return ds, nil
		}
	}
}
