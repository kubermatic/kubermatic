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

package resources

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources/etcd"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
)

const (
	openshiftAPIServerConfigMapName     = "openshift-config-apiserver"
	openshiftKubeAPIServerConfigMapName = "openshift-config-kube-apiserver"
	openshiftContolPlaneConfigKeyName   = "master-config.yaml"
)

var (
	openshiftKubeAPIServerTemplate = template.Must(template.New("base").Funcs(sprig.TxtFuncMap()).Parse(openshiftKubeAPIServerConfigTemplate))
	openshiftAPIServerTemplate     = template.Must(template.New("base").Funcs(sprig.TxtFuncMap()).Parse(openshiftAPIServerConfigTemplate))
)

type masterConfigData interface {
	Cluster() *kubermaticv1.Cluster
	GetApiserverExternalNodePort(context.Context) (int32, error)
	GetKubernetesCloudProviderName() string
}

type openshiftAPIServerCreatorData interface {
	Cluster() *kubermaticv1.Cluster
}

func OpenshiftAPIServerConfigMapCreator(data openshiftAPIServerCreatorData) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return openshiftAPIServerConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			apiServerConfigBuffer := bytes.Buffer{}
			templateInput := struct {
				ETCDEndpoints []string
			}{
				ETCDEndpoints: etcd.GetClientEndpoints(data.Cluster().Status.NamespaceName),
			}
			if err := openshiftAPIServerTemplate.Execute(&apiServerConfigBuffer, templateInput); err != nil {
				return nil, fmt.Errorf("failed to execute template: %v", err)
			}
			cm.Data[openshiftContolPlaneConfigKeyName] = apiServerConfigBuffer.String()
			return cm, nil
		}
	}
}

func OpenshiftKubeAPIServerConfigMapCreator(data masterConfigData) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return openshiftKubeAPIServerConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}

			var podCIDR, serviceCIDR string
			if len(data.Cluster().Spec.ClusterNetwork.Pods.CIDRBlocks) > 0 {
				podCIDR = data.Cluster().Spec.ClusterNetwork.Pods.CIDRBlocks[0]
			}
			if len(data.Cluster().Spec.ClusterNetwork.Services.CIDRBlocks) > 0 {
				serviceCIDR = data.Cluster().Spec.ClusterNetwork.Services.CIDRBlocks[0]
			}

			apiServerConfigBuffer := bytes.Buffer{}
			templateInput := struct {
				PodCIDR          string
				ServiceCIDR      string
				ListenPort       string
				ETCDEndpoints    []string
				AdvertiseAddress string
				CloudProvider    string
			}{
				PodCIDR:          podCIDR,
				ServiceCIDR:      serviceCIDR,
				ListenPort:       fmt.Sprint(data.Cluster().Address.Port),
				ETCDEndpoints:    etcd.GetClientEndpoints(data.Cluster().Status.NamespaceName),
				AdvertiseAddress: data.Cluster().Address.IP,
				CloudProvider:    data.GetKubernetesCloudProviderName(),
			}
			if err := openshiftKubeAPIServerTemplate.Execute(&apiServerConfigBuffer, templateInput); err != nil {
				return nil, fmt.Errorf("failed to execute template: %v", err)
			}
			cm.Data[openshiftContolPlaneConfigKeyName] = apiServerConfigBuffer.String()
			return cm, nil

		}
	}
}

