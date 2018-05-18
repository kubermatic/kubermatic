package apiserver

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
	defaultApiserverMemoryRequest = resource.MustParse("256Mi")
	defaultApiserverCPURequest    = resource.MustParse("100m")
	defaultApiserverMemoryLimit   = resource.MustParse("1Gi")
	defaultApiserverCPULimit      = resource.MustParse("500m")

	defaultOpenVPNMemoryRequest = resource.MustParse("30Mi")
	defaultOpenVPNCPURequest    = resource.MustParse("10m")
	defaultOpenVPNMemoryLimit   = resource.MustParse("64Mi")
	defaultOpenVPNCPULimit      = resource.MustParse("40m")
)

const (
	name = "apiserver"

	defaultNodePortRange = "30000-32767"
)

// Deployment returns the kubernetes Apiserver Deployment
func Deployment(data *resources.TemplateData, existing *appsv1.Deployment) (*appsv1.Deployment, error) {
	var dep *appsv1.Deployment
	if existing != nil {
		dep = existing
	} else {
		dep = &appsv1.Deployment{}
	}

	dep.Name = resources.ApiserverDeploymenName
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
			"prometheus.io/port":   "8080",
		},
	}

	externalNodePort, err := data.GetApiserverExternalNodePort()
	if err != nil {
		return nil, err
	}

	dep.Spec.Template.Spec.Volumes = getVolumes()
	dep.Spec.Template.Spec.InitContainers = []corev1.Container{
		{
			Name:            "etcd-running",
			Image:           data.ImageRegistry("quay.io") + "/coreos/etcd:v3.2.7",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command: []string{
				"/bin/sh",
				"-ec",
				"until ETCDCTL_API=3 /usr/local/bin/etcdctl --dial-timeout=2s --endpoints=[http://etcd-client:2379] get foo; do echo waiting for etcd; sleep 2; done;",
			},
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		},
	}
	dep.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:            "openvpn-client",
			Image:           data.ImageRegistry("docker.io") + "/kubermatic/openvpn:v0.2",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/usr/sbin/openvpn"},
			Args: []string{
				"--client",
				"--proto", "tcp",
				"--dev", "tun",
				"--auth-nocache",
				"--remote", "openvpn-server", "1194",
				"--nobind",
				"--connect-timeout", "5",
				"--connect-retry", "1",
				"--ca", "/etc/kubernetes/ca-cert/ca.crt",
				"--cert", "/etc/openvpn/certs/client.crt",
				"--key", "/etc/openvpn/certs/client.key",
				"--remote-cert-tls", "server",
				"--link-mtu", "1432",
				"--cipher", "AES-256-GCM",
				"--auth", "SHA1",
				"--keysize", "256",
				"--script-security", "2",
				"--up", "/bin/touch /tmp/running",
				"--log", "/dev/stdout",
			},
			SecurityContext: &corev1.SecurityContext{
				Privileged: resources.Bool(true),
			},
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: defaultOpenVPNMemoryRequest,
					corev1.ResourceCPU:    defaultOpenVPNCPURequest,
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: defaultOpenVPNMemoryLimit,
					corev1.ResourceCPU:    defaultOpenVPNCPULimit,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					MountPath: "/etc/openvpn/certs",
					Name:      resources.OpenVPNClientCertificatesSecretName,
					ReadOnly:  true,
				},
				{
					MountPath: "/etc/kubernetes/ca-cert",
					Name:      resources.CACertSecretName,
					ReadOnly:  true,
				},
			},
		},
		{
			Name:            name,
			Image:           data.ImageRegistry("gcr.io") + "/google_containers/hyperkube-amd64:" + data.Version.Values["k8s-version"],
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/hyperkube", "apiserver"},
			Args:            getApiserverFlags(data, externalNodePort),
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: defaultApiserverMemoryRequest,
					corev1.ResourceCPU:    defaultApiserverCPURequest,
				},
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: defaultApiserverMemoryLimit,
					corev1.ResourceCPU:    defaultApiserverCPULimit,
				},
			},
			Ports: []corev1.ContainerPort{
				{
					ContainerPort: externalNodePort,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					ContainerPort: 8080,
					Protocol:      corev1.ProtocolTCP,
				},
			},
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.FromInt(8080),
					},
				},
				FailureThreshold: 3,
				PeriodSeconds:    5,
				SuccessThreshold: 1,
				TimeoutSeconds:   15,
			},
			LivenessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: "/healthz",
						Port: intstr.FromInt(8080),
					},
				},
				InitialDelaySeconds: 15,
				FailureThreshold:    8,
				PeriodSeconds:       10,
				SuccessThreshold:    1,
				TimeoutSeconds:      15,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					MountPath: "/etc/kubernetes/tls",
					Name:      resources.ApiserverTLSSecretName,
					ReadOnly:  true,
				},
				{
					Name:      resources.TokensSecretName,
					MountPath: "/etc/kubernetes/tokens",
					ReadOnly:  true,
				},
				{
					Name:      resources.KubeletClientCertificatesSecretName,
					MountPath: "/etc/kubernetes/kubelet",
					ReadOnly:  true,
				},
				{
					Name:      resources.CACertSecretName,
					MountPath: "/etc/kubernetes/ca-cert",
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

func getApiserverFlags(data *resources.TemplateData, externalNodePort int32) []string {
	nodePortRange := data.NodePortRange
	if nodePortRange == "" {
		nodePortRange = defaultNodePortRange
	}

	flags := []string{
		"--advertise-address", data.Cluster.Address.IP,
		"--secure-port", fmt.Sprintf("%d", externalNodePort),
		"--kubernetes-service-node-port", fmt.Sprintf("%d", externalNodePort),
		"--insecure-bind-address", "0.0.0.0",
		"--insecure-port", "8080",
		"--etcd-servers", "http://etcd-client:2379",
		"--storage-backend", "etcd3",
		"--admission-control", "NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,ResourceQuota,NodeRestriction",
		"--authorization-mode", "Node,RBAC",
		"--external-hostname", data.Cluster.Address.ExternalName,
		"--token-auth-file", "/etc/kubernetes/tokens/tokens.csv",
		"--enable-bootstrap-token-auth", "true",
		"--service-account-key-file", "/etc/kubernetes/service-account-key/sa.key",
		"--service-cluster-ip-range", "10.10.10.0/24",
		"--service-node-port-range", nodePortRange,
		"--allow-privileged",
		"--audit-log-maxage", "30",
		"--audit-log-maxbackup", "3",
		"--audit-log-maxsize", "100",
		"--audit-log-path", "/var/log/audit.log",
		"--tls-cert-file", "/etc/kubernetes/tls/apiserver-tls.crt",
		"--tls-private-key-file", "/etc/kubernetes/tls/apiserver-tls.key",
		"--client-ca-file", "/etc/kubernetes/ca-cert/ca.crt",
		"--kubelet-client-certificate", "/etc/kubernetes/kubelet/kubelet-client.crt",
		"--kubelet-client-key", "/etc/kubernetes/kubelet/kubelet-client.key",
		"--kubelet-preferred-address-types", "ExternalIP,InternalIP",
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
		resources.AppLabelKey: "apiserver",
	}

	secretDependencies := []string{
		resources.TokensSecretName,
		resources.ApiserverTLSSecretName,
		resources.KubeletClientCertificatesSecretName,
		resources.CACertSecretName,
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
			Name: resources.ApiserverTLSSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.ApiserverTLSSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.TokensSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.TokensSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
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
			Name: resources.KubeletClientCertificatesSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.KubeletClientCertificatesSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
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
