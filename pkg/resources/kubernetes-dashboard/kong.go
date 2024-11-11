/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package kubernetesdashboard

import (
	"fmt"
	"strings"

	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
)

// These values are used by the KKP Dashboard to setup the internal proxying.

var (
	// ProxyPort is the port on the kong proxy pod that ingests traffic.
	ProxyPodPort = 8443
	// ProxyPodSelector is selecting the Kong pods.
	ProxyPodSelector = labels.Set(resources.BaseAppLabels(kongDeploymentName, nil)).String()
)

const (
	kongDeploymentName     = "kubernetes-dashboard-kong"
	kongContainerName      = "kubernetes-dashboard-kong"
	kongServiceName        = "kubernetes-dashboard-kong-proxy"
	KongConfigMapName      = "kong-dbless-config"
	kongServiceAccountName = "kubernetes-dashboard-kong"

	kongConfig = `
_format_version: "3.0"
services:
  - name: auth
    host: kubernetes-dashboard-auth
    port: 8000
    protocol: http
    routes:
      - name: authLogin
        paths:
          - /api/v1/login
        strip_path: false
      - name: authCsrf
        paths:
          - /api/v1/csrftoken/login
        strip_path: false
      - name: authMe
        paths:
          - /api/v1/me
        strip_path: false
  - name: api
    host: kubernetes-dashboard-api
    port: 8000
    protocol: http
    routes:
      - name: api
        paths:
          - /api
        strip_path: false
      - name: metrics
        paths:
          - /metrics
        strip_path: false
  - name: web
    host: kubernetes-dashboard-web
    port: 8000
    protocol: http
    routes:
      - name: root
        paths:
          - /
        strip_path: false
`
)

func KongServiceAccountReconciler() reconciling.NamedServiceAccountReconcilerFactory {
	return func() (string, reconciling.ServiceAccountReconciler) {
		return kongServiceAccountName, func(existing *corev1.ServiceAccount) (*corev1.ServiceAccount, error) {
			return existing, nil
		}
	}
}

func KongConfigMapReconciler() reconciling.NamedConfigMapReconcilerFactory {
	return func() (string, reconciling.ConfigMapReconciler) {
		return KongConfigMapName, func(existing *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if existing.Data == nil {
				existing.Data = map[string]string{}
			}
			existing.Data["kong.yml"] = strings.TrimSpace(kongConfig)
			return existing, nil
		}
	}
}

func KongProxyServiceReconciler() reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return kongServiceName, func(existing *corev1.Service) (*corev1.Service, error) {
			existing.Spec.Ports = []corev1.ServicePort{
				{
					Name:       "kong-proxy-tls",
					Port:       443,
					TargetPort: intstr.FromInt(8443),
					Protocol:   corev1.ProtocolTCP,
				},
			}

			existing.Spec.Selector = resources.BaseAppLabels(kongDeploymentName, nil)

			return existing, nil
		}
	}
}

func mustParseQuantity(s string) *resource.Quantity {
	q := resource.MustParse(s)
	return &q
}

