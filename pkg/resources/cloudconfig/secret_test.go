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

package cloudconfig

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestKubeVirtInfraSecretReconciler(t *testing.T) {
	testCases := []struct {
		name               string
		infraKubeconfig    string
		expectedSecretData string
		expectedError      error
	}{
		{name: "with kubeconfig", infraKubeconfig: "ZmFrZWt1YmVjb25maWcK", expectedSecretData: "ZmFrZWt1YmVjb25maWcK", expectedError: nil},
		{name: "with empty kubeconfig", infraKubeconfig: "", expectedSecretData: "", expectedError: errors.New("configVar is nil")},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			fakeTemplateData := createFakeTemplateData(test.infraKubeconfig)
			_, creator := KubeVirtInfraSecretReconciler(fakeTemplateData)()
			secret, err := creator(&corev1.Secret{})
			assert.Equal(t, test.expectedError, err)
			var actualSecretData string
			if secret != nil {
				actualSecretData = string(secret.Data[resources.KubeVirtInfraSecretKey])
			}
			assert.Equal(t, test.expectedSecretData, actualSecretData)
		})
	}
}

func createFakeTemplateData(kubeconfig string) *resources.TemplateData {
	return resources.NewTemplateDataBuilder().WithCluster(&kubermaticv1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fake-Kubevirt-Cluster",
		},
		Spec: kubermaticv1.ClusterSpec{
			Cloud: kubermaticv1.CloudSpec{Kubevirt: &kubermaticv1.KubevirtCloudSpec{
				Kubeconfig:    kubeconfig,
				CSIKubeconfig: kubeconfig,
			}},
		},
	}).Build()
}
