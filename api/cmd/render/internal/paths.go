package internal

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

const (
	AssetPathSecrets                     = "tls"
	AssetPathCAKey                       = "tls/ca.key"
	AssetPathCACert                      = "tls/ca.crt"
	AssetPathAPIServerKey                = "tls/apiserver.key"
	AssetPathAPIServerCert               = "tls/apiserver.crt"
	AssetPathEtcdClientCA                = "tls/etcd-client-ca.crt"
	AssetPathEtcdClientCert              = "tls/etcd-client.crt"
	AssetPathEtcdClientKey               = "tls/etcd-client.key"
	AssetPathEtcdServerCA                = "tls/etcd/server-ca.crt"
	AssetPathEtcdServerCert              = "tls/etcd/server.crt"
	AssetPathEtcdServerKey               = "tls/etcd/server.key"
	AssetPathEtcdPeerCA                  = "tls/etcd/peer-ca.crt"
	AssetPathEtcdPeerCert                = "tls/etcd/peer.crt"
	AssetPathEtcdPeerKey                 = "tls/etcd/peer.key"
	AssetPathServiceAccountPrivKey       = "tls/service-account.key"
	AssetPathServiceAccountPubKey        = "tls/service-account.pub"
	AssetPathKubeletKey                  = "tls/kubelet.key"
	AssetPathKubeletCert                 = "tls/kubelet.crt"
	AssetPathAuth                        = "auth"
	AssetPathKubeConfig                  = "auth/kubeconfig"
	AssetPathManifests                   = "manifests"
	AssetPathProxy                       = "manifests/kube-proxy.yaml"
	AssetPathProxySA                     = "manifests/kube-proxy-sa.yaml"
	AssetPathProxyRoleBinding            = "manifests/kube-proxy-role-binding.yaml"
	AssetPathCalico                      = "manifests/calico.yaml"
	AssetPathCalicoPolicyOnly            = "manifests/calico-policy-only.yaml"
	AssetPathCalicoCfg                   = "manifests/calico-config.yaml"
	AssetPathCalicoSA                    = "manifests/calico-service-account.yaml"
	AssetPathCalicoRole                  = "manifests/calico-role.yaml"
	AssetPathCalicoRoleBinding           = "manifests/calico-role-binding.yaml"
	AssetPathCalicoBGPConfigsCRD         = "manifests/calico-bgp-configs-crd.yaml"
	AssetPathCalicoFelixConfigsCRD       = "manifests/calico-felix-configs-crd.yaml"
	AssetPathCalicoNetworkPoliciesCRD    = "manifests/calico-network-policies-crd.yaml"
	AssetPathCalicoIPPoolsCRD            = "manifests/calico-ip-pools-crd.yaml"
	AssetPathAPIServerSecret             = "manifests/kube-apiserver-secret.yaml"
	AssetPathAPIServer                   = "manifests/kube-apiserver.yaml"
	AssetPathControllerManager           = "manifests/kube-controller-manager.yaml"
	AssetPathControllerManagerSecret     = "manifests/kube-controller-manager-secret.yaml"
	AssetPathControllerManagerDisruption = "manifests/kube-controller-manager-disruption.yaml"
	AssetPathScheduler                   = "manifests/kube-scheduler.yaml"
	AssetPathSchedulerDisruption         = "manifests/kube-scheduler-disruption.yaml"
	AssetPathKubeDNSDeployment           = "manifests/kube-dns-deployment.yaml"
	AssetPathKubeDNSSvc                  = "manifests/kube-dns-svc.yaml"
	AssetPathSystemNamespace             = "manifests/kube-system-ns.yaml"
	AssetPathCheckpointer                = "manifests/pod-checkpointer.yaml"
	AssetPathEtcdOperator                = "manifests/etcd-operator.yaml"
	AssetPathEtcdSvc                     = "manifests/etcd-service.yaml"
	AssetPathEtcdClientSecret            = "manifests/etcd-client-tls.yaml"
	AssetPathEtcdPeerSecret              = "manifests/etcd-peer-tls.yaml"
	AssetPathEtcdServerSecret            = "manifests/etcd-server-tls.yaml"
	AssetPathKenc                        = "manifests/kube-etcd-network-checkpointer.yaml"
	AssetPathKubeSystemSARoleBinding     = "manifests/kube-system-rbac-role-binding.yaml"
	AssetPathBootstrapManifests          = "bootstrap-manifests"
	AssetPathBootstrapAPIServer          = "bootstrap-manifests/bootstrap-apiserver.yaml"
	AssetPathBootstrapControllerManager  = "bootstrap-manifests/bootstrap-controller-manager.yaml"
	AssetPathBootstrapScheduler          = "bootstrap-manifests/bootstrap-scheduler.yaml"
	AssetPathBootstrapEtcd               = "bootstrap-manifests/bootstrap-etcd.yaml"
	AssetPathEtcd                        = "etcd"
	AssetPathBootstrapEtcdService        = "etcd/bootstrap-etcd-service.json"
	AssetPathMigrateEtcdCluster          = "etcd/migrate-etcd-cluster.json"
)

