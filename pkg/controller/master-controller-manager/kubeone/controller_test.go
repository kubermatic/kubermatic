/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package kubeone

import (
	"context"
	"strings"
	"testing"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/machine-controller/sdk/providerconfig"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimefakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func secretRef(name string) *providerconfig.GlobalSecretKeySelector {
	return &providerconfig.GlobalSecretKeySelector{
		ObjectReference: corev1.ObjectReference{Name: name, Namespace: resources.KubermaticNamespace},
	}
}

// TestKubeOneScriptConfigMapsAreUpdated ensures that script ConfigMaps created by an older
// KKP version are updated when a KubeOne job is generated, so that already imported clusters
// receive script changes (e.g. the ssh-agent socket location required by newer KubeOne images).
func TestKubeOneScriptConfigMapsAreUpdated(t *testing.T) {
	externalCluster := &kubermaticv1.ExternalCluster{
		ObjectMeta: metav1.ObjectMeta{Name: "test"},
		Spec: kubermaticv1.ExternalClusterSpec{
			CloudSpec: kubermaticv1.ExternalClusterCloudSpec{
				ProviderName: kubermaticv1.ExternalClusterKubeOneProvider,
				KubeOne: &kubermaticv1.ExternalClusterKubeOneCloudSpec{
					ProviderName:         resources.KubeOneVSphere,
					CredentialsReference: secretRef("credentials"),
					SSHReference:         secretRef("ssh"),
					ManifestReference:    secretRef("manifest"),
				},
			},
		},
	}
	namespace := externalCluster.GetKubeOneNamespaceName()

	testCases := []struct {
		action        string
		configMapName string
		command       string
	}{
		{action: ImportAction, configMapName: KubeOneImportConfigMap, command: "kubeone kubeconfig"},
		{action: UpgradeControlPlaneAction, configMapName: KubeOneUpgradeConfigMap, command: "kubeone apply"},
		{action: MigrateContainerRuntimeAction, configMapName: KubeOneMigrateConfigMap, command: "kubeone migrate to-containerd"},
	}

	for _, tc := range testCases {
		t.Run(tc.action, func(t *testing.T) {
			ctx := context.Background()
			client := ctrlruntimefakeclient.NewClientBuilder().WithObjects(
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{Name: tc.configMapName, Namespace: namespace},
					Data:       map[string]string{"script.sh": "eval `ssh-agent` > /dev/null\n"},
				},
				&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "credentials", Namespace: resources.KubermaticNamespace}},
				&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "ssh", Namespace: resources.KubermaticNamespace}},
				&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "manifest", Namespace: resources.KubermaticNamespace}},
			).Build()

			r := &reconciler{Client: client, log: zap.NewNop().Sugar()}
			data := resources.NewTemplateDataBuilder().Build()

			if _, err := r.generateKubeOneActionJob(ctx, r.log, data, externalCluster, tc.action); err != nil {
				t.Fatalf("failed to generate kubeone job: %v", err)
			}

			configMap := &corev1.ConfigMap{}
			if err := client.Get(ctx, types.NamespacedName{Name: tc.configMapName, Namespace: namespace}, configMap); err != nil {
				t.Fatalf("failed to get ConfigMap: %v", err)
			}

			script := configMap.Data["script.sh"]
			if !strings.Contains(script, "ssh-agent -a /tmp/kubeone-agent.sock") {
				t.Errorf("expected the ssh-agent socket to be bound to /tmp, got script:\n%s", script)
			}
			if !strings.Contains(script, tc.command) {
				t.Errorf("expected the script to run %q, got script:\n%s", tc.command, script)
			}
		})
	}
}
