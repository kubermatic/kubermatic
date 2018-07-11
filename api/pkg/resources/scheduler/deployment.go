package scheduler

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
	defaultMemoryRequest = resource.MustParse("64Mi")
	defaultCPURequest    = resource.MustParse("20m")
	defaultMemoryLimit   = resource.MustParse("128Mi")
	defaultCPULimit      = resource.MustParse("100m")
)

const (
	name = "scheduler"
)

// Deployment returns the kubernetes Controller-Manager Deployment
func Deployment(data *resources.TemplateData, existing *appsv1.Deployment) (*appsv1.Deployment, error) {
	var dep *appsv1.Deployment
	if existing != nil {
		dep = existing
	} else {
		dep = &appsv1.Deployment{}
	}

	dep.Name = resources.SchedulerDeploymentName
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

	dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Labels: map[string]string{
			resources.AppLabelKey: name,
		},
		Annotations: map[string]string{
			"prometheus.io/scrape": "true",
			"prometheus.io/path":   "/metrics",
			"prometheus.io/port":   "10251",
		},
	}

	// get openvpn sidecar container and apiserverServiceIP
	apiIP, apiPort, err := data.ClusterIPPortByServiceName(resources.ApiserverExternalServiceName)
	if err != nil {
		return nil, err
	}
	openvpnSidecar, err := resources.OpenVPNSidecarContainer(data, "openvpn-client")
	if err != nil {
		return nil, fmt.Errorf("failed to get openvpn sidecar: %v", err)
	}

	// Configure user cluster DNS resolver for this pod.
	dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data.Cluster)
	if err != nil {
		return nil, err
	}

	kcDir := "/etc/kubernetes/scheduler"
	dep.Spec.Template.Spec.Volumes = getVolumes()
	dep.Spec.Template.Spec.InitContainers = []corev1.Container{
		{
			Name:            "apiserver-running",
			Image:           data.ImageRegistry(resources.RegistryDocker) + "/busybox",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command: []string{
				"/bin/sh",
				"-ec",
				fmt.Sprintf("until wget -O - -T 1 https://%s:%d/healthz; do echo waiting for apiserver; sleep 2; done", apiIP, apiPort),
				// * unfortunately no curl in busybox image
				// * "fortunately" busybox wget does not care about TLS verification (neither peername, nor ca)
				// * might still be enough for only waiting for `apiserver-running`
				//
				// curl could do a nice trick with --resolve, which eases handing in the peername
				// but still connect to a different ip:
				// curl --resolve kubernetes:31834:10.47.248.241 --cacert /ca.crt --cert /1.crt --key /1.key  -i https://kubernetes:31834/healthz
				// (but busybox image has no curl by default)
			},
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		},
	}
	dep.Spec.Template.Spec.Containers = []corev1.Container{
		*openvpnSidecar,
		{
			Name:            name,
			Image:           data.ImageRegistry(resources.RegistryKubernetesGCR) + "/google_containers/hyperkube-amd64:v" + data.Cluster.Spec.Version,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/hyperkube", "scheduler"},
			Args: []string{
				"--kubeconfig", fmt.Sprintf("%s/%s", kcDir, resources.SchedulerKubeconfigSecretName),
				"--v", "4",
			},
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      resources.SchedulerKubeconfigSecretName,
					MountPath: kcDir,
					ReadOnly:  true,
				},
			},
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
						Port: intstr.FromInt(10251),
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
						Port: intstr.FromInt(10251),
					},
				},
				InitialDelaySeconds: 15,
				PeriodSeconds:       10,
				SuccessThreshold:    1,
				TimeoutSeconds:      15,
			},
		},
	}

	return dep, nil
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.OpenVPNClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.OpenVPNClientCertificatesSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.CACertSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.CACertSecretName,
				},
			},
		},
		{
			Name: resources.SchedulerKubeconfigSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.SchedulerKubeconfigSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
	}
}
