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

package kubernetes_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateOrUpdateKubeconfigSecretForCluster(t *testing.T) {
	testCases := []struct {
		name            string
		externalCluster *kubermaticapiv1.ExternalCluster
		kubeconfig      *clientcmdapi.Config
		existingObjects []runtime.Object
		expectedSecret  *corev1.Secret
	}{
		{
			name:            "test: create a new secret",
			existingObjects: []runtime.Object{},
			externalCluster: genExternalCluster("test"),
			kubeconfig:      genKubeconfig("localhost", "test"),
			expectedSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
					Name:            genExternalCluster("test").GetKubeconfigSecretName(),
					Namespace:       resources.KubermaticNamespace,
				},
				Data: map[string][]byte{resources.ExternalClusterKubeconfig: convertKubeconfig(genKubeconfig("localhost", "test"), t)},
				Type: corev1.SecretTypeOpaque,
			},
		},
		{
			name: "test: update existing secret",
			existingObjects: []runtime.Object{
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						ResourceVersion: "1",
						Name:            genExternalCluster("test").GetKubeconfigSecretName(),
						Namespace:       resources.KubermaticNamespace,
					},
					Data: map[string][]byte{resources.ExternalClusterKubeconfig: convertKubeconfig(genKubeconfig("localhost", "test"), t)},
					Type: corev1.SecretTypeOpaque,
				},
			},
			externalCluster: genExternalCluster("test"),
			kubeconfig:      genKubeconfig("192.168.1.1", "updated"),
			expectedSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
					Name:            genExternalCluster("test").GetKubeconfigSecretName(),
					Namespace:       resources.KubermaticNamespace,
				},
				Data: map[string][]byte{resources.ExternalClusterKubeconfig: convertKubeconfig(genKubeconfig("192.168.1.1", "updated"), t)},
				Type: corev1.SecretTypeOpaque,
			},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := fakectrlruntimeclient.NewFakeClientWithScheme(scheme.Scheme, tc.existingObjects...)
			fakeImpersonationClient := func(impCfg restclient.ImpersonationConfig) (ctrlruntimeclient.Client, error) {
				return client, nil
			}
			provider, err := kubernetes.NewExternalClusterProvider(fakeImpersonationClient, client)
			if err != nil {
				t.Fatal(err)
			}

			if err := provider.CreateOrUpdateKubeconfigSecretForCluster(context.Background(), tc.externalCluster, tc.kubeconfig); err != nil {
				t.Fatal(err)
			}

			secret := &corev1.Secret{}
			if err := client.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: tc.externalCluster.GetKubeconfigSecretName(), Namespace: resources.KubermaticNamespace}, secret); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(secret, tc.expectedSecret) {
				t.Fatalf(" diff: %s", diff.ObjectGoPrintSideBySide(tc.expectedSecret, secret))
			}
		})
	}
}

func genExternalCluster(name string) *kubermaticapiv1.ExternalCluster {
	return &kubermaticapiv1.ExternalCluster{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec: kubermaticapiv1.ExternalClusterSpec{
			HumanReadableName: name,
		},
	}
}

func genKubeconfig(server, cluster string) *clientcmdapi.Config {
	return &clientcmdapi.Config{
		Clusters: map[string]*clientcmdapi.Cluster{"test": {
			Server: server,
		}},
		Contexts: map[string]*clientcmdapi.Context{"default": {
			Cluster: cluster,
		}},
		CurrentContext: "default",
	}
}

func convertKubeconfig(kubeconfig *clientcmdapi.Config, t *testing.T) []byte {
	rawData, err := json.Marshal(kubeconfig)
	if err != nil {
		t.Fatal(err)
	}
	return rawData
}
