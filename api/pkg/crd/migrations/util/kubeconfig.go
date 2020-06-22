/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package util

import (
	"errors"
	"fmt"

	"github.com/kubermatic/kubermatic/api/pkg/provider"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// SingleSeedKubeconfig takes a kubeconfig and returns a new kubeconfig that only has the
// required parts to connect to the given seed. An error is returned when the given seed
// has no valid matching context in the kubeconfig.
func SingleSeedKubeconfig(kubeconfig *clientcmdapi.Config, seedName string) (*clientcmdapi.Config, error) {
	if kubeconfig == nil {
		return nil, errors.New("no kubeconfig defined")
	}

	context, exists := kubeconfig.Contexts[seedName]
	if !exists {
		return nil, fmt.Errorf("no context named %s found in kubeconfig", seedName)
	}
	clusterName := context.Cluster
	authName := context.AuthInfo

	cluster, exists := kubeconfig.Clusters[clusterName]
	if !exists {
		return nil, fmt.Errorf("no cluster named %s found in kubeconfig", clusterName)
	}

	auth, exists := kubeconfig.AuthInfos[authName]
	if !exists {
		return nil, fmt.Errorf("no user named %s found in kubeconfig", authName)
	}

	config := clientcmdapi.NewConfig()
	config.CurrentContext = seedName
	config.Contexts[seedName] = context
	config.Clusters[clusterName] = cluster
	config.AuthInfos[authName] = auth

	return config, nil
}

func CreateKubeconfigSecret(kubeconfig *clientcmdapi.Config, name string, namespace string) (*corev1.Secret, string, error) {
	encoded, err := clientcmd.Write(*kubeconfig)
	if err != nil {
		return nil, "", fmt.Errorf("failed to serialize kubeconfig: %v", err)
	}

	fieldPath := provider.DefaultKubeconfigFieldPath

	secret := &corev1.Secret{}
	secret.Name = name
	secret.Namespace = namespace
	secret.Data = map[string][]byte{
		fieldPath: encoded,
	}

	return secret, fieldPath, nil
}
