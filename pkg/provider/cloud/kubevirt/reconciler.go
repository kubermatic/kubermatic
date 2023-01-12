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

	"k8c.io/kubermatic/v2/pkg/resources/reconciling"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	csiServiceAccountNamespace = metav1.NamespaceDefault
	csiResourceName            = "kubevirt-csi"
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

func (r *reconciler) ReconcileCSIAccess(ctx context.Context) ([]byte, error) {
	// soon we are going to change the default namespace to cluster namespace
	namespace := "default"
	csiResourceName := "kubevirt-csi"

	saCreators := []reconciling.NamedServiceAccountCreatorGetter{
		csiServiceAccountCreator(csiResourceName),
	}
	if err := reconciling.ReconcileServiceAccounts(ctx, saCreators, namespace, r.Client); err != nil {
		return nil, err
	}

	roleCreators := []reconciling.NamedRoleCreatorGetter{
		csiRoleCreator(csiResourceName),
	}
	if err := reconciling.ReconcileRoles(ctx, roleCreators, namespace, r.Client); err != nil {
		return nil, err
	}

	roleBindingCreators := []reconciling.NamedRoleBindingCreatorGetter{
		csiRoleBindingCreator(csiResourceName, namespace),
	}
	if err := reconciling.ReconcileRoleBindings(ctx, roleBindingCreators, namespace, r.Client); err != nil {
		return nil, err
	}

	return r.GenerateKubeConfigForSA(ctx, csiResourceName, namespace)
}

func csiSecretTokenCreator(name string) reconciling.NamedSecretCreatorGetter {
	return func() (string, reconciling.SecretCreator) {
		return name, func(s *corev1.Secret) (*corev1.Secret, error) {
			s.SetAnnotations(map[string]string{
				corev1.ServiceAccountNameKey: csiResourceName,
			})

			s.Type = corev1.SecretTypeServiceAccountToken
			return s, nil
		}
	}
}

func (r *reconciler) GenerateKubeConfigForSA(ctx context.Context, name string, namespace string) ([]byte, error) {
	sa := corev1.ServiceAccount{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, &sa)
	if err != nil {
		return nil, err
	}

	var sName string
	if len(sa.Secrets) == 0 {
		// k8s 1.24 by default disabled automatic token creation for service accounts
		sName = csiResourceName
		seCreators := []reconciling.NamedSecretCreatorGetter{
			csiSecretTokenCreator(csiResourceName),
		}
		if err := reconciling.ReconcileSecrets(ctx, seCreators, csiServiceAccountNamespace, r.Client); err != nil {
			return nil, err
		}
	} else {
		sName = sa.Secrets[0].Name
	}

	s := corev1.Secret{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: sName, Namespace: namespace}, &s)
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
