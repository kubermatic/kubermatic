package usercluster

import (
	"fmt"
	"net/url"
	"strings"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"

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
			corev1.ResourceCPU:    resource.MustParse("25m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("512Mi"),
			corev1.ResourceCPU:    resource.MustParse("500m"),
		},
	}
)

const (
	name = "usercluster-controller"
)

// userclusterControllerData is the subet of the deploymentData interface
// that is actually required by the usercluster deployment
// This makes importing the the deployment elsewhere (openshift controller)
// easier as only have to implement the parts that are actually in use
type userclusterControllerData interface {
	GetPodTemplateLabels(string, []corev1.Volume, map[string]string) (map[string]string, error)
	ImageRegistry(string) string
	InClusterApiserverURL() (*url.URL, error)
	Cluster() *kubermaticv1.Cluster
}

// DeploymentCreator returns the function to create and update the user cluster controller deployment
func DeploymentCreator(data userclusterControllerData) resources.DeploymentCreator {
	return func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
		dep.Name = resources.UserClusterControllerDeploymentName
		dep.Labels = resources.BaseAppLabel(name, nil)

		dep.Spec.Replicas = resources.Int32(1)
		dep.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: resources.BaseAppLabel(name, nil),
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
		dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

		volumes := getVolumes()
		podLabels, err := data.GetPodTemplateLabels(name, volumes, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create pod labels: %v", err)
		}

		dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
			Labels: podLabels,
			Annotations: map[string]string{
				"prometheus.io/scrape": "true",
				"prometheus.io/path":   "/metrics",
				"prometheus.io/port":   "8085",
			},
		}

		dep.Spec.Template.Spec.Volumes = volumes

		apiserverIsRunningContainer, err := apiserver.IsRunningInitContainer(data)
		if err != nil {
			return nil, err
		}
		dep.Spec.Template.Spec.InitContainers = []corev1.Container{*apiserverIsRunningContainer}

		dep.Spec.Template.Spec.Containers = []corev1.Container{
			{
				Name:            name,
				Image:           data.ImageRegistry(resources.RegistryQuay) + "/kubermatic/api:" + resources.KUBERMATICCOMMIT,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Command:         []string{"/usr/local/bin/user-cluster-controller-manager"},
				Args: append([]string{
					"-kubeconfig", "/etc/kubernetes/kubeconfig/kubeconfig",
					"-internal-address", "0.0.0.0:8085",
					"-logtostderr",
					"-v", "4"}, getNetworkArgs(data)...),
				TerminationMessagePath:   corev1.TerminationMessagePathDefault,
				TerminationMessagePolicy: corev1.TerminationMessageReadFile,
				Resources:                defaultResourceRequirements,
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      resources.InternalUserClusterAdminKubeconfigSecretName,
						MountPath: "/etc/kubernetes/kubeconfig",
						ReadOnly:  true,
					},
				},
			},
		}

		return dep, nil
	}
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.InternalUserClusterAdminKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.InternalUserClusterAdminKubeconfigSecretName,
					// We have to make the secret readable for all for now because owner/group cannot be changed.
					// ( upstream proposal: https://github.com/kubernetes/kubernetes/pull/28733 )
					DefaultMode: resources.Int32(resources.DefaultAllReadOnlyMode),
				},
			},
		},
	}
}

func getNetworkArgs(data userclusterControllerData) []string {
	networkFlags := make([]string, len(data.Cluster().Spec.MachineNetworks)*2)
	i := 0

	for _, n := range data.Cluster().Spec.MachineNetworks {
		networkFlags[i] = "--ipam-controller-network"
		i++
		networkFlags[i] = fmt.Sprintf("%s,%s,%s", n.CIDR, n.Gateway, strings.Join(n.DNSServers, ","))
		i++
	}

	return networkFlags
}
