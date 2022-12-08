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
	"context"
	"encoding/base64"

	kubevirt "k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/reconciler/pkg/reconciling"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Secretsreators returns the CSI secrets for KubeVirt.
func SecretsReconcilers(ctx context.Context, data *resources.TemplateData) []reconciling.NamedSecretReconcilerFactory {
	creators := []reconciling.NamedSecretReconcilerFactory{
		InfraAccessSecretReconciler(ctx, data),
	}
	return creators
}

// InfraAccessSecretReconciler returns the CSI secrets for KubeVirt.
func InfraAccessSecretReconciler(ctx context.Context, data *resources.TemplateData) reconciling.NamedSecretReconcilerFactory {
	return func() (name string, create reconciling.SecretReconciler) {
		return resources.KubeVirtCSISecretName, func(se *corev1.Secret) (*corev1.Secret, error) {
			se.Labels = resources.BaseAppLabels(resources.KubeVirtCSISecretName, nil)
			if se.Data == nil {
				se.Data = map[string][]byte{}
			}
			credentials, err := resources.GetCredentials(data)
			if err != nil {
				return nil, err
			}
			infraKubeconfig := credentials.Kubevirt.KubeConfig
			infraClient, err := kubevirt.NewClient(infraKubeconfig, kubevirt.ClientOptions{})
			if err != nil {
				return nil, err
			}

			// Get the infra csi SA and compute csiKubeConfig from it
			csiSA := corev1.ServiceAccount{}
			err = infraClient.Get(ctx, types.NamespacedName{Name: resources.KubeVirtCSIServiceAccountName, Namespace: data.Cluster().Status.NamespaceName}, &csiSA)
			if err != nil {
				return nil, err
			}

			// k8s 1.24 by default disabled automatic token creation for service accounts
			// if created < 1.24, tokenName is retrieved from the SA
			// if not, it's created by KKP with a fixed name
			tokenName := resources.KubeVirtCSIServiceAccountName
			if len(csiSA.Secrets) > 0 {
				tokenName = csiSA.Secrets[0].Name
			}

			csiInfraTokenSecret := corev1.Secret{}
			err = infraClient.Get(ctx, types.NamespacedName{Name: tokenName, Namespace: data.Cluster().Status.NamespaceName}, &csiInfraTokenSecret)
			if err != nil {
				return nil, err
			}

			token, err := base64.StdEncoding.DecodeString(string(csiInfraTokenSecret.Data["token"]))
			if err != nil {
				token = csiInfraTokenSecret.Data["token"]
			}

			csiKubeconfig, err := kubevirt.GenerateKubeconfigWithToken(infraClient.RestConfig, &csiSA, string(token))
			if err != nil {
				return nil, err
			}
			se.Data[resources.KubeVirtCSISecretKey] = csiKubeconfig
			return se, nil
		}
	}
}
