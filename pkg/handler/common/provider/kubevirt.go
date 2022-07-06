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
	"net/http"

	kubevirtv1 "kubevirt.io/api/core/v1"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt"
	kubevirtcli "k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt/kubevirtcli/client/versioned"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var NewKubeVirtClientSet = func(kubeconfig string) (kubevirtcli.Interface, kubernetes.Interface, error) {
	config, err := base64.StdEncoding.DecodeString(kubeconfig)
	if err != nil {
		// should not happen, always sent base64 encoded
		return nil, nil, err
	}
	clientConfig, err := clientcmd.RESTConfigFromKubeConfig(config)
	if err != nil {
		return nil, nil, err
	}

	kubevirtcli, err := kubevirtcli.NewForConfig(clientConfig)
	if err != nil {
		return nil, nil, err
	}
	k8scli, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return kubevirtcli, nil, err
	}
	return kubevirtcli, k8scli, nil

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

func KubeVirtVMIPresets(kubeconfig string) (apiv2.VirtualMachineInstancePresetList, error) {

	kvClient, _, err := NewKubeVirtClientSet(kubeconfig)
	if err != nil {
		return nil, err
	}
	vmiPresets, err := kvClient.KubevirtV1().VirtualMachineInstancePresets(metav1.NamespaceDefault).List(context.Background(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	res := apiv2.VirtualMachineInstancePresetList{}
	for _, vmiPreset := range vmiPresets.Items {
		res = append(res, *newAPIVirtualMachineInstancePreset(&vmiPreset))
	}

	return res, nil
}

func KubeVirtVMIPresetsWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	projectID, clusterID string) (interface{}, error) {
	kvKubeconfig, err := getKvKubeConfigFromCredentials(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	return KubeVirtVMIPresets(kvKubeconfig)
}

func newAPIVirtualMachineInstancePreset(vmiPreset *kubevirtv1.VirtualMachineInstancePreset) *apiv2.VirtualMachineInstancePreset {
	return &apiv2.VirtualMachineInstancePreset{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                string(vmiPreset.ObjectMeta.UID),
			Name:              vmiPreset.ObjectMeta.Name,
			Annotations:       vmiPreset.Annotations,
			CreationTimestamp: apiv1.Time(vmiPreset.CreationTimestamp),
			DeletionTimestamp: (*apiv1.Time)(vmiPreset.DeletionTimestamp),
		},
		Spec: apiv2.VirtualMachineInstancePresetSpec{
			Selector: vmiPreset.Spec.Selector,
			Domain: &apiv2.DomainSpec{
				Resources: corev1.ResourceRequirements{
					Limits:   vmiPreset.Spec.Domain.Resources.Limits,
					Requests: vmiPreset.Spec.Domain.Resources.Requests,
				},
				CPU:             vmiPreset.Spec.Domain.CPU,
				Memory:          vmiPreset.Spec.Domain.Memory,
				Machine:         vmiPreset.Spec.Domain.Machine,
				Firmware:        vmiPreset.Spec.Domain.Firmware,
				Clock:           vmiPreset.Spec.Domain.Clock,
				Features:        vmiPreset.Spec.Domain.Features,
				Devices:         vmiPreset.Spec.Domain.Devices,
				IOThreadsPolicy: vmiPreset.Spec.Domain.IOThreadsPolicy,
				Chassis:         vmiPreset.Spec.Domain.Chassis,
			},
		},
	}
}

func newAPIStorageClass(sc *storagev1.StorageClass) *apiv2.StorageClass {
	return &apiv2.StorageClass{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                string(sc.ObjectMeta.UID),
			Name:              sc.ObjectMeta.Name,
			Annotations:       sc.Annotations,
			CreationTimestamp: apiv1.Time(sc.CreationTimestamp),
			DeletionTimestamp: (*apiv1.Time)(sc.DeletionTimestamp),
		},
		Provisioner:          sc.Provisioner,
		Parameters:           sc.Parameters,
		ReclaimPolicy:        sc.ReclaimPolicy,
		MountOptions:         sc.MountOptions,
		AllowVolumeExpansion: sc.AllowVolumeExpansion,
		VolumeBindingMode:    sc.VolumeBindingMode,
		AllowedTopologies:    sc.AllowedTopologies,
	}
}

func KubeVirtStorageClasses(kubeconfig string) (apiv2.StorageClassList, error) {

	_, cli, err := NewKubeVirtClientSet(kubeconfig)
	if err != nil {
		return nil, err
	}
	storageClassList, err := cli.StorageV1().StorageClasses().List(context.Background(), v1.ListOptions{})
	if err != nil {
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

	return KubeVirtStorageClasses(kvKubeconfig)
}