// TODO Use OpenShiftAPIServerConfig type from github.com/openshift/api/openshiftcontrolplane/v1/types.go
const openshiftAPIServerConfigTemplate = `aggregatorConfig:
  allowedNames:
  - apiserver-aggregator
  - kube-apiserver-proxy
  - system:kube-apiserver-proxy
  - system:openshift-aggregator
  clientCA: /var/run/configmaps/aggregator-client-ca/ca.crt
  extraHeaderPrefixes:
  - X-Remote-Extra-
  groupHeaders:
  - X-Remote-Group
  usernameHeaders:
  - X-Remote-User
apiServerArguments:
  minimal-shutdown-duration:
  - 3s
apiVersion: openshiftcontrolplane.config.openshift.io/v1
auditConfig:
  # TODO: Doesn't make much sense in a production setup, but useful for debugging
  auditFilePath: /var/log/openshift-apiserver/audit.log
  enabled: true
  logFormat: json
  maximumFileSizeMegabytes: 100
  maximumRetainedFiles: 10
  policyConfiguration:
    apiVersion: audit.k8s.io/v1beta1
    kind: Policy
    omitStages:
    - RequestReceived
    rules:
    - level: None
      resources:
      - group: ''
        resources:
        - events
    - level: None
      resources:
      - group: oauth.openshift.io
        resources:
        - oauthaccesstokens
        - oauthauthorizetokens
    - level: None
      nonResourceURLs:
      - /api*
      - /version
      - /healthz
      userGroups:
      - system:authenticated
      - system:unauthenticated
    - level: Metadata
      omitStages:
      - RequestReceived
imagePolicyConfig:
  internalRegistryHostname: image-registry.openshift-image-registry.svc:5000
kind: OpenShiftAPIServerConfig
projectConfig:
  projectRequestMessage: ""
routingConfig:
  # TODO: Fix
  subdomain: apps.openshift-test.aws.k8c.io
storageConfig:
  ca: /etc/etcd/pki/client/ca.crt
  certFile: /etc/etcd/pki/client/apiserver-etcd-client.crt
  keyFile: /etc/etcd/pki/client/apiserver-etcd-client.key
  urls:{{ range .ETCDEndpoints }}
  - "{{ . }}"
{{- end }}
servingInfo:
  {{- /* TODO: Use consts from resources package */}}
  certFile: /var/run/secrets/serving-cert/apiserver-tls.crt
  keyFile: /var/run/secrets/serving-cert/apiserver-tls.key
kubeClientConfig:
  kubeConfig: /etc/origin/master/kubeconfig/kubeconfig
`

