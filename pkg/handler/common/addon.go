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

package common

import (
	"context"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/middleware"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"

	"k8s.io/apimachinery/pkg/runtime"
	k8sjson "k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	addonEnsureLabelKey = "addons.kubermatic.io/ensure"
	trueFlag            = "true"
)

func PatchAddonEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, addon apiv1.Addon, projectID, clusterID, addonID string) (interface{}, error) {
	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	apiAddon, err := getAddon(ctx, userInfoGetter, cluster, projectID, addonID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	rawVars, err := convertExternalVariablesToInternal(addon.Spec.Variables)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	apiAddon.Spec.Variables = *rawVars

	if apiAddon.Labels == nil {
		apiAddon.Labels = map[string]string{}
	}
	apiAddon.Labels[addonEnsureLabelKey] = "false"
	if addon.Spec.ContinuouslyReconcile {
		apiAddon.Labels[addonEnsureLabelKey] = trueFlag
	}

	apiAddon, err = updateAddon(ctx, userInfoGetter, cluster, apiAddon, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	result, err := convertInternalAddonToExternal(apiAddon)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	return result, nil
}

func CreateAddonEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, addon apiv1.Addon, projectID, clusterID string) (interface{}, error) {
	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	rawVars, err := convertExternalVariablesToInternal(addon.Spec.Variables)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	labels := map[string]string{}
	if addon.Spec.ContinuouslyReconcile {
		labels[addonEnsureLabelKey] = trueFlag
	}
	apiAddon, err := createAddon(ctx, userInfoGetter, cluster, rawVars, labels, projectID, addon.Name)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	result, err := convertInternalAddonToExternal(apiAddon)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	return result, nil
}

func ListAddonEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID, clusterID string) (interface{}, error) {
	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	addons, err := listAddons(ctx, userInfoGetter, cluster, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	result, err := convertInternalAddonsToExternal(addons)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	return result, nil
}

func GetAddonEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID, clusterID, addonID string) (interface{}, error) {
	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	addon, err := getAddon(ctx, userInfoGetter, cluster, projectID, addonID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	result, err := convertInternalAddonToExternal(addon)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	return result, nil
}

func ListInstallableAddonEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, accessibleAddons sets.String, projectID, clusterID string) (interface{}, error) {
	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}

	addons, err := listAddons(ctx, userInfoGetter, cluster, projectID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	installedAddons := sets.NewString()
	for _, addon := range addons {
		installedAddons.Insert(addon.Name)
	}

	return accessibleAddons.Difference(installedAddons).UnsortedList(), nil
}

func DeleteAddonEndpoint(ctx context.Context, userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, projectID, clusterID, addonID string) (interface{}, error) {
	cluster, err := GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, projectID, clusterID, nil)
	if err != nil {
		return nil, err
	}
	return nil, common.KubernetesErrorToHTTPError(deleteAddon(ctx, userInfoGetter, cluster, projectID, addonID))
}

func GetAddonConfigEndpoint(addonConfigProvider provider.AddonConfigProvider, addonID string) (interface{}, error) {
	addon, err := addonConfigProvider.Get(addonID)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return convertInternalAddonConfigToExternal(addon)
}

func ListAddonConfigsEndpoint(addonConfigProvider provider.AddonConfigProvider) (interface{}, error) {
	list, err := addonConfigProvider.List()
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return convertInternalAddonConfigsToExternal(list)
}

func deleteAddon(ctx context.Context, userInfoGetter provider.UserInfoGetter, cluster *kubermaticapiv1.Cluster, projectID, addonID string) error {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return err
	}
	if adminUserInfo.IsAdmin {
		privilegedAddonProvider := ctx.Value(middleware.PrivilegedAddonProviderContextKey).(provider.PrivilegedAddonProvider)
		return privilegedAddonProvider.DeleteUnsecured(cluster, addonID)
	}
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return err
	}
	addonProvider := ctx.Value(middleware.AddonProviderContextKey).(provider.AddonProvider)
	return addonProvider.Delete(userInfo, cluster, addonID)
}

func updateAddon(ctx context.Context, userInfoGetter provider.UserInfoGetter, cluster *kubermaticapiv1.Cluster, addon *kubermaticapiv1.Addon, projectID string) (*kubermaticapiv1.Addon, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		privilegedAddonProvider := ctx.Value(middleware.PrivilegedAddonProviderContextKey).(provider.PrivilegedAddonProvider)
		return privilegedAddonProvider.UpdateUnsecured(cluster, addon)
	}
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}
	addonProvider := ctx.Value(middleware.AddonProviderContextKey).(provider.AddonProvider)
	return addonProvider.Update(userInfo, cluster, addon)
}

