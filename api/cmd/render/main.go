package main

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path"
	"strings"
	"text/template"

	"encoding/base64"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/cmd/render/internal"
	"github.com/kubermatic/kubermatic/api/pkg/controller/resources"
	"github.com/kubermatic/kubermatic/api/pkg/controller/version"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubernetes-incubator/bootkube/pkg/tlsutil"
	v12 "k8s.io/api/core/v1"
	v13 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// The name of the k8s service that selects self-hosted etcd pods
	EtcdServiceName = "etcd-service"

	SecretEtcdPeer   = "etcd-peer-tls"
	SecretEtcdServer = "etcd-server-tls"
	SecretEtcdClient = "etcd-client-tls"

	NetworkCalico = "experimental-calico"
	NetworkCanal  = "experimental-canal"

	secretNamespace     = "kube-system"
	secretAPIServerName = "kube-apiserver"
	secretCMName        = "kube-controller-manager"
)

const (
	apiOffset              = 1
	dnsOffset              = 10
	etcdOffset             = 15
	bootEtcdOffset         = 20
	defaultApiServers      = "https://127.0.0.1:443"
	defaultEtcdServers     = "https://127.0.0.1:2379"
	defaultEtcdServiceName = "etcd-service"
	defaultAltNames        = ""
	defaultServiceBaseIP   = "10.3.0.0"
	defaultPodCIDR         = "10.2.0.0/16"
	defaultServiceCIDR     = "10.3.0.0/24"
	bootstrapSecretsDir    = "/etc/kubernetes/bootstrap-secrets" // Overridden for testing.
)

var (
	masterResources = flag.String("master-resources", "", "The master resources path (Required).")
	versionsFile    = flag.String("versions", "versions.yaml", "The versions.yaml file path")
	updatesFile     = flag.String("updates", "updates.yaml", "The updates.yaml file path")
	clusterManifest = flag.String("cluster-manifest", "", "Manifest of the cluster to be rendered")
	outputFolder    = flag.String("out", "./_out", "Output path")
)

const (
	AddonManagerDeploymentName      = "addon-manager-dep.yaml"
	ApiserverDeploymentName         = "apiserver-dep.yaml"
	ApiserverExternalServiceName    = "apiserver-external-service.yaml"
	ApiserverIngressName            = "apiserver-ingress.yaml"
	ApiserverSecretName             = "apiserver-secret.yaml"
	ApiserverServiceName            = "apiserver-service.yaml"
	ControllerManagerDeploymentName = "controller-manager-dep.yaml"
	ControllerManagerSecretName     = "controller-manager-secret.yaml"
	EtcdClusterName                 = "etcd-cluster.yaml"
	EtcdOperatorDeploymentName      = "etcd-operator-dep.yaml"
	EtcdOperatorRolebindingName     = "etcd-operator-rolebinding.yaml"
	EtcdOperatorServiceAccountName  = "etcd-operator-serviceaccount.yaml"
	NodeControllerDeploymentName    = "node-controller-dep.yaml"
	SchedulerDeploymentName         = "scheduler-dep.yaml"
)

func must(err error) {
	if err != nil {
		glog.Fatal(err)
	}
}

func mustReadKubermaticManifest() *v1.Cluster {
	buf, err := ioutil.ReadFile(*clusterManifest)
	must(err)
	var sa v1.Cluster
	must(yaml.Unmarshal(buf, &sa))
	return &sa
}

// ImageVersions holds all the images (and their versions) that are rendered into the templates.
type ImageVersions struct {
	Etcd            string
	EtcdOperator    string
	Flannel         string
	FlannelCNI      string
	Calico          string
	CalicoCNI       string
	Hyperkube       string
	Kenc            string
	KubeDNS         string
	KubeDNSMasq     string
	KubeDNSSidecar  string
	PodCheckpointer string
}

