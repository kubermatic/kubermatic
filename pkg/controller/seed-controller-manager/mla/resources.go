/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package mla

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	grafanasdk "github.com/kubermatic/grafanasdk"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
)

const nginxConfig = `worker_processes  1;
error_log  /dev/stderr;
pid        /tmp/nginx.pid;
worker_rlimit_nofile 8192;

events {
  worker_connections  1024;
}

http {
  client_body_temp_path /tmp/client_temp;
  proxy_temp_path       /tmp/proxy_temp_path;
  fastcgi_temp_path     /tmp/fastcgi_temp;
  uwsgi_temp_path       /tmp/uwsgi_temp;
  scgi_temp_path        /tmp/scgi_temp;

  default_type application/octet-stream;
  log_format   main '$remote_addr - $remote_user [$time_local]  $status '
	'"$request" $body_bytes_sent "$http_referer" '
	'"$http_user_agent" "$http_x_forwarded_for" $http_x_scope_orgid';
  access_log   /dev/stderr  main;
  sendfile     on;
  tcp_nopush   on;
  resolver kube-dns.kube-system.svc.cluster.local;

  # write path - exposed to user clusters
  server {
	listen             8080;
	proxy_set_header X-Scope-OrgID {{ .TenantID}};

	# Loki Config
	location = /loki/api/v1/push {
	  proxy_pass       http://loki-distributed-distributor.{{ .Namespace}}.svc.cluster.local:3100$request_uri;
	}

	# Cortex Config
	location = /api/v1/push {
	  proxy_pass      http://cortex-distributor.{{ .Namespace}}.svc.cluster.local:8080$request_uri;
	}
  }

  # read path - cluster-local access only
  server {
	listen             8081;
	proxy_set_header   X-Scope-OrgID {{ .TenantID}};

	# k8s probes
	location = / {
	  return 200 'OK';
	  auth_basic off;
	}

	# location = /api/prom/tail {
	#   proxy_pass       http://loki-distributed-querier.{{ .Namespace}}.svc.cluster.local:3100$request_uri;
	#   proxy_set_header Upgrade $http_upgrade;
	#   proxy_set_header Connection "upgrade";
	# }

	# location = /loki/api/v1/tail {
	#   proxy_pass       http://loki-distributed-querier.{{ .Namespace}}.svc.cluster.local:3100$request_uri;
	#   proxy_set_header Upgrade $http_upgrade;
	#   proxy_set_header Connection "upgrade";
	# }

	location ~ /loki/api/.* {
	  proxy_pass       http://loki-distributed-query-frontend.{{ .Namespace}}.svc.cluster.local:3100$request_uri;
	}

	# Cortex Config
	location ~ /api/prom/.* {
	  proxy_pass       http://cortex-query-frontend.{{ .Namespace}}.svc.cluster.local:8080$request_uri;
	}
  }

  # public read and write path - used for alertmanager only
  server {
	listen             8082;
	proxy_set_header   X-Scope-OrgID {{ .TenantID}};

	# Alertmanager Config
	location ~ /api/prom/alertmanager.* {
	  proxy_pass      http://cortex-alertmanager.{{ .Namespace}}.svc.cluster.local:8080$request_uri;
	}
	location ~ /api/v1/alerts {
	  proxy_pass      http://cortex-alertmanager.{{ .Namespace}}.svc.cluster.local:8080$request_uri;
	}
	location ~ /multitenant_alertmanager/status {
	  proxy_pass      http://cortex-alertmanager.{{ .Namespace}}.svc.cluster.local:8080$request_uri;
	}
  }
}
`

const (
	gatewayName         = "mla-gateway"
	gatewayAlertName    = "mla-gateway-alert"
	gatewayExternalName = resources.MLAGatewayExternalServiceName
)

type configTemplateData struct {
	Namespace string
	TenantID  string
}

func renderTemplate(tpl string, data interface{}) (string, error) {
	t, err := template.New("base").Funcs(sprig.TxtFuncMap()).Parse(tpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse as Go template: %v", err)
	}

	output := bytes.Buffer{}
	if err := t.Execute(&output, data); err != nil {
		return "", fmt.Errorf("failed to render template: %v", err)
	}

	return strings.TrimSpace(output.String()), nil
}

func GatewayConfigMapCreator(c *kubermaticv1.Cluster) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return gatewayName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				configData := configTemplateData{
					Namespace: resources.MLANamespace,
					TenantID:  c.Name,
				}
				config, err := renderTemplate(nginxConfig, configData)
				if err != nil {
					return nil, fmt.Errorf("failed to render Prometheus config: %v", err)
				}

				cm.Data = map[string]string{
					"nginx.conf": config}
			}
			return cm, nil
		}
	}
}

func GatewayAlertServiceCreator() reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return gatewayAlertName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Spec.Type = corev1.ServiceTypeClusterIP
			s.Spec.Selector = map[string]string{common.NameLabel: "mla"}

			if len(s.Spec.Ports) == 0 {
				s.Spec.Ports = make([]corev1.ServicePort, 1)
			}

			s.Spec.Ports[0].Name = "http-alert"
			s.Spec.Ports[0].Protocol = corev1.ProtocolTCP
			s.Spec.Ports[0].Port = 80
			s.Spec.Ports[0].TargetPort = intstr.FromString("http-alert")

			return s, nil
		}
	}
}

