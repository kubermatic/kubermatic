package prometheus

import (
	"fmt"
	"path"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	Name = "prometheus"

	defaultImage = "quay.io/prometheus/prometheus"

	volumeConfigName = "config"
	volumeDataName   = "data"
)

var defaultMemoryRequest = resource.MustParse("200Mi")

func StatefulSet(data *resources.Data) (*appsv1.StatefulSet, error) {
	cm, err := data.ConfigMapLister.ConfigMaps(data.Cluster.Status.NamespaceName).Get(Name)
	if err != nil {
		return nil, err
	}

	image := defaultImage
	if data.ImageRepository != "" {
		image = path.Join(data.ImageRepository, "prometheus/prometheus")
	}
	image = fmt.Sprintf("%s:v2.2.1", image)

	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            Name,
			Labels:          map[string]string{},
			Annotations:     map[string]string{},
			OwnerReferences: []metav1.OwnerReference{data.GetClusterRef()},
		},
		Spec: appsv1.StatefulSetSpec{
			UpdateStrategy: appsv1.StatefulSetUpdateStrategy{
				Type: appsv1.RollingUpdateStatefulSetStrategyType,
			},
			Replicas:    resources.Int32(1),
			ServiceName: Name,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":     Name,
					"cluster": data.Cluster.Name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":             Name,
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
					ServiceAccountName:            Name,
					TerminationGracePeriodSeconds: resources.Int64(600),
					Containers: []corev1.Container{
						{
							Name:  Name,
							Image: image,
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
										Name: Name,
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
