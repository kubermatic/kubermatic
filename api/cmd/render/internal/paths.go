package internal

import (
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

	setError(ioutil.WriteFile(path.Join(basename, AssetPathCAKey), m.CAKey, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathCACert), m.CACert, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathAPIServerKey), m.APIServerKey, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathAPIServerCert), m.APIServerCert, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathEtcdClientCA), m.EtcdClientCA, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathEtcdClientCert), m.EtcdClientCert, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathEtcdClientKey), m.EtcdClientKey, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathEtcdServerCA), m.EtcdServerCA, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathEtcdServerCert), m.EtcdServerCert, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathEtcdServerKey), m.EtcdServerKey, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathEtcdPeerCA), m.EtcdPeerCA, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathEtcdPeerCert), m.EtcdPeerCert, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathEtcdPeerKey), m.EtcdPeerKey, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathServiceAccountPrivKey), m.ServiceAccountPrivKey, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathServiceAccountPubKey), m.ServiceAccountPubKey, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathKubeletKey), m.KubeletKey, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathKubeletCert), m.KubeletCert, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathKubeConfig), m.KubeConfig, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathProxy), m.Proxy, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathAPIServerSecret), m.APIServerSecret, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathAPIServer), m.APIServer, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathControllerManager), m.ControllerManager, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathControllerManagerSecret), m.ControllerManagerSecret, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathControllerManagerDisruption), m.ControllerManagerDisruption, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathScheduler), m.Scheduler, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathSchedulerDisruption), m.SchedulerDisruption, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathKubeDNSDeployment), m.KubeDNSDeployment, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathKubeDNSSvc), m.KubeDNSSvc, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathSystemNamespace), m.SystemNamespace, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathCheckpointer), m.Checkpointer, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathEtcdOperator), m.EtcdOperator, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathEtcdSvc), m.EtcdSvc, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathEtcdClientSecret), m.EtcdClientSecret, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathEtcdPeerSecret), m.EtcdPeerSecret, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathEtcdServerSecret), m.EtcdServerSecret, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathKenc), m.Kenc, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathKubeSystemSARoleBinding), m.KubeSystemSARoleBinding, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathBootstrapAPIServer), m.BootstrapAPIServer, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathBootstrapControllerManager), m.BootstrapControllerManager, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathBootstrapScheduler), m.BootstrapScheduler, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathBootstrapEtcd), m.BootstrapEtcd, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathBootstrapEtcdService), m.BootstrapEtcdService, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathMigrateEtcdCluster), m.MigrateEtcdCluster, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathCalico), m.Calico, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathCalicoPolicyOnly), m.CalicoPolicyOnly, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathCalicoCfg), m.CalicoCfg, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathCalicoSA), m.CalicoSA, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathCalicoRole), m.CalicoRole, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathCalicoRoleBinding), m.CalicoRoleBinding, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathCalicoBGPConfigsCRD), m.CalicoBGPConfigsCRD, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathCalicoFelixConfigsCRD), m.CalicoFelixConfigsCRD, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathCalicoNetworkPoliciesCRD), m.CalicoNetworkPoliciesCRD, fileMode))
	setError(ioutil.WriteFile(path.Join(basename, AssetPathCalicoIPPoolsCRD), m.CalicoIPPoolsCRD, fileMode))
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
