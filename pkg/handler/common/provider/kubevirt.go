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
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	kubevirtv1 "kubevirt.io/api/core/v1"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	kvmanifests "k8c.io/kubermatic/v2/pkg/resources/cloudcontroller/kubevirtmanifests"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	"k8s.io/apimachinery/pkg/api/resource"
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
		return "", utilerrors.NewNotFound("cloud spec for ", clusterID)
	}

	assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
	if !ok {
		return "", utilerrors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
	}

	secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
	kvKubeconfig, err := kubevirt.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString([]byte(kvKubeconfig)), nil
}

func KubeVirtVMIPresets(ctx context.Context, kubeconfig string, cluster *kubermaticv1.Cluster, settingsProvider provider.SettingsProvider) (apiv2.VirtualMachineInstancePresetList, error) {
	client, err := NewKubeVirtClient(kubeconfig)
	if err != nil {
		return nil, err
	}

	vmiPresets := kubevirtv1.VirtualMachineInstancePresetList{}
	if err := client.List(ctx, &vmiPresets, ctrlruntimeclient.InNamespace(metav1.NamespaceDefault)); err != nil {
		return nil, err
	}

	// Add a standard preset to the list
	vmiPresets.Items = append(vmiPresets.Items, kubevirt.GetKubermaticStandardPresets(client, &kvmanifests.StandardPresetGetter{})...)

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
	settings, err := settingsProvider.GetGlobalSettings(ctx)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	return filterVMIPresets(res, settings.Spec.MachineDeploymentVMResourceQuota), nil
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
	projectID, clusterID string, settingsProvider provider.SettingsProvider) (interface{}, error) {
	kvKubeconfig, err := getKvKubeConfigFromCredentials(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}

	return KubeVirtVMIPresets(ctx, kvKubeconfig, cluster, settingsProvider)
}

func KubeVirtVMIPreset(ctx context.Context, kubeconfig, flavor string) (*kubevirtv1.VirtualMachineInstancePreset, error) {
	client, err := NewKubeVirtClient(kubeconfig)
	if err != nil {
		return nil, err
	}

	vmiPresets := &kubevirtv1.VirtualMachineInstancePresetList{}
	if err := client.List(ctx, vmiPresets, ctrlruntimeclient.InNamespace(metav1.NamespaceDefault)); err != nil {
		return nil, err
	}

	// Add a standard preset to the list
	vmiPresets.Items = append(vmiPresets.Items, kubevirt.GetKubermaticStandardPresets(client, &kvmanifests.StandardPresetGetter{})...)

	for _, vmiPreset := range vmiPresets.Items {
		if strings.EqualFold(vmiPreset.Name, flavor) {
			return &vmiPreset, nil
		}
	}
	return nil, fmt.Errorf("KubeVirt VMI preset %q not found", flavor)
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

func KubeVirtStorageClasses(ctx context.Context, kubeconfig string) (apiv2.StorageClassList, error) {
	client, err := NewKubeVirtClient(kubeconfig)
	if err != nil {
		return nil, err
	}

	return kubevirt.ListStorageClasses(ctx, client, nil)
}

func KubeVirtStorageClassesWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	projectID, clusterID string) (interface{}, error) {
	kvKubeconfig, err := getKvKubeConfigFromCredentials(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	return KubeVirtStorageClasses(ctx, kvKubeconfig)
}

func filterVMIPresets(vmiPresets apiv2.VirtualMachineInstancePresetList, quota kubermaticv1.MachineDeploymentVMResourceQuota) apiv2.VirtualMachineInstancePresetList {
	filteredVMIPresets := apiv2.VirtualMachineInstancePresetList{}

	// Range over the records and apply all the filters to each record.
	// If the record passes all the filters, add it to the final slice.
	for _, vmiPreset := range vmiPresets {
		presetSpec := kubevirtv1.VirtualMachineInstancePresetSpec{}
		if err := json.Unmarshal([]byte(vmiPreset.Spec), &presetSpec); err != nil {
			log.Logger.Errorf("skipping VMIPreset:%s, parsing preset.Spec failed:%v", vmiPreset.Name, err)
			continue
		}

		cpu, memory, err := GetKubeVirtPresetResourceDetails(presetSpec)
		if err != nil {
			log.Logger.Errorf("skipping VMIPreset:%s, fetching presetResourceDetails failed:%v", vmiPreset.Name, err)
			continue
		}

		if handlercommon.FilterCPU(int(cpu.AsApproximateFloat64()), quota.MinCPU, quota.MaxCPU) && handlercommon.FilterMemory(int(memory.Value()/(1<<30)), quota.MinRAM, quota.MaxRAM) {
			filteredVMIPresets = append(filteredVMIPresets, vmiPreset)
		}
	}
	return filteredVMIPresets
}

// GetKubeVirtPresetResourceDetails extracts cpu and mem resource requests from the kubevirt preset
// for CPU, take the value by priority:
// - check if spec.cpu is set, if socket and threads are set then do the calculation, use that
// - if resource request is set, use that
// - if resource limit is set, use that
// for memory, take the value by priority:
// - if resource request is set, use that
// - if resource limit is set, use that.
func GetKubeVirtPresetResourceDetails(presetSpec kubevirtv1.VirtualMachineInstancePresetSpec) (resource.Quantity, resource.Quantity, error) {
	var err error
	// Get CPU
	cpuReq := resource.Quantity{}

	if presetSpec.Domain.CPU != nil {
		if !presetSpec.Domain.Resources.Requests.Cpu().IsZero() || !presetSpec.Domain.Resources.Limits.Cpu().IsZero() {
			return resource.Quantity{}, resource.Quantity{}, errors.New("should not specify both spec.domain.cpu and spec.domain.resources.[requests/limits].cpu in VMIPreset")
		}
		cores := presetSpec.Domain.CPU.Cores
		if cores == 0 {
			cores = 1
		}
		// if threads and sockets are set, calculate VCPU
		threads := presetSpec.Domain.CPU.Threads
		if threads == 0 {
			threads = 1
		}
		sockets := presetSpec.Domain.CPU.Sockets
		if sockets == 0 {
			sockets = 1
		}

		cpuReq, err = resource.ParseQuantity(strconv.Itoa(int(cores * threads * sockets)))
		if err != nil {
			return resource.Quantity{}, resource.Quantity{}, fmt.Errorf("error parsing calculated KubeVirt VCPU: %w", err)
		}
	} else {
		if !presetSpec.Domain.Resources.Requests.Cpu().IsZero() {
			cpuReq = *presetSpec.Domain.Resources.Requests.Cpu()
		}
		if !presetSpec.Domain.Resources.Limits.Cpu().IsZero() {
			cpuReq = *presetSpec.Domain.Resources.Limits.Cpu()
		}
	}

	// get MEM
	memReq := resource.Quantity{}
	if presetSpec.Domain.Resources.Requests.Memory().IsZero() && presetSpec.Domain.Resources.Limits.Memory().IsZero() {
		return resource.Quantity{}, resource.Quantity{}, errors.New("spec.domain.resources.[requests/limits].memory must be set in VMIPreset")
	}
	if !presetSpec.Domain.Resources.Requests.Memory().IsZero() {
		memReq = *presetSpec.Domain.Resources.Requests.Memory()
	}
	if !presetSpec.Domain.Resources.Limits.Memory().IsZero() {
		memReq = *presetSpec.Domain.Resources.Limits.Memory()
	}

	return cpuReq, memReq, nil
}