// DefaultImages are the defualt images bootkube components use.
var DefaultImages = ImageVersions{
	Etcd:            "quay.io/coreos/etcd:v3.1.8",
	EtcdOperator:    "quay.io/coreos/etcd-operator:v0.5.0",
	Flannel:         "quay.io/coreos/flannel:v0.9.1-amd64",
	FlannelCNI:      "quay.io/coreos/flannel-cni:v0.3.0",
	Calico:          "quay.io/calico/node:v2.6.3",
	CalicoCNI:       "quay.io/calico/cni:v1.11.1",
	Hyperkube:       "gcr.io/google_containers/hyperkube:v1.8.4",
	Kenc:            "quay.io/coreos/kenc:0.0.2",
	KubeDNS:         "gcr.io/google_containers/k8s-dns-kube-dns-amd64:1.14.5",
	KubeDNSMasq:     "gcr.io/google_containers/k8s-dns-dnsmasq-nanny-amd64:1.14.5",
	KubeDNSSidecar:  "gcr.io/google_containers/k8s-dns-sidecar-amd64:1.14.5",
	PodCheckpointer: "quay.io/coreos/pod-checkpointer:e22cc0e3714378de92f45326474874eb602ca0ac",
}

// AssetConfig holds all configuration needed when generating
// the default set of assets.
type Config struct {
	EtcdCACert             *x509.Certificate
	EtcdClientCert         *x509.Certificate
	EtcdClientKey          *rsa.PrivateKey
	EtcdServers            []*url.URL
	EtcdUseTLS             bool
	APIServers             []*url.URL
	CACert                 *x509.Certificate
	CAPrivKey              *rsa.PrivateKey
	AltNames               *tlsutil.AltNames
	PodCIDR                *net.IPNet
	ServiceCIDR            *net.IPNet
	APIServiceIP           net.IP
	BootEtcdServiceIP      net.IP
	DNSServiceIP           net.IP
	EtcdServiceIP          net.IP
	EtcdServiceName        string
	SelfHostedEtcd         bool
	CloudProvider          string
	BootstrapSecretsSubdir string
	Images                 ImageVersions
}

func main() {
	flag.Parse()
	defer glog.Flush()
	os.MkdirAll(*outputFolder, os.ModeDir|os.ModePerm)

	cluster := mustReadKubermaticManifest()
	templates := internal.DefaultInternalTemplateContent()
	// load versions
	versions, err := version.LoadVersions(*versionsFile)
	if err != nil {
		glog.Fatal(fmt.Sprintf("failed to load version yaml %q: %v", *versionsFile, err))
	}
	// load updates
	updates, err := version.LoadUpdates(*updatesFile)
	if err != nil {
		glog.Fatal(fmt.Sprintf("failed to load version yaml %q: %v", *versionsFile, err))
	}
	manifests := &internal.Manifests{}

	mustRenderBootkubeTemplatesInto(manifests, cluster, templates, versions, updates, "")
	mustRenderKubermaticTemplatesInto(manifests, cluster, templates, versions, updates, "")
	//mustRenderKubermaticTemplateFiles(cluster, templates, versions, updates, "")

	must(manifests.WriteToDir(*outputFolder))
	glog.Info("Ended Successfully!")
}

func mustCreateAssetFromTemplate(tpl []byte, data interface{}) []byte {
	t, err := template.New("t").Parse(string(tpl))
	must(err)
	b := &bytes.Buffer{}
	t.Execute(b, data)
	return b.Bytes()
}

