package ipamcontroller

import (
	"fmt"
	"strings"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Deployment returns the ipamcontroller deployment
func Deployment(data *resources.TemplateData, existing *appsv1.Deployment) (*appsv1.Deployment, error) {
	var dep *appsv1.Deployment
	if existing != nil {
		dep = existing
	} else {
		dep = &appsv1.Deployment{}
	}

	dep.Name = resources.IPAMControllerDeploymentName
	dep.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	dep.Labels = resources.GetLabels(resources.IPAMControllerDeploymentName)

	dep.Spec.Replicas = resources.Int32(3)
	dep.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			resources.AppLabelKey: resources.IPAMControllerDeploymentName,
		},
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
		"--cidr-range", strings.Join(data.Cluster.Spec.MachineNetwork.CIDRBlocks, ","),
		"--gateway", data.Cluster.Spec.MachineNetwork.Gateway,
		"--dns-servers", strings.Join(data.Cluster.Spec.MachineNetwork.DNSServers, ","),
		"--kubeconfig", fmt.Sprintf("%s/%s", kcDir, resources.IPAMControllerKubeconfigSecretName),
		"--master", "foo",
	}

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
			Name: resources.DockerCfgSecretName,
		},
	}

	dep.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:            resources.IPAMControllerDeploymentName,
			Image:           data.ImageRegistry(resources.RegistryDocker) + "/kubermatic/api:" + resources.KUBERMATICTAG,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/usr/local/bin/ipam-controller"},
			Args:            flags,
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
