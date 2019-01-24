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
	defaultResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("5Mi"),
			corev1.ResourceCPU:    resource.MustParse("5m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("20Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
	}

	ipForwardRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("16Mi"),
			corev1.ResourceCPU:    resource.MustParse("5m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("32Mi"),
			corev1.ResourceCPU:    resource.MustParse("10m"),
		},
	}
)

const (
	name = "openvpn-server"
)

// Deployment returns the kubernetes Controller-Manager Deployment
func Deployment(data resources.DeploymentDataProvider, existing *appsv1.Deployment) (*appsv1.Deployment, error) {
	var dep *appsv1.Deployment
	if existing != nil {
		dep = existing
	} else {
		dep = &appsv1.Deployment{}
	}

	dep.Name = resources.OpenVPNServerDeploymentName
	dep.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	dep.Labels = resources.BaseAppLabel(name, nil)

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
	dep.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}

	volumes := getVolumes()
	podLabels, err := data.GetPodTemplateLabels(name, volumes, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create pod labels: %v", err)
	}

	dep.Spec.Template.ObjectMeta = metav1.ObjectMeta{
		Labels: podLabels,
	}

	_, podNet, err := net.ParseCIDR(data.Cluster().Spec.ClusterNetwork.Pods.CIDRBlocks[0])
	if err != nil {
		return nil, err
	}

	_, serviceNet, err := net.ParseCIDR(data.Cluster().Spec.ClusterNetwork.Services.CIDRBlocks[0])
	if err != nil {
		return nil, err
	}

	pushRoutes := []string{
		// pod and service routes
		"--push", fmt.Sprintf("route %s %s", podNet.IP.String(), net.IP(podNet.Mask).String()),
		"--route", podNet.IP.String(), net.IP(podNet.Mask).String(),
		"--push", fmt.Sprintf("route %s %s", serviceNet.IP.String(), net.IP(serviceNet.Mask).String()),
		"--route", serviceNet.IP.String(), net.IP(serviceNet.Mask).String(),
	}

	// node access network route
	_, nodeAccessNetwork, err := net.ParseCIDR(data.NodeAccessNetwork())
	if err != nil {
		return nil, fmt.Errorf("failed to parse node access network %s: %v", data.NodeAccessNetwork(), err)
	}
	pushRoutes = append(pushRoutes, []string{
		"--push", fmt.Sprintf("route %s %s", nodeAccessNetwork.IP.String(), net.IP(nodeAccessNetwork.Mask).String()),
		"--route", nodeAccessNetwork.IP.String(), net.IP(nodeAccessNetwork.Mask).String(),
	}...)

	dep.Spec.Template.Spec.Volumes = volumes
	dep.Spec.Template.Spec.InitContainers = []corev1.Container{
		{
			Name:            "iptables-init",
			Image:           data.ImageRegistry(resources.RegistryDocker) + "/kubermatic/openvpn:v0.4",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/bin/bash"},
			Args: []string{
				"-c", `# do not give a 10.20.0.0/24 route to clients (nodes) but
# masquerade to openvpn-server's IP instead:
iptables -t nat -A POSTROUTING -o tun0 -s 10.20.0.0/24 -j MASQUERADE

# Only allow outbound traffic to services, pods, nodes
iptables -P FORWARD DROP
iptables -A FORWARD -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A FORWARD -i tun0 -o tun0 -s 10.20.0.0/24 -d ` + podNet.String() + ` -j ACCEPT
iptables -A FORWARD -i tun0 -o tun0 -s 10.20.0.0/24 -d ` + serviceNet.String() + ` -j ACCEPT
iptables -A FORWARD -i tun0 -o tun0 -s 10.20.0.0/24 -d ` + nodeAccessNetwork.String() + ` -j ACCEPT

iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT
iptables -A INPUT -i tun0 -p icmp -j ACCEPT
iptables -A INPUT -i tun0 -j DROP
`,
			},
			SecurityContext: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{"NET_ADMIN"},
				},
			},
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			Resources:                defaultResourceRequirements,
		},
	}

	vpnArgs := []string{
		"--proto", "tcp",
		"--dev", "tun",
		"--mode", "server",
		"--lport", "1194",
		"--server", "10.20.0.0", "255.255.255.0",
		"--ca", "/etc/kubernetes/pki/ca/ca.crt",
		"--cert", "/etc/openvpn/pki/server/server.crt",
		"--key", "/etc/openvpn/pki/server/server.key",
		"--dh", "none",
		"--duplicate-cn",
		"--client-config-dir", "/etc/openvpn/clients",
		"--status", "/run/openvpn-status",
		"--link-mtu", "1432",
		"--cipher", "AES-256-GCM",
		"--auth", "SHA1",
		"--keysize", "256",
		"--script-security", "2",
		"--ping", "5",
		"--verb", "3",
		"--log", "/dev/stdout",
	}
	vpnArgs = append(vpnArgs, pushRoutes...)

	dep.Spec.Template.Spec.Containers = []corev1.Container{
		{
			Name:                     name,
			Image:                    data.ImageRegistry(resources.RegistryDocker) + "/kubermatic/openvpn:v0.4",
			ImagePullPolicy:          corev1.PullIfNotPresent,
			Command:                  []string{"/usr/sbin/openvpn"},
			Args:                     vpnArgs,
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
			Resources: defaultResourceRequirements,
			ReadinessProbe: &corev1.Probe{
				Handler: corev1.Handler{
					Exec: &corev1.ExecAction{
						Command: []string{
							"test",
							"-s",
							"/run/openvpn-status",
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
					Name:      resources.OpenVPNServerCertificatesSecretName,
					MountPath: "/etc/openvpn/pki/server",
					ReadOnly:  true,
				},
				{
					Name:      resources.CASecretName,
					MountPath: "/etc/kubernetes/pki/ca",
					ReadOnly:  true,
				},
				{
					Name:      resources.OpenVPNClientConfigsConfigMapName,
					MountPath: "/etc/openvpn/clients",
					ReadOnly:  true,
				},
			},
		},
		{
			Name:            "ip-forward",
			Image:           data.ImageRegistry(resources.RegistryDocker) + "/kubermatic/openvpn:v0.4",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/bin/bash"},
			Args: []string{
				"-c",
				// Always set IP forwarding as a CNI plugin might reset this to 0 (Like Calico 3).
				"while true; do sysctl -w net.ipv4.ip_forward=1; sleep 30; done",
			},
			TerminationMessagePath:   corev1.TerminationMessagePathDefault,
			TerminationMessagePolicy: corev1.TerminationMessageReadFile,
			SecurityContext: &corev1.SecurityContext{
				Privileged: resources.Bool(true),
			},
			Resources: ipForwardRequirements,
		},
	}

	return dep, nil
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.CASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  resources.OpenVPNCASecretName,
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
					Items: []corev1.KeyToPath{
						{
							Path: resources.OpenVPNCACertKey,
							Key:  resources.OpenVPNCACertKey,
						},
					},
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
			Name: resources.OpenVPNClientConfigsConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: resources.OpenVPNClientConfigsConfigMapName,
					},
					DefaultMode: resources.Int32(resources.DefaultOwnerReadOnlyMode),
				},
			},
		},
	}
}