func mustRenderBootkubeTemplatesInto(m *internal.Manifests, cluster *v1.Cluster, content *internal.TemplateContent, versions map[string]*api.MasterVersion, updates []api.MasterUpdate, dc string) {
	conf := translateClusterToBootkube(cluster)
	buf, err := yaml.Marshal(cluster.GetKubeconfig())
	must(err)
	m.KubeConfig = buf

	// Core components
	m.ControllerManager = mustCreateAssetFromTemplate(content.ControllerManagerTemplate, conf)
	m.APIServer = mustCreateAssetFromTemplate(content.APIServerTemplate, conf)
	m.Proxy = mustCreateAssetFromTemplate(content.ProxyTemplate, conf)
	m.ProxyRoleBinding = mustCreateAssetFromTemplate(content.ProxyRoleBindingTemplate, conf)
	m.ProxySA = mustCreateAssetFromTemplate(content.ProxySATemplate, conf)
	m.KubeDNSSvc = mustCreateAssetFromTemplate(content.DNSSvcTemplate, conf)
	m.BootstrapAPIServer = mustCreateAssetFromTemplate(content.BootstrapAPIServerTemplate, conf)
	m.BootstrapControllerManager = mustCreateAssetFromTemplate(content.BootstrapControllerManagerTemplate, conf)
	m.BootstrapScheduler = mustCreateAssetFromTemplate(content.BootstrapSchedulerTemplate, conf)

	// Canal
	m.CalicoCfg = mustCreateAssetFromTemplate(content.CalicoCfgTemplate, conf)
	m.CalicoRole = mustCreateAssetFromTemplate(content.CalicoRoleTemplate, conf)
	m.CalicoRoleBinding = mustCreateAssetFromTemplate(content.CalicoRoleBindingTemplate, conf)
	m.CalicoSA = mustCreateAssetFromTemplate(content.CalicoSATemplate, conf)
	m.CalicoPolicyOnly = mustCreateAssetFromTemplate(content.CalicoPolicyOnlyTemplate, conf)
	m.CalicoBGPConfigsCRD = mustCreateAssetFromTemplate(content.CalicoBGPConfigsCRDTemplate, conf)
	m.CalicoFelixConfigsCRD = mustCreateAssetFromTemplate(content.CalicoFelixConfigsCRDTemplate, conf)
	m.CalicoNetworkPoliciesCRD = mustCreateAssetFromTemplate(content.CalicoNetworkPoliciesCRDTemplate, conf)
	m.CalicoIPPoolsCRD = mustCreateAssetFromTemplate(content.CalicoIPPoolsCRDTemplate, conf)

	// Self hosted etcd
	m.EtcdOperator = mustCreateAssetFromTemplate(content.EtcdOperatorTemplate, conf)
	m.EtcdSvc = mustCreateAssetFromTemplate(content.EtcdSvcTemplate, conf)
	m.Kenc = mustCreateAssetFromTemplate(content.KencTemplate, conf)
	m.BootstrapEtcd = mustCreateAssetFromTemplate(content.BootstrapEtcdTemplate, conf)
	m.BootstrapEtcdService = mustCreateAssetFromTemplate(content.BootstrapEtcdSvcTemplate, conf)
	m.MigrateEtcdCluster = mustCreateAssetFromTemplate(content.EtcdCRDTemplate, conf)

	// Static templates
	m.Scheduler = mustCreateAssetFromTemplate(content.SchedulerTemplate, conf)
	m.SchedulerDisruption = mustCreateAssetFromTemplate(content.SchedulerDisruptionTemplate, conf)
	m.ControllerManagerDisruption = mustCreateAssetFromTemplate(content.ControllerManagerDisruptionTemplate, conf)
	m.KubeDNSDeployment = mustCreateAssetFromTemplate(content.DNSDeploymentTemplate, conf)
	m.Checkpointer = mustCreateAssetFromTemplate(content.CheckpointerTemplate, conf)
	m.CheckpointerSA = mustCreateAssetFromTemplate(content.CheckpointerSATemplate, conf)
	m.CheckpointerRole = mustCreateAssetFromTemplate(content.CheckpointerRoleTemplate, conf)
	m.CheckpointerRoleBinding = mustCreateAssetFromTemplate(content.CheckpointerRoleBindingTemplate, conf)
	m.KubeSystemSARoleBinding = mustCreateAssetFromTemplate(content.KubeSystemSARoleBindingTemplate, conf)

	m.CACert = cluster.Status.RootCA.Cert
	m.CAKey = cluster.Status.RootCA.Key
	m.KubeletCert = cluster.Status.KubeletCert.Cert
	m.KubeletKey = cluster.Status.KubeletCert.Key
	m.APIServerCert = cluster.Status.ApiserverCert.Cert
	m.APIServerKey = cluster.Status.ApiserverCert.Key
	//m.ServiceAccountPrivKey
	//m.ServiceAccountPubKey

	// Shitty parts
	m.KubeConfigInCluster, m.KubeConfig = newKubeConfigs(m, conf, content)
	m.EtcdPeerSecret = etcdPeerSecrets(m)
	m.EtcdServerSecret = etcdServerSecrets(m)
	m.EtcdClientSecret = etcdClientSecrets(m)
	m.ControllerManagerSecret = controllerManagerSecrets(m)
	m.APIServerSecret = apiServerSecrets(m)

	// TODO(realfake): bootstrap etcd is missing
}