func GatewayInternalServiceCreator() reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return gatewayName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Spec.Type = corev1.ServiceTypeClusterIP
			s.Spec.Selector = map[string]string{common.NameLabel: "mla"}

			if len(s.Spec.Ports) == 0 {
				s.Spec.Ports = make([]corev1.ServicePort, 1)
			}

			s.Spec.Ports[0].Name = "http-int"
			s.Spec.Ports[0].Protocol = corev1.ProtocolTCP
			s.Spec.Ports[0].Port = 80
			s.Spec.Ports[0].TargetPort = intstr.FromString("http-int")

			return s, nil
		}
	}
}

func GatewayExternalServiceCreator(c *kubermaticv1.Cluster) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
		return gatewayExternalName, func(s *corev1.Service) (*corev1.Service, error) {
			if s.Annotations == nil {
				s.Annotations = map[string]string{}
			}
			s.Spec.Selector = map[string]string{common.NameLabel: "mla"}

			switch c.Spec.ExposeStrategy {
			case kubermaticv1.ExposeStrategyNodePort:
				// Exposes MLA GW via ModePort.
				s.Spec.Type = corev1.ServiceTypeNodePort
				s.Annotations[nodeportproxy.DefaultExposeAnnotationKey] = "true"
				delete(s.Annotations, nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey)
			case kubermaticv1.ExposeStrategyLoadBalancer:
				// When using exposeStrategy==LoadBalancer, only one LB service is used to expose multiple user cluster
				// -related services (APIServer, OpenVPN, MLAGw). NodePortProxy in namespaced mode is used to redirect
				// the traffic to the right service.
				s.Spec.Type = corev1.ServiceTypeNodePort
				s.Annotations[nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey] = "true"
				delete(s.Annotations, nodeportproxy.DefaultExposeAnnotationKey)
			case kubermaticv1.ExposeStrategyTunneling:
				// Exposes MLA GW via SNI.
				s.Spec.Type = corev1.ServiceTypeClusterIP
				s.Annotations[nodeportproxy.DefaultExposeAnnotationKey] = nodeportproxy.SNIType.String()
				// Maps SNI host with the port name of this service.
				s.Annotations[nodeportproxy.PortHostMappingAnnotationKey] =
					fmt.Sprintf(`{"http-ext": %q}`, resources.MLAGatewaySNIPrefix+c.Address.ExternalName)
				delete(s.Annotations, nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey)
			default:
				return nil, fmt.Errorf("unsupported expose strategy: %q", c.Spec.ExposeStrategy)
			}

			if len(s.Spec.Ports) == 0 {
				s.Spec.Ports = make([]corev1.ServicePort, 1)
			}

			s.Spec.Ports[0].Name = "http-ext"
			s.Spec.Ports[0].Protocol = corev1.ProtocolTCP
			s.Spec.Ports[0].Port = 80
			s.Spec.Ports[0].TargetPort = intstr.FromString("http-ext")

			return s, nil
		}
	}
}

const (
	image   = "nginxinc/nginx-unprivileged"
	version = "1.19-alpine"
)

func GatewayDeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return gatewayName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Spec.Replicas = pointer.Int32Ptr(1)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					common.NameLabel: "mla",
				},
			}
			d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}
			d.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				FSGroup:      pointer.Int64Ptr(1001),
				RunAsGroup:   pointer.Int64Ptr(2001),
				RunAsUser:    pointer.Int64Ptr(1001),
				RunAsNonRoot: pointer.BoolPtr(true),
			}
			d.Spec.Template.Labels = d.Spec.Selector.MatchLabels
			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:            "nginx",
					Image:           data.ImageRegistry(resources.RegistryDocker) + "/" + image + ":" + version,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Ports: []corev1.ContainerPort{
						{
							Name:          "http-ext",
							ContainerPort: 8080,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "http-int",
							ContainerPort: 8081,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          "http-alert",
							ContainerPort: 8082,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/",
								Port:   intstr.FromString("http-int"),
								Scheme: corev1.URISchemeHTTP,
							},
						},
						InitialDelaySeconds: 15,
						TimeoutSeconds:      1,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						FailureThreshold:    3,
					},
					SecurityContext: &corev1.SecurityContext{
						Capabilities: &corev1.Capabilities{
							Drop: []corev1.Capability{"ALL"},
						},
						ReadOnlyRootFilesystem:   pointer.BoolPtr(true),
						AllowPrivilegeEscalation: pointer.BoolPtr(false),
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "config",
							MountPath: "/etc/nginx",
						},
						{
							Name:      "tmp",
							MountPath: "/tmp",
						},
						{
							Name:      "docker-entrypoint-d-override",
							MountPath: "/docker-entrypoint.d",
						},
					},
				},
			}
			d.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: "config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: gatewayName,
							},
						},
					},
				},
				{
					Name:         "tmp",
					VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
				{
					Name:         "docker-entrypoint-d-override",
					VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
				},
			}
			return d, nil
		}
	}
}

func getOrgByProjectID(ctx context.Context, client ctrlruntimeclient.Client, grafanaClient *grafanasdk.Client, projectID string) (grafanasdk.Org, error) {
	project := &kubermaticv1.Project{}
	if err := client.Get(ctx, types.NamespacedName{Name: projectID}, project); err != nil {
		return grafanasdk.Org{}, fmt.Errorf("failed to get project: %w", err)
	}

	orgID, ok := project.GetAnnotations()[grafanaOrgAnnotationKey]
	if !ok {
		return grafanasdk.Org{}, fmt.Errorf("project should have grafana org annotation set")
	}
	id, err := strconv.ParseUint(orgID, 10, 32)
	if err != nil {
		return grafanasdk.Org{}, err
	}
	return grafanaClient.GetOrgById(ctx, uint(id))
}
