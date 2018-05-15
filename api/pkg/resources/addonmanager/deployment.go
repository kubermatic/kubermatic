package addonmanager

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	defaultMemoryRequest = resource.MustParse("20Mi")
	defaultCPURequest    = resource.MustParse("50m")
	defaultMemoryLimit   = resource.MustParse("64Mi")
	defaultCPULimit      = resource.MustParse("100m")
)

// Deployment returns the addon-manager Deployment
func Deployment(data *resources.TemplateData, existing *appsv1.Deployment) (*appsv1.Deployment, error) {
	var dep *appsv1.Deployment
	if existing != nil {
		dep = existing
	} else {
		dep = &appsv1.Deployment{}
	}

	dep.Name = resources.AddonManagerDeploymentName
	dep.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

	dep.Spec.Replicas = resources.Int32(1)
	dep.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			"role": "addon-manager",
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
			"role":    "addon-manager",
			"release": data.Version.Values["addon-manager-version"],
		},
	}

	dep.Spec.Template.Spec.InitContainers = []corev1.Container{
		{
			Name:            "apiserver-running",
			Image:           data.ImageRegistry("docker.io") + "/busybox",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command: []string{
				"/bin/sh",
				"-ec",
				"until wget -T 1 http://apiserver:8080/healthz; do echo waiting for apiserver; sleep 2; done;",
			},
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		},
	}

	dep.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:            "addon-manager",
			Image:           data.ImageRegistry("docker.io") + "/kubermatic/addon-manager:" + data.Version.Values["addon-manager-version"],
			ImagePullPolicy: corev1.PullIfNotPresent,
			Env: []corev1.EnvVar{
				{
					Name:  "KUBECTL_OPTS",
					Value: "--server=http://apiserver:8080",
				},
				{
					Name:  "ADDON_MANAGER_LEADER_ELECTION",
					Value: "false",
				},
			},
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: defaultMemoryRequest,
					corev1.ResourceCPU:    defaultCPURequest,
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: defaultMemoryLimit,
					corev1.ResourceCPU:    defaultCPULimit,
				},
			},
		},
	}

	return dep, nil
}