func newKubeConfigs(manifests *internal.Manifests, c *Config, t *internal.TemplateContent) (internal, external []byte) {
	cfg := struct {
		Server      string
		CACert      string
		KubeletCert string
		KubeletKey  string
	}{
		Server:      c.APIServers[0].String(),
		CACert:      base64.StdEncoding.EncodeToString(manifests.CACert),
		KubeletCert: base64.StdEncoding.EncodeToString(manifests.KubeletCert),
		KubeletKey:  base64.StdEncoding.EncodeToString(manifests.KubeletKey),
	}
	return mustCreateAssetFromTemplate(t.KubeConfigInClusterTemplate, cfg), mustCreateAssetFromTemplate(t.KubeConfigTemplate, cfg)
}

func etcdPeerSecrets(c *internal.Manifests) []byte {
	s := v12.Secret{
		Type: v12.SecretTypeOpaque,
		TypeMeta: v13.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: v13.ObjectMeta{
			Name:      SecretEtcdPeer,
			Namespace: secretNamespace,
		},
	}
	s.Data = map[string][]byte{
		path.Base(internal.AssetPathEtcdPeerCA):   []byte(base64.StdEncoding.EncodeToString(c.EtcdPeerCA)),
		path.Base(internal.AssetPathEtcdPeerCert): []byte(base64.StdEncoding.EncodeToString(c.EtcdPeerCert)),
		path.Base(internal.AssetPathEtcdPeerKey):  []byte(base64.StdEncoding.EncodeToString(c.EtcdPeerKey)),
	}
	data, err := yaml.Marshal(s)
	must(err)
	return data
}

func etcdServerSecrets(c *internal.Manifests) []byte {
	s := v12.Secret{
		Type: v12.SecretTypeOpaque,
		TypeMeta: v13.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: v13.ObjectMeta{
			Name:      SecretEtcdServer,
			Namespace: secretNamespace,
		},
	}
	s.Data = map[string][]byte{
		path.Base(internal.AssetPathEtcdServerCA):   []byte(base64.StdEncoding.EncodeToString(c.EtcdServerCA)),
		path.Base(internal.AssetPathEtcdServerCert): []byte(base64.StdEncoding.EncodeToString(c.EtcdServerCert)),
		path.Base(internal.AssetPathEtcdServerKey):  []byte(base64.StdEncoding.EncodeToString(c.EtcdServerKey)),
	}
	data, err := yaml.Marshal(s)
	must(err)
	return data
}

func etcdClientSecrets(c *internal.Manifests) []byte {
	s := v12.Secret{
		Type: v12.SecretTypeOpaque,
		TypeMeta: v13.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: v13.ObjectMeta{
			Name:      SecretEtcdClient,
			Namespace: secretNamespace,
		},
	}
	s.Data = map[string][]byte{
		path.Base(internal.AssetPathEtcdClientCA):   []byte(base64.StdEncoding.EncodeToString(c.EtcdClientCA)),
		path.Base(internal.AssetPathEtcdClientCert): []byte(base64.StdEncoding.EncodeToString(c.EtcdClientCert)),
		path.Base(internal.AssetPathEtcdClientKey):  []byte(base64.StdEncoding.EncodeToString(c.EtcdClientKey)),
	}
	data, err := yaml.Marshal(s)
	must(err)
	return data
}

func controllerManagerSecrets(c *internal.Manifests) []byte {
	s := v12.Secret{
		Type: v12.SecretTypeOpaque,
		TypeMeta: v13.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: v13.ObjectMeta{
			Name:      secretCMName,
			Namespace: secretNamespace,
		},
	}
	s.Data = map[string][]byte{
		path.Base(internal.AssetPathServiceAccountPrivKey): []byte(base64.StdEncoding.EncodeToString(c.ServiceAccountPrivKey)),
		path.Base(internal.AssetPathCACert):                []byte(base64.StdEncoding.EncodeToString(c.CACert)),
	}
	data, err := yaml.Marshal(s)
	must(err)
	return data
}

