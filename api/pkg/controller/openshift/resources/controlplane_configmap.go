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
	openshiftAPIServerConfigMapName        = "openshift-config-apiserver"
	openshiftControllerMangerConfigMapName = "openshift-config-controller-manager"
	openshiftContolPlaneConfigKeyName      = "master-config.yaml"
)

func OpenshiftAPIServerConfigMapCreator(ctx context.Context, data openshiftData) resources.NamedConfigMapCreatorGetter {
	return func() (string, resources.ConfigMapCreator) {
		return openshiftAPIServerConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			masterConfig, err := getMasterConfig(ctx, data, "apiserver")
			if err != nil {
				return nil, fmt.Errorf("failed to get master config :%v", err)
			}
			cm.Labels = resources.BaseAppLabel(openshiftAPIServerConfigMapName, nil)
			cm.Data[openshiftContolPlaneConfigKeyName] = masterConfig
			cm.Data["policy.json"] = policyJSON
			cm.Data["scheduler.json"] = schedulerJSON

			return cm, nil
		}
	}
}

func OpenshiftControllerMangerConfigMapCreator(ctx context.Context, data openshiftData) resources.NamedConfigMapCreatorGetter {
	return func() (string, resources.ConfigMapCreator) {
		return openshiftControllerMangerConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}
			masterConfig, err := getMasterConfig(ctx, data, "controller-manager")
			if err != nil {
				return nil, fmt.Errorf("failed to get master config :%v", err)
			}
			cm.Labels = resources.BaseAppLabel(openshiftControllerMangerConfigMapName, nil)
			cm.Data[openshiftContolPlaneConfigKeyName] = masterConfig
			cm.Data["policy.json"] = policyJSON
			cm.Data["scheduler.json"] = schedulerJSON
			cm.Data["recycler_pod.yaml"] = recyclerPod

			return cm, nil
		}
	}
}

type openshiftConfigInput struct {
	ETCDEndpoints    []string
	ServiceCIDR      string
	ClusterCIDR      string
	ClusterURL       string
	DNSDomain        string
	ListenPort       string
	ControlPlaneType string
	MasterIP         string
}

func getMasterConfig(ctx context.Context, data openshiftData, controlPlaneType string) (string, error) {
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
		ETCDEndpoints:    etcd.GetClientEndpoints(data.Cluster().Status.NamespaceName),
		ServiceCIDR:      data.Cluster().Spec.ClusterNetwork.Services.CIDRBlocks[0],
		ClusterCIDR:      data.Cluster().Spec.ClusterNetwork.Pods.CIDRBlocks[0],
		ClusterURL:       data.Cluster().Address.URL,
		DNSDomain:        data.Cluster().Spec.ClusterNetwork.DNSDomain,
		ListenPort:       fmt.Sprintf("%d", apiserverListenPort),
		ControlPlaneType: controlPlaneType,
		MasterIP:         data.Cluster().Address.IP,
	}
	if err := tmpl.Execute(&controlPlaneConfigBuffer, templateInput); err != nil {
		return "", fmt.Errorf("failed to execute template: %v", err)
	}
	return controlPlaneConfigBuffer.String(), nil
}

const schedulerJSON = `
{"apiVersion":"v1","kind":"Policy","predicates":[{"name":"NoVolumeZoneConflict"},{"name":"MaxEBSVolumeCount"},{"name":"MaxGCEPDVolumeCount"},{"name":"MaxAzureDiskVolumeCount"},{"name":"MatchInterPodAffinity"},{"name":"NoDiskConflict"},{"name":"GeneralPredicates"},{"name":"PodToleratesNodeTaints"},{"name":"CheckNodeMemoryPressure"},{"name":"CheckNodeDiskPressure"},{"name":"CheckVolumeBinding"},{"argument":{"serviceAffinity":{"labels":["region"]}},"name":"Region"}],"priorities":[{"name":"SelectorSpreadPriority","weight":1},{"name":"InterPodAffinityPriority","weight":1},{"name":"LeastRequestedPriority","weight":1},{"name":"BalancedResourceAllocation","weight":1},{"name":"NodePreferAvoidPodsPriority","weight":10000},{"name":"NodeAffinityPriority","weight":1},{"name":"TaintTolerationPriority","weight":1},{"argument":{"serviceAntiAffinity":{"label":"zone"}},"name":"Zone","weight":2}]}`

