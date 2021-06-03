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
	"crypto/x509"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/resources/nodeportproxy"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	certutil "k8s.io/client-go/util/cert"
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
	listen                  8080 ssl;
	proxy_set_header        X-Scope-OrgID {{ .TenantID }};

	ssl_certificate         {{ .SSLCertFile }};
	ssl_certificate_key     {{ .SSLKeyFile }};
	ssl_verify_client       on;
	ssl_client_certificate  {{ .SSLCACertFile }};
	ssl_protocols           TLSv1.3;

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
}
`

const (
	gatewayName         = "mla-gateway"
	gatewayExternalName = resources.MLAGatewayExternalServiceName

	extPortName = "http-ext"
	intPortName = "http-int"

	configVolumeName         = "config"
	configVolumePath         = "/etc/nginx"
	certificatesVolumeName   = "gw-certificates"
	certificatesVolumePath   = "/etc/ssl/mla-gateway"
	caCertificatesVolumeName = "ca-certificates"
	caCertificatesVolumePath = "/etc/ssl/mla-gateway-ca"
)

type configTemplateData struct {
	Namespace     string
	TenantID      string
	SSLCertFile   string
	SSLKeyFile    string
	SSLCACertFile string
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

func GatewayConfigMapCreator(c *kubermaticv1.Cluster, mlaNamespace string) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return gatewayName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				configData := configTemplateData{
					Namespace:     mlaNamespace,
					TenantID:      c.Name,
					SSLCertFile:   fmt.Sprintf("%s/%s", certificatesVolumePath, resources.MLAGatewayCertSecretKey),
					SSLKeyFile:    fmt.Sprintf("%s/%s", certificatesVolumePath, resources.MLAGatewayKeySecretKey),
					SSLCACertFile: fmt.Sprintf("%s/%s", caCertificatesVolumePath, resources.MLAGatewayCACertKey),
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

func GatewayInternalServiceCreator() reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
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

func GatewayExternalServiceCreator(c *kubermaticv1.Cluster) reconciling.NamedServiceCreatorGetter {
	return func() (string, reconciling.ServiceCreator) {
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
					fmt.Sprintf(`{%q: %q}`, extPortName, resources.MLAGatewaySNIPrefix+c.Address.ExternalName)
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
	version = "1.19-alpine"
)

func GatewayDeploymentCreator(data *resources.TemplateData) reconciling.NamedDeploymentCreatorGetter {
	return func() (string, reconciling.DeploymentCreator) {
		return gatewayName, func(d *appsv1.Deployment) (*appsv1.Deployment, error) {
			d.Spec.Replicas = pointer.Int32Ptr(1)
			d.Spec.Selector = &metav1.LabelSelector{
				MatchLabels: map[string]string{
					common.NameLabel: gatewayName,
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
						Handler: corev1.Handler{
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
						ReadOnlyRootFilesystem:   pointer.BoolPtr(true),
						AllowPrivilegeEscalation: pointer.BoolPtr(false),
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
						{
							Name:      "docker-entrypoint-d-override",
							MountPath: "/docker-entrypoint.d",
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
							DefaultMode: pointer.Int32Ptr(0400),
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
							DefaultMode: pointer.Int32Ptr(0400),
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

// GatewayCACreator returns a function to create the ECDSA-based CA to be used for MLA Gateway.
func GatewayCACreator() reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.MLAGatewayCASecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			if data, exists := se.Data[resources.MLAGatewayCACertKey]; exists {
				certs, err := certutil.ParseCertsPEM(data)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate %s from existing secret %s: %v",
						resources.MLAGatewayCACertKey, resources.MLAGatewayCASecretName, err)
				}
				if !resources.CertWillExpireSoon(certs[0]) {
					return se, nil
				}
			}

			cert, key, err := certificates.GetECDSACACertAndKey()
			if err != nil {
				return nil, fmt.Errorf("failed to generate MLA CA: %v", err)
			}
			se.Data[resources.MLAGatewayCACertKey] = cert
			se.Data[resources.MLAGatewayCAKeyKey] = key

			return se, nil
		}
	}
}

// GatewayCertificateCreator returns a function to create/update a secret with the MLA gateway TLS certificate.
func GatewayCertificateCreator(c *kubermaticv1.Cluster, mlaGatewayCAGetter func() (*resources.ECDSAKeyPair, error)) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return resources.MLAGatewayCertificatesSecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}

			ca, err := mlaGatewayCAGetter()
			if err != nil {
				return nil, fmt.Errorf("failed to get MLA Gateway ca: %v", err)
			}
			commonName := resources.MLAGatewaySNIPrefix + c.Address.ExternalName
			altNames := certutil.AltNames{
				DNSNames: []string{
					commonName,
					c.Address.ExternalName, // required for NodePort expose strategy
				},
			}
			if b, exists := se.Data[resources.MLAGatewayCertSecretKey]; exists {
				certs, err := certutil.ParseCertsPEM(b)
				if err != nil {
					return nil, fmt.Errorf("failed to parse certificate (key=%s) from existing secret: %v", resources.MLAGatewayCertSecretKey, err)
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
				return nil, fmt.Errorf("unable to sign the server certificate: %v", err)
			}

			se.Data[resources.MLAGatewayCertSecretKey] = cert
			se.Data[resources.MLAGatewayKeySecretKey] = key

			return se, nil
		}
	}
}