func apiServerSecrets(c *internal.Manifests) []byte {
	s := v12.Secret{
		Type: v12.SecretTypeOpaque,
		TypeMeta: v13.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: v13.ObjectMeta{
			Name:      secretAPIServerName,
			Namespace: secretNamespace,
		},
	}
	s.Data = map[string][]byte{
		path.Base(internal.AssetPathAPIServerKey):         []byte(base64.StdEncoding.EncodeToString(c.APIServerKey)),
		path.Base(internal.AssetPathAPIServerCert):        []byte(base64.StdEncoding.EncodeToString(c.APIServerCert)),
		path.Base(internal.AssetPathServiceAccountPubKey): []byte(base64.StdEncoding.EncodeToString(c.ServiceAccountPubKey)),
		path.Base(internal.AssetPathCACert):               []byte(base64.StdEncoding.EncodeToString(c.CACert)),

		// UseTLS
		path.Base(internal.AssetPathEtcdPeerCA): []byte(base64.StdEncoding.EncodeToString(c.EtcdPeerCA)),
	}
	data, err := yaml.Marshal(s)
	must(err)
	return data
}

func translateClusterToBootkube(cluster *v1.Cluster) *Config {
	providerName, err := provider.ClusterCloudProviderName(cluster.Spec.Cloud)
	must(err)

	apiServers, err := parseURLs(defaultServiceCIDR)

	altNames, err := parseAltNames(defaultAltNames)
	if altNames == nil {
		// Fall back to parsing from api-server list
		altNames = altNamesFromURLs(apiServers)
	}

	_, podNet, err := net.ParseCIDR(defaultPodCIDR)
	must(err)

	_, serviceNet, err := net.ParseCIDR(defaultServiceCIDR)
	must(err)

	if podNet.Contains(serviceNet.IP) || serviceNet.Contains(podNet.IP) {
		must(fmt.Errorf("Pod CIDR %s and service CIDR %s must not overlap", podNet.String(), serviceNet.String()))
	}

	apiServiceIP, err := offsetServiceIP(serviceNet, apiOffset)
	must(err)

	dnsServiceIP, err := offsetServiceIP(serviceNet, dnsOffset)

	bootEtcdServiceIP, err := offsetServiceIP(serviceNet, bootEtcdOffset)
	must(err)

	etcdServiceIP, err := offsetServiceIP(serviceNet, etcdOffset)
	must(err)

	var etcdServers []*url.URL
	etcdServerUrl, err := url.Parse(fmt.Sprintf("https://%s:2379", etcdServiceIP))
	must(err)
	etcdServers = append(etcdServers, etcdServerUrl)

	key, crt, err := parseCertAndPrivateKeyFromString(string(cluster.Status.ApiserverCert.Cert), string(cluster.Status.ApiserverCert.Key))
	must(err)

	// TODO: Find better option than asking users to make manual changes
	if serviceNet.IP.String() != defaultServiceBaseIP {
		fmt.Printf("You have selected a non-default service CIDR %s - be sure your kubelet service file uses --cluster-dns=%s\n", serviceNet.String(), dnsServiceIP.String())
	}

	c := &Config{
		EtcdServers:            etcdServers,
		EtcdUseTLS:             false,
		APIServers:             apiServers,
		CACert:                 crt,
		CAPrivKey:              key,
		AltNames:               altNames,
		PodCIDR:                podNet,
		ServiceCIDR:            serviceNet,
		APIServiceIP:           apiServiceIP,
		BootEtcdServiceIP:      bootEtcdServiceIP,
		DNSServiceIP:           dnsServiceIP,
		EtcdServiceIP:          etcdServiceIP,
		EtcdServiceName:        defaultEtcdServiceName,
		SelfHostedEtcd:         true,
		CloudProvider:          providerName,
		BootstrapSecretsSubdir: path.Base(bootstrapSecretsDir),
		Images:                 DefaultImages,
	}

	// Add kube-apiserver service IP
	c.AltNames.IPs = append(c.AltNames.IPs, c.APIServiceIP)

	return c
}

