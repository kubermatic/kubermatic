package prometheus

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	name = "prometheus"

	volumeConfigName = "config"
	volumeDataName   = "data"
)

var (
	defaultCPURequest    = resource.MustParse("50m")
	defaultMemoryRequest = resource.MustParse("128Mi")
	defaultCPULimit      = resource.MustParse("100m")
	defaultMemoryLimit   = resource.MustParse("512Mi")
)

// StatefulSet returns the prometheus StatefulSet
func StatefulSet(data *resources.TemplateData, existing *appsv1.StatefulSet) (*appsv1.StatefulSet, error) {
	var set *appsv1.StatefulSet
	if existing != nil {
		set = existing
	} else {
		set = &appsv1.StatefulSet{}
	}

	set.Name = resources.PrometheusStatefulSetName
	set.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}

	requiredBaseLabels := map[string]string{"cluster": data.Cluster.Name}
	set.Labels = resources.BaseAppLabel(name, requiredBaseLabels)
	set.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: resources.BaseAppLabel(name, requiredBaseLabels),
	}

	set.Spec.Replicas = resources.Int32(1)
	set.Spec.UpdateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
	set.Spec.ServiceName = resources.PrometheusServiceName

	volumes := getVolumes()
	podLabels, err := data.GetPodTemplateLabels(name, volumes, requiredBaseLabels)
	if err != nil {
		return nil, fmt.Errorf("failed to create pod labels: %v", err)
	}

	set.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Labels: podLabels,
	}
	set.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyAlways
	set.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
		FSGroup:      resources.Int64(2000),
		RunAsNonRoot: resources.Bool(true),
		RunAsUser:    resources.Int64(1000),
	}
	set.Spec.Template.Spec.ServiceAccountName = resources.PrometheusServiceAccountName
	set.Spec.Template.Spec.TerminationGracePeriodSeconds = resources.Int64(600)

	set.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:                     name,
			Image:                    data.ImageRegistry(resources.RegistryQuay) + "/prometheus/prometheus:v2.2.0",
			ImagePullPolicy:          corev1.PullIfNotPresent,
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			Args: []string{
				"--config.file=/etc/prometheus/config/prometheus.yaml",
				"--storage.tsdb.path=/var/prometheus/data",
				"--storage.tsdb.retention=12h",
				"--web.enable-lifecycle",
				"--storage.tsdb.no-lockfile",
				"--web.route-prefix=/",
			},
			Ports: []corev1.ContainerPort{
				{
					Name:          "web",
					ContainerPort: 9090,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    defaultCPURequest,
					corev1.ResourceMemory: defaultMemoryRequest,
				},
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    defaultCPULimit,
					corev1.ResourceMemory: defaultMemoryLimit,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      volumeConfigName,
					MountPath: "/etc/prometheus/config",
					ReadOnly:  true,
				},
				{
					Name:      volumeDataName,
					MountPath: "/var/prometheus/data",
				},
				{
					Name:      resources.ApiserverEtcdClientCertificateSecretName,
					MountPath: "/etc/etcd/apiserver",
					ReadOnly:  true,
				},
			},
			LivenessProbe: &corev1.Probe{
				PeriodSeconds:       5,
				TimeoutSeconds:      3,
				FailureThreshold:    10,
				InitialDelaySeconds: 30,
				SuccessThreshold:    1,
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/-/healthy",
						Port: intstr.FromString("web"),
					},
				},
			},
			ReadinessProbe: &corev1.Probe{
				PeriodSeconds:       5,
				TimeoutSeconds:      3,
				FailureThreshold:    6,
				InitialDelaySeconds: 30,
				SuccessThreshold:    1,
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/-/healthy",
						Port: intstr.FromString("web"),
					},
				},
			},
		},
	}

	set.Spec.Template.Spec.Volumes = volumes

	return set, nil
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: volumeConfigName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.PrometheusConfigConfigMapName,
					},
				},
			},
		},
		{
			Name: volumeDataName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: resources.ApiserverEtcdClientCertificateSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.ApiserverEtcdClientCertificateSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
	}
}
