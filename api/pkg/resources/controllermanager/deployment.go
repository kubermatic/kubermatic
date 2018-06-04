package controllermanager

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	defaultMemoryRequest = resource.MustParse("100Mi")
	defaultCPURequest    = resource.MustParse("100m")
	defaultMemoryLimit   = resource.MustParse("512Mi")
	defaultCPULimit      = resource.MustParse("250m")
)

const (
	name = "controller-manager"
)

// Deployment returns the kubernetes Controller-Manager Deployment
func Deployment(data *resources.TemplateData, existing *appsv1.Deployment) (*appsv1.Deployment, error) {
	var dep *appsv1.Deployment
	if existing != nil {
		dep = existing
	} else {
		dep = &appsv1.Deployment{}
	}

	dep.Name = resources.ControllerManagerDeploymentName
	dep.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	dep.Labels = resources.GetLabels(name)

	dep.Spec.Replicas = resources.Int32(1)
	dep.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			resources.AppLabelKey: name,
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

	podLabels, err := getTemplatePodLabels(data)
	if err != nil {
		return nil, err
	}

	dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Labels: podLabels,
		Annotations: map[string]string{
			"prometheus.io/scrape": "true",
			"prometheus.io/path":   "/metrics",
			"prometheus.io/port":   "10252",
		},
	}

	dep.Spec.Template.Spec.Volumes = getVolumes()
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
			Name:            name,
			Image:           data.ImageRegistry("gcr.io") + "/google_containers/hyperkube-amd64:v" + data.Cluster.Spec.Version,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/hyperkube", "controller-manager"},
			Args:            getFlags(data),
			Env:             getEnvVars(data),
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
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.FromInt(10252),
					},
				},
				FailureThreshold: 3,
				PeriodSeconds:    10,
				SuccessThreshold: 1,
				TimeoutSeconds:   15,
			},
			LivenessProbe: &corev1.Probe{
				FailureThreshold: 8,
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.FromInt(10252),
					},
				},
				InitialDelaySeconds: 15,
				PeriodSeconds:       10,
				SuccessThreshold:    1,
				TimeoutSeconds:      15,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      resources.CACertSecretName,
					MountPath: "/etc/kubernetes/ca-cert",
					ReadOnly:  true,
				},
				{
					Name:      resources.CAKeySecretName,
					MountPath: "/etc/kubernetes/ca-key",
					ReadOnly:  true,
				},
				{
					Name:      resources.ServiceAccountKeySecretName,
					MountPath: "/etc/kubernetes/service-account-key",
					ReadOnly:  true,
				},
				{
					Name:      resources.CloudConfigConfigMapName,
					MountPath: "/etc/kubernetes/cloud",
					ReadOnly:  true,
				},
			},
		},
	}

	return dep, nil
}

func getFlags(data *resources.TemplateData) []string {
	flags := []string{
		"--master", "http://apiserver:8080",
		"--service-account-private-key-file", "/etc/kubernetes/service-account-key/sa.key",
		"--root-ca-file", "/etc/kubernetes/ca-cert/ca.crt",
		"--cluster-signing-cert-file", "/etc/kubernetes/ca-cert/ca.crt",
		"--cluster-signing-key-file", "/etc/kubernetes/ca-key/ca.key",
		"--cluster-cidr", data.Cluster.Spec.ClusterNetwork.Pods.CIDRBlocks[0],
		"--configure-cloud-routes", "false",
		"--allocate-node-cidrs", "true",
		"--controllers", "*,bootstrapsigner,tokencleaner",
		"--v", "4",
	}
	if data.Cluster.Spec.Cloud.AWS != nil {
		flags = append(flags, "--cloud-provider", "aws")
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")
	}
	if data.Cluster.Spec.Cloud.Openstack != nil {
		flags = append(flags, "--cloud-provider", "openstack")
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")
	}
	return flags
}

func getTemplatePodLabels(data *resources.TemplateData) (map[string]string, error) {
	podLabels := map[string]string{
		resources.AppLabelKey: name,
	}

	secretDependencies := []string{
		resources.CACertSecretName,
		resources.CAKeySecretName,
		resources.ServiceAccountKeySecretName,
	}
	for _, name := range secretDependencies {
		revision, err := data.SecretRevision(name)
		if err != nil {
			return nil, err
		}
		podLabels[fmt.Sprintf("%s-secret-revision", name)] = revision
	}

	cloudConfigRevision, err := data.ConfigMapRevision(resources.CloudConfigConfigMapName)
	if err != nil {
		return nil, err
	}
	podLabels[fmt.Sprintf("%s-configmap-revision", resources.CloudConfigConfigMapName)] = cloudConfigRevision

	return podLabels, nil
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.CACertSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.CACertSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.CAKeySecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.CAKeySecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.ServiceAccountKeySecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.ServiceAccountKeySecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.CloudConfigConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.CloudConfigConfigMapName,
					},
					DefaultMode: resources.Int32(420),
				},
			},
		},
	}
}

func getEnvVars(data *resources.TemplateData) []corev1.EnvVar {
	var vars []corev1.EnvVar
	if data.Cluster.Spec.Cloud.AWS != nil {
		vars = append(vars, corev1.EnvVar{Name: "AWS_ACCESS_KEY_ID", Value: data.Cluster.Spec.Cloud.AWS.AccessKeyID})
		vars = append(vars, corev1.EnvVar{Name: "AWS_SECRET_ACCESS_KEY", Value: data.Cluster.Spec.Cloud.AWS.SecretAccessKey})
		vars = append(vars, corev1.EnvVar{Name: "AWS_VPC_ID", Value: data.Cluster.Spec.Cloud.AWS.VPCID})
		vars = append(vars, corev1.EnvVar{Name: "AWS_AVAILABILITY_ZONE", Value: data.Cluster.Spec.Cloud.AWS.AvailabilityZone})
	}
	return vars
}