func mustRenderKubermaticTemplatesInto(m *internal.Manifests, cluster *v1.Cluster, content *internal.TemplateContent, versions map[string]*api.MasterVersion, updates []api.MasterUpdate, dc string) {
	ns := cluster.Status.NamespaceName
	masterVersion, found := versions[cluster.Spec.MasterVersion]
	if !found {
		must(fmt.Errorf("unknown new cluster %q master version %q", cluster.Name, cluster.Spec.MasterVersion))
	}

	// Deployments
	{
		deps := map[*[]byte]string{
			&(m.EtcdOperator):      masterVersion.EtcdOperatorDeploymentYaml,
			&(m.APIServer):         masterVersion.ApiserverDeploymentYaml,
			&(m.ControllerManager): masterVersion.ControllerDeploymentYaml,
			&(m.Scheduler):         masterVersion.SchedulerDeploymentYaml,
			// TODO(realfake): Implement
			//&(m.NOTIMPLEMENTED): masterVersion.NodeControllerDeploymentYaml,
			//&(m.NOTIMPLEMENTED): masterVersion.AddonManagerDeploymentYaml,
		}

		for file, app := range deps {
			dep, err := resources.LoadDeploymentFile(cluster, masterVersion, *masterResources, dc, app)
			must(err)

			dep.SetNamespace(ns)
			buf, err := yaml.Marshal(dep)
			must(err)

			b := &bytes.Buffer{}
			b.Write(*file)
			b.WriteString("\n---\n")
			b.Write(buf)
			*file = b.Bytes()
		}
	}

	// Services
	{
		services := map[*[]byte]string{
			&(m.APIServer): "apiserver",
			&(m.APIServer): "apiserver-external",
		}
		for file, app := range services {
			dep, err := resources.LoadServiceFile(cluster, app, *masterResources)
			must(err)

			dep.SetNamespace(ns)
			buf, err := yaml.Marshal(dep)
			must(err)

			b := &bytes.Buffer{}
			b.Write(*file)
			b.WriteString("\n---\n")
			b.Write(buf)
			*file = b.Bytes()
		}
	}

	// Secrets
	{
		secrets := map[*[]byte]string{
			&(m.APIServerSecret):         "apiserver",
			&(m.ControllerManagerSecret): "controller-manager",
		}
		for file, app := range secrets {
			dep, err := resources.LoadSecretFile(cluster, app, *masterResources)
			must(err)

			dep.SetNamespace(ns)
			buf, err := yaml.Marshal(dep)
			must(err)

			b := &bytes.Buffer{}
			b.Write(*file)
			b.WriteString("\n---\n")
			b.Write(buf)
			*file = b.Bytes()
		}
	}

	/*
		// Ingress
		{
			secrets := map[*[]byte]string{
				ApiserverIngressName: "apiserver",
			}
			for file, app := range secrets {
				dep, err := resources.LoadIngressFile(cluster, app, *masterResources)
				must(err)

				dep.SetNamespace(ns)
				buf, err := yaml.Marshal(dep)
				must(err)

				b := &bytes.Buffer{}
				b.Write(*file)
				b.WriteString("\n---\n")
				b.Write(buf)
				*file = b.Bytes()
			}
		}
	*/

	// Etcd crd's
	{
		secrets := map[*[]byte]string{
			&(m.EtcdOperator): masterVersion.EtcdClusterYaml,
		}
		for file, app := range secrets {
			dep, err := resources.LoadEtcdClusterFile(masterVersion, *masterResources, app)
			must(err)

			dep.SetNamespace(ns)
			buf, err := yaml.Marshal(dep)
			must(err)

			b := &bytes.Buffer{}
			b.Write(*file)
			b.WriteString("\n---\n")
			b.Write(buf)
			*file = b.Bytes()
		}
	}

	// RoleBinding
	{
		secrets := map[*[]byte]string{
			&(m.EtcdOperator): "etcd-operator",
		}
		for file, app := range secrets {
			dep, err := resources.LoadClusterRoleBindingFile(ns, app, *masterResources)
			must(err)

			dep.SetNamespace(ns)
			buf, err := yaml.Marshal(dep)
			must(err)

			b := &bytes.Buffer{}
			b.Write(*file)
			b.WriteString("\n---\n")
			b.Write(buf)
			*file = b.Bytes()
		}
	}

	// Service Account
	{
		secrets := map[*[]byte]string{
			&(m.EtcdOperator): "etcd-operator",
		}
		for file, app := range secrets {
			dep, err := resources.LoadServiceAccountFile(app, *masterResources)
			must(err)

			dep.SetNamespace(ns)
			buf, err := yaml.Marshal(dep)
			must(err)

			b := &bytes.Buffer{}
			b.Write(*file)
			b.WriteString("\n---\n")
			b.Write(buf)
			*file = b.Bytes()
		}
	}
	fmt.Println(string(m.EtcdOperator))
}