type TemplateContent struct {
	KubeConfigTemplate                  []byte
	KubeSystemSARoleBindingTemplate     []byte
	APIServerTemplate                   []byte
	BootstrapAPIServerTemplate          []byte
	KencTemplate                        []byte
	CheckpointerTemplate                []byte
	ControllerManagerTemplate           []byte
	BootstrapControllerManagerTemplate  []byte
	ControllerManagerDisruptionTemplate []byte
	SchedulerTemplate                   []byte
	BootstrapSchedulerTemplate          []byte
	SchedulerDisruptionTemplate         []byte
	ProxyTemplate                       []byte
	ProxySATemplate                     []byte
	ProxyRoleBindingTemplate            []byte
	DNSDeploymentTemplate               []byte
	DNSSvcTemplate                      []byte
	EtcdOperatorTemplate                []byte
	EtcdSvcTemplate                     []byte
	BootstrapEtcdTemplate               []byte
	BootstrapEtcdSvcTemplate            []byte
	EtcdCRDTemplate                     []byte
	CalicoTemplate                      []byte
	CalicoPolicyOnlyTemplate            []byte
	CalicoCfgTemplate                   []byte
	CalicoSATemplate                    []byte
	CalicoRoleTemplate                  []byte
	CalicoRoleBindingTemplate           []byte
	CalicoBGPConfigsCRDTemplate         []byte
	CalicoFelixConfigsCRDTemplate       []byte
	CalicoNetworkPoliciesCRDTemplate    []byte
	CalicoIPPoolsCRDTemplate            []byte
}

type Manifests struct {
	CAKey                       []byte
	CACert                      []byte
	APIServerKey                []byte
	APIServerCert               []byte
	EtcdClientCA                []byte
	EtcdClientCert              []byte
	EtcdClientKey               []byte
	EtcdServerCA                []byte
	EtcdServerCert              []byte
	EtcdServerKey               []byte
	EtcdPeerCA                  []byte
	EtcdPeerCert                []byte
	EtcdPeerKey                 []byte
	ServiceAccountPrivKey       []byte
	ServiceAccountPubKey        []byte
	KubeletKey                  []byte
	KubeletCert                 []byte
	KubeConfig                  []byte
	Proxy                       []byte
	ProxySA                     []byte
	ProxyRoleBinding            []byte
	KubeFlannel                 []byte
	KubeFlannelCfg              []byte
	APIServerSecret             []byte
	APIServer                   []byte
	ControllerManager           []byte
	ControllerManagerSecret     []byte
	ControllerManagerDisruption []byte
	Scheduler                   []byte
	SchedulerDisruption         []byte
	KubeDNSDeployment           []byte
	KubeDNSSvc                  []byte
	SystemNamespace             []byte
	Checkpointer                []byte
	EtcdOperator                []byte
	EtcdSvc                     []byte
	EtcdClientSecret            []byte
	EtcdPeerSecret              []byte
	EtcdServerSecret            []byte
	Kenc                        []byte
	KubeSystemSARoleBinding     []byte
	BootstrapAPIServer          []byte
	BootstrapControllerManager  []byte
	BootstrapScheduler          []byte
	BootstrapEtcd               []byte
	BootstrapEtcdService        []byte
	MigrateEtcdCluster          []byte
	Calico                      []byte
	CalicoPolicyOnly            []byte
	CalicoCfg                   []byte
	CalicoSA                    []byte
	CalicoRole                  []byte
	CalicoRoleBinding           []byte
	CalicoBGPConfigsCRD         []byte
	CalicoFelixConfigsCRD       []byte
	CalicoNetworkPoliciesCRD    []byte
	CalicoIPPoolsCRD            []byte
}

