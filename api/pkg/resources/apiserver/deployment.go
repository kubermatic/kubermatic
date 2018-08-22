package apiserver

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/vpnsidecar"

	"github.com/Masterminds/semver"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("256Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("1Gi"),
			corev1.ResourceCPU:    resource.MustParse("500m"),
		},
	}
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

	dep.Name = resources.ApiserverDeploymentName
	dep.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	dep.Labels = resources.BaseAppLabel(name, nil)

	dep.Spec.Replicas = resources.Int32(1)
	if data.Cluster.Spec.ComponentsOverride.Apiserver.Replicas != nil {
		dep.Spec.Replicas = data.Cluster.Spec.ComponentsOverride.Apiserver.Replicas
	}

	dep.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: resources.BaseAppLabel(name, nil),
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

	volumes := getVolumes()
	podLabels, err := data.GetPodTemplateLabels(name, volumes, nil)
	if err != nil {
		return nil, err
	}

	externalNodePort, err := data.GetApiserverExternalNodePort()
	if err != nil {
		return nil, err
	}

	dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Labels: podLabels,
		Annotations: map[string]string{
			"prometheus.io-0/scrape": "true",
			"prometheus.io-0/path":   "/metrics",
			"prometheus.io-0/port":   "8080",
		},
	}

	etcdClientServiceIP, err := data.ClusterIPByServiceName(resources.EtcdClientServiceName)
	if err != nil {
		return nil, err
	}
	etcd := fmt.Sprintf("https://%s:2379", etcdClientServiceIP)

	// Configure user cluster DNS resolver for this pod.
	dep.Spec.Template.Spec.DNSPolicy, dep.Spec.Template.Spec.DNSConfig, err = resources.UserClusterDNSPolicyAndConfig(data)
	if err != nil {
		return nil, err
	}
	dep.Spec.Template.Spec.Volumes = volumes
	dep.Spec.Template.Spec.InitContainers = []corev1.Container{
		{
			Name:            "etcd-running",
			Image:           data.ImageRegistry(resources.RegistryQuay) + "/coreos/etcd:v3.2.7",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command: []string{
				"/bin/sh",
				"-ec",
				fmt.Sprintf("until ETCDCTL_API=3 /usr/local/bin/etcdctl --cacert=/etc/etcd/pki/client/ca.crt --cert=/etc/etcd/pki/client/apiserver-etcd-client.crt --key=/etc/etcd/pki/client/apiserver-etcd-client.key --dial-timeout=2s --endpoints=[%s] get foo; do echo waiting for etcd; sleep 2; done;", etcd),
			},
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      resources.ApiserverEtcdClientCertificateSecretName,
					MountPath: "/etc/etcd/pki/client",
					ReadOnly:  true,
				},
			},
		},
	}

	openvpnSidecar, err := vpnsidecar.OpenVPNSidecarContainer(data, "openvpn-client")
	if err != nil {
		return nil, fmt.Errorf("failed to get openvpn-client sidecar: %v", err)
	}

	dnatControllerSidecar, err := vpnsidecar.DnatControllerContainer(data, "dnat-controller")
	if err != nil {
		return nil, fmt.Errorf("failed to get dnat-controller sidecar: %v", err)
	}

	flags, err := getApiserverFlags(data, externalNodePort, etcd)
	if err != nil {
		return nil, err
	}

	resourceRequirements := defaultResourceRequirements
	if data.Cluster.Spec.ComponentsOverride.Apiserver.Resources != nil {
		resourceRequirements = *data.Cluster.Spec.ComponentsOverride.Apiserver.Resources
	}

	dep.Spec.Template.Spec.Containers = []corev1.Container{
		*openvpnSidecar,
		*dnatControllerSidecar,
		{
			Name:            name,
			Image:           data.ImageRegistry(resources.RegistryKubernetesGCR) + "/google_containers/hyperkube-amd64:v" + data.Cluster.Spec.Version,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/hyperkube", "apiserver"},
			Env:             getEnvVars(data),
			Args:            flags,
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			Resources:                resourceRequirements,
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
					Name:      resources.CASecretName,
					MountPath: "/etc/kubernetes/pki/ca",
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
				{
					Name:      resources.ApiserverEtcdClientCertificateSecretName,
					MountPath: "/etc/etcd/pki/client",
					ReadOnly:  true,
				},
				{
					Name:      resources.ApiserverFrontProxyClientCertificateSecretName,
					MountPath: "/etc/kubernetes/pki/front-proxy/client",
					ReadOnly:  true,
				},
				{
					Name:      resources.FrontProxyCASecretName,
					MountPath: "/etc/kubernetes/pki/front-proxy/ca",
					ReadOnly:  true,
				},
			},
		},
	}

	return dep, nil
}

