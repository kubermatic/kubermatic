package resources

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/resources/apiserver"
	"github.com/kubermatic/kubermatic/api/pkg/resources/etcd"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

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
	OIDCIssuerURL() string
	OIDCClientID() string
	OIDCClientSecret() string
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
				PodCIDR       string
				ServiceCIDR   string
				ListenPort    string
				ETCDEndpoints []string
			}{
				PodCIDR:       podCIDR,
				ServiceCIDR:   serviceCIDR,
				ListenPort:    fmt.Sprint(data.Cluster().Address.Port),
				ETCDEndpoints: etcd.GetClientEndpoints(data.Cluster().Status.NamespaceName),
			}
			if err := openshiftKubeAPIServerTemplate.Execute(&apiServerConfigBuffer, templateInput); err != nil {
				return nil, fmt.Errorf("failed to execute template: %v", err)
			}
			cm.Data[openshiftContolPlaneConfigKeyName] = apiServerConfigBuffer.String()
			return cm, nil

		}
	}
}

type oidcData struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
}

type openshiftConfigInput struct {
	ETCDEndpoints     []string
	ServiceCIDR       string
	ClusterCIDR       string
	ClusterURL        string
	DNSDomain         string
	ListenPort        string
	ControlPlaneType  string
	MasterIP          string
	CloudProviderName string
	OIDC              oidcData
}

func getMasterConfig(ctx context.Context, data masterConfigData, controlPlaneType string) (string, error) {
	controlPlaneConfigBuffer := bytes.Buffer{}
	tmpl, err := template.New("base").Funcs(sprig.TxtFuncMap()).Parse(openshiftControlPlaneConfigTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse openshiftControlPlaneConfigTemplate: %v", err)
	}
	apiserverListenPort, err := data.GetApiserverExternalNodePort(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get nodePort for apiserver: %v", err)
	}
	templateInput := openshiftConfigInput{
		ETCDEndpoints:     etcd.GetClientEndpoints(data.Cluster().Status.NamespaceName),
		ServiceCIDR:       data.Cluster().Spec.ClusterNetwork.Services.CIDRBlocks[0],
		ClusterCIDR:       data.Cluster().Spec.ClusterNetwork.Pods.CIDRBlocks[0],
		ClusterURL:        data.Cluster().Address.URL,
		DNSDomain:         data.Cluster().Spec.ClusterNetwork.DNSDomain,
		ListenPort:        fmt.Sprintf("%d", apiserverListenPort),
		ControlPlaneType:  controlPlaneType,
		MasterIP:          data.Cluster().Address.IP,
		CloudProviderName: apiserver.GetKubernetesCloudProviderName(data.Cluster()),
		OIDC: oidcData{
			IssuerURL:    data.OIDCIssuerURL(),
			ClientID:     data.OIDCClientID(),
			ClientSecret: data.OIDCClientSecret(),
		},
	}
	if err := tmpl.Execute(&controlPlaneConfigBuffer, templateInput); err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}
	return controlPlaneConfigBuffer.String(), nil
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
  cloud-provider:
  # TODO: Re-Enable
  # - aws
  enable-aggregator-routing:
  # Thist _must_ stay false, if its true, the kube-apiserver will try to resolve endpoints for
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
  enable-bootstrap-token-auth:
  - "true"
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
  # TODO: What is this? Looks like an additional auth webhook source?
  # oauthMetadataFile: /etc/kubernetes/static-pod-resources/configmaps/oauth-metadata/oauthMetadata
  requestHeader:
    clientCA: /etc/kubernetes/pki/front-proxy/client/ca.crt
    clientCommonNames:
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
  urls: {{ range .ETCDEndpoints }}
  - "{{ . }}"
{{- end }}
userAgentMatchingConfig:
  defaultRejectionMessage: ''
  deniedClients: null
  requiredClients: null
