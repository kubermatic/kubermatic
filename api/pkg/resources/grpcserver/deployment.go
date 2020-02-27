package grpcserver

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"github.com/pkg/errors"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	name    = "grpc-server"
	version = "TODO"
)

// DeploymentCreator returns the function to create and update the grpc-server deployment.
func DeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		dc := func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			labels := resources.BaseAppLabels(name, nil)

			dep.Name = resources.GRPCServerDeploymentName
			dep.Labels = labels

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: labels,
			}

			dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{
				{
					Name: resources.ImagePullSecretName,
				},
			}

			volumes := []corev1.Volume{
				{
					Name: resources.GRPCServerSecretName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.GRPCServerSecretName,
						},
					},
				},
			}

			podLabels, err := data.GetPodTemplateLabels(name, volumes, nil)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create popd labels")
			}

			dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
				Labels: podLabels,
				Annotations: map[string]string{
					// do not specify a port so that Prometheus automatically
					// scrapes both the metrics and the telemetry endpoints
					"prometheus.io/scrape": "true",
				},
			}

			dep.Spec.Template.Spec.Volumes = volumes

			tag := fmt.Sprintf("%s-%s", name, version)
			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:  name,
					Image: data.ImageRegistry(resources.RegistryQuay) + "/kubermatic/grpc-connector:" + tag,
					Args: []string{
						"--ssh-public-key-file", "/etc/grpc-server/ssh.pub",
						"--tls-certificate", "/etc/grpc-server/server.crt",
						"--tls-key", "/etc/grpc-server/server.key",
						"--tls-ca", "/etc/grpc-server/ca.crt",
						"--proxy-port", "22,6443",
						"--kube-api-server-address", fmt.Sprintf("%s.%s.svc.cluster.local", resources.GRPCTunnelServiceName, data.ClusterNamespaceName()),
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      resources.GRPCServerSecretName,
							MountPath: "/etc/grpc-server",
							ReadOnly:  true,
						},
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "metrics",
							ContainerPort: 8080,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "telemetry",
							ContainerPort: 8081,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/healthz",
								Port:   intstr.FromInt(8080),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						FailureThreshold: 3,
						PeriodSeconds:    10,
						SuccessThreshold: 1,
						TimeoutSeconds:   15,
					},
				},
			}

			req := map[string]*corev1.ResourceRequirements{
				name: {
					Requests: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("12Mi"),
						corev1.ResourceCPU:    resource.MustParse("10m"),
					},
					Limits: corev1.ResourceList{
						corev1.ResourceMemory: resource.MustParse("1Gi"),
						corev1.ResourceCPU:    resource.MustParse("100m"),
					},
				},
			}

			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, req, nil, dep.Annotations)
			if err != nil {
				return nil, errors.Wrap(err, "failed to set resource requirements")
			}

			wrappedPodSpec, err := apiserver.IsRunningWrapper(data, dep.Spec.Template.Spec, sets.NewString(name))
			if err != nil {
				return nil, errors.Wrap(err, "failed to add apiserver.IsRunningWrapper")
			}

			dep.Spec.Template.Spec = *wrappedPodSpec

			return dep, nil
		}

		return resources.GRPCServerDeploymentName, dc
	}
}
