package openvpn

import (
	"github.com/kubermatic/kubermatic/api/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMap returns a ConfigMap containing the openvpn config
func ConfigMap(data *resources.TemplateData, existing *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	var cm *corev1.ConfigMap
	if existing != nil {
		cm = existing
	} else {
		cm = &corev1.ConfigMap{}
	}

	cm.Name = resources.OpenVPNClientConfigConfigMapName
	cm.OwnerReferences = []metav1.OwnerReference{data.GetClusterRef()}
	cm.Labels = resources.GetLabels(name)

	cm.Data = map[string]string{
		"user-cluster-client": `iroute 172.25.0.0 255.255.0.0
iroute 10.10.10.0 255.255.255.0
`,
	}

	return cm, nil
}