func (m *Manifests) WriteToDir(basename string) error {
	const fileMode = os.ModePerm
	const dirMode = os.ModeDir | os.ModePerm
	var err error
	writeFile := func(filePath string, content []byte) error {
		if len(content) != 0 {
			fmt.Printf("Writing asset: out-files/%s\n", filePath)
			return ioutil.WriteFile(path.Join(basename, filePath), content, fileMode)
		}
		return nil
	}

	setError := func(e error) {
		if e != nil && err != nil {
			err = e
		}
	}
	setError(os.MkdirAll(basename, dirMode))
	setError(os.MkdirAll(path.Join(basename, AssetPathManifests), dirMode))
	setError(os.MkdirAll(path.Join(basename, AssetPathBootstrapManifests), dirMode))
	setError(os.MkdirAll(path.Join(basename, AssetPathSecrets), dirMode))
	setError(os.MkdirAll(path.Join(basename, AssetPathAuth), dirMode))
	setError(os.MkdirAll(path.Join(basename, AssetPathEtcd), dirMode))
	if err != nil {
		return err
	}

	setError(writeFile(AssetPathCAKey, m.CAKey))
	setError(writeFile(AssetPathCACert, m.CACert))
	setError(writeFile(AssetPathAPIServerKey, m.APIServerKey))
	setError(writeFile(AssetPathAPIServerCert, m.APIServerCert))
	setError(writeFile(AssetPathEtcdClientCA, m.EtcdClientCA))
	setError(writeFile(AssetPathEtcdClientCert, m.EtcdClientCert))
	setError(writeFile(AssetPathEtcdClientKey, m.EtcdClientKey))
	setError(writeFile(AssetPathEtcdServerCA, m.EtcdServerCA))
	setError(writeFile(AssetPathEtcdServerCert, m.EtcdServerCert))
	setError(writeFile(AssetPathEtcdServerKey, m.EtcdServerKey))
	setError(writeFile(AssetPathEtcdPeerCA, m.EtcdPeerCA))
	setError(writeFile(AssetPathEtcdPeerCert, m.EtcdPeerCert))
	setError(writeFile(AssetPathEtcdPeerKey, m.EtcdPeerKey))
	setError(writeFile(AssetPathServiceAccountPrivKey, m.ServiceAccountPrivKey))
	setError(writeFile(AssetPathServiceAccountPubKey, m.ServiceAccountPubKey))
	setError(writeFile(AssetPathKubeletKey, m.KubeletKey))
	setError(writeFile(AssetPathKubeletCert, m.KubeletCert))
	setError(writeFile(AssetPathKubeConfig, m.KubeConfig))
	setError(writeFile(AssetPathProxy, m.Proxy))
	setError(writeFile(AssetPathProxySA, m.ProxySA))
	setError(writeFile(AssetPathProxyRoleBinding, m.ProxyRoleBinding))
	setError(writeFile(AssetPathProxy, m.Proxy))
	setError(writeFile(AssetPathAPIServerSecret, m.APIServerSecret))
	setError(writeFile(AssetPathAPIServer, m.APIServer))
	setError(writeFile(AssetPathControllerManager, m.ControllerManager))
	setError(writeFile(AssetPathControllerManagerSecret, m.ControllerManagerSecret))
	setError(writeFile(AssetPathControllerManagerDisruption, m.ControllerManagerDisruption))
	setError(writeFile(AssetPathScheduler, m.Scheduler))
	setError(writeFile(AssetPathSchedulerDisruption, m.SchedulerDisruption))
	setError(writeFile(AssetPathKubeDNSDeployment, m.KubeDNSDeployment))
	setError(writeFile(AssetPathKubeDNSSvc, m.KubeDNSSvc))
	setError(writeFile(AssetPathSystemNamespace, m.SystemNamespace))
	setError(writeFile(AssetPathCheckpointer, m.Checkpointer))
	setError(writeFile(AssetPathEtcdOperator, m.EtcdOperator))
	setError(writeFile(AssetPathEtcdSvc, m.EtcdSvc))
	setError(writeFile(AssetPathEtcdClientSecret, m.EtcdClientSecret))
	setError(writeFile(AssetPathEtcdPeerSecret, m.EtcdPeerSecret))
	setError(writeFile(AssetPathEtcdServerSecret, m.EtcdServerSecret))
	setError(writeFile(AssetPathKenc, m.Kenc))
	setError(writeFile(AssetPathKubeSystemSARoleBinding, m.KubeSystemSARoleBinding))
	setError(writeFile(AssetPathBootstrapAPIServer, m.BootstrapAPIServer))
	setError(writeFile(AssetPathBootstrapControllerManager, m.BootstrapControllerManager))
	setError(writeFile(AssetPathBootstrapScheduler, m.BootstrapScheduler))
	setError(writeFile(AssetPathBootstrapEtcd, m.BootstrapEtcd))
	setError(writeFile(AssetPathBootstrapEtcdService, m.BootstrapEtcdService))
	setError(writeFile(AssetPathMigrateEtcdCluster, m.MigrateEtcdCluster))
	setError(writeFile(AssetPathCalico, m.Calico))
	setError(writeFile(AssetPathCalicoPolicyOnly, m.CalicoPolicyOnly))
	setError(writeFile(AssetPathCalicoCfg, m.CalicoCfg))
	setError(writeFile(AssetPathCalicoSA, m.CalicoSA))
	setError(writeFile(AssetPathCalicoRole, m.CalicoRole))
	setError(writeFile(AssetPathCalicoRoleBinding, m.CalicoRoleBinding))
	setError(writeFile(AssetPathCalicoBGPConfigsCRD, m.CalicoBGPConfigsCRD))
	setError(writeFile(AssetPathCalicoFelixConfigsCRD, m.CalicoFelixConfigsCRD))
	setError(writeFile(AssetPathCalicoNetworkPoliciesCRD, m.CalicoNetworkPoliciesCRD))
	setError(writeFile(AssetPathCalicoIPPoolsCRD, m.CalicoIPPoolsCRD))
	return err
}

