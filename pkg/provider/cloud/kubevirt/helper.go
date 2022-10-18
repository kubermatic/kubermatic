/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubevirt

import (
	corev1 "k8s.io/api/core/v1"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
)

func generateKubeconfigWithToken(restConfig *restclient.Config, sa *corev1.ServiceAccount, token string) ([]byte, error) {
	config := api.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters: map[string]*api.Cluster{
			"infra-cluster": {
				CertificateAuthorityData: restConfig.CAData,
				Server:                   restConfig.Host,
			},
		},
		Contexts: map[string]*api.Context{
			"default": {
				Cluster:  "infra-cluster",
				AuthInfo: sa.Name,
			},
		},
		CurrentContext: "default",
		AuthInfos: map[string]*api.AuthInfo{
			sa.Name: {
				Token: token,
			},
		},
	}

	kubeconfig, err := clientcmd.Write(config)
	if err != nil {
		return nil, err
	}

	return kubeconfig, nil
}
