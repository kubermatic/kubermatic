package prometheus

import (
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

	cm, err := data.ConfigMapLister.ConfigMaps(data.Cluster.Status.NamespaceName).Get(resources.PrometheusConfigConfigMapName)
	if err != nil {
		return nil, err
	}

	set.Name = resources.PrometheusStatefulSetName
	set.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	set.Labels = resources.GetLabels(name)

	set.Spec.Replicas = resources.Int32(1)
	set.Spec.UpdateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
	set.Spec.ServiceName = resources.PrometheusServiceName
	set.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			resources.AppLabelKey: name,
			"cluster":             data.Cluster.Name,
		},
	}

	set.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Labels: map[string]string{
			resources.AppLabelKey: name,
			"cluster":             data.Cluster.Name,
			"config-revision":     cm.ObjectMeta.ResourceVersion,
		},
	}
	set.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyAlways
	set.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
		FSGroup:      resources.Int64(2000),
		RunAsNonRoot: resources.Bool(true),
		RunAsUser:    resources.Int64(1000),
	}
	set.Spec.Template.Spec.ServiceAccountName = resources.PrometheusServiceAccountName
	set.Spec.Template.Spec.TerminationGracePeriodSeconds = resources.Int64(600)

	// Checking if its already set. Otherwise initialize it with 1.
	// Overriding the whole container slice would reset existing defaults, causing a patch.
	if len(set.Spec.Template.Spec.Containers) == 0 {
		set.Spec.Template.Spec.Containers = make([]corev1.Container, 1)
	}
	set.Spec.Template.Spec.Containers[0].Name = name
	set.Spec.Template.Spec.Containers[0].Image = data.ImageRegistry(resources.RegistryQuay) + "/prometheus/prometheus:v2.2.0"
	set.Spec.Template.Spec.Containers[0].Args = []string{
		"--config.file=/etc/prometheus/config/prometheus.yaml",
		"--storage.tsdb.path=/var/prometheus/data",
		"--storage.tsdb.retention=12h",
		"--web.enable-lifecycle",
		"--storage.tsdb.no-lockfile",
		"--web.route-prefix=/",
	}
	set.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
		{
			Name:          "web",
			ContainerPort: 9090,
			Protocol:      corev1.ProtocolTCP,
		},
	}
	set.Spec.Template.Spec.Containers[0].Resources = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    defaultCPURequest,
			corev1.ResourceMemory: defaultMemoryRequest,
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    defaultCPULimit,
			corev1.ResourceMemory: defaultMemoryLimit,
		},
	}

	set.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
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
	}

	if set.Spec.Template.Spec.Containers[0].LivenessProbe == nil {
		set.Spec.Template.Spec.Containers[0].LivenessProbe = &corev1.Probe{}
	}
	set.Spec.Template.Spec.Containers[0].LivenessProbe.PeriodSeconds = 5
	set.Spec.Template.Spec.Containers[0].LivenessProbe.TimeoutSeconds = 3
	set.Spec.Template.Spec.Containers[0].LivenessProbe.FailureThreshold = 10
	set.Spec.Template.Spec.Containers[0].LivenessProbe.InitialDelaySeconds = 30
	set.Spec.Template.Spec.Containers[0].LivenessProbe.Handler = corev1.Handler{
		HTTPGet: &corev1.HTTPGetAction{
			Path: "/-/healthy",
			Port: intstr.FromString("web"),
		},
	}

	if set.Spec.Template.Spec.Containers[0].ReadinessProbe == nil {
		set.Spec.Template.Spec.Containers[0].ReadinessProbe = &corev1.Probe{}
	}
	set.Spec.Template.Spec.Containers[0].ReadinessProbe.PeriodSeconds = 5
	set.Spec.Template.Spec.Containers[0].ReadinessProbe.TimeoutSeconds = 3
	set.Spec.Template.Spec.Containers[0].ReadinessProbe.FailureThreshold = 6
	set.Spec.Template.Spec.Containers[0].ReadinessProbe.Handler = corev1.Handler{
		HTTPGet: &corev1.HTTPGetAction{
			Path: "/-/healthy",
			Port: intstr.FromString("web"),
		},
	}

	set.Spec.Template.Spec.Volumes = []corev1.Volume{
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

	return set, nil
}