func KongDeploymentReconciler(data kubernetesDashboardData) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return kongDeploymentName, func(dep *appsv1.Deployment) (*appsv1.Deployment, error) {
			baseLabels := resources.BaseAppLabels(kongDeploymentName, nil)
			kubernetes.EnsureLabels(dep, baseLabels)

			dep.Spec.Replicas = resources.Int32(1)
			dep.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: baseLabels,
			}

			kubernetes.EnsureAnnotations(&dep.Spec.Template, map[string]string{
				resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
				// these volumes should not block the autoscaler from evicting the pod
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: "kubernetes-dashboard-kong-prefix-dir,kubernetes-dashboard-kong-tmp",
			})

			var environment = []corev1.EnvVar{
				{Name: "KONG_ADMIN_ACCESS_LOG", Value: "/dev/stdout"},
				{Name: "KONG_ADMIN_ERROR_LOG", Value: "/dev/stderr"},
				{Name: "KONG_ADMIN_GUI_ACCESS_LOG", Value: "/dev/stdout"},
				{Name: "KONG_ADMIN_GUI_ERROR_LOG", Value: "/dev/stderr"},
				{Name: "KONG_ADMIN_LISTEN", Value: "127.0.0.1:8444 http2 ssl, [::1]:8444 http2 ssl"},
				{Name: "KONG_CLUSTER_LISTEN", Value: "off"},
				{Name: "KONG_DATABASE", Value: "off"},
				{Name: "KONG_DECLARATIVE_CONFIG", Value: "/kong_dbless/kong.yml"},
				{Name: "KONG_DNS_ORDER", Value: "LAST,A,CNAME,AAAA,SRV"},
				{Name: "KONG_LUA_PACKAGE_PATH", Value: "/opt/?.lua;/opt/?/init.lua;;"},
				{Name: "KONG_NGINX_WORKER_PROCESSES", Value: "1"},
				{Name: "KONG_PLUGINS", Value: "off"},
				{Name: "KONG_PORTAL_API_ACCESS_LOG", Value: "/dev/stdout"},
				{Name: "KONG_PORTAL_API_ERROR_LOG", Value: "/dev/stderr"},
				{Name: "KONG_PORT_MAPS", Value: "443:8443"},
				{Name: "KONG_PREFIX", Value: "/kong_prefix/"},
				{Name: "KONG_PROXY_ACCESS_LOG", Value: "/dev/stdout"},
				{Name: "KONG_PROXY_ERROR_LOG", Value: "/dev/stderr"},
				{Name: "KONG_PROXY_LISTEN", Value: "0.0.0.0:8443 http2 ssl, [::]:8443 http2 ssl"},
				{Name: "KONG_PROXY_STREAM_ACCESS_LOG", Value: "/dev/stdout basic"},
				{Name: "KONG_PROXY_STREAM_ERROR_LOG", Value: "/dev/stderr"},
				{Name: "KONG_ROUTER_FLAVOR", Value: "traditional"},
				{Name: "KONG_STATUS_ACCESS_LOG", Value: "off"},
				{Name: "KONG_STATUS_ERROR_LOG", Value: "/dev/stderr"},
				{Name: "KONG_STATUS_LISTEN", Value: "0.0.0.0:8100, [::]:8100"},
				{Name: "KONG_STREAM_LISTEN", Value: "off"},
			}

			clusterVersion := data.Cluster().Status.Versions.ControlPlane
			if clusterVersion == "" {
				clusterVersion = data.Cluster().Spec.Version
			}

			kongVersion, err := KongVersion(clusterVersion)
			if err != nil {
				return nil, fmt.Errorf("failed to determine Kong version: %w", err)
			}

			dep.Spec.Template.Spec.ServiceAccountName = kongServiceAccountName
			dep.Spec.Template.Spec.AutomountServiceAccountToken = ptr.To(false)

			dep.Spec.Template.Spec.InitContainers = []corev1.Container{
				{
					Name:            "clear-stale-pid",
					Image:           registry.Must(data.RewriteImage("docker.io/kong:" + kongVersion)),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"rm",
						"-vrf",
						"$KONG_PREFIX/pids",
					},
					Env: environment,
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: ptr.To(false),
						ReadOnlyRootFilesystem:   ptr.To(true),
						RunAsNonRoot:             ptr.To(true),
						RunAsUser:                ptr.To(int64(1000)),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{
								"ALL",
							},
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "kubernetes-dashboard-kong-prefix-dir",
							MountPath: "/kong_prefix/",
						},
						{
							Name:      "kubernetes-dashboard-kong-tmp",
							MountPath: "/tmp",
						},
						{
							Name:      "kong-custom-dbless-config-volume",
							MountPath: "/kong_dbless/",
						},
					},
				},
			}

			dep.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            kongContainerName,
					Image:           registry.Must(data.RewriteImage("docker.io/kong:" + kongVersion)),
					ImagePullPolicy: corev1.PullIfNotPresent,
					Env: append(
						environment,
						corev1.EnvVar{
							Name:  "KONG_NGINX_DAEMON",
							Value: "off",
						},
					),
					SecurityContext: &corev1.SecurityContext{
						AllowPrivilegeEscalation: ptr.To(false),
						ReadOnlyRootFilesystem:   ptr.To(true),
						RunAsNonRoot:             ptr.To(true),
						RunAsUser:                ptr.To(int64(1000)),
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{
								"ALL",
							},
						},
					},
					Lifecycle: &corev1.Lifecycle{
						PreStop: &corev1.LifecycleHandler{
							Exec: &corev1.ExecAction{
								Command: []string{
									"kong",
									"quit",
									"--wait=15",
								},
							},
						},
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "proxy-tls",
							ContainerPort: 8443,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "status",
							ContainerPort: 8100,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "kubernetes-dashboard-kong-prefix-dir",
							MountPath: "/kong_prefix/",
						},
						{
							Name:      "kubernetes-dashboard-kong-tmp",
							MountPath: "/tmp",
						},
						{
							Name:      "kong-custom-dbless-config-volume",
							MountPath: "/kong_dbless/",
						},
					},
					ReadinessProbe: &corev1.Probe{
						FailureThreshold: 3,
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/status/ready",
								Port:   intstr.FromString("status"),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						InitialDelaySeconds: 5,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      5,
					},
					LivenessProbe: &corev1.Probe{
						FailureThreshold: 3,
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/status",
								Port:   intstr.FromString("status"),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						InitialDelaySeconds: 5,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						TimeoutSeconds:      5,
					},
				},
			}

			dep.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "kubernetes-dashboard-kong-prefix-dir",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{
							SizeLimit: mustParseQuantity("256Mi"),
						},
					},
				},
				{
					Name: "kubernetes-dashboard-kong-tmp",
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{
							SizeLimit: mustParseQuantity("1Gi"),
						},
					},
				},
				{
					Name: "kubernetes-dashboard-kong-token",
					VolumeSource: corev1.VolumeSource{
						Projected: &corev1.ProjectedVolumeSource{
							DefaultMode: ptr.To(int32(0644)),
							Sources: []corev1.VolumeProjection{
								{
									ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
										ExpirationSeconds: ptr.To(int64(3607)),
										Path:              "token",
									},
								},
								{
									ConfigMap: &corev1.ConfigMapProjection{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "kube-root-ca.crt",
										},
										Items: []corev1.KeyToPath{
											{
												Key:  "ca.crt",
												Path: "ca.crt",
											},
										},
									},
								},
								{
									DownwardAPI: &corev1.DownwardAPIProjection{
										Items: []corev1.DownwardAPIVolumeFile{
											{
												Path: "namespace",
												FieldRef: &corev1.ObjectFieldSelector{
													APIVersion: "v1",
													FieldPath:  "metadata.namespace",
												},
											},
										},
									},
								},
							},
						},
					},
				},
				{
					Name: "kong-custom-dbless-config-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "kong-dbless-config",
							},
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
