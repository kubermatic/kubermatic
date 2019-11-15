package usersshkeys

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	daemonSetName = "user-ssh-keys-agent"
	mainRepoName  = "kubermatic"
	tag           = "latest"
)

var (
	daemonSetMaxUnavailable = intstr.FromInt(1)
	hostPathType            = corev1.HostPathUnset
)

func DaemonSetCreator() reconciling.NamedDaemonSetCreatorGetter {
	return func() (string, reconciling.DaemonSetCreator) {
		return daemonSetName, func(ds *appsv1.DaemonSet) (*appsv1.DaemonSet, error) {
			ds.Spec.UpdateStrategy = appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
				RollingUpdate: &appsv1.RollingUpdateDaemonSet{
					MaxUnavailable: &daemonSetMaxUnavailable,
				},
			}
			labels := map[string]string{"app": "user-ssh-keys-agent"}
			ds.Spec.Selector = &metav1.LabelSelector{MatchLabels: labels}
			ds.Spec.Template.ObjectMeta.Labels = labels

			ds.Spec.Template.Spec.ServiceAccountName = serviceAccountName

			ds.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            daemonSetName,
					ImagePullPolicy: corev1.PullAlways,
					Image:           resources.RegistryQuay + "/" + mainRepoName + "/" + daemonSetName + ":" + tag,
					Command:         []string{fmt.Sprintf("/bin/%v", daemonSetName)},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "root",
							MountPath: "/root/.ssh/authorized_keys",
						},
						{
							Name:      "core",
							MountPath: "/home/core/.ssh/authorized_keys",
						},
						{
							Name:      "centos",
							MountPath: "/home/centos/.ssh/authorized_keys",
						},
						{
							Name:      "ubuntu",
							MountPath: "/home/ubuntu/.ssh/authorized_keys",
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
							Path: "/home/core/.ssh/authorized_keys",
							Type: &hostPathType,
						},
					},
				},
				{
					Name: "centos",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/home/centos/.ssh/authorized_keys",
							Type: &hostPathType,
						},
					},
				},
				{
					Name: "ubuntu",
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: "/home/ubuntu/.ssh/authorized_keys",
							Type: &hostPathType,
						},
					},
				},
			}

			return ds, nil
		}
	}
}