kubeClientConfig:
  kubeConfig: /etc/origin/master/loopback-kubeconfig/kubeconfig`

//TODO: Replace template with actual types in
// https://github.com/openshift/origin/pkg/cmd/server/apis/config/v1/types.go
const openshiftControlPlaneConfigTemplate = `admissionConfig:
  pluginConfig:
    BuildDefaults:
      configuration:
        apiVersion: v1
        env: []
        kind: BuildDefaultsConfig
        resources:
          limits: {}
          requests: {}
    BuildOverrides:
      configuration:
        apiVersion: v1
        kind: BuildOverridesConfig
    openshift.io/ImagePolicy:
      configuration:
        apiVersion: v1
        executionRules:
        - matchImageAnnotations:
          - key: images.openshift.io/deny-execution
            value: 'true'
          name: execution-denied
          onResources:
          - resource: pods
          - resource: builds
          reject: true
          skipOnResolutionFailure: true
        kind: ImagePolicyConfig
{{- if eq .ControlPlaneType "apiserver" }}
aggregatorConfig:
  proxyClientInfo:
    certFile: /etc/kubernetes/pki/front-proxy/client/apiserver-proxy-client.crt
    keyFile: /etc/kubernetes/pki/front-proxy/client/apiserver-proxy-client.key
{{- end }}
apiLevels:
- v1
apiVersion: v1
{{- if eq .ControlPlaneType "apiserver" }}
authConfig:
  requestHeader:
    clientCA: /etc/kubernetes/pki/front-proxy/client/ca.crt
    clientCommonNames:
    - aggregator-front-proxy
    extraHeaderPrefixes:
    - X-Remote-Extra-
    groupHeaders:
    - X-Remote-Group
    usernameHeaders:
    - X-Remote-User
{{- end }}
controllerConfig:
  election:
    lockName: openshift-master-controllers
  serviceServingCert:
    signer:
     # CA used for signing serving certs on demand for user workloads
     # https://github.com/openshift/service-serving-cert-signer
     # In theory this shouldn't be required, in practise the APIServer
     # panics on startup if not passed
     certFile: /etc/origin/master/service-signer-ca/ca.crt
     keyFile: /etc/origin/master/service-signer-ca/ca.key
controllers: '*'
corsAllowedOrigins:
## TODO: Fix up to contain all public addresses
- (?i)//127\.0\.0\.1(:|\z)
- (?i)//localhost(:|\z)
- (?i)//kubernetes\.default(:|\z)
- (?i)//kubernetes\.default\.svc\.cluster\.local(:|\z)
- (?i)//kubernetes(:|\z)
- (?i)//openshift\.default(:|\z)
- (?i)//openshift\.default\.svc(:|\z)
- (?i)//172\.30\.0\.1(:|\z)
- (?i)//openshift\.default\.svc\.cluster\.local(:|\z)
- (?i)//kubernetes\.default\.svc(:|\z)
- (?i)//openshift(:|\z)
dnsConfig:
  bindAddress: 0.0.0.0:8053
  bindNetwork: tcp4
etcdClientInfo:
{{- if eq .ControlPlaneType "apiserver" }}
  ca: /etc/etcd/pki/client/ca.crt
  certFile: /etc/etcd/pki/client/apiserver-etcd-client.crt
  keyFile: /etc/etcd/pki/client/apiserver-etcd-client.key
{{- end }}
  # Mandatory field, controller manager fails startup if unset
  urls: {{ range .ETCDEndpoints }}
  - "{{ . }}"
{{- end }}
etcdStorageConfig:
  kubernetesStoragePrefix: kubernetes.io
  kubernetesStorageVersion: v1
  openShiftStoragePrefix: openshift.io
  openShiftStorageVersion: v1
imageConfig:
  format: docker.io/openshift/origin-${component}:${version}
  latest: false
imagePolicyConfig:
  internalRegistryHostname: docker-registry.default.svc:5000
kind: MasterConfig
kubeletClientInfo:
{{- if eq .ControlPlaneType "apiserver" }}
  ca: /etc/kubernetes/pki/ca/ca.crt
  certFile: /etc/kubernetes/kubelet/kubelet-client.crt
  keyFile: /etc/kubernetes/kubelet/kubelet-client.key
{{- end }}
  # Port is required for the controller manager to start up
  port: 10250
kubernetesMasterConfig:
{{- if eq .ControlPlaneType "apiserver" }}
  apiServerArguments:
    storage-backend:
    - etcd3
    storage-media-type:
    - application/vnd.kubernetes.protobuf
    kubelet-preferred-address-types:
    - ExternalIP
    - InternalIP
{{- if .CloudProviderName }}
    cloud-provider:
      - "{{ .CloudProviderName }}"
    cloud-config:
      - "/etc/kubernetes/cloud/config"
{{- end }}
{{- end }}
  controllerArguments:
    cluster-signing-cert-file:
    - /etc/kubernetes/pki/ca/ca.crt
    cluster-signing-key-file:
    - /etc/kubernetes/pki/ca/ca.key
    pv-recycler-pod-template-filepath-hostpath:
    - /etc/origin/master/recycler_pod.yaml
    pv-recycler-pod-template-filepath-nfs:
    - /etc/origin/master/recycler_pod.yaml
{{- if .CloudProviderName }}
    cloud-provider:
      - "{{ .CloudProviderName }}"
    cloud-config:
      - "/etc/kubernetes/cloud/config"
{{- end }}
  # For some reason this field results in an error: Encountered config error json: unknown field "masterCount" in object *config.MasterConfig, raw JSON:
  #masterCount: 1
  masterIP: "{{ .MasterIP }}"
  podEvictionTimeout: null
{{- if eq .ControlPlaneType "apiserver" }}
  proxyClientInfo:
    certFile: /etc/kubernetes/pki/front-proxy/client/apiserver-proxy-client.crt
    keyFile: /etc/kubernetes/pki/front-proxy/client/apiserver-proxy-client.key
{{- end }}
  schedulerArguments: null
  schedulerConfigFile: /etc/origin/master/scheduler.json
  servicesNodePortRange: ''
  servicesSubnet: "{{ .ServiceCIDR }}"
