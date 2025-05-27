/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package vpnsidecar

import (
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"

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
			corev1.ResourceCPU:    resource.MustParse("1"),
		},
	}
)

type openvpnData interface {
	RewriteImage(string) (string, error)
	Cluster() *kubermaticv1.Cluster
}

// OpenVPNSidecarContainer returns a `corev1.Container` for
// running alongside a master component, providing vpn access
// to user cluster networks.
// Also required but not provided by this func:
// * volumes: resources.OpenVPNClientCertificatesSecretName, resources.CACertSecretName.
func OpenVPNSidecarContainer(data openvpnData, name string) (*corev1.Container, error) {
	return &corev1.Container{
		Name:    name,
		Image:   registry.Must(data.RewriteImage(resources.RegistryQuay + "/kubermatic/openvpn:v2.5.2-r0")),
		Command: []string{"/usr/sbin/openvpn"},
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
		Resources: vpnClientResourceRequirements,
		VolumeMounts: []corev1.VolumeMount{
			{
				MountPath: "/etc/openvpn/pki/client",
				Name:      resources.OpenVPNClientCertificatesSecretName,
				ReadOnly:  true,
			},
		},
	}, nil
}
