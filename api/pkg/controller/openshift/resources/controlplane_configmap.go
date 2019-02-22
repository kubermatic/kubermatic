package resources

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/Masterminds/sprig"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/etcd"

	corev1 "k8s.io/api/core/v1"
)

const (
	openshiftControlPlaneConfigConfigMapName = "openshift-config"
	openshiftControlPlaneCertsBasePath       = "/etc/origin/certs"
	openshiftControlPlaneConfigbasePath      = "/etc/origin/master"
	openshiftContolPlaneConfigKeyName        = "master-config.yaml"
)

func OpenshiftControlPlaneConfigMapCreator(ctx context.Context,
	data openshiftData) (string, resources.ConfigMapCreator) {
	return openshiftControlPlaneConfigConfigMapName,
		func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			masterConfig, err := getMasterConfig(ctx, data)
			if err != nil {
				return nil, fmt.Errorf("failed to get master config :%v", err)
			}
			cm.Name = openshiftControlPlaneConfigConfigMapName
			cm.Labels = resources.BaseAppLabel(openshiftControlPlaneConfigConfigMapName, nil)
			cm.Data[openshiftContolPlaneConfigKeyName] = masterConfig
			cm.Data["policy.json"] = policyJSON

			return cm, nil
		}
}

type openshiftConfigInput struct {
	ETCDEndpoints []string
	ServiceCIDR   string
	ClusterCIDR   string
	ClusterURL    string
	DNSDomain     string
	CertsBasePath string
}

func getMasterConfig(ctx context.Context, data openshiftData) (string, error) {
	controlPlaneConfigBuffer := bytes.Buffer{}
	tmpl, err := template.New("base").Funcs(sprig.TxtFuncMap()).Parse(openshiftControlPlaneConfigTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse openshiftControlPlaneConfigTemplate: %v", err)
	}
	templateInput := openshiftConfigInput{
		ETCDEndpoints: etcd.GetClientEndpoints(data.Cluster().Status.NamespaceName),
		ServiceCIDR:   data.Cluster().Spec.ClusterNetwork.Services.CIDRBlocks[0],
		ClusterCIDR:   data.Cluster().Spec.ClusterNetwork.Pods.CIDRBlocks[0],
		ClusterURL:    data.Cluster().Address.URL,
		DNSDomain:     data.Cluster().Spec.ClusterNetwork.DNSDomain,
		CertsBasePath: openshiftControlPlaneCertsBasePath,
	}
	if err := tmpl.Execute(&controlPlaneConfigBuffer, templateInput); err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}
	return controlPlaneConfigBuffer.String(), nil
}