func DefaultInternalTemplateContent() *TemplateContent {
	return &TemplateContent{
		KubeConfigTemplate:                  KubeConfigTemplate,
		KubeSystemSARoleBindingTemplate:     KubeSystemSARoleBindingTemplate,
		APIServerTemplate:                   APIServerTemplate,
		BootstrapAPIServerTemplate:          BootstrapAPIServerTemplate,
		KencTemplate:                        KencTemplate,
		CheckpointerTemplate:                CheckpointerTemplate,
		ControllerManagerTemplate:           ControllerManagerTemplate,
		BootstrapControllerManagerTemplate:  BootstrapControllerManagerTemplate,
		ControllerManagerDisruptionTemplate: ControllerManagerDisruptionTemplate,
		SchedulerTemplate:                   SchedulerTemplate,
		BootstrapSchedulerTemplate:          BootstrapSchedulerTemplate,
		SchedulerDisruptionTemplate:         SchedulerDisruptionTemplate,
		ProxyTemplate:                       ProxyTemplate,
		ProxySATemplate:                     ProxyServiceAccount,
		ProxyRoleBindingTemplate:            ProxyClusterRoleBinding,
		DNSDeploymentTemplate:               DNSDeploymentTemplate,
		DNSSvcTemplate:                      DNSSvcTemplate,
		EtcdOperatorTemplate:                EtcdOperatorTemplate,
		EtcdSvcTemplate:                     EtcdSvcTemplate,
		BootstrapEtcdTemplate:               BootstrapEtcdTemplate,
		BootstrapEtcdSvcTemplate:            BootstrapEtcdSvcTemplate,
		EtcdCRDTemplate:                     EtcdCRDTemplate,
		CalicoTemplate:                      CalicoNodeTemplate,
		CalicoPolicyOnlyTemplate:            CalicoPolicyOnlyTemplate,
		CalicoCfgTemplate:                   CalicoCfgTemplate,
		CalicoSATemplate:                    CalicoServiceAccountTemplate,
		CalicoRoleTemplate:                  CalicoRoleTemplate,
		CalicoRoleBindingTemplate:           CalicoRoleBindingTemplate,
		CalicoBGPConfigsCRDTemplate:         CalicoBGPConfigsCRD,
		CalicoFelixConfigsCRDTemplate:       CalicoFelixConfigsCRD,
		CalicoNetworkPoliciesCRDTemplate:    CalicoNetworkPoliciesCRD,
		CalicoIPPoolsCRDTemplate:            CalicoIPPoolsCRD,
	}
}