func getApiserverFlags(data *resources.TemplateData, externalNodePort int32, etcd string) ([]string, error) {
	nodePortRange := data.NodePortRange
	if nodePortRange == "" {
		nodePortRange = defaultNodePortRange
	}

	clusterVersionSemVer, err := semver.NewVersion(data.Cluster.Spec.Version)
	if err != nil {
		return nil, err
	}

	admissionControlFlagName, admissionControlFlagValue := getAdmissionControlFlags(data)

	flags := []string{
		"--advertise-address", data.Cluster.Address.IP,
		"--secure-port", fmt.Sprintf("%d", externalNodePort),
		"--kubernetes-service-node-port", fmt.Sprintf("%d", externalNodePort),
		"--insecure-bind-address", "0.0.0.0",
		"--insecure-port", "8080",
		"--etcd-servers", etcd,
		"--etcd-cafile", "/etc/etcd/pki/client/ca.crt",
		"--etcd-certfile", "/etc/etcd/pki/client/apiserver-etcd-client.crt",
		"--etcd-keyfile", "/etc/etcd/pki/client/apiserver-etcd-client.key",
		"--storage-backend", "etcd3",
		admissionControlFlagName, admissionControlFlagValue,
		"--authorization-mode", "Node,RBAC",
		"--external-hostname", data.Cluster.Address.ExternalName,
		"--token-auth-file", "/etc/kubernetes/tokens/tokens.csv",
		"--enable-bootstrap-token-auth", "true",
		"--service-account-key-file", "/etc/kubernetes/service-account-key/sa.key",
		// There are efforts upstream adding support for multiple cidr's. Until that has landed, we'll take the first entry
		"--service-cluster-ip-range", data.Cluster.Spec.ClusterNetwork.Services.CIDRBlocks[0],
		"--service-node-port-range", nodePortRange,
		"--allow-privileged",
		"--audit-log-maxage", "30",
		"--audit-log-maxbackup", "3",
		"--audit-log-maxsize", "100",
		"--audit-log-path", "/var/log/audit.log",
		"--tls-cert-file", "/etc/kubernetes/tls/apiserver-tls.crt",
		"--tls-private-key-file", "/etc/kubernetes/tls/apiserver-tls.key",
		"--proxy-client-cert-file", "/etc/kubernetes/pki/front-proxy/client/" + resources.ApiserverProxyClientCertificateCertSecretKey,
		"--proxy-client-key-file", "/etc/kubernetes/pki/front-proxy/client/" + resources.ApiserverProxyClientCertificateKeySecretKey,
		"--client-ca-file", "/etc/kubernetes/pki/ca/ca.crt",
		"--kubelet-client-certificate", "/etc/kubernetes/kubelet/kubelet-client.crt",
		"--kubelet-client-key", "/etc/kubernetes/kubelet/kubelet-client.key",
		"--v", "4",
		"--requestheader-client-ca-file", "/etc/kubernetes/pki/front-proxy/ca/ca.crt",
		"--requestheader-allowed-names", "apiserver-aggregator",
		"--requestheader-extra-headers-prefix", "X-Remote-Extra-",
		"--requestheader-group-headers", "X-Remote-Group",
		"--requestheader-username-headers", "X-Remote-User",
	}
	if clusterVersionSemVer.Minor() >= 9 {
		flags = append(flags, "--feature-gates", "Initializers=true")
		flags = append(flags, "--runtime-config", "admissionregistration.k8s.io/v1alpha1")
	}
	if data.Cluster.Spec.Cloud.AWS != nil {
		flags = append(flags, "--cloud-provider", "aws")
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")
	}
	if data.Cluster.Spec.Cloud.Openstack != nil {
		flags = append(flags, "--cloud-provider", "openstack")
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")
	}
	if data.Cluster.Spec.Cloud.VSphere != nil {
		flags = append(flags, "--cloud-provider", "vsphere")
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")
	}
	if data.Cluster.Spec.Cloud.Azure != nil {
		flags = append(flags, "--cloud-provider", "azure")
		flags = append(flags, "--cloud-config", "/etc/kubernetes/cloud/config")
	}

	if data.Cluster.Spec.Cloud.BringYourOwn != nil {
		flags = append(flags, "--kubelet-preferred-address-types", "Hostname,InternalIP,ExternalIP")
	} else {
		flags = append(flags, "--kubelet-preferred-address-types", "ExternalIP,InternalIP")
	}
	return flags, nil
}

func getAdmissionControlFlags(data *resources.TemplateData) (string, string) {
	// We use these as default in case semver parsing fails
	admissionControlFlagValue := "NamespaceLifecycle,LimitRanger,ServiceAccount,DefaultStorageClass,DefaultTolerationSeconds,NodeRestriction"
	admissionControlFlagName := "--admission-control"

	clusterVersionSemVer, err := semver.NewVersion(data.Cluster.Spec.Version)
	if err != nil {
		return admissionControlFlagName, admissionControlFlagValue
	}

	// Enable {Mutating,Validating}AdmissionWebhook for 1.9+
	if clusterVersionSemVer.Minor() >= 9 {
		admissionControlFlagValue += ",Initializers,MutatingAdmissionWebhook,ValidatingAdmissionWebhook"
	}

	// Use the newer "--enable-admission-plugins" which doesn't care about order for 1.10+
	if clusterVersionSemVer.Minor() >= 10 {
		admissionControlFlagName = "--enable-admission-plugins"
	}

	// Order of these flags matter pre 1.10 and MutatingAdmissionWebhook may manipulate the object, so "ResourceQuota
	// must come last
	admissionControlFlagValue += ",ResourceQuota"

	return admissionControlFlagName, admissionControlFlagValue
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
			Name: resources.CASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.CASecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
					Items: []corev1.KeyToPath{
						{
							Path: resources.CACertSecretKey,
							Key:  resources.CACertSecretKey,
						},
					},
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
		{
			Name: resources.ApiserverEtcdClientCertificateSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.ApiserverEtcdClientCertificateSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.ApiserverFrontProxyClientCertificateSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.ApiserverFrontProxyClientCertificateSecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
		{
			Name: resources.FrontProxyCASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.FrontProxyCASecretName,
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