const openshiftKubeAPIServerConfigTemplate = `admission:
  pluginConfig:
    network.openshift.io/RestrictedEndpointsAdmission:
      configuration:
        apiVersion: network.openshift.io/v1
        kind: RestrictedEndpointsAdmissionConfig
        restrictedCIDRs:
        - {{ .PodCIDR }}
        - {{ .ServiceCIDR }}
aggregatorConfig:
  proxyClientInfo:
    certFile: /etc/kubernetes/pki/front-proxy/client/apiserver-proxy-client.crt
    keyFile: /etc/kubernetes/pki/front-proxy/client/apiserver-proxy-client.key
apiServerArguments:
{{- if .CloudProvider }}
  cloud-provider:
  - {{ .CloudProvider}}
  cloud-config:
  - /etc/kubernetes/cloud/config
{{- end }}
  enable-aggregator-routing:
  # This _must_ stay false, if its true, the kube-apiserver will try to resolve endpoints for
  # the services that service the extension apis and error out because they are of type
  # ExternalName
  - 'false'
  feature-gates:
  - ExperimentalCriticalPodAnnotation=true
  - RotateKubeletServerCertificate=true
  - SupportPodPidsLimit=true
  - LocalStorageCapacityIsolation=false
  http2-max-streams-per-connection:
  - '2000'
  minimal-shutdown-duration:
  - 70s
  storage-backend:
  - etcd3
  storage-media-type:
  - application/vnd.kubernetes.protobuf
  advertise-address:
  - {{ .AdvertiseAddress }}
  kubelet-preferred-address-types:
  - InternalIP
apiVersion: kubecontrolplane.config.openshift.io/v1
auditConfig:
  # TODO: Doesn't make much sense in a production setup, but useful for debugging
  auditFilePath: /var/log/kube-apiserver/audit.log
  enabled: true
  logFormat: json
  maximumFileSizeMegabytes: 100
  maximumRetainedFiles: 10
  policyConfiguration:
    apiVersion: audit.k8s.io/v1beta1
    kind: Policy
    omitStages:
    - RequestReceived
    rules:
    - level: None
      resources:
      - group: ''
        resources:
        - events
    - level: None
      resources:
      - group: oauth.openshift.io
        resources:
        - oauthaccesstokens
        - oauthauthorizetokens
    - level: None
      nonResourceURLs:
      - /api*
      - /version
      - /healthz
      - /readyz
      userGroups:
      - system:authenticated
      - system:unauthenticated
    - level: Metadata
      omitStages:
      - RequestReceived
authConfig:
  oauthMetadataFile: /etc/kubernetes/oauth-metadata/oauthMetadata
  requestHeader:
    clientCA: /etc/kubernetes/pki/front-proxy/client/ca.crt
    clientCommonNames:
    - apiserver-aggregator
    - kube-apiserver-proxy
    - system:kube-apiserver-proxy
    - system:openshift-aggregator
    extraHeaderPrefixes:
    - X-Remote-Extra-
    groupHeaders:
    - X-Remote-Group
    usernameHeaders:
    - X-Remote-User
  webhookTokenAuthenticators: null
consolePublicURL: ''
corsAllowedOrigins:
- //127\.0\.0\.1(:|$)
- //localhost(:|$)
imagePolicyConfig:
  internalRegistryHostname: image-registry.openshift-image-registry.svc:5000
kind: KubeAPIServerConfig
kubeletClientInfo:
  ca: /etc/kubernetes/pki/ca/ca.crt
  certFile: /etc/kubernetes/kubelet/kubelet-client.crt
  keyFile: /etc/kubernetes/kubelet/kubelet-client.key
  port: 10250
projectConfig:
  defaultNodeSelector: ''
serviceAccountPublicKeyFiles:
- /etc/kubernetes/service-account-key/sa.pub
servicesNodePortRange: 30000-32767
servicesSubnet: {{ .ServiceCIDR }}
servingInfo:
  bindAddress: 0.0.0.0:{{ .ListenPort }}
  bindNetwork: tcp4
  clientCA: /etc/kubernetes/pki/ca/ca.crt
  certFile: /etc/kubernetes/tls/apiserver-tls.crt
  keyFile: /etc/kubernetes/tls/apiserver-tls.key
  maxRequestsInFlight: 1200
  namedCertificates: []
 # TODO: What are they needed for? Additional serving certs?
 # - certFile: /etc/kubernetes/static-pod-certs/secrets/localhost-serving-cert-certkey/tls.crt
 #   keyFile: /etc/kubernetes/static-pod-certs/secrets/localhost-serving-cert-certkey/tls.key
 # - certFile: /etc/kubernetes/static-pod-certs/secrets/service-network-serving-certkey/tls.crt
 #   keyFile: /etc/kubernetes/static-pod-certs/secrets/service-network-serving-certkey/tls.key
 # - certFile: /etc/kubernetes/static-pod-certs/secrets/external-loadbalancer-serving-certkey/tls.crt
 #   keyFile: /etc/kubernetes/static-pod-certs/secrets/external-loadbalancer-serving-certkey/tls.key
 # - certFile: /etc/kubernetes/static-pod-certs/secrets/internal-loadbalancer-serving-certkey/tls.crt
 #   keyFile: /etc/kubernetes/static-pod-certs/secrets/internal-loadbalancer-serving-certkey/tls.key
  requestTimeoutSeconds: 3600
storageConfig:
  ca: /etc/etcd/pki/client/ca.crt
  certFile: /etc/etcd/pki/client/apiserver-etcd-client.crt
  keyFile: /etc/etcd/pki/client/apiserver-etcd-client.key
  urls:{{ range .ETCDEndpoints }}
  - "{{ . }}"
{{- end }}
userAgentMatchingConfig:
  defaultRejectionMessage: ''
  deniedClients: null
  requiredClients: null
kubeClientConfig:
  kubeConfig: /etc/origin/master/loopback-kubeconfig/kubeconfig`
