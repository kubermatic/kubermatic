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

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
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

const (
	defaultKubeconfig      = "YXBpVmVyc2lvbjogdjEKY2x1c3RlcnM6Ci0gY2x1c3RlcjoKICAgIGNlcnRpZmljYXRlLWF1dGhvcml0eS1kYXRhOiBZWEJwVm1WeWMybHZiam9nZGpFS1kyeDFjM1JsY25NNkNpMGdZMngxYzNSbGNqb0tJQ0FnSUdObGNuUnBabWxqWVhSbExXRjFkR2h2Y21sMGVTMWtZWFJoT2lCaFltTUtJQ0FnSUhObGNuWmxjam9nYUhSMGNITTZMeTlzYzJoNmRtTm5PR3RrTG1WMWNtOXdaUzEzWlhOME15MWpMbVJsZGk1cmRXSmxjbTFoZEdsakxtbHZPak14TWpjMUNpQWdibUZ0WlRvZ2JITm9lblpqWnpoclpBcGpiMjUwWlhoMGN6b0tMU0JqYjI1MFpYaDBPZ29nSUNBZ1kyeDFjM1JsY2pvZ2JITm9lblpqWnpoclpBb2dJQ0FnZFhObGNqb2daR1ZtWVhWc2RBb2dJRzVoYldVNklHUmxabUYxYkhRS1kzVnljbVZ1ZEMxamIyNTBaWGgwT2lCa1pXWmhkV3gwQ210cGJtUTZJRU52Ym1acFp3cHdjbVZtWlhKbGJtTmxjem9nZTMwS2RYTmxjbk02Q2kwZ2JtRnRaVG9nWkdWbVlYVnNkQW9nSUhWelpYSTZDaUFnSUNCMGIydGxiam9nWVdGaExtSmlZZ289CiAgICBzZXJ2ZXI6IGh0dHBzOi8vbG9jYWxob3N0OjMwODA4CiAgbmFtZTogaHZ3OWs0c2djbApjb250ZXh0czoKLSBjb250ZXh0OgogICAgY2x1c3RlcjogaHZ3OWs0c2djbAogICAgdXNlcjogZGVmYXVsdAogIG5hbWU6IGRlZmF1bHQKY3VycmVudC1jb250ZXh0OiBkZWZhdWx0CmtpbmQ6IENvbmZpZwpwcmVmZXJlbmNlczoge30KdXNlcnM6Ci0gbmFtZTogZGVmYXVsdAogIHVzZXI6CiAgICB0b2tlbjogejlzaDc2LjI0ZGNkaDU3czR6ZGt4OGwK"
	defaultAccessKeyID     = "abc"
	defaultSecretAccessKey = "abc"
	defaultServiceAccount  = "abc"
)

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

func TestCreateOrUpdateCloudSecretForCluster(t *testing.T) {
	testCases := []struct {
		name            string
		projectID       string
		clusterID       string
		externalCluster *kubermaticapiv1.Cluster
		existingObjects []ctrlruntimeclient.Object
		expectedSecret  *corev1.Secret
		cloudSpec       *apiv2.ExternalClusterCloudSpec
	}{
		{
			name:            "test: create a new eks secret",
			projectID:       "projectID",
			clusterID:       "test",
			existingObjects: []ctrlruntimeclient.Object{},
			cloudSpec: &apiv2.ExternalClusterCloudSpec{
				EKS: &apiv2.EKSCloudSpec{
					Name:            "test",
					AccessKeyID:     defaultAccessKeyID,
					SecretAccessKey: defaultSecretAccessKey,
				},
			},
			externalCluster: genEKSCluster("test", "eu-central-1", "projectID"),
			expectedSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
					Name:            genEKSCluster("test", "eu-central-1", "projectID").GetSecretName(),
					Namespace:       resources.KubermaticNamespace,
				},
				Data: map[string][]byte{resources.ExternalEKSClusterAccessKeyID: []byte(defaultAccessKeyID), resources.ExternalEKSClusterSecretAccessKey: []byte(defaultSecretAccessKey)},
				Type: corev1.SecretTypeOpaque,
			},
		},
		{
			name:            "test: create a new gke secret",
			projectID:       "projectID",
			clusterID:       "test",
			existingObjects: []ctrlruntimeclient.Object{},
			cloudSpec: &apiv2.ExternalClusterCloudSpec{
				GKE: &apiv2.GKECloudSpec{
					Name:           "test",
					ServiceAccount: defaultAccessKeyID,
				},
			},
			externalCluster: genGKECluster("test", "eu-central-1", "projectID"),
			expectedSecret: &corev1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					ResourceVersion: "1",
					Name:            genGKECluster("test", "eu-central-1", "projectID").GetSecretName(),
					Namespace:       resources.KubermaticNamespace,
				},
				Data: map[string][]byte{resources.ExternalGKEClusterSeriveAccount: []byte(defaultAccessKeyID)},
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
			credentialRef, err := provider.CreateOrUpdateCredentialSecretForCluster(context.Background(), tc.cloudSpec, tc.projectID, tc.clusterID)
			if err != nil {
				t.Fatal(err)
			}

			secret := &corev1.Secret{}
			if err := client.Get(context.Background(), ctrlruntimeclient.ObjectKey{Name: credentialRef.Name, Namespace: resources.KubermaticNamespace}, secret); err != nil {
				t.Fatal(err)
			}
			tc.expectedSecret.ObjectMeta.Labels = secret.ObjectMeta.Labels
			tc.expectedSecret.Data = secret.Data
			if !reflect.DeepEqual(tc.expectedSecret, secret) {
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

func genEKSCluster(name, region, projectID string) *kubermaticapiv1.Cluster {
	cluster := &kubermaticapiv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: projectID},
		},
		Spec: kubermaticapiv1.ClusterSpec{
			Cloud: kubermaticapiv1.CloudSpec{},
		},
	}
	cluster.Spec.Cloud.AWS = &kubermaticapiv1.AWSCloudSpec{
		AccessKeyID:     defaultAccessKeyID,
		SecretAccessKey: defaultSecretAccessKey,
	}
	return cluster
}

func genGKECluster(name, region, projectID string) *kubermaticapiv1.Cluster {
	cluster := &kubermaticapiv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{kubermaticv1.ProjectIDLabelKey: projectID},
		},
		Spec: kubermaticapiv1.ClusterSpec{
			Cloud: kubermaticapiv1.CloudSpec{},
		},
	}
	cluster.Spec.Cloud.GCP = &kubermaticapiv1.GCPCloudSpec{
		ServiceAccount: defaultServiceAccount,
	}
	return cluster
}
