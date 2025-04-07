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

package openvpn

import (
	"fmt"
	"net"
	"path"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var (
	openvpnResourceRequirements = corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("5Mi"),
			corev1.ResourceCPU:    resource.MustParse("5m"),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceMemory: resource.MustParse("50Mi"),
			corev1.ResourceCPU:    resource.MustParse("1"),
		},
	}

	defaultResourceRequirements = map[string]*corev1.ResourceRequirements{
		name: openvpnResourceRequirements.DeepCopy(),
		"ip-fixup": {
			Requests: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("16Mi"),
				corev1.ResourceCPU:    resource.MustParse("5m"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceMemory: resource.MustParse("32Mi"),
				corev1.ResourceCPU:    resource.MustParse("50m"),
			},
		},
		"openvpn-exporter": openvpnResourceRequirements.DeepCopy(),
	}
)

const (
	name         = "openvpn-server"
	statusPath   = "/run/openvpn/openvpn-status"
	exporterPort = 9176
	version      = "v2.5.2-r0"
)

type openVPNDeploymentReconcilerData interface {
	Cluster() *kubermaticv1.Cluster
	NodeAccessNetwork() string
	RewriteImage(string) (string, error)
}

// DeploymentReconciler returns the function to create and update the openvpn server deployment.
func DeploymentReconciler(data openVPNDeploymentReconcilerData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return resources.OpenVPNServerDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			baseLabels := resources.BaseAppLabels(name, nil)
			kubernetes.EnsureLabels(dep, baseLabels)

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}
			dep.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
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

			procMountType := corev1.DefaultProcMount

			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				"prometheus.io/path":                   "/metrics",
				"prometheus.io/port":                   fmt.Sprintf("%d", exporterPort),
				"prometheus.io/scrape":                 "true",
				resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
				// these volumes should not block the autoscaler from evicting the pod
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: "openvpn-status",
			})

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
				return nil, fmt.Errorf("failed to parse node access network %s: %w", data.NodeAccessNetwork(), err)
			}
			pushRoutes = append(pushRoutes, []string{
				"--push", fmt.Sprintf("route %s %s", nodeAccessNetwork.IP.String(), net.IP(nodeAccessNetwork.Mask).String()),
				"--route", nodeAccessNetwork.IP.String(), net.IP(nodeAccessNetwork.Mask).String(),
			}...)

			dep.Spec.Template.Spec.Volumes = getVolumes()

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{
				{
					Name:    "iptables-init",
					Image:   registry.Must(data.RewriteImage(resources.RegistryQuay + "/kubermatic/openvpn:" + version)),
					Command: []string{"/bin/bash"},
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
							Add: []corev1.Capability{
								"NET_ADMIN",
								"NET_RAW",
							},
						},
						ProcMount: &procMountType,
					},
					Resources: openvpnResourceRequirements,
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
				"--status", statusPath,
				"--status-version", "3",
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
					Name:    name,
					Image:   registry.Must(data.RewriteImage(resources.RegistryQuay + "/kubermatic/openvpn:v2.5.2-r0")),
					Command: []string{"/usr/sbin/openvpn"},
					Args:    vpnArgs,
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 1194,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: resources.Bool(true),
						ProcMount:  &procMountType,
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"test",
									"-s",
									statusPath,
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
						{
							Name:      "openvpn-status",
							MountPath: path.Dir(statusPath),
						},
					},
				},
				{
					Name:    "ip-fixup",
					Image:   registry.Must(data.RewriteImage(resources.RegistryQuay + "/kubermatic/openvpn:v2.5.2-r0")),
					Command: []string{"/bin/bash"},
					Args: []string{
						"-c",
						// Always set IP forwarding as a CNI plugin might reset this to 0 (Like Calico 3).
						`while true; do sysctl -w net.ipv4.ip_forward=1;
  if ! iptables -t mangle -C INPUT -p tcp --tcp-flags SYN,RST SYN --dport 1194 -j TCPMSS --set-mss 1300 &>/dev/null; then
   iptables -t mangle -A INPUT -p tcp --tcp-flags SYN,RST SYN --dport 1194 -j TCPMSS --set-mss 1300
  fi
  sleep 30;
done`,
					},
					SecurityContext: &corev1.SecurityContext{
						Privileged: resources.Bool(true),
						ProcMount:  &procMountType,
					},
				},
				{
					Name:    "openvpn-exporter",
					Image:   registry.Must(data.RewriteImage(resources.RegistryDocker + "/kumina/openvpn-exporter:v0.2.2")),
					Command: []string{"/bin/openvpn_exporter"},
					Args: []string{
						"-openvpn.status_paths",
						statusPath,
					},
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: exporterPort,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "openvpn-status",
							MountPath: path.Dir(statusPath),
						},
					},
				},
			}
			err = resources.SetResourceRequirements(dep.Spec.Template.Spec.Containers, defaultResourceRequirements, nil, dep.Annotations)
			if err != nil {
				return nil, fmt.Errorf("failed to set resource requirements: %w", err)
			}

			return dep, nil
		}
	}
}

func getVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: resources.CASecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: resources.OpenVPNCASecretName,
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
					SecretName: resources.OpenVPNServerCertificatesSecretName,
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
				},
			},
		},
		{
			Name: "openvpn-status",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}
}