const recyclerPod = `
apiVersion: v1
kind: Pod
metadata:
  name: recyler-pod-
  namespace: openshift-infra
spec:
  activeDeadlineSeconds: 60
  containers:
  - args:
    - /scrub
    command:
    - /usr/bin/openshift-recycle
    image: docker.io/openshift/origin-recycler:v3.11
    name: recyler-container
    securityContext:
      runAsUser: 0
    volumeMounts:
    - mountPath: /scrub
      name: vol
  restartPolicy: Never
  serviceAccountName: pv-recycler-controller
  volumes:
  - name: vol
`

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
{{ if eq .ControlPlaneType "apiserver" }}
aggregatorConfig:
  proxyClientInfo:
    certFile: /etc/kubernetes/pki/front-proxy/client/apiserver-proxy-client.crt
    keyFile: /etc/kubernetes/pki/front-proxy/client/apiserver-proxy-client.key
{{ end }}
apiLevels:
- v1
apiVersion: v1
{{ if eq .ControlPlaneType "apiserver" }}
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
{{ end }}
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
{{ if eq .ControlPlaneType "apiserver" }}
  ca: /etc/etcd/pki/client/ca.crt
  certFile: /etc/etcd/pki/client/apiserver-etcd-client.crt
  keyFile: /etc/etcd/pki/client/apiserver-etcd-client.key
{{ end }}
  # Mandatory field, controller manager fails startup if unset
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
{{ if eq .ControlPlaneType "apiserver" }}
  ca: /etc/kubernetes/pki/ca/ca.crt
  certFile: /etc/kubernetes/kubelet/kubelet-client.crt
  keyFile: /etc/kubernetes/kubelet/kubelet-client.key
{{ end }}
  # Port is required for the controller manager to start up
  port: 10250
kubernetesMasterConfig:
{{ if eq .ControlPlaneType "apiserver" }}
  apiServerArguments:
    storage-backend:
    - etcd3
    storage-media-type:
    - application/vnd.kubernetes.protobuf
    kubelet-preferred-address-types:
    - ExternalIP
    - InternalIP
{{ end }}
  controllerArguments:
    cluster-signing-cert-file:
    - /etc/kubernetes/pki/ca/ca.crt
    cluster-signing-key-file:
    - /etc/kubernetes/pki/ca/ca.key
    pv-recycler-pod-template-filepath-hostpath:
    - /etc/origin/master/recycler_pod.yaml
    pv-recycler-pod-template-filepath-nfs:
    - /etc/origin/master/recycler_pod.yaml
  # For some reason this field results in an error: Encountered config error json: unknown field "masterCount" in object *config.MasterConfig, raw JSON:
  #masterCount: 1
  masterIP: "{{ .MasterIP }}"
  podEvictionTimeout: null
{{ if eq .ControlPlaneType "apiserver" }}
  proxyClientInfo:
    certFile: /etc/kubernetes/pki/front-proxy/client/apiserver-proxy-client.crt
    keyFile: /etc/kubernetes/pki/front-proxy/client/apiserver-proxy-client.key
{{ end }}
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
# TODO: Get this running with dex
#oauthConfig:
#  # TODO: Could be made nicer to be something listening on 443
#  assetPublicURL: "{{ .ClusterURL }}/console"
#  grantConfig:
#    method: auto
#  identityProviders:
#  - challenge: true
#    login: true
#    mappingMethod: claim
#    name: allow_all
#    provider:
#      apiVersion: v1
#      kind: AllowAllPasswordIdentityProvider
#  masterCA: /etc/kubernetes/pki/ca/ca.crt
#  masterPublicURL: "{{ .ClusterURL }}"
#  masterURL: "{{ .ClusterURL }}"
#  sessionConfig:
#    sessionMaxAgeSeconds: 3600
#    sessionName: ssn
#    sessionSecretsFile: /etc/origin/master/session-secrets.yaml
#  tokenConfig:
#    accessTokenMaxAgeSeconds: 86400
#    authorizeTokenMaxAgeSeconds: 500
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
  dynamicProvisioningEnabled: true
`