func mustRenderKubermaticTemplateFiles(cluster *v1.Cluster, content *internal.TemplateContent, versions map[string]*api.MasterVersion, updates []api.MasterUpdate, dc string) {
	ns := cluster.Status.NamespaceName
	masterVersion, found := versions[cluster.Spec.MasterVersion]
	if !found {
		must(fmt.Errorf("unknown new cluster %q master version %q", cluster.Name, cluster.Spec.MasterVersion))
	}

	// Deployments
	{
		deps := map[string]string{
			EtcdOperatorDeploymentName:      masterVersion.EtcdOperatorDeploymentYaml,
			ApiserverDeploymentName:         masterVersion.ApiserverDeploymentYaml,
			ControllerManagerDeploymentName: masterVersion.ControllerDeploymentYaml,
			SchedulerDeploymentName:         masterVersion.SchedulerDeploymentYaml,
			NodeControllerDeploymentName:    masterVersion.NodeControllerDeploymentYaml,
			AddonManagerDeploymentName:      masterVersion.AddonManagerDeploymentYaml,
		}

		for file, app := range deps {
			dep, err := resources.LoadDeploymentFile(cluster, masterVersion, *masterResources, dc, app)
			must(err)

			dep.SetNamespace(ns)
			buf, err := yaml.Marshal(dep)
			must(err)

			must(ioutil.WriteFile(path.Join(*outputFolder, file), buf, os.ModePerm))
		}
	}

	// Services
	{
		services := map[string]string{
			ApiserverServiceName:         "apiserver",
			ApiserverExternalServiceName: "apiserver-external",
		}
		for file, app := range services {
			dep, err := resources.LoadServiceFile(cluster, app, *masterResources)
			must(err)

			dep.SetNamespace(ns)
			buf, err := yaml.Marshal(dep)
			must(err)

			must(ioutil.WriteFile(path.Join(*outputFolder, file), buf, os.ModePerm))
		}
	}

	// Secrets
	{
		secrets := map[string]string{
			ApiserverSecretName:         "apiserver",
			ControllerManagerSecretName: "controller-manager",
		}
		for file, app := range secrets {
			dep, err := resources.LoadSecretFile(cluster, app, *masterResources)
			must(err)

			dep.SetNamespace(ns)
			buf, err := yaml.Marshal(dep)
			must(err)

			must(ioutil.WriteFile(path.Join(*outputFolder, file), buf, os.ModePerm))
		}
	}

	// Ingress
	{
		secrets := map[string]string{
			ApiserverIngressName: "apiserver",
		}
		for file, app := range secrets {
			dep, err := resources.LoadIngressFile(cluster, app, *masterResources)
			must(err)

			dep.SetNamespace(ns)
			buf, err := yaml.Marshal(dep)
			must(err)

			must(ioutil.WriteFile(path.Join(*outputFolder, file), buf, os.ModePerm))
		}
	}

	// Etcd crd's
	{
		secrets := map[string]string{
			EtcdClusterName: masterVersion.EtcdClusterYaml,
		}
		for file, app := range secrets {
			dep, err := resources.LoadEtcdClusterFile(masterVersion, *masterResources, app)
			must(err)

			dep.SetNamespace(ns)
			buf, err := yaml.Marshal(dep)
			must(err)

			must(ioutil.WriteFile(path.Join(*outputFolder, file), buf, os.ModePerm))
		}
	}

	// RoleBinding
	{
		secrets := map[string]string{
			EtcdOperatorRolebindingName: "etcd-operator",
		}
		for file, app := range secrets {
			dep, err := resources.LoadClusterRoleBindingFile(ns, app, *masterResources)
			must(err)

			dep.SetNamespace(ns)
			buf, err := yaml.Marshal(dep)
			must(err)

			must(ioutil.WriteFile(path.Join(*outputFolder, file), buf, os.ModePerm))
		}
	}

	// Service Account
	{
		secrets := map[string]string{
			EtcdOperatorServiceAccountName: "etcd-operator",
		}
		for file, app := range secrets {
			dep, err := resources.LoadServiceAccountFile(app, *masterResources)
			must(err)

			dep.SetNamespace(ns)
			buf, err := yaml.Marshal(dep)
			must(err)

			must(ioutil.WriteFile(path.Join(*outputFolder, file), buf, os.ModePerm))
		}
	}
}

