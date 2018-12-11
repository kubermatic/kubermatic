package machinecontroller

import (
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	"k8s.io/client-go/util/cert"
)

// ClusterInfoConfigMapCreator returns a ConfigMap containing the config for the OpenVPN client. It lives inside the user-cluster
func ClusterInfoConfigMapCreator(data resources.ConfigMapDataProvider) resources.ConfigMapCreator {
	return func(existing *corev1.ConfigMap) (*corev1.ConfigMap, error) {
		cm := existing
		if cm == nil {
			cm = &corev1.ConfigMap{}
		}
		if cm.Data == nil {
			cm.Data = map[string]string{}
		}

		cm.Name = resources.ClusterInfoConfigMapName
		cm.Namespace = metav1.NamespacePublic
		cm.Labels = resources.BaseAppLabel(name, nil)

		caKp, err := data.GetRootCA()
		if err != nil {
			return nil, err
		}

		kubeconfig := clientcmdapi.Config{}
		kubeconfig.Clusters = map[string]*clientcmdapi.Cluster{
			"": {
				Server:                   data.Cluster().Address.URL,
				CertificateAuthorityData: cert.EncodeCertPEM(caKp.Cert),
			},
		}

		bconfig, err := clientcmd.Write(kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("failed to encode kubeconfig: %v", err)
		}
		cm.Data["kubeconfig"] = string(bconfig)

		return cm, nil
	}
}