//TODO: Replace template with actual types in
// https://github.com/openshift/origin/pkg/cmd/server/apis/config/v1/types.go
const openshiftControlPlaneConfigTemplate = `
admissionConfig:
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
aggregatorConfig:
  proxyClientInfo:
    certFile: {{ .CertsBasePath }}/aggregator-front-proxy.crt
    keyFile: {{ .CertsBasePath }}/aggregator-front-proxy.key
apiLevels:
- v1
apiVersion: v1
authConfig:
  requestHeader:
    clientCA: {{ .CertsBasePath }}/front-proxy-ca.crt
    clientCommonNames:
    - aggregator-front-proxy
    extraHeaderPrefixes:
    - X-Remote-Extra-
    groupHeaders:
    - X-Remote-Group
    usernameHeaders:
    - X-Remote-User
controllerConfig:
  election:
    lockName: openshift-master-controllers
  serviceServingCert:
    signer:
      certFile: {{ .CertsBasePath }}/service-signer.crt
      keyFile: {{ .CertsBasePath }}/service-signer.key
controllers: '*'
corsAllowedOrigins:
## TODO: Fix up
- (?i)//127\.0\.0\.1(:|\z)
- (?i)//localhost(:|\z)
- (?i)//116\.203\.105\.73(:|\z)
- (?i)//kubernetes\.default(:|\z)
- (?i)//kubernetes\.default\.svc\.cluster\.local(:|\z)
- (?i)//kubernetes(:|\z)
- (?i)//openshift\.default(:|\z)
- (?i)//openshift\.default\.svc(:|\z)
- (?i)//172\.30\.0\.1(:|\z)
- (?i)//alvaro\-openshift\-controller(:|\z)
- (?i)//openshift\.default\.svc\.cluster\.local(:|\z)
- (?i)//kubernetes\.default\.svc(:|\z)
- (?i)//openshift(:|\z)
dnsConfig:
  bindAddress: 0.0.0.0:8053
  bindNetwork: tcp4
etcdClientInfo:
  ca: {{ .CertsBasePath }}/master.etcd-ca.crt
  certFile: {{ .CertsBasePath }}/master.etcd-client.crt
  keyFile: {{ .CertsBasePath }}/master.etcd-client.key
  urls: {{ range .ETCDEndpoints }}
  - "{{ . }}"
{{ end }}
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
  ca: {{ .CertsBasePath }}/ca-bundle.crt
  certFile: {{ .CertsBasePath }}/master.kubelet-client.crt
  keyFile: {{ .CertsBasePath }}/master.kubelet-client.key
  port: 10250
kubernetesMasterConfig:
  apiServerArguments:
    storage-backend:
    - etcd3
    storage-media-type:
    - application/vnd.kubernetes.protobuf
  controllerArguments:
    cluster-signing-cert-file:
    - /etc/origin/master/ca.crt
    cluster-signing-key-file:
    - /etc/origin/master/ca.key
    pv-recycler-pod-template-filepath-hostpath:
    - /etc/origin/master/recycler_pod.yaml
    pv-recycler-pod-template-filepath-nfs:
    - /etc/origin/master/recycler_pod.yaml
  masterCount: 1
	//TODO: Should we put something here?
  masterIP: ""
  podEvictionTimeout: null
  proxyClientInfo:
    certFile: {{ .CertsBasePath }}/master.proxy-client.crt
    keyFile: {{ .CertsBasePath }}/master.proxy-client.key
  schedulerArguments: null
  schedulerConfigFile: /etc/origin/master/scheduler.json
  servicesNodePortRange: ''
  servicesSubnet: "{{ .ServiceCIDR }}"
  staticNodeNames: []
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
  # TODO: Generate this
	# Must have: Subject: O=system:masters, CN=system:openshift-master
	# Must have X509v3 Extended Key Usage: TLS Web Client Authentication
  openshiftLoopbackKubeConfig: openshift-master.kubeconfig
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
oauthConfig:
  # TODO: Could be made nicer to be something listening on 443
  assetPublicURL: "{{ .ClusterURL }}/console"
  grantConfig:
    method: auto
  identityProviders:
  - challenge: true
    login: true
    mappingMethod: claim
    name: allow_all
    provider:
      apiVersion: v1
      kind: AllowAllPasswordIdentityProvider
  masterCA: /etc/origin/master/ca-bundle.crt
  masterPublicURL: "{{ .ClusterURL }}"
  masterURL: "{{ .ClusterURL }}"
  sessionConfig:
    sessionMaxAgeSeconds: 3600
    sessionName: ssn
    sessionSecretsFile: /etc/origin/master/session-secrets.yaml
  tokenConfig:
    accessTokenMaxAgeSeconds: 86400
    authorizeTokenMaxAgeSeconds: 500
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
  masterCA: {{ .CertsBasePath }}/ca-bundle.crt
  privateKeyFile: {{ .CertsBasePath }}/serviceaccounts.private.key
  publicKeyFiles:
  - serviceaccounts.public.key
servingInfo:
  bindAddress: 0.0.0.0:8443
  bindNetwork: tcp4
  certFile: master.server.crt
  clientCA: {{ .CertsBasePath }}/ca.crt
  keyFile: {{ .CertsBasePath }}/master.server.key
  maxRequestsInFlight: 500
  requestTimeoutSeconds: 3600
volumeConfig:
  dynamicProvisioningEnabled: true
`
