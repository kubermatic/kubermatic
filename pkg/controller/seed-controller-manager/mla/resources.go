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
	"crypto/sha1"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/resources/registry"
	"k8c.io/reconciler/pkg/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/utils/ptr"
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
  sendfile     on;
  tcp_nopush   on;
  resolver kube-dns.kube-system.svc.cluster.local;

{{ if .LokiWriteLimit }}
  limit_req_zone $binary_remote_addr zone=loki_write_limit:1m rate={{ .LokiWriteLimit }}r/s;
{{ end }}
{{ if .LokiReadLimit }}
  limit_req_zone $binary_remote_addr zone=loki_read_limit:1m rate={{ .LokiReadLimit }}r/s;
{{ end }}
{{ if .CortexReadLimit }}
  limit_req_zone $binary_remote_addr zone=cortex_read_limit:1m rate={{ .CortexReadLimit }}r/s;
{{ end }}

  # write path - exposed to user clusters
  server {
	listen                  8080 ssl;
	proxy_set_header        X-Scope-OrgID {{ .TenantID }};

	ssl_certificate         {{ .SSLCertFile }};
	ssl_certificate_key     {{ .SSLKeyFile }};
	ssl_verify_client       on;
	ssl_client_certificate  {{ .SSLCACertFile }};
	ssl_protocols           TLSv1.3;

	# Loki
	location = /loki/api/v1/push {
{{ if .LokiWriteLimit }}
      limit_req zone=loki_write_limit{{ if .LokiWriteLimitBurst }} burst={{ .LokiWriteLimitBurst }} nodelay{{ end }};
{{ end }}
	  proxy_pass       http://loki-distributed-distributor.{{ .Namespace }}.svc.cluster.local:3100$request_uri;
	}

	# Cortex
	location = /api/v1/push {
	  proxy_pass      http://cortex-distributor.{{ .Namespace }}.svc.cluster.local:8080$request_uri;
	}
  }

  # read path - cluster-local access only
  server {
	listen             8081;
	proxy_set_header   X-Scope-OrgID {{ .TenantID }};

	# k8s probes
	location = / {
	  return 200 'OK';
	  auth_basic off;
	}

	# Alertmanager Alerts
	location ~ /api/prom/alertmanager/.* {
	  proxy_pass      http://cortex-alertmanager.{{ .Namespace }}.svc.cluster.local:8080$request_uri;
	}

	# Alertmanager Config
	location = /api/prom/api/v1/alerts {
	  proxy_pass      http://cortex-alertmanager.{{ .Namespace }}.svc.cluster.local:8080/api/v1/alerts;
	}

	# Loki
	location ~ /loki/api/.* {
{{ if .LokiReadLimit }}
      limit_req zone=loki_read_limit{{ if .LokiReadLimitBurst }} burst={{ .LokiReadLimitBurst }} nodelay{{ end }};
{{ end }}
	  proxy_pass       http://loki-distributed-query-frontend.{{ .Namespace }}.svc.cluster.local:3100$request_uri;
	}

	# Loki Ruler
	location ~ /prometheus/.* {
	  proxy_pass       http://loki-distributed-ruler.{{ .Namespace }}.svc.cluster.local:3100$request_uri;
	}

	# Cortex
	location ~ /api/prom/.* {
{{ if .CortexReadLimit }}
      limit_req zone=cortex_read_limit{{ if .CortexReadLimitBurst }} burst={{ .CortexReadLimitBurst }} nodelay{{ end }};
{{ end }}
	  proxy_pass       http://cortex-query-frontend.{{ .Namespace }}.svc.cluster.local:8080$request_uri;
	}

	# Cortex Ruler, alternative for /api/v1/rules which is also used by loki)
	location = /api/prom/api/v1/rules {
	  proxy_pass       http://cortex-ruler.{{ .Namespace }}.svc.cluster.local:8080/prometheus/api/v1/rules;
	}
  }
}
`

const (
	gatewayName         = "mla-gateway"
	gatewayExternalName = resources.MLAGatewayExternalServiceName

	extPortName = "http-ext"
	intPortName = "http-int"

	configHashAnnotation     = "mla.k8c.io/config-hash"
	configVolumeName         = "config"
	configVolumePath         = "/etc/nginx"
	certificatesVolumeName   = "gw-certificates"
	certificatesVolumePath   = "/etc/ssl/mla-gateway"
	caCertificatesVolumeName = "ca-certificates"
	caCertificatesVolumePath = "/etc/ssl/mla-gateway-ca"
)

type configTemplateData struct {
	Namespace            string
	TenantID             string
	SSLCertFile          string
	SSLKeyFile           string
	SSLCACertFile        string
	CortexReadLimit      int32
	CortexReadLimitBurst int32
	LokiWriteLimit       int32
	LokiWriteLimitBurst  int32
	LokiReadLimit        int32
	LokiReadLimitBurst   int32
}

func renderTemplate(tpl string, data interface{}) (string, error) {
	t, err := template.New("base").Funcs(sprig.TxtFuncMap()).Parse(tpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse as Go template: %w", err)
	}

	output := bytes.Buffer{}
	if err := t.Execute(&output, data); err != nil {
		return "", fmt.Errorf("failed to render template: %w", err)
	}

	return strings.TrimSpace(output.String()), nil
}

func GatewayConfigMapReconciler(c *kubermaticv1.Cluster, mlaNamespace string, s *kubermaticv1.MLAAdminSetting) reconciling.NamedConfigMapReconcilerFactory {
	return func() (string, reconciling.ConfigMapReconciler) {
		return gatewayName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			configData := configTemplateData{
				Namespace:     mlaNamespace,
				TenantID:      c.Name,
				SSLCertFile:   fmt.Sprintf("%s/%s", certificatesVolumePath, resources.MLAGatewayCertSecretKey),
				SSLKeyFile:    fmt.Sprintf("%s/%s", certificatesVolumePath, resources.MLAGatewayKeySecretKey),
				SSLCACertFile: fmt.Sprintf("%s/%s", caCertificatesVolumePath, resources.MLAGatewayCACertKey),
			}
			if s != nil && s.Spec.MonitoringRateLimits != nil {
				// NOTE: Cortex write path rate-limiting is implemented directly by Cortex configuration
				configData.CortexReadLimit = s.Spec.MonitoringRateLimits.QueryRate
				configData.CortexReadLimitBurst = s.Spec.MonitoringRateLimits.QueryBurstSize
			}
			if s != nil && s.Spec.LoggingRateLimits != nil {
				configData.LokiWriteLimit = s.Spec.LoggingRateLimits.IngestionRate
				configData.LokiWriteLimitBurst = s.Spec.LoggingRateLimits.IngestionBurstSize
				configData.LokiReadLimit = s.Spec.LoggingRateLimits.QueryRate
				configData.LokiReadLimitBurst = s.Spec.LoggingRateLimits.QueryBurstSize
			}
			config, err := renderTemplate(nginxConfig, configData)
			if err != nil {
				return nil, fmt.Errorf("failed to render Prometheus config: %w", err)
			}
			cm.Data["nginx.conf"] = config
			return cm, nil
		}
	}
}

func GatewayInternalServiceReconciler() reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return gatewayName, func(s *corev1.Service) (*corev1.Service, error) {
			s.Spec.Type = corev1.ServiceTypeClusterIP
			s.Spec.Selector = map[string]string{common.NameLabel: gatewayName}

			if len(s.Spec.Ports) == 0 {
				s.Spec.Ports = make([]corev1.ServicePort, 1)
			}

			s.Spec.Ports[0].Name = intPortName
			s.Spec.Ports[0].Protocol = corev1.ProtocolTCP
			s.Spec.Ports[0].Port = 80
			s.Spec.Ports[0].TargetPort = intstr.FromString(intPortName)

			return s, nil
		}
	}
}

func GatewayExternalServiceReconciler(c *kubermaticv1.Cluster) reconciling.NamedServiceReconcilerFactory {
	return func() (string, reconciling.ServiceReconciler) {
		return gatewayExternalName, func(s *corev1.Service) (*corev1.Service, error) {
			if s.Annotations == nil {
				s.Annotations = map[string]string{}
			}
			s.Spec.Selector = map[string]string{common.NameLabel: gatewayName}

			switch c.Spec.ExposeStrategy {
			case kubermaticv1.ExposeStrategyNodePort:
				// Exposes MLA GW via NodePort.
				s.Spec.Type = corev1.ServiceTypeNodePort
				s.Annotations[nodeportproxy.DefaultExposeAnnotationKey] = nodeportproxy.NodePortType.String()
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
					fmt.Sprintf(`{%q: %q}`, extPortName, resources.MLAGatewaySNIPrefix+c.Status.Address.ExternalName)
				delete(s.Annotations, nodeportproxy.NodePortProxyExposeNamespacedAnnotationKey)
			default:
				return nil, fmt.Errorf("unsupported expose strategy: %q", c.Spec.ExposeStrategy)
			}

			if len(s.Spec.Ports) == 0 {
				s.Spec.Ports = make([]corev1.ServicePort, 1)
			}

			s.Spec.Ports[0].Name = extPortName
			s.Spec.Ports[0].Protocol = corev1.ProtocolTCP
			s.Spec.Ports[0].Port = 80
			s.Spec.Ports[0].TargetPort = intstr.FromString(extPortName)

			if c.Spec.ExposeStrategy == kubermaticv1.ExposeStrategyTunneling {
				s.Spec.Ports[0].NodePort = 0 // allows switching from other expose strategies
			}

			return s, nil
		}
	}
}

const (
	image   = "nginxinc/nginx-unprivileged"
	version = "1.20.1-alpine"

	nginxScript = `
