package ipamcontroller

import (
	"fmt"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("32Mi"),
			corev1.ResourceCPU:    resource.MustParse("10m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
	}
)

// DeploymentCreator returns the function to create and update the IPAM controller deployment
func DeploymentCreator(data resources.DeploymentDataProvider) resources.DeploymentCreator {
	return func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
		dep.Name = resources.IPAMControllerDeploymentName
		dep.Labels = resources.BaseAppLabel(resources.IPAMControllerDeploymentName, nil)

		dep.Spec.Replicas = resources.Int32(1)
		dep.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: resources.BaseAppLabel(resources.IPAMControllerDeploymentName, nil),
		}
		dep.Spec.Strategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
		dep.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{
			MaxSurge: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 1,
			},
			MaxUnavailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 0,
			},
		}

		dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
			Labels: map[string]string{
				resources.AppLabelKey: resources.IPAMControllerDeploymentName,
			},
		}

		kcDir := "/etc/kubernetes/ipamcontroller"
		flags := []string{
			"--kubeconfig", fmt.Sprintf("%s/kubeconfig", kcDir),
			"-v", "4",
			"-logtostderr",
		}

		flags = append(flags, getNetworkArgs(data)...)

		dep.Spec.Template.Spec.Volumes = []corev1.Volume{
			{
				Name: resources.IPAMControllerKubeconfigSecretName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: resources.IPAMControllerKubeconfigSecretName,
						// We have to make the secret readable for all for now because owner/group cannot be changed.
						// ( upstream proposal: https://github.com/kubernetes/kubernetes/pull/28733 )
						DefaultMode: resources.Int32(resources.DefaultAllReadOnlyMode),
					},
				},
			},
		}

		dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
			{
				Name: resources.ImagePullSecretName,
			},
		}

		dep.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Name:                     resources.IPAMControllerDeploymentName,
				Image:                    data.ImageRegistry(resources.RegistryQuay) + "/kubermatic/api:" + resources.KUBERMATICCOMMIT,
				ImagePullPolicy:          corev1.PullIfNotPresent,
				TerminationMessagePath:   corev1.TerminationMessagePathDefault,
				TerminationMessagePolicy: corev1.TerminationMessageReadFile,
				Command:                  []string{"/usr/local/bin/ipam-controller"},
				Args:                     flags,
				Resources:                defaultResourceRequirements,
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      resources.IPAMControllerKubeconfigSecretName,
						MountPath: kcDir,
						ReadOnly:  true,
					},
				},
			},
		}

		return dep, nil
	}
}

func getNetworkArgs(data resources.DeploymentDataProvider) []string {
	networkFlags := make([]string, len(data.Cluster().Spec.MachineNetworks)*2)
	i := 0

	for _, n := range data.Cluster().Spec.MachineNetworks {
		networkFlags[i] = "--network"
		i++
		networkFlags[i] = fmt.Sprintf("%s,%s,%s", n.CIDR, n.Gateway, strings.Join(n.DNSServers, ","))
		i++
	}

	return networkFlags
}
