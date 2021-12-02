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
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	kubevirtv1 "kubevirt.io/api/core/v1"
)

const (
	// soon the namespace will be changed to the cluster namespace
	kubevirtns = "default"
)

var NewKubevirtClientSet = func(kubeconfig string) (kubevirtcli.Interface, kubernetes.Interface, error) {
	config, err := base64.StdEncoding.DecodeString(kubeconfig)
	if err != nil {
		// if the decoding failed, the kubeconfig is sent already decoded without the need of decoding it,
		// for example the value has been read from Vault during the ci tests, which is saved as json format.
		config = []byte(kubeconfig)
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
	return kvKubeconfig, nil

}

// LIST VmiPreset

func KubevirtVmiPresets(kubeconfig string) (apiv2.VirtualMachineInstancePresetList, error) {

	kvclient, _, err := NewKubevirtClientSet(kubeconfig)
	if err != nil {
		return nil, err
	}
	vmipresetlist, err := kvclient.KubevirtV1().VirtualMachineInstancePresets(kubevirtns).List(context.Background(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}
	res := apiv2.VirtualMachineInstancePresetList{}
	for _, vmiPreset := range vmipresetlist.Items {
		res = append(res, *newAPIVirtualMachineInstancePreset(&vmiPreset))
	}

	return res, nil
}

func KubevirtVmiPresetsWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	projectID, clusterID string) (interface{}, error) {
	kvKubeconfig, err := getKvKubeConfigFromCredentials(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	return KubevirtVmiPresets(kvKubeconfig)
}

// GET VmiPreset
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

func KubevirtVmiPreset(kubeconfig, presetName string) (*apiv2.VirtualMachineInstancePreset, error) {

	kvclient, _, err := NewKubevirtClientSet(kubeconfig)
	if err != nil {
		return nil, err
	}
	vmiPreset, err := kvclient.KubevirtV1().VirtualMachineInstancePresets(kubevirtns).Get(context.Background(), presetName, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return newAPIVirtualMachineInstancePreset(vmiPreset), nil
}

func KubevirtVmiPresetWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	projectID, clusterID, presetName string) (interface{}, error) {
	kvKubeconfig, err := getKvKubeConfigFromCredentials(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	return KubevirtVmiPreset(kvKubeconfig, presetName)
}

// LIST StorageClass

func KubevirtStorageClasses(kubeconfig string) (*apiv2.StorageClassList, error) {

	_, cli, err := NewKubevirtClientSet(kubeconfig)
	if err != nil {
		return nil, err
	}
	storageClassList, err := cli.StorageV1().StorageClasses().List(context.Background(), v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return &apiv2.StorageClassList{StorageClassList: storageClassList}, nil
}

func KubevirtStorageClassesWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	projectID, clusterID string) (interface{}, error) {
	kvKubeconfig, err := getKvKubeConfigFromCredentials(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	return KubevirtStorageClasses(kvKubeconfig)

}

// GET StorageClass

func KubevirtStorageClass(kubeconfig, storageClass string) (*apiv2.StorageClass, error) {

	_, cli, err := NewKubevirtClientSet(kubeconfig)
	if err != nil {
		return nil, err
	}

	storageclass, err := cli.StorageV1().StorageClasses().Get(context.Background(), storageClass, v1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return &apiv2.StorageClass{StorageClass: storageclass}, nil
}

func KubevirtStorageClassWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	projectID, clusterID, storageClass string) (interface{}, error) {
	kvKubeconfig, err := getKvKubeConfigFromCredentials(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	return KubevirtStorageClass(kvKubeconfig, storageClass)

}
