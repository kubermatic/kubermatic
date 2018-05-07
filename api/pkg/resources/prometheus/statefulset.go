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

var defaultMemoryRequest = resource.MustParse("200Mi")

// StatefulSet returns the prometheus statefulset
func StatefulSet(data *resources.TemplateData) (*appsv1.StatefulSet, error) {
	cm, err := data.ConfigMapLister.ConfigMaps(data.Cluster.Status.NamespaceName).Get(name)
	if err != nil {
		return nil, err
	}

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Labels:          map[string]string{},
			Annotations:     map[string]string{},
			OwnerReferences: []metav1.OwnerReference{data.GetClusterRef()},
		},
		Spec: appsv1.StatefulSetSpec{
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			Replicas:    resources.Int32(1),
			ServiceName: name,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":     name,
					"cluster": data.Cluster.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":             name,
						"cluster":         data.Cluster.Name,
						"config-revision": cm.ObjectMeta.ResourceVersion,
					},
				},
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyAlways,
					SecurityContext: &corev1.PodSecurityContext{
						FSGroup:      resources.Int64(2000),
						RunAsNonRoot: resources.Bool(true),
						RunAsUser:    resources.Int64(1000),
					},
					ServiceAccountName:            name,
					TerminationGracePeriodSeconds: resources.Int64(600),
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: "quay.io/prometheus/prometheus:v2.2.1",
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
									corev1.ResourceMemory: defaultMemoryRequest,
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
							},
							LivenessProbe: &corev1.Probe{
								PeriodSeconds:       5,
								TimeoutSeconds:      3,
								FailureThreshold:    10,
								InitialDelaySeconds: 30,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/-/healthy",
										Port: intstr.FromString("web"),
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								PeriodSeconds:    5,
								TimeoutSeconds:   3,
								FailureThreshold: 6,
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/-/healthy",
										Port: intstr.FromString("web"),
									},
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: volumeConfigName,
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: name,
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
					},
				},
			},
		},
	}, nil
}
