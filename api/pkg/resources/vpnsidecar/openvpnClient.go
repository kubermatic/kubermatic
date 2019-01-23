package vpnsidecar

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

var (
	vpnClientResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("5Mi"),
			corev1.ResourceCPU:    resource.MustParse("5m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("32Mi"),
			corev1.ResourceCPU:    resource.MustParse("100m"),
		},
	}
)

// OpenVPNSidecarContainer returns a `corev1.Container` for
// running alongside a master component, providing vpn access
// to user cluster networks.
// Also required but not provided by this func:
// * volumes: resources.OpenVPNClientCertificatesSecretName, resources.CACertSecretName
func OpenVPNSidecarContainer(data resources.DeploymentDataProvider, name string) (*corev1.Container, error) {
	return &corev1.Container{
		Name:            name,
		Image:           data.ImageRegistry("docker.io") + "/kubermatic/openvpn:v0.4",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"/usr/sbin/openvpn"},
		Args: []string{
			"--client",
			"--proto", "tcp",
			"--dev", "tun",
			"--auth-nocache",
			"--remote", resources.GetAbsoluteServiceDNSName(resources.OpenVPNServerServiceName, data.Cluster().Status.NamespaceName), "1194",
			"--nobind",
			"--connect-timeout", "5",
			"--connect-retry", "1",
			"--ca", "/etc/openvpn/pki/client/ca.crt",
			"--cert", "/etc/openvpn/pki/client/client.crt",
			"--key", "/etc/openvpn/pki/client/client.key",
			"--remote-cert-tls", "server",
			"--link-mtu", "1432",
			"--cipher", "AES-256-GCM",
			"--auth", "SHA1",
			"--keysize", "256",
			"--script-security", "2",
			"--status", "/run/openvpn-status",
			"--log", "/dev/stdout",
		},
		SecurityContext: &corev1.SecurityContext{
			Privileged: resources.Bool(true),
		},
		TerminationMessagePath:   corev1.TerminationMessagePathDefault,
		TerminationMessagePolicy: corev1.TerminationMessageReadFile,
		Resources:                vpnClientResourceRequirements,
		VolumeMounts: []corev1.VolumeMount{
			{
				MountPath: "/etc/openvpn/pki/client",
				Name:      resources.OpenVPNClientCertificatesSecretName,
				ReadOnly:  true,
			},
		},
	}, nil
}
