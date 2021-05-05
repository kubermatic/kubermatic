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
	"reflect"
	"testing"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/client-go/kubernetes/scheme"
	restclient "k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const defaultKubeconfig = "YXBpVmVyc2lvbjogdjEKY2x1c3RlcnM6Ci0gY2x1c3RlcjoKICAgIGNlcnRpZmljYXRlLWF1dGhvcml0eS1kYXRhOiBZWEJwVm1WeWMybHZiam9nZGpFS1kyeDFjM1JsY25NNkNpMGdZMngxYzNSbGNqb0tJQ0FnSUdObGNuUnBabWxqWVhSbExXRjFkR2h2Y21sMGVTMWtZWFJoT2lCaFltTUtJQ0FnSUhObGNuWmxjam9nYUhSMGNITTZMeTlzYzJoNmRtTm5PR3RrTG1WMWNtOXdaUzEzWlhOME15MWpMbVJsZGk1cmRXSmxjbTFoZEdsakxtbHZPak14TWpjMUNpQWdibUZ0WlRvZ2JITm9lblpqWnpoclpBcGpiMjUwWlhoMGN6b0tMU0JqYjI1MFpYaDBPZ29nSUNBZ1kyeDFjM1JsY2pvZ2JITm9lblpqWnpoclpBb2dJQ0FnZFhObGNqb2daR1ZtWVhWc2RBb2dJRzVoYldVNklHUmxabUYxYkhRS1kzVnljbVZ1ZEMxamIyNTBaWGgwT2lCa1pXWmhkV3gwQ210cGJtUTZJRU52Ym1acFp3cHdjbVZtWlhKbGJtTmxjem9nZTMwS2RYTmxjbk02Q2kwZ2JtRnRaVG9nWkdWbVlYVnNkQW9nSUhWelpYSTZDaUFnSUNCMGIydGxiam9nWVdGaExtSmlZZ289CiAgICBzZXJ2ZXI6IGh0dHBzOi8vbG9jYWxob3N0OjMwODA4CiAgbmFtZTogaHZ3OWs0c2djbApjb250ZXh0czoKLSBjb250ZXh0OgogICAgY2x1c3RlcjogaHZ3OWs0c2djbAogICAgdXNlcjogZGVmYXVsdAogIG5hbWU6IGRlZmF1bHQKY3VycmVudC1jb250ZXh0OiBkZWZhdWx0CmtpbmQ6IENvbmZpZwpwcmVmZXJlbmNlczoge30KdXNlcnM6Ci0gbmFtZTogZGVmYXVsdAogIHVzZXI6CiAgICB0b2tlbjogejlzaDc2LjI0ZGNkaDU3czR6ZGt4OGwK"

func TestCreateOrUpdateKubeconfigSecretForCluster(t *testing.T) {
	testCases := []struct {
		name            string
		externalCluster *kubermaticapiv1.ExternalCluster
		kubeconfig      string
		existingObjects []ctrlruntimeclient.Object
		expectedSecret  *corev1.Secret
	}{
		{
			name:            "test: create a new secret",
			existingObjects: []ctrlruntimeclient.Object{},
			externalCluster: genExternalCluster("test", "projectID"),
			kubeconfig:      defaultKubeconfig,
			expectedSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
					Name:            genExternalCluster("test", "projectID").GetKubeconfigSecretName(),
					Namespace:       resources.KubermaticNamespace,
					Labels:          map[string]string{kubermaticapiv1.ProjectIDLabelKey: "projectID"},
				},
				Data: map[string][]byte{resources.ExternalClusterKubeconfig: []byte(defaultKubeconfig)},
				Type: corev1.SecretTypeOpaque,
			},
		},
		{
			name: "test: update existing secret",
			existingObjects: []ctrlruntimeclient.Object{
				&corev1.Secret{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Secret",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						ResourceVersion: "1",
						Name:            genExternalCluster("test", "projectID").GetKubeconfigSecretName(),
						Namespace:       resources.KubermaticNamespace,
					},
					Data: map[string][]byte{resources.ExternalClusterKubeconfig: []byte("abc")},
					Type: corev1.SecretTypeOpaque,
				},
			},
			externalCluster: genExternalCluster("test", "projectID"),
			kubeconfig:      defaultKubeconfig,
			expectedSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "2",
					Name:            genExternalCluster("test", "projectID").GetKubeconfigSecretName(),
					Namespace:       resources.KubermaticNamespace,
					Labels:          map[string]string{kubermaticapiv1.ProjectIDLabelKey: "projectID"},
				},
				Data: map[string][]byte{resources.ExternalClusterKubeconfig: []byte(defaultKubeconfig)},
				Type: corev1.SecretTypeOpaque,
			},
		},
	}

	for idx := range testCases {
		tc := testCases[idx]
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			client := fakectrlruntimeclient.
				NewClientBuilder().
				WithScheme(scheme.Scheme).
				WithObjects(tc.existingObjects...).
				Build()

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

func genExternalCluster(name, projectID string) *kubermaticapiv1.ExternalCluster {
	return &kubermaticapiv1.ExternalCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{kubermaticapiv1.ProjectIDLabelKey: projectID},
		},
		Spec: kubermaticapiv1.ExternalClusterSpec{
			HumanReadableName: name,
		},
	}
}
