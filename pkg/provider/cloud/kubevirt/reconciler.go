/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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
	"context"
	"encoding/base64"
	"fmt"

	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type reconciler struct {
	ctrlruntimeclient.Client
	RestConfig *restclient.Config

	ClusterName string
}

func NewReconciler(kubeconfig string, clusterName string) (*reconciler, error) {
	client, restConfig, err := NewClientWithRestConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	return &reconciler{Client: client, RestConfig: restConfig, ClusterName: clusterName}, nil
}

func (r *reconciler) ReconcileCSIServiceAccount(ctx context.Context) ([]byte, error) {
	saCreators := []reconciling.NamedServiceAccountCreatorGetter{
		csiServiceAccountCreator(csiResourceName),
	}
	if err := reconciling.EnsureNamedObjects(ctx, r, csiServiceAccountNamespace, saCreators); err != nil {
		return nil, err
	}

	return r.GenerateKubeConfigForSA(ctx, csiResourceName, csiServiceAccountNamespace)
}

func (r *reconciler) GenerateKubeConfigForSA(ctx context.Context, name string, namespace string) ([]byte, error) {
	sa := corev1.ServiceAccount{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &sa)
	if err != nil {
		return nil, err
	}

	if len(sa.Secrets) == 0 {
		return nil, fmt.Errorf("not found auth token for service account: %s", sa.Name)
	}

	s := corev1.Secret{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: sa.Secrets[0].Name, Namespace: namespace}, &s)
	if err != nil {
		return nil, err
	}

	if _, ok := s.Data["token"]; !ok {
		return nil, err
	}
	token, err := base64.StdEncoding.DecodeString(string(s.Data["token"]))
	if err != nil {
		token = s.Data["token"]
	}

	return generateKubeConfigWithToken(r.RestConfig, &sa, string(token))
}

func generateKubeConfigWithToken(restConfig *restclient.Config, sa *corev1.ServiceAccount, token string) ([]byte, error) {
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

	kubeConfig, err := clientcmd.Write(config)
	if err != nil {
		return nil, err
	}

	return kubeConfig, nil
}