func createAddon(ctx context.Context, userInfoGetter provider.UserInfoGetter, cluster *kubermaticapiv1.Cluster, rawVars *runtime.RawExtension, labels map[string]string, projectID, name string) (*kubermaticapiv1.Addon, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		privilegedAddonProvider := ctx.Value(middleware.PrivilegedAddonProviderContextKey).(provider.PrivilegedAddonProvider)
		return privilegedAddonProvider.NewUnsecured(cluster, name, rawVars, labels)
	}
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}
	addonProvider := ctx.Value(middleware.AddonProviderContextKey).(provider.AddonProvider)
	return addonProvider.New(userInfo, cluster, name, rawVars, labels)

}

func getAddon(ctx context.Context, userInfoGetter provider.UserInfoGetter, cluster *kubermaticapiv1.Cluster, projectID, addonID string) (*kubermaticapiv1.Addon, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		privilegedAddonProvider := ctx.Value(middleware.PrivilegedAddonProviderContextKey).(provider.PrivilegedAddonProvider)
		return privilegedAddonProvider.GetUnsecured(cluster, addonID)
	}
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}
	addonProvider := ctx.Value(middleware.AddonProviderContextKey).(provider.AddonProvider)
	return addonProvider.Get(userInfo, cluster, addonID)
}

func listAddons(ctx context.Context, userInfoGetter provider.UserInfoGetter, cluster *kubermaticapiv1.Cluster, projectID string) ([]*kubermaticapiv1.Addon, error) {
	adminUserInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, err
	}
	if adminUserInfo.IsAdmin {
		privilegedAddonProvider := ctx.Value(middleware.PrivilegedAddonProviderContextKey).(provider.PrivilegedAddonProvider)
		return privilegedAddonProvider.ListUnsecured(cluster)
	}
	userInfo, err := userInfoGetter(ctx, projectID)
	if err != nil {
		return nil, err
	}
	addonProvider := ctx.Value(middleware.AddonProviderContextKey).(provider.AddonProvider)
	return addonProvider.List(userInfo, cluster)
}

func convertInternalAddonToExternal(internalAddon *kubermaticapiv1.Addon) (*apiv1.Addon, error) {
	result := &apiv1.Addon{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                internalAddon.Name,
			Name:              internalAddon.Name,
			CreationTimestamp: apiv1.NewTime(internalAddon.CreationTimestamp.Time),
			DeletionTimestamp: func() *apiv1.Time {
				if internalAddon.DeletionTimestamp != nil {
					deletionTimestamp := apiv1.NewTime(internalAddon.DeletionTimestamp.Time)
					return &deletionTimestamp
				}
				return nil
			}(),
		},
		Spec: apiv1.AddonSpec{
			IsDefault: internalAddon.Spec.IsDefault,
		},
	}
	if len(internalAddon.Spec.Variables.Raw) > 0 {
		if err := k8sjson.Unmarshal(internalAddon.Spec.Variables.Raw, &result.Spec.Variables); err != nil {
			return nil, err
		}
	}
	if internalAddon.Labels != nil && internalAddon.Labels[addonEnsureLabelKey] == trueFlag {
		result.Spec.ContinuouslyReconcile = true
	}

	return result, nil
}

func convertInternalAddonsToExternal(internalAddons []*kubermaticapiv1.Addon) ([]*apiv1.Addon, error) {
	result := []*apiv1.Addon{}

	for _, addon := range internalAddons {
		converted, err := convertInternalAddonToExternal(addon)
		if err != nil {
			return nil, err
		}
		result = append(result, converted)
	}

	return result, nil
}

func convertInternalAddonConfigToExternal(internalAddonConfig *kubermaticapiv1.AddonConfig) (*apiv1.AddonConfig, error) {
	return &apiv1.AddonConfig{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                internalAddonConfig.Name,
			Name:              internalAddonConfig.Name,
			CreationTimestamp: apiv1.NewTime(internalAddonConfig.CreationTimestamp.Time),
			DeletionTimestamp: func() *apiv1.Time {
				if internalAddonConfig.DeletionTimestamp != nil {
					deletionTimestamp := apiv1.NewTime(internalAddonConfig.DeletionTimestamp.Time)
					return &deletionTimestamp
				}
				return nil
			}(),
		},
		Spec: internalAddonConfig.Spec,
	}, nil
}

func convertInternalAddonConfigsToExternal(internalAddonConfigs *kubermaticapiv1.AddonConfigList) ([]*apiv1.AddonConfig, error) {
	result := []*apiv1.AddonConfig{}

	for _, internalAddonConfig := range internalAddonConfigs.Items {
		converted, err := convertInternalAddonConfigToExternal(&internalAddonConfig)
		if err != nil {
			return nil, err
		}
		result = append(result, converted)
	}

	return result, nil
}

func convertExternalVariablesToInternal(external map[string]interface{}) (*runtime.RawExtension, error) {
	result := &runtime.RawExtension{}
	raw, err := k8sjson.Marshal(external)
	if err != nil {
		return nil, err
	}
	result.Raw = raw
	return result, nil
}
