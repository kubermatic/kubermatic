package kubeone

import (
	"encoding/json"
	"fmt"

	kubeone "github.com/kubermatic/kubeone/pkg/apis/kubeone/v1alpha1"
	"github.com/kubermatic/kubermatic/api/pkg/resources"
	"github.com/kubermatic/kubermatic/api/pkg/resources/reconciling"
	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigMapCreator returns a ConfigMapCreator containing the grpc-server config for the supplied data.
func ConfigMapCreator(data *resources.TemplateData) reconciling.NamedConfigMapCreatorGetter {
	return func() (string, reconciling.ConfigMapCreator) {
		cc := func(cm *corev1.ConfigMap) (*corev1.ConfigMap, error) {
			name := data.ClusterNamespaceName()

			config := kubeone.KubeOneCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kubeone.io/v1alpha1",
					Kind:       "KubeOneCluster",
				},
				Name: name,
				Hosts: []kubeone.HostConfig{
					{
						SSHPrivateKeyFile: "/secret/ssh",
						PublicAddress:     fmt.Sprintf("%s.%s.svc.cluster.local", resources.GRPCTunnelServiceName, name),
						PrivateAddress:    "::",
					},
				},
				CloudProvider:     kubeone.CloudProviderSpec{Name: kubeone.CloudProviderNameNone},
				Versions:          kubeone.VersionConfig{Kubernetes: data.ClusterVersion()},
				MachineController: &kubeone.MachineControllerConfig{Deploy: false},
			}

			data, err := json.Marshal(config)
			if err != nil {
				return nil, errors.Wrap(err, "failed to marshal kubeone config to json string")
			}

			cm.Data = map[string]string{
				"cluster": string(data),
			}

			return cm, nil
		}

		return resources.KubeoneConfigMapName, cc
	}
}
