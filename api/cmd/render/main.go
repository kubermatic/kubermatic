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
	"text/template"

	"github.com/ghodss/yaml"
	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/cmd/render/internal"
	"github.com/kubermatic/kubermatic/api/pkg/controller/resources"
	"github.com/kubermatic/kubermatic/api/pkg/controller/version"
	"github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubernetes-incubator/bootkube/pkg/tlsutil"
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
	Flannel:         "quay.io/coreos/flannel:v0.8.0-amd64",
	FlannelCNI:      "quay.io/coreos/flannel-cni:v0.3.0",
	Calico:          "quay.io/calico/node:v2.5.1",
	CalicoCNI:       "quay.io/calico/cni:v1.10.0",
	Hyperkube:       "quay.io/coreos/hyperkube:v1.7.5_coreos.0",
	Kenc:            "quay.io/coreos/kenc:0.0.2",
	KubeDNS:         "gcr.io/google_containers/k8s-dns-kube-dns-amd64:1.14.4",
	KubeDNSMasq:     "gcr.io/google_containers/k8s-dns-dnsmasq-nanny-amd64:1.14.4",
	KubeDNSSidecar:  "gcr.io/google_containers/k8s-dns-sidecar-amd64:1.14.4",
	PodCheckpointer: "quay.io/coreos/pod-checkpointer:0cd390e0bc1dcdcc714b20eda3435c3d00669d0e",
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
	SelfHostKubelet        bool
	SelfHostedEtcd         bool
	CalicoNetworkPolicy    bool
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
	m.ControllerManager = mustCreateAssetFromTemplate(internal.ControllerManagerTemplate, conf)
	m.APIServer = mustCreateAssetFromTemplate(internal.APIServerTemplate, conf)
	m.Proxy = mustCreateAssetFromTemplate(internal.ProxyTemplate, conf)
	m.KubeFlannelCfg = mustCreateAssetFromTemplate(internal.KubeFlannelCfgTemplate, conf)
	m.KubeFlannel = mustCreateAssetFromTemplate(internal.KubeFlannelTemplate, conf)
	m.KubeDNSSvc = mustCreateAssetFromTemplate(internal.DNSSvcTemplate, conf)
	m.BootstrapAPIServer = mustCreateAssetFromTemplate(internal.BootstrapAPIServerTemplate, conf)
	m.BootstrapControllerManager = mustCreateAssetFromTemplate(internal.BootstrapControllerManagerTemplate, conf)
	m.BootstrapScheduler = mustCreateAssetFromTemplate(internal.BootstrapSchedulerTemplate, conf)

	// Self hosted etcd
	m.EtcdOperator = mustCreateAssetFromTemplate(internal.EtcdOperatorTemplate, conf)
	m.EtcdSvc = mustCreateAssetFromTemplate(internal.EtcdSvcTemplate, conf)
	m.Kenc = mustCreateAssetFromTemplate(internal.KencTemplate, conf)
	m.BootstrapEtcd = mustCreateAssetFromTemplate(internal.BootstrapEtcdTemplate, conf)
	m.BootstrapEtcdService = mustCreateAssetFromTemplate(internal.BootstrapEtcdSvcTemplate, conf)
	m.MigrateEtcdCluster = mustCreateAssetFromTemplate(internal.EtcdCRDTemplate, conf)

	// Static templates
	m.Scheduler = mustCreateAssetFromTemplate(internal.SchedulerTemplate, conf)
	m.SchedulerDisruption = mustCreateAssetFromTemplate(internal.SchedulerDisruptionTemplate, conf)
	m.ControllerManagerDisruption = mustCreateAssetFromTemplate(internal.ControllerManagerDisruptionTemplate, conf)
	m.KubeDNSDeployment = mustCreateAssetFromTemplate(internal.DNSDeploymentTemplate, conf)
	m.Checkpointer = mustCreateAssetFromTemplate(internal.CheckpointerTemplate, conf)
	m.KubeSystemSARoleBinding = mustCreateAssetFromTemplate(internal.KubeSystemSARoleBindingTemplate, conf)

	// TODO(realfake): bootstrap etcd is missing
}
func translateClusterToBootkube(cluster *v1.Cluster) *Config {
	providerName, err := provider.ClusterCloudProviderName(cluster.Spec.Cloud)
	must(err)
	c := &Config{
		EtcdCACert:             nil,
		EtcdClientCert:         nil,
		EtcdClientKey:          nil,
		EtcdServers:            nil,
		EtcdUseTLS:             false,
		APIServers:             nil,
		CACert:                 nil,
		CAPrivKey:              nil,
		AltNames:               nil,
		PodCIDR:                nil,
		ServiceCIDR:            nil,
		APIServiceIP:           nil,
		BootEtcdServiceIP:      nil,
		DNSServiceIP:           nil,
		EtcdServiceIP:          nil,
		EtcdServiceName:        "",
		SelfHostKubelet:        false,
		SelfHostedEtcd:         true,
		CalicoNetworkPolicy:    false,
		CloudProvider:          providerName,
		BootstrapSecretsSubdir: "",
		Images:                 DefaultImages,
	}
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