masterClients:
  externalKubernetesClientConnectionOverrides:
    acceptContentTypes: application/vnd.kubernetes.protobuf,application/json
    burst: 400
    contentType: application/vnd.kubernetes.protobuf
    qps: 200
  externalKubernetesKubeConfig: ''
  openshiftLoopbackClientConnectionOverrides:
    acceptContentTypes: application/vnd.kubernetes.protobuf,application/json
    burst: 600
    contentType: application/vnd.kubernetes.protobuf
    qps: 300
  openshiftLoopbackKubeConfig: /etc/origin/master/loopback-kubeconfig/kubeconfig
masterPublicURL: "{{ .ClusterURL }}"
networkConfig:
  clusterNetworks:
  - cidr: "{{ .ClusterCIDR }}"
    # The number of bits to allocate per node subnet, e.G. 8 == hosts get a /24
    hostSubnetLength: 8
  externalIPNetworkCIDRs:
  - 0.0.0.0/0
  networkPluginName: redhat/openshift-ovs-subnet
  serviceNetworkCIDR: "{{ .ServiceCIDR }}"
{{- if eq .ControlPlaneType "apiserver" }}
# TODO: Get this running with dex
oauthConfig:
  alwaysShowProviderSelection: false
  grantConfig:
    method: auto
    serviceAccountMethod: prompt
  identityProviders:
    - challenge: true
      login: true
      mappingMethod: claim
      name: openid-connect
      provider:
        apiVersion: v1
        kind: OpenIDIdentityProvider
        clientID: {{ .OIDC.ClientID }}
        clientSecret: {{ .OIDC.ClientSecret }}
        ca: /etc/kubernetes/dex/ca/caBundle.pem
        claims:
          id:
            - sub
          preferredUsername:
            - name
          name:
            - name
          email:
            - email
        extraScopes:
          - email
          - profile
        urls:
          authorize: {{ .OIDC.IssuerURL }}/auth
          token: {{ .OIDC.IssuerURL }}/token
  masterCA: /etc/kubernetes/pki/ca/ca.crt
  assetPublicURL: "{{ .ClusterURL }}/console"
  masterPublicURL: "{{ .ClusterURL }}"
  masterURL: "{{ .ClusterURL }}"
  sessionConfig:
    sessionMaxAgeSeconds: 300
    sessionName: ssn
    sessionSecretsFile: ""
  templates: null
  tokenConfig:
    accessTokenMaxAgeSeconds: 86400
    authorizeTokenMaxAgeSeconds: 300
{{- end }}
pauseControllers: false
policyConfig:
  bootstrapPolicyFile: /etc/origin/master/policy.json
  openshiftInfrastructureNamespace: openshift-infra
  openshiftSharedResourcesNamespace: openshift
projectConfig:
  defaultNodeSelector: node-role.kubernetes.io/compute=true
  projectRequestMessage: ''
  projectRequestTemplate: ''
  securityAllocator:
    mcsAllocatorRange: s0:/2
    mcsLabelsPerProject: 5
    uidAllocatorRange: 1000000000-1999999999/10000
routingConfig:
  subdomain: router.default.svc.{{ .DNSDomain }}
serviceAccountConfig:
  limitSecretReferences: false
  managedNames:
  - default
  - builder
  - deployer
  masterCA: /etc/kubernetes/pki/ca/ca.crt
  privateKeyFile: /etc/kubernetes/service-account-key/sa.key
  publicKeyFiles:
  - /etc/kubernetes/service-account-key/sa.pub
servingInfo:
  bindAddress: 0.0.0.0:{{ .ListenPort }}
  bindNetwork: tcp4
  clientCA: /etc/kubernetes/pki/ca/ca.crt
  certFile: /etc/kubernetes/tls/apiserver-tls.crt
  keyFile: /etc/kubernetes/tls/apiserver-tls.key
  maxRequestsInFlight: 500
  requestTimeoutSeconds: 3600
volumeConfig:
  dynamicProvisioningEnabled: true`