set -e

calc_state() {
  find /etc/ssl -type f -exec cat {} + | sha1sum -
}

state=$(calc_state)

nginx &

# watch for changes (i.e. renewed certificates), then reload nginx
while true; do
  newState=$(calc_state)
  if [ "$newState" != "$state" ]; then
    nginx -s reload
  fi
  state=$newState
  sleep 60
done
`
)

func GatewayDeploymentReconciler(data *resources.TemplateData, settings *kubermaticv1.MLAAdminSetting) reconciling.NamedDeploymentReconcilerFactory {
	return func() (string, reconciling.DeploymentReconciler) {
		return gatewayName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Spec.Replicas = ptr.To[int32](1)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					common.NameLabel: gatewayName,
				},
			}
			d.Spec.Template.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: resources.ImagePullSecretName}}
			d.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
				FSGroup:      ptr.To[int64](1001),
				RunAsGroup:   ptr.To[int64](2001),
				RunAsUser:    ptr.To[int64](1001),
				RunAsNonRoot: ptr.To(true),
			}

			// hash for the annotation used to force deployment rollout upon configuration change
			configHash := sha1.New()
			configData, err := json.Marshal(settings)
			if err != nil {
				return nil, fmt.Errorf("failed to encode MLAAdminSetting: %w", err)
			}
			configHash.Write(configData)

			kubernetes.EnsureLabels(&d.Spec.Template, d.Spec.Selector.MatchLabels)
			kubernetes.EnsureAnnotations(&d.Spec.Template, map[string]string{
				configHashAnnotation:                   fmt.Sprintf("%x", configHash.Sum(nil)),
				resources.ClusterLastRestartAnnotation: data.Cluster().Annotations[resources.ClusterLastRestartAnnotation],
				// these volumes should not block the autoscaler from evicting the pod
				resources.ClusterAutoscalerSafeToEvictVolumesAnnotation: "tmp",
			})

			d.Spec.Template.Spec.Containers = []corev1.Container{
				{
					Name:  "nginx",
					Image: registry.Must(data.RewriteImage(resources.RegistryDocker + "/" + image + ":" + version)),
					Command: []string{
						"/bin/sh",
						"-c",
						strings.TrimSpace(nginxScript),
					},
					ImagePullPolicy: corev1.PullIfNotPresent,
					Ports: []corev1.ContainerPort{
						{
							Name:          extPortName,
							ContainerPort: 8080,
							Protocol:      corev1.ProtocolTCP,
						},
						{
							Name:          intPortName,
							ContainerPort: 8081,
							Protocol:      corev1.ProtocolTCP,
						},
					},
					ReadinessProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							HTTPGet: &corev1.HTTPGetAction{
								Path:   "/",
								Port:   intstr.FromString(intPortName),
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
						ReadOnlyRootFilesystem:   ptr.To(true),
						AllowPrivilegeEscalation: ptr.To(false),
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      configVolumeName,
							MountPath: configVolumePath,
						},
						{
							Name:      certificatesVolumeName,
							MountPath: certificatesVolumePath,
						},
						{
							Name:      caCertificatesVolumeName,
							MountPath: caCertificatesVolumePath,
						},
						{
							Name:      "tmp",
							MountPath: "/tmp",
						},
					},
				},
			}
			d.Spec.Template.Spec.Volumes = []corev1.Volume{
				{
					Name: configVolumeName,
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: gatewayName,
							},
						},
					},
				},
				{
					Name: certificatesVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  resources.MLAGatewayCertificatesSecretName,
							DefaultMode: ptr.To[int32](0400),
						},
					},
				},
				{
					Name: caCertificatesVolumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: resources.MLAGatewayCASecretName,
							Items: []corev1.KeyToPath{
								{
									Key:  resources.MLAGatewayCACertKey,
									Path: resources.MLAGatewayCACertKey,
								},
							},
							DefaultMode: ptr.To[int32](0400),
						},
					},
				},
				{
					Name:         "tmp",
					VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
				},
			}

			return d, nil
		}
	}
}

// GatewayCAReconciler returns a function to create the ECDSA-based CA to be used for MLA Gateway.
func GatewayCAReconciler() reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.MLAGatewayCASecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			if data, exists := se.Data[resources.MLAGatewayCACertKey]; exists {
				certs, err := certutil.ParseCertsPEM(data)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate %s from existing secret %s: %w",
						resources.MLAGatewayCACertKey, resources.MLAGatewayCASecretName, err)
				}
				if !resources.CertWillExpireSoon(certs[0]) {
					return se, nil
				}
			}

			cert, key, err := certificates.GetECDSACACertAndKey()
			if err != nil {
				return nil, fmt.Errorf("failed to generate MLA CA: %w", err)
			}
			se.Data[resources.MLAGatewayCACertKey] = cert
			se.Data[resources.MLAGatewayCAKeyKey] = key

			return se, nil
		}
	}
}

// GatewayCertificateReconciler returns a function to create/update a secret with the MLA gateway TLS certificate.
func GatewayCertificateReconciler(c *kubermaticv1.Cluster, mlaGatewayCAGetter func() (*resources.ECDSAKeyPair, error)) reconciling.NamedSecretReconcilerFactory {
	return func() (string, reconciling.SecretReconciler) {
		return resources.MLAGatewayCertificatesSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := mlaGatewayCAGetter()
			if err != nil {
				return nil, fmt.Errorf("failed to get MLA Gateway ca: %w", err)
			}

			address := c.Status.Address
			if address.ExternalName == "" {
				return nil, fmt.Errorf("unable to issue MLA Gateway certificate: cluster ExternalName is empty")
			}
			commonName := resources.MLAGatewaySNIPrefix + address.ExternalName
			altNames := certutil.AltNames{
				DNSNames: []string{
					commonName,
					address.ExternalName, // required for NodePort expose strategy
				},
			}
			cIP := net.ParseIP(address.IP)
			if cIP != nil {
				altNames.IPs = []net.IP{cIP} // required for LoadBalancer expose strategy
			}
			if b, exists := se.Data[resources.MLAGatewayCertSecretKey]; exists {
				certs, err := certutil.ParseCertsPEM(b)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %w", resources.MLAGatewayCertSecretKey, err)
				}
				if resources.IsServerCertificateValidForAllOf(certs[0], commonName, altNames, ca.Cert) {
					return se, nil
				}
			}
			config := certutil.Config{
				CommonName: commonName,
				AltNames:   altNames,
				Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			}
			cert, key, err := certificates.GetSignedECDSACertAndKey(certificates.Duration365d, config, ca.Cert, ca.Key)
			if err != nil {
				return nil, fmt.Errorf("unable to sign the server certificate: %w", err)
			}

			se.Data[resources.MLAGatewayCertSecretKey] = cert
			se.Data[resources.MLAGatewayKeySecretKey] = key

			return se, nil
		}
	}
}
