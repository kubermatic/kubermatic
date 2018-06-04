package openvpn

import (
	"fmt"
	"net"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	name = "openvpn-server"

	defaultInitMemoryRequest = resource.MustParse("128Mi")
	defaultInitCPURequest    = resource.MustParse("250m")
	defaultInitMemoryLimit   = resource.MustParse("512Mi")
	defaultInitCPULimit      = resource.MustParse("500m")

	defaultMemoryRequest = resource.MustParse("64Mi")
	defaultCPURequest    = resource.MustParse("25m")
	defaultMemoryLimit   = resource.MustParse("256Mi")
	defaultCPULimit      = resource.MustParse("250m")
)

// Deployment returns the kubernetes Controller-Manager Deployment
func Deployment(data *resources.TemplateData, existing *appsv1.Deployment) (*appsv1.Deployment, error) {
	var dep *appsv1.Deployment
	if existing != nil {
		dep = existing
	} else {
		dep = &appsv1.Deployment{}
	}

	dep.Name = resources.OpenVPNServerDeploymentName
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
			IntVal: 1,
		},
	}

	podLabels, err := getTemplatePodLabels(data)
	if err != nil {
		return nil, err
	}

	dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Labels: podLabels,
	}

	podNetIP, podNet, err := net.ParseCIDR(data.Cluster.Spec.ClusterNetwork.Pods.CIDRBlocks[0])
	if err != nil {
		return nil, err
	}

	serviceNetIP, serviceNet, err := net.ParseCIDR(data.Cluster.Spec.ClusterNetwork.Services.CIDRBlocks[0])
	if err != nil {
		return nil, err
	}

	dep.Spec.Template.Spec.Volumes = getVolumes()
	dep.Spec.Template.Spec.InitContainers = []corev1.Container{
		{
			Name:            "openssl-dhparam",
			Image:           data.ImageRegistry("docker.io") + "/kubermatic/openvpn:v0.2",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/usr/bin/openssl"},
			Args: []string{
				"dhparam",
				"-out", "/etc/openvpn/dh/dh2048.pem",
				"2048",
			},
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: defaultInitMemoryRequest,
					corev1.ResourceCPU:    defaultInitCPURequest,
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: defaultInitMemoryLimit,
					corev1.ResourceCPU:    defaultInitCPULimit,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "diffie-hellman-params",
					MountPath: "/etc/openvpn/dh",
				},
			},
		},
	}
	dep.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:            name,
			Image:           data.ImageRegistry("docker.io") + "/kubermatic/openvpn:v0.2",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/usr/sbin/openvpn"},
			Args: []string{
				"--proto", "tcp",
				"--dev", "tun",
				"--mode", "server",
				"--lport", "1194",
				"--server", "10.20.0.0", "255.255.255.0",
				"--ca", "/etc/kubernetes/ca-cert/ca.crt",
				"--cert", "/etc/openvpn/certs/server.crt",
				"--key", "/etc/openvpn/certs/server.key",
				"--dh", "/etc/openvpn/dh/dh2048.pem",
				"--duplicate-cn",
				"--route", podNetIP.String(), net.IP(podNet.Mask).String(),
				"--route", serviceNetIP.String(), net.IP(serviceNet.Mask).String(),
				"--push", fmt.Sprintf("route %s %s", podNetIP.String(), net.IP(podNet.Mask).String()),
				"--push", fmt.Sprintf("route %s %s", serviceNetIP.String(), net.IP(serviceNet.Mask).String()),
				"--client-to-client",
				"--client-config-dir", "/etc/openvpn/clients",
				"--link-mtu", "1432",
				"--cipher", "AES-256-GCM",
				"--auth", "SHA1",
				"--keysize", "256",
				"--script-security", "2",
				"--up", "/bin/touch /tmp/running",
				"--ping", "5",
				"--verb", "3",
				"--log", "/dev/stdout",
			},
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: 1194,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			SecurityContext: &corev1.SecurityContext{
				Privileged: resources.Bool(true),
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
					Exec: &corev1.ExecAction{
						Command: []string{
							"cat",
							"/tmp/running",
						},
					},
				},
				FailureThreshold:    3,
				InitialDelaySeconds: 5,
				PeriodSeconds:       5,
				SuccessThreshold:    1,
				TimeoutSeconds:      1,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "diffie-hellman-params",
					MountPath: "/etc/openvpn/dh",
					ReadOnly:  true,
				},
				{
					Name:      resources.OpenVPNServerCertificatesSecretName,
					MountPath: "/etc/openvpn/certs",
					ReadOnly:  true,
				},
				{
					Name:      resources.CACertSecretName,
					MountPath: "/etc/kubernetes/ca-cert",
					ReadOnly:  true,
				},
				{
					Name:      resources.OpenVPNClientConfigConfigMapName,
					MountPath: "/etc/openvpn/clients",
					ReadOnly:  true,
				},
			},
		},
	}

	return dep, nil
}

func getTemplatePodLabels(data *resources.TemplateData) (map[string]string, error) {
	podLabels := map[string]string{
		resources.AppLabelKey: name,
	}

	cloudConfigRevision, err := data.ConfigMapRevision(resources.CloudConfigConfigMapName)
	if err != nil {
		return nil, err
	}
	podLabels[fmt.Sprintf("%s-configmap-revision", resources.CloudConfigConfigMapName)] = cloudConfigRevision

	caCertSecretRevision, err := data.SecretRevision(resources.CACertSecretName)
	if err != nil {
		return nil, err
	}
	podLabels[fmt.Sprintf("%s-secret-revision", resources.CACertSecretName)] = caCertSecretRevision

	return podLabels, nil
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: "diffie-hellman-params",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
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
			Name: resources.OpenVPNServerCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.OpenVPNServerCertificatesSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.OpenVPNClientConfigConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.OpenVPNClientConfigConfigMapName,
					},
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
	}
}
