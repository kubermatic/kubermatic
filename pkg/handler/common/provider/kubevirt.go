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
	kvinstancetypev1alpha1 "kubevirt.io/api/instancetype/v1alpha1"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	handlercommon "k8c.io/kubermatic/v2/pkg/handler/common"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt"
	kvmanifests "k8c.io/kubermatic/v2/pkg/provider/cloud/kubevirt/manifests"
	kubernetesprovider "k8c.io/kubermatic/v2/pkg/provider/kubernetes"
	"k8c.io/kubermatic/v2/pkg/resources/reconciling"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var NewKubeVirtClient = kubevirt.NewClient

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

// kubeVirtPresets returns the kubevirtv1.VirtualMachineInstancePreset from the `default` namespace, concatenated with Kubermatic standard presets.
func kubeVirtPresets(ctx context.Context, client ctrlruntimeclient.Client, kubeconfig string) (*kubevirtv1.VirtualMachineInstancePresetList, error) {
	// From `default` namespace.
	vmiPresets := &kubevirtv1.VirtualMachineInstancePresetList{}
	if err := client.List(ctx, vmiPresets, ctrlruntimeclient.InNamespace(metav1.NamespaceDefault)); err != nil {
		return nil, err
	}

	// Add standard presets to the list.
	vmiPresets.Items = append(vmiPresets.Items, kubevirt.GetKubermaticStandardPresets(client, &kvmanifests.StandardPresetGetter{})...)

	return vmiPresets, nil
}

func KubeVirtVMIPresets(ctx context.Context, kubeconfig string, cluster *kubermaticv1.Cluster, settingsProvider provider.SettingsProvider) (apiv2.VirtualMachineInstancePresetList, error) {
	client, err := NewKubeVirtClient(kubeconfig, kubevirt.ClientOptions{})
	if err != nil {
		return nil, err
	}

	// KubeVirt presets concatenated with Kubermatic standards.
	vmiPresets, err := kubeVirtPresets(ctx, client, kubeconfig)
	if err != nil {
		return nil, err
	}

	// Convert to API objects.
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

	// Quota filtering
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
	client, err := NewKubeVirtClient(kubeconfig, kubevirt.ClientOptions{})
	if err != nil {
		return nil, err
	}

	// KubeVirt presets concatenated with Kubermatic standards.
	vmiPresets, err := kubeVirtPresets(ctx, client, kubeconfig)
	if err != nil {
		return nil, err
	}

	for _, vmiPreset := range vmiPresets.Items {
		if strings.EqualFold(vmiPreset.Name, flavor) {
			return &vmiPreset, nil
		}
	}
	return nil, fmt.Errorf("KubeVirt VMI preset %q not found", flavor)
}

func KubeVirtInstancetypesWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	projectID, clusterID string, settingsProvider provider.SettingsProvider) (interface{}, error) {
	kvKubeconfig, err := getKvKubeConfigFromCredentials(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}

	return KubeVirtInstancetypes(ctx, kvKubeconfig, cluster, settingsProvider)
}

