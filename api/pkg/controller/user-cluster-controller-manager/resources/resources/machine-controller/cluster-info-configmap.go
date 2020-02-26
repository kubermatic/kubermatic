package machinecontroller

import (
	"crypto/x509"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/certificates/triple"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// ClusterInfoConfigMapCreator returns the func to create/update the ConfigMap
func ClusterInfoConfigMapCreator(url string, caCert *x509.Certificate) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		return resources.ClusterInfoConfigMapName, func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			if cm.Data == nil {
				cm.Data = map[string]string{}
			}

			cm.Labels = resources.BaseAppLabels(Name, nil)

			kubeconfig := clientcmdapi.Config{}
			kubeconfig.Clusters = map[string]*clientcmdapi.Cluster{
				"": {
					Server:                   url,
					CertificateAuthorityData: triple.EncodeCertPEM(caCert),
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
}