func parseURLs(s string) ([]*url.URL, error) {
	var out []*url.URL
	for _, u := range strings.Split(s, ",") {
		parsed, err := url.Parse(u)
		if err != nil {
			return nil, err
		}
		out = append(out, parsed)
	}
	return out, nil
}

func parseAltNames(s string) (*tlsutil.AltNames, error) {
	if s == "" {
		return nil, nil
	}
	var alt tlsutil.AltNames
	for _, an := range strings.Split(s, ",") {
		switch {
		case strings.HasPrefix(an, "DNS="):
			alt.DNSNames = append(alt.DNSNames, strings.TrimPrefix(an, "DNS="))
		case strings.HasPrefix(an, "IP="):
			ip := net.ParseIP(strings.TrimPrefix(an, "IP="))
			if ip == nil {
				return nil, fmt.Errorf("Invalid IP alt name: %s", an)
			}
			alt.IPs = append(alt.IPs, ip)
		default:
			return nil, fmt.Errorf("Invalid alt name: %s", an)
		}
	}
	return &alt, nil
}

func altNamesFromURLs(urls []*url.URL) *tlsutil.AltNames {
	var an tlsutil.AltNames
	for _, u := range urls {
		host, _, err := net.SplitHostPort(u.Host)
		if err != nil {
			host = u.Host
		}
		ip := net.ParseIP(host)
		if ip == nil {
			an.DNSNames = append(an.DNSNames, host)
		} else {
			an.IPs = append(an.IPs, ip)
		}
	}
	return &an
}

// offsetServiceIP returns an IP offset by up to 255.
// TODO: do numeric conversion to generalize this utility.
func offsetServiceIP(ipnet *net.IPNet, offset int) (net.IP, error) {
	ip := make(net.IP, len(ipnet.IP))
	copy(ip, ipnet.IP)
	for i := 0; i < offset; i++ {
		incIPv4(ip)
	}
	if ipnet.Contains(ip) {
		return ip, nil
	}
	return net.IP([]byte("")), fmt.Errorf("Service IP %v is not in %s", ip, ipnet)
}

func incIPv4(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func parseCertAndPrivateKeyFromString(caCertPem, privKey string) (*rsa.PrivateKey, *x509.Certificate, error) {
	// Parse CA Private key.
	key, err := tlsutil.ParsePEMEncodedPrivateKey([]byte(privKey))
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse CA private key: %v", err)
	}
	// Parse CA Cert.
	cert, err := parseCertFromString(caCertPem)
	if err != nil {
		return nil, nil, err
	}
	return key, cert, nil
}

func parseCertFromString(caCertPem string) (*x509.Certificate, error) {
	cert, err := tlsutil.ParsePEMEncodedCACert([]byte(caCertPem))
	if err != nil {
		return nil, fmt.Errorf("unable to parse CA Cert: %v", err)
	}
	return cert, nil
}