func KubeVirtPreferencesWithClusterCredentialsEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider,
	projectID, clusterID string, settingsProvider provider.SettingsProvider) (interface{}, error) {
	kvKubeconfig, err := getKvKubeConfigFromCredentials(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID)
	if err != nil {
		return nil, err
	}

	cluster, err := handlercommon.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
	if err != nil {
		return nil, err
	}

	return KubeVirtPreferences(ctx, kvKubeconfig, cluster, settingsProvider)
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
	client, err := NewKubeVirtClient(kubeconfig, kubevirt.ClientOptions{})
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

func isCPUSpecified(cpu *kubevirtv1.CPU) bool {
	return cpu != nil && (cpu.Cores != 0 || cpu.Threads != 0 || cpu.Sockets != 0)
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

	if isCPUSpecified(presetSpec.Domain.CPU) {
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

// kubeVirtInstancetypes returns the kvinstancetypev1alpha1.VirtualMachineInstanceType:
// - custom (cluster-wide)
// - concatenated with kubermatic standard from yaml manifests.
func kubeVirtInstancetypes(ctx context.Context, client ctrlruntimeclient.Client, kubeconfig string) (instancetypeListWrapper, error) {
	instancetypes := instancetypeListWrapper{}
	customInstancetypes := kvinstancetypev1alpha1.VirtualMachineClusterInstancetypeList{}
	standardInstancetypes := kvinstancetypev1alpha1.VirtualMachineInstancetypeList{}
	// "custom" (cluster-wide)
	if err := client.List(ctx, &customInstancetypes); err != nil {
		return instancetypes, err
	}
	// "standard" (namespaced)
	standardInstancetypes.Items = kubevirt.GetKubermaticStandardInstancetypes(client, &kvmanifests.StandardInstancetypeGetter{})

	// Wrap
	if len(customInstancetypes.Items) > 0 || len(standardInstancetypes.Items) > 0 {
		instancetypes.items = make([]instancetypeWrapper, 0)
	}
	for i := range customInstancetypes.Items {
		w := customInstancetypeWrapper{&customInstancetypes.Items[i]}
		instancetypes.items = append(instancetypes.items, &w)
	}
	for i := range standardInstancetypes.Items {
		w := standardInstancetypeWrapper{&standardInstancetypes.Items[i]}
		instancetypes.items = append(instancetypes.items, &w)
	}

	return instancetypes, nil
}

func newAPIInstancetype(w instancetypeWrapper) (*apiv2.VirtualMachineInstancetype, error) {
	spec, err := json.Marshal(w.Spec())
	if err != nil {
		return nil, err
	}

	return &apiv2.VirtualMachineInstancetype{
		Name: w.GetObjectMeta().GetName(),
		Spec: string(spec),
	}, nil
}

func newAPIPreference(w preferenceWrapper) (*apiv2.VirtualMachinePreference, error) {
	spec, err := json.Marshal(w.Spec())
	if err != nil {
		return nil, err
	}

	return &apiv2.VirtualMachinePreference{
		Name: w.GetObjectMeta().GetName(),
		Spec: string(spec),
	}, nil
}

// KubeVirtInstancetypes returns the apiv2.VirtualMachineInstanceType:
// - custom (cluster-wide)
// - concatenated with kubermatic standard from yaml manifests
// The list is filtered based on the Resource Quota.
func KubeVirtInstancetypes(ctx context.Context, kubeconfig string, cluster *kubermaticv1.Cluster, settingsProvider provider.SettingsProvider) (*apiv2.VirtualMachineInstancetypeList, error) {
	client, err := NewKubeVirtClient(kubeconfig, kubevirt.ClientOptions{})
	if err != nil {
		return nil, err
	}

	instancetypes, err := kubeVirtInstancetypes(ctx, client, kubeconfig)
	if err != nil {
		return nil, err
	}

	// conversion to api type
	res, err := instancetypes.toApi()
	if err != nil {
		return nil, err
	}

	// Reconcile Kubermatic Standard (update flow)
	for _, it := range instancetypes.items {
		if it.Category() == apiv2.InstancetypeKubermatic {
			if cluster != nil {
				instancetypeCreators := []reconciling.NamedKvInstancetypeV1alpha1VirtualMachineInstancetypeCreatorGetter{
					instancetypeCreator(it),
				}
				if err := reconciling.ReconcileKvInstancetypeV1alpha1VirtualMachineInstancetypes(ctx, instancetypeCreators, cluster.Status.NamespaceName, client); err != nil {
					return nil, err
				}
			}
		}
	}

	settings, err := settingsProvider.GetGlobalSettings(ctx)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return filterInstancetypes(res, settings.Spec.MachineDeploymentVMResourceQuota), nil
}

// kubeVirtPreferences returns the kvinstancetypev1alpha1.VirtualMachinePreference:
// - custom (cluster-wide)
// - concatenated with kubermatic standard from yaml manifests.
func kubeVirtPreferences(ctx context.Context, client ctrlruntimeclient.Client, kubeconfig string) (preferenceListWrapper, error) {
	preferences := preferenceListWrapper{}
	customPreferences := kvinstancetypev1alpha1.VirtualMachineClusterPreferenceList{}
	standardPreferences := kvinstancetypev1alpha1.VirtualMachinePreferenceList{}
	// "custom" (cluster-wide)
	if err := client.List(ctx, &customPreferences); err != nil {
		return preferences, err
	}
	// "standard" (namespaced)
	standardPreferences.Items = kubevirt.GetKubermaticStandardPreferences(client, &kvmanifests.StandardPreferenceGetter{})

	// Wrap
	if len(customPreferences.Items) > 0 || len(standardPreferences.Items) > 0 {
		preferences.items = make([]preferenceWrapper, 0)
	}
	for i := range customPreferences.Items {
		w := customPreferenceWrapper{&customPreferences.Items[i]}
		preferences.items = append(preferences.items, &w)
	}
	for i := range standardPreferences.Items {
		w := standardPreferenceWrapper{&standardPreferences.Items[i]}
		preferences.items = append(preferences.items, &w)
	}

	return preferences, nil
}

// KubeVirtPreferences returns the apiv2.VirtualMachinePreference:
// - custom (cluster-wide)
// - concatenated with kubermatic standard from yaml manifests.
// No filtering due to quota is needed.
func KubeVirtPreferences(ctx context.Context, kubeconfig string, cluster *kubermaticv1.Cluster, settingsProvider provider.SettingsProvider) (*apiv2.VirtualMachinePreferenceList, error) {
	client, err := NewKubeVirtClient(kubeconfig, kubevirt.ClientOptions{})
	if err != nil {
		return nil, err
	}

	preferences, err := kubeVirtPreferences(ctx, client, kubeconfig)
	if err != nil {
		return nil, err
	}

	// conversion to api type
	res, err := preferences.toApi()
	if err != nil {
		return nil, err
	}

	// Reconcile Kubermatic Standard (update flow)
	for _, it := range preferences.items {
		if it.Category() == apiv2.InstancetypeKubermatic {
			if cluster != nil {
				preferenceCreators := []reconciling.NamedKvInstancetypeV1alpha1VirtualMachinePreferenceCreatorGetter{
					preferenceCreator(it),
				}
				if err := reconciling.ReconcileKvInstancetypeV1alpha1VirtualMachinePreferences(ctx, preferenceCreators, cluster.Status.NamespaceName, client); err != nil {
					return nil, err
				}
			}
		}
	}

	return res, nil
}

func instancetypeCreator(w instancetypeWrapper) reconciling.NamedKvInstancetypeV1alpha1VirtualMachineInstancetypeCreatorGetter {
	return func() (string, reconciling.KvInstancetypeV1alpha1VirtualMachineInstancetypeCreator) {
		return w.GetObjectMeta().GetName(), func(it *kvinstancetypev1alpha1.VirtualMachineInstancetype) (*kvinstancetypev1alpha1.VirtualMachineInstancetype, error) {
			it.Labels = w.GetObjectMeta().GetLabels()
			it.Spec = w.Spec()
			return it, nil
		}
	}
}

func preferenceCreator(w preferenceWrapper) reconciling.NamedKvInstancetypeV1alpha1VirtualMachinePreferenceCreatorGetter {
	return func() (string, reconciling.KvInstancetypeV1alpha1VirtualMachinePreferenceCreator) {
		return w.GetObjectMeta().GetName(), func(it *kvinstancetypev1alpha1.VirtualMachinePreference) (*kvinstancetypev1alpha1.VirtualMachinePreference, error) {
			it.Labels = w.GetObjectMeta().GetLabels()
			it.Spec = w.Spec()
			return it, nil
		}
	}
}

func filterInstancetypes(instancetypes *apiv2.VirtualMachineInstancetypeList, quota kubermaticv1.MachineDeploymentVMResourceQuota) *apiv2.VirtualMachineInstancetypeList {
	filtered := &apiv2.VirtualMachineInstancetypeList{}

	// Range over the records and apply all the filters to each record.
	// If the record passes all the filters, add it to the final slice.
	for category, types := range instancetypes.Instancetypes {
		for _, instancetype := range types {
			spec := kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec{}
			if err := json.Unmarshal([]byte(instancetype.Spec), &spec); err != nil {
				log.Logger.Errorf("skipping VirtualMachineInstancetype:%s, parsing preset.Spec failed:%v", instancetype.Name, err)
				continue
			}

			if handlercommon.FilterCPU(int(spec.CPU.Guest), quota.MinCPU, quota.MaxCPU) && handlercommon.FilterMemory(int(spec.Memory.Guest.Value()/(1<<30)), quota.MinRAM, quota.MaxRAM) {
				if filtered.Instancetypes == nil {
					filtered.Instancetypes = make(map[apiv2.VirtualMachineInstancetypeCategory][]apiv2.VirtualMachineInstancetype, 0)
				}
				if filtered.Instancetypes[category] == nil {
					filtered.Instancetypes[category] = make([]apiv2.VirtualMachineInstancetype, 0)
				}
				filtered.Instancetypes[category] = append(filtered.Instancetypes[category], instancetype)
			}
		}
	}
	return filtered
}

// instancetypeWrapper to wrap functions needed to convert to API type:
//   - kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec, kvinstancetypev1alpha1.VirtualMachineClusterInstancetypeSpec
type instancetypeWrapper interface {
	Spec() kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec
	Category() apiv2.VirtualMachineInstancetypeCategory
	GetObjectMeta() metav1.Object
}

type customInstancetypeWrapper struct {
	*kvinstancetypev1alpha1.VirtualMachineClusterInstancetype
}

func (it *customInstancetypeWrapper) Category() apiv2.VirtualMachineInstancetypeCategory {
	return apiv2.InstancetypeCustom
}

func (it *customInstancetypeWrapper) Spec() kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec {
	return it.VirtualMachineClusterInstancetype.Spec
}

type standardInstancetypeWrapper struct {
	*kvinstancetypev1alpha1.VirtualMachineInstancetype
}

func (it *standardInstancetypeWrapper) Category() apiv2.VirtualMachineInstancetypeCategory {
	return apiv2.InstancetypeKubermatic
}

func (it *standardInstancetypeWrapper) Spec() kvinstancetypev1alpha1.VirtualMachineInstancetypeSpec {
	return it.VirtualMachineInstancetype.Spec
}

type instancetypeListWrapper struct {
	items []instancetypeWrapper
}

// toApi converts to apiv2 types.
func (l *instancetypeListWrapper) toApi() (*apiv2.VirtualMachineInstancetypeList, error) {
	res := &apiv2.VirtualMachineInstancetypeList{}
	if len(l.items) > 0 {
		res.Instancetypes = make(map[apiv2.VirtualMachineInstancetypeCategory][]apiv2.VirtualMachineInstancetype)
		for _, it := range l.items {
			instancetype, err := newAPIInstancetype(it)
			if err != nil {
				return nil, err
			}
			res.Instancetypes[it.Category()] = append(res.Instancetypes[it.Category()], *instancetype)
		}
	}
	return res, nil
}

// preferenceWrapper to wrap functions needed to convert to API type:
//   - kvinstancetypev1alpha1.VirtualMachinePreferenceSpec, kvinstancetypev1alpha1.VirtualMachineClusterPreferenceSpec
type preferenceWrapper interface {
	Spec() kvinstancetypev1alpha1.VirtualMachinePreferenceSpec
	Category() apiv2.VirtualMachineInstancetypeCategory
	GetObjectMeta() metav1.Object
}

type customPreferenceWrapper struct {
	*kvinstancetypev1alpha1.VirtualMachineClusterPreference
}

func (p *customPreferenceWrapper) Category() apiv2.VirtualMachineInstancetypeCategory {
	return apiv2.InstancetypeCustom
}

func (p *customPreferenceWrapper) Spec() kvinstancetypev1alpha1.VirtualMachinePreferenceSpec {
	return p.VirtualMachineClusterPreference.Spec
}

type standardPreferenceWrapper struct {
	*kvinstancetypev1alpha1.VirtualMachinePreference
}

func (p *standardPreferenceWrapper) Category() apiv2.VirtualMachineInstancetypeCategory {
	return apiv2.InstancetypeKubermatic
}

func (p *standardPreferenceWrapper) Spec() kvinstancetypev1alpha1.VirtualMachinePreferenceSpec {
	return p.VirtualMachinePreference.Spec
}

type preferenceListWrapper struct {
	items []preferenceWrapper
}

// toApi converts to apiv2 types.
func (l *preferenceListWrapper) toApi() (*apiv2.VirtualMachinePreferenceList, error) {
	res := &apiv2.VirtualMachinePreferenceList{}
	if len(l.items) > 0 {
		res.Preferences = make(map[apiv2.VirtualMachineInstancetypeCategory][]apiv2.VirtualMachinePreference)
		for _, it := range l.items {
			preference, err := newAPIPreference(it)
			if err != nil {
				return nil, err
			}
			res.Preferences[it.Category()] = append(res.Preferences[it.Category()], *preference)
		}
	}
	return res, nil
}
