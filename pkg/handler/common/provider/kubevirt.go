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

package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"

	kubevirtv1 "kubevirt.io/api/core/v1"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	"k8c.io/kubermatic/v2/pkg/util/errors"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var NewKubeVirtClient = func(kubeconfig string) (ctrlruntimeclient.Client, error) {
	config, err := base64.StdEncoding.DecodeString(kubeconfig)
	if err != nil {
		// should not happen, always sent base64 encoded
		return nil, err
	}

	clientConfig, err := clientcmd.RESTConfigFromKubeConfig(config)
	if err != nil {
		return nil, err
	}

	client, err := ctrlruntimeclient.New(clientConfig, ctrlruntimeclient.Options{})
	if err != nil {
		return nil, err
	}

	if err := kubevirtv1.AddToScheme(client.Scheme()); err != nil {
		return nil, err
	}

	return client, nil
}

func getKvKubeConfigFromCredentials(ctx context.Context, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	userInfoGetter provider.UserInfoGetter, projectID, clusterID string) (string, error) {
	clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return "", err
	}

	if cluster.Spec.Cloud.Kubevirt == nil {
		return "", errors.NewNotFound("cloud spec for ", clusterID)
	}

	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return "", errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	kvKubeconfig, err := kubevirt.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString([]byte(kvKubeconfig)), nil
}

func KubeVirtVMIPresets(ctx context.Context, kubeconfig string, cluster *kubermaticv1.Cluster) (apiv2.VirtualMachineInstancePresetList, error) {
	client, err := NewKubeVirtClient(kubeconfig)
	if err != nil {
		return nil, err
	}

	vmiPresets := kubevirtv1.VirtualMachineInstancePresetList{}
	if err := client.List(ctx, &vmiPresets, ctrlruntimeclient.InNamespace(metav1.NamespaceDefault)); err != nil {
		return nil, err
	}

	// Add a standard preset to the list
	vmiPresets.Items = append(vmiPresets.Items, *kubevirt.GetKubermaticStandardPreset())

	res := apiv2.VirtualMachineInstancePresetList{}
	for _, vmiPreset := range vmiPresets.Items {
		preset, err := newAPIVirtualMachineInstancePreset(&vmiPreset)
		if err != nil {
			return nil, err
		}
		res = append(res, *preset)

		// Reconcile each Preset in the dedicated Namespace.
		// Update flow: cluster is not nil, reconciliation of Presets is done here.
		// Creation flow: cluster is nil, reconciliation in then done by the ReconcileCluster.
		if cluster != nil {
			presetCreators := []reconciling.NamedKubeVirtV1VirtualMachineInstancePresetCreatorGetter{
				presetCreator(&vmiPreset),
			}
			if err := reconciling.ReconcileKubeVirtV1VirtualMachineInstancePresets(ctx, presetCreators, cluster.Status.NamespaceName, client); err != nil {
				return nil, err
			}
		}
	}

	return res, nil
}

func presetCreator(preset *kubevirtv1.VirtualMachineInstancePreset) reconciling.NamedKubeVirtV1VirtualMachineInstancePresetCreatorGetter {
	return func() (string, reconciling.KubeVirtV1VirtualMachineInstancePresetCreator) {
		return preset.Name, func(p *kubevirtv1.VirtualMachineInstancePreset) (*kubevirtv1.VirtualMachineInstancePreset, error) {
			p.Labels = preset.Labels
			p.Spec = preset.Spec
			return p, nil
		}
	}
}

func KubeVirtVMIPresetsWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	projectID, clusterID string) (interface{}, error) {
	kvKubeconfig, err := getKvKubeConfigFromCredentials(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}

	return KubeVirtVMIPresets(ctx, kvKubeconfig, cluster)
}

func newAPIVirtualMachineInstancePreset(vmiPreset *kubevirtv1.VirtualMachineInstancePreset) (*apiv2.VirtualMachineInstancePreset, error) {
	spec, err := json.Marshal(vmiPreset.Spec)
	if err != nil {
		return nil, err
	}

	return &apiv2.VirtualMachineInstancePreset{
		Name:      vmiPreset.ObjectMeta.Name,
		Namespace: vmiPreset.ObjectMeta.Namespace,
		Spec:      string(spec),
	}, nil
}

func newAPIStorageClass(sc *storagev1.StorageClass) *apiv2.StorageClass {
	return &apiv2.StorageClass{
		Name: sc.ObjectMeta.Name,
	}
}

func KubeVirtStorageClasses(ctx context.Context, kubeconfig string) (apiv2.StorageClassList, error) {
	client, err := NewKubeVirtClient(kubeconfig)
	if err != nil {
		return nil, err
	}

	storageClassList := storagev1.StorageClassList{}
	if err := client.List(ctx, &storageClassList); err != nil {
		return nil, err
	}

	res := apiv2.StorageClassList{}
	for _, sc := range storageClassList.Items {
		res = append(res, *newAPIStorageClass(&sc))
	}

	return res, nil
}

func KubeVirtStorageClassesWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	projectID, clusterID string) (interface{}, error) {
	kvKubeconfig, err := getKvKubeConfigFromCredentials(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	return KubeVirtStorageClasses(ctx, kvKubeconfig)
}
