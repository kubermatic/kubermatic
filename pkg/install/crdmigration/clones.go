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

package crdmigration

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	newv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"
	"k8c.io/kubermatic/v2/pkg/semver"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func DuplicateResources(ctx context.Context, logger logrus.FieldLogger, opt *Options) error {
	// clone master cluster resources
	if err := cloneResourcesInCluster(ctx, logger.WithField("master", true), opt.MasterClient); err != nil {
		return fmt.Errorf("processing the master cluster failed: %w", err)
	}

	// clone seed cluster resources
	for seedName, seedClient := range opt.SeedClients {
		if err := cloneResourcesInCluster(ctx, logger.WithField("seed", seedName), seedClient); err != nil {
			return fmt.Errorf("processing the seed cluster failed: %w", err)
		}
	}

	return nil
}

func cloneResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) error {
	logger.Info("Duplicating resources into new API group…")

	// reset our runtime UID cache
	uidCache = map[string]types.UID{}

	type cloneFn func(context.Context, logrus.FieldLogger, ctrlruntimeclient.Client) (int, error)

	// the order in which resources are migrated is important, as they are interlinked via owner references
	tasks := []struct {
		Kind   string
		cloner cloneFn
	}{
		{Kind: "KubermaticConfiguration", cloner: cloneKubermaticConfigurationResourcesInCluster},
		{Kind: "User", cloner: cloneUserResourcesInCluster},
		{Kind: "Project", cloner: cloneProjectResourcesInCluster},
		{Kind: "Cluster", cloner: cloneClusterResourcesInCluster},
		{Kind: "Addon", cloner: cloneAddonResourcesInCluster},
		{Kind: "AddonConfig", cloner: cloneAddonConfigResourcesInCluster},
		{Kind: "AdmissionPlugin", cloner: cloneAdmissionPluginResourcesInCluster},
		{Kind: "Alertmanager", cloner: cloneAlertmanagerResourcesInCluster},
		{Kind: "AllowedRegistrie", cloner: cloneAllowedRegistryResourcesInCluster},
		{Kind: "ClusterTemplate", cloner: cloneClusterTemplateResourcesInCluster},
		{Kind: "ClusterTemplateInstance", cloner: cloneClusterTemplateInstanceResourcesInCluster},
		{Kind: "ConstraintTemplate", cloner: cloneConstraintTemplateResourcesInCluster},
		{Kind: "Constraint", cloner: cloneConstraintResourcesInCluster},
		{Kind: "EtcdBackupConfig", cloner: cloneEtcdBackupConfigResourcesInCluster},
		{Kind: "EtcdRestore", cloner: cloneEtcdRestoreResourcesInCluster},
		{Kind: "ExternalCluster", cloner: cloneExternalClusterResourcesInCluster},
		{Kind: "KubermaticSetting", cloner: cloneKubermaticSettingResourcesInCluster},
		{Kind: "MLAAdminSetting", cloner: cloneMLAAdminSettingResourcesInCluster},
		{Kind: "Preset", cloner: clonePresetResourcesInCluster},
		{Kind: "RuleGroup", cloner: cloneRuleGroupResourcesInCluster},
		{Kind: "Seed", cloner: cloneSeedResourcesInCluster},
		{Kind: "UserProjectBinding", cloner: cloneUserProjectBindingResourcesInCluster},
		{Kind: "UserSSHKey", cloner: cloneUserSSHKeyResourcesInCluster},
	}

	for _, task := range tasks {
		logger.Debugf("Duplicating %s objects…", task.Kind)

		cloned, err := task.cloner(ctx, logger, client)
		if err != nil {
			return fmt.Errorf("failed to clone %s: %w", task.Kind, err)
		}

		logger.Infof("Duplicated %d %s objects.", cloned, task.Kind)
	}

	return nil
}

// uidCache is a primitive runtime cache for the UIDs of
// objects created via ensureObject. It is used to fill in
// the UID for owner and object references.
var uidCache = map[string]types.UID{}

func getUIDCacheKey(kind, namespace, name string) string {
	if isNamespacedKind(kind) {
		return fmt.Sprintf("%s/%s/%s", kind, namespace, name)
	}

	return fmt.Sprintf("%s/%s", kind, name)
}

func getUIDCacheKeyForObject(obj ctrlruntimeclient.Object, client ctrlruntimeclient.Client) string {
	gvk, err := apiutil.GVKForObject(obj, client.Scheme())
	if err != nil {
		panic(err)
	}

	return getUIDCacheKey(gvk.Kind, obj.GetNamespace(), obj.GetName())
}

func ensureObject(ctx context.Context, client ctrlruntimeclient.Client, obj ctrlruntimeclient.Object, cacheUID bool) error {
	if err := client.Create(ctx, obj); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return err
		}
	}

	// re-fetch the object to
	//    1. fill in its UID so we can cache it
	//    2. get the ResourceVersion, so we can later update any subresources like "status"
	// Get()ing an object fills in the APIVersion and Kind too, but only if the client
	// is backed by a cache (e.g. ctrlruntime's delegatingClient). Since the clients our
	// seedClientGetter provides do not have caches attached, this won't work.
	if err := client.Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(obj), obj); err != nil {
		return err
	}

	if cacheUID {
		uidCache[getUIDCacheKeyForObject(obj, client)] = obj.GetUID()
	}

	return nil
}

func convertObjectMeta(om metav1.ObjectMeta) metav1.ObjectMeta {
	om = *om.DeepCopy()
	om.OwnerReferences = migrateOwnerReferences(om.OwnerReferences, om.Namespace)
	om.UID = ""
	om.Generation = 0
	om.ResourceVersion = ""
	om.CreationTimestamp = metav1.Time{}

	return om
}

func migrateOwnerReferences(ownerRefs []metav1.OwnerReference, namespace string) []metav1.OwnerReference {
	result := []metav1.OwnerReference{}

	for _, ref := range ownerRefs {
		newRef := ref.DeepCopy()

		if newRef.APIVersion == "kubermatic.k8s.io/v1" {
			newRef.APIVersion = "kubermatic.k8c.io/v1"

			cacheKey := getUIDCacheKey(newRef.Kind, namespace, newRef.Name)
			uid, exists := uidCache[cacheKey]
			if !exists {
				panic(fmt.Sprintf("Cannot find UID for %s in cache. Make sure to create %s first.", cacheKey, newRef.Kind))
			}

			newRef.UID = uid
		}

		result = append(result, *newRef)
	}

	return result
}

func migrateObjectReference(objectRef corev1.ObjectReference, namespace string) corev1.ObjectReference {
	newRef := *objectRef.DeepCopy()

	if newRef.APIVersion == "kubermatic.k8s.io/v1" {
		newRef.APIVersion = "kubermatic.k8c.io/v1"

		if newRef.Namespace != "" {
			namespace = newRef.Namespace
		}

		cacheKey := getUIDCacheKey(newRef.Kind, namespace, newRef.Name)
		uid, exists := uidCache[cacheKey]
		if !exists {
			panic(fmt.Sprintf("Cannot find UID for %s in cache. Make sure to create %s first.", cacheKey, newRef.Kind))
		}

		newRef.UID = uid
	}

	return newRef
}

func cloneKubermaticConfigurationResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &operatorv1alpha1.KubermaticConfigurationList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.KubermaticConfiguration{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.KubermaticConfigurationSpec{
				CABundle:        oldObject.Spec.CABundle,
				ImagePullSecret: oldObject.Spec.ImagePullSecret,
				Auth:            newv1.KubermaticAuthConfiguration(oldObject.Spec.Auth),
				FeatureGates:    map[string]bool{},
				UI:              newv1.KubermaticUIConfiguration(oldObject.Spec.UI),
				API:             newv1.KubermaticAPIConfiguration(oldObject.Spec.API),
				SeedController: newv1.KubermaticSeedControllerConfiguration{
					DockerRepository:          oldObject.Spec.SeedController.DockerRepository,
					BackupStoreContainer:      oldObject.Spec.SeedController.BackupStoreContainer,
					BackupDeleteContainer:     oldObject.Spec.SeedController.BackupDeleteContainer,
					BackupCleanupContainer:    oldObject.Spec.SeedController.BackupCleanupContainer,
					BackupRestore:             newv1.LegacyKubermaticBackupRestoreConfiguration(oldObject.Spec.SeedController.BackupRestore), //nolint:staticcheck
					MaximumParallelReconciles: oldObject.Spec.SeedController.MaximumParallelReconciles,
					PProfEndpoint:             oldObject.Spec.SeedController.PProfEndpoint,
					Resources:                 oldObject.Spec.SeedController.Resources,
					DebugLog:                  oldObject.Spec.SeedController.DebugLog,
					Replicas:                  oldObject.Spec.SeedController.Replicas,
				},
				MasterController: newv1.KubermaticMasterControllerConfiguration{
					DockerRepository: oldObject.Spec.MasterController.DockerRepository,
					ProjectsMigrator: newv1.KubermaticProjectsMigratorConfiguration(oldObject.Spec.MasterController.ProjectsMigrator),
					PProfEndpoint:    oldObject.Spec.MasterController.PProfEndpoint,
					Resources:        oldObject.Spec.MasterController.Resources,
					DebugLog:         oldObject.Spec.MasterController.DebugLog,
					Replicas:         oldObject.Spec.MasterController.Replicas,
				},
				UserCluster: newv1.KubermaticUserClusterConfiguration{
					KubermaticDockerRepository:          oldObject.Spec.UserCluster.KubermaticDockerRepository,
					DNATControllerDockerRepository:      oldObject.Spec.UserCluster.DNATControllerDockerRepository,
					EtcdLauncherDockerRepository:        oldObject.Spec.UserCluster.EtcdLauncherDockerRepository,
					OverwriteRegistry:                   oldObject.Spec.UserCluster.OverwriteRegistry,
					Addons:                              newv1.KubermaticAddonsConfiguration(oldObject.Spec.UserCluster.Addons.Kubernetes),
					NodePortRange:                       oldObject.Spec.UserCluster.NodePortRange,
					Monitoring:                          newv1.KubermaticUserClusterMonitoringConfiguration(oldObject.Spec.UserCluster.Monitoring),
					DisableAPIServerEndpointReconciling: oldObject.Spec.UserCluster.DisableAPIServerEndpointReconciling,
					EtcdVolumeSize:                      oldObject.Spec.UserCluster.EtcdVolumeSize,
					APIServerReplicas:                   oldObject.Spec.UserCluster.APIServerReplicas,
					MachineController:                   newv1.MachineControllerConfiguration(oldObject.Spec.UserCluster.MachineController),
				},
				ExposeStrategy: oldObject.Spec.ExposeStrategy,
				Ingress:        newv1.KubermaticIngressConfiguration(oldObject.Spec.Ingress),
				Versions:       convertKubermaticVersioningConfiguration(oldObject.Spec.Versions.Kubernetes),
				VerticalPodAutoscaler: newv1.KubermaticVPAConfiguration{
					Recommender:         newv1.KubermaticVPAComponent(oldObject.Spec.VerticalPodAutoscaler.Recommender),
					Updater:             newv1.KubermaticVPAComponent(oldObject.Spec.VerticalPodAutoscaler.Updater),
					AdmissionController: newv1.KubermaticVPAComponent(oldObject.Spec.VerticalPodAutoscaler.AdmissionController),
				},
				Proxy: newv1.KubermaticProxyConfiguration(oldObject.Spec.Proxy),
			},
		}

		for _, feature := range oldObject.Spec.FeatureGates.List() {
			newObject.Spec.FeatureGates[feature] = true
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func convertKubermaticVersioningConfiguration(old operatorv1alpha1.KubermaticVersioningConfiguration) newv1.KubermaticVersioningConfiguration {
	result := newv1.KubermaticVersioningConfiguration{}

	if old.Default != nil {
		result.Default = semver.Semver(old.Default.String())
	}

	for _, v := range old.Versions {
		result.Versions = append(result.Versions, semver.Semver(v.String()))
	}

	for _, u := range old.Updates {
		result.Updates = append(result.Updates, newv1.Update(u))
	}

	for _, i := range old.ProviderIncompatibilities {
		result.ProviderIncompatibilities = append(result.ProviderIncompatibilities, newv1.Incompatibility{
			Provider:  i.Provider,
			Version:   i.Version,
			Condition: newv1.ConditionType(i.Condition),
			Operation: newv1.OperationType(i.Operation),
		})
	}

	return result
}

func cloneClusterResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.ClusterList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.Cluster{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Address:    newv1.ClusterAddress(oldObject.Address),
			Spec:       convertClusterSpec(oldObject.Spec),
			Status: newv1.ClusterStatus{
				KubermaticVersion:      oldObject.Status.KubermaticVersion,
				NamespaceName:          oldObject.Status.NamespaceName,
				CloudMigrationRevision: oldObject.Status.CloudMigrationRevision,
				LastUpdated:            oldObject.Status.LastUpdated,
				UserName:               oldObject.Status.UserName,
				UserEmail:              oldObject.Status.UserEmail,
				ExtendedHealth: newv1.ExtendedClusterHealth{
					Apiserver:                    newv1.HealthStatus(oldObject.Status.ExtendedHealth.Apiserver),
					Scheduler:                    newv1.HealthStatus(oldObject.Status.ExtendedHealth.Scheduler),
					Controller:                   newv1.HealthStatus(oldObject.Status.ExtendedHealth.Controller),
					MachineController:            newv1.HealthStatus(oldObject.Status.ExtendedHealth.MachineController),
					Etcd:                         newv1.HealthStatus(oldObject.Status.ExtendedHealth.Etcd),
					OpenVPN:                      newv1.HealthStatus(oldObject.Status.ExtendedHealth.OpenVPN),
					CloudProviderInfrastructure:  newv1.HealthStatus(oldObject.Status.ExtendedHealth.CloudProviderInfrastructure),
					UserClusterControllerManager: newv1.HealthStatus(oldObject.Status.ExtendedHealth.UserClusterControllerManager),
					GatekeeperController:         newv1.HealthStatus(oldObject.Status.ExtendedHealth.GatekeeperController),
					GatekeeperAudit:              newv1.HealthStatus(oldObject.Status.ExtendedHealth.GatekeeperAudit),
				},
			},
		}

		if err := ensureObject(ctx, client, &newObject, true); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}

		if err := client.Status().Update(ctx, &newObject); err != nil {
			return 0, fmt.Errorf("failed to update status on %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func convertClusterSpec(old kubermaticv1.ClusterSpec) newv1.ClusterSpec {
	result := newv1.ClusterSpec{
		Cloud: newv1.CloudSpec{
			DatacenterName: old.Cloud.DatacenterName,

			// Azure and VSphere need special treatment further down
			AWS:          (*newv1.AWSCloudSpec)(old.Cloud.AWS),
			Alibaba:      (*newv1.AlibabaCloudSpec)(old.Cloud.Alibaba),
			Anexia:       (*newv1.AnexiaCloudSpec)(old.Cloud.Anexia),
			BringYourOwn: (*newv1.BringYourOwnCloudSpec)(old.Cloud.BringYourOwn),
			Digitalocean: (*newv1.DigitaloceanCloudSpec)(old.Cloud.Digitalocean),
			Fake:         (*newv1.FakeCloudSpec)(old.Cloud.Fake),
			GCP:          (*newv1.GCPCloudSpec)(old.Cloud.GCP),
			Hetzner:      (*newv1.HetznerCloudSpec)(old.Cloud.Hetzner),
			Kubevirt:     (*newv1.KubevirtCloudSpec)(old.Cloud.Kubevirt),
			Openstack:    (*newv1.OpenstackCloudSpec)(old.Cloud.Openstack),
			Packet:       (*newv1.PacketCloudSpec)(old.Cloud.Packet),
		},
		ClusterNetwork: newv1.ClusterNetworkingConfig{
			Pods:                     newv1.NetworkRanges(old.ClusterNetwork.Pods),
			Services:                 newv1.NetworkRanges(old.ClusterNetwork.Services),
			DNSDomain:                old.ClusterNetwork.DNSDomain,
			ProxyMode:                old.ClusterNetwork.ProxyMode,
			IPVS:                     (*newv1.IPVSConfiguration)(old.ClusterNetwork.IPVS),
			NodeLocalDNSCacheEnabled: old.ClusterNetwork.NodeLocalDNSCacheEnabled,
			KonnectivityEnabled:      old.ClusterNetwork.KonnectivityEnabled,
		},
		Version:                              old.Version,
		MasterVersion:                        old.MasterVersion,
		HumanReadableName:                    old.HumanReadableName,
		ExposeStrategy:                       newv1.ExposeStrategy(old.ExposeStrategy),
		Pause:                                old.Pause,
		PauseReason:                          old.PauseReason,
		ComponentsOverride:                   convertComponentSettings(old.ComponentsOverride),
		OIDC:                                 newv1.OIDCSettings(old.OIDC),
		Features:                             old.Features,
		UpdateWindow:                         (*newv1.UpdateWindow)(old.UpdateWindow),
		UsePodSecurityPolicyAdmissionPlugin:  old.UsePodSecurityPolicyAdmissionPlugin,
		UsePodNodeSelectorAdmissionPlugin:    old.UsePodNodeSelectorAdmissionPlugin,
		EnableUserSSHKeyAgent:                old.EnableUserSSHKeyAgent,
		PodNodeSelectorAdmissionPluginConfig: old.PodNodeSelectorAdmissionPluginConfig,
		AdmissionPlugins:                     old.AdmissionPlugins,
		AuditLogging:                         (*newv1.AuditLoggingSettings)(old.AuditLogging),
		OPAIntegration:                       (*newv1.OPAIntegrationSettings)(old.OPAIntegration),
		ServiceAccount:                       (*newv1.ServiceAccountSettings)(old.ServiceAccount),
		MLA:                                  (*newv1.MLASettings)(old.MLA),
		ContainerRuntime:                     old.ContainerRuntime,
	}

	if old := old.Cloud.Azure; old != nil {
		result.Cloud.Azure = &newv1.AzureCloudSpec{
			CredentialsReference:  old.CredentialsReference,
			TenantID:              old.TenantID,
			SubscriptionID:        old.SubscriptionID,
			ClientID:              old.ClientID,
			ClientSecret:          old.ClientSecret,
			ResourceGroup:         old.ResourceGroup,
			VNetResourceGroup:     old.VNetResourceGroup,
			VNetName:              old.VNetName,
			SubnetName:            old.SubnetName,
			RouteTableName:        old.RouteTableName,
			SecurityGroup:         old.SecurityGroup,
			AssignAvailabilitySet: old.AssignAvailabilitySet,
			AvailabilitySet:       old.AvailabilitySet,
			LoadBalancerSKU:       newv1.LBSKU(old.LoadBalancerSKU),
		}
	}

	if old := old.Cloud.VSphere; old != nil {
		result.Cloud.VSphere = &newv1.VSphereCloudSpec{
			CredentialsReference: old.CredentialsReference,
			Username:             old.Username,
			Password:             old.Password,
			VMNetName:            old.VMNetName,
			Folder:               old.Folder,
			Datastore:            old.Datastore,
			DatastoreCluster:     old.DatastoreCluster,
			StoragePolicy:        old.StoragePolicy,
			ResourcePool:         old.ResourcePool,
			InfraManagementUser:  newv1.VSphereCredentials(old.InfraManagementUser),
		}
	}

	if old := old.CNIPlugin; old != nil {
		result.CNIPlugin = &newv1.CNIPluginSettings{
			Type:    newv1.CNIPluginType(old.Type),
			Version: old.Version,
		}
	}

	for _, network := range old.MachineNetworks {
		result.MachineNetworks = append(result.MachineNetworks, newv1.MachineNetworkingConfig(network))
	}

	return result
}

func cloneAddonResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.AddonList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list Addon objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.Addon{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.AddonSpec{
				Name:      oldObject.Spec.Name,
				Cluster:   migrateObjectReference(oldObject.Spec.Cluster, ""),
				IsDefault: oldObject.Spec.IsDefault,
				Variables: oldObject.Spec.Variables,
			},
		}

		for _, t := range oldObject.Spec.RequiredResourceTypes {
			newObject.Spec.RequiredResourceTypes = append(newObject.Spec.RequiredResourceTypes, newv1.GroupVersionKind(t))
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneAddonConfigResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.AddonConfigList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.AddonConfig{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.AddonConfigSpec{
				ShortDescription: oldObject.Spec.ShortDescription,
				Description:      oldObject.Spec.Description,
				Logo:             oldObject.Spec.Logo,
				LogoFormat:       oldObject.Spec.LogoFormat,
				Controls:         []newv1.AddonFormControl{},
			},
		}

		for _, ctrl := range oldObject.Spec.Controls {
			newObject.Spec.Controls = append(newObject.Spec.Controls, newv1.AddonFormControl(ctrl))
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneAdmissionPluginResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.AdmissionPluginList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.AdmissionPlugin{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.AdmissionPluginSpec{
				PluginName:  oldObject.Spec.PluginName,
				FromVersion: oldObject.Spec.DeepCopy().FromVersion,
			},
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneAlertmanagerResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.AlertmanagerList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.Alertmanager{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.AlertmanagerSpec{
				ConfigSecret: oldObject.Spec.DeepCopy().ConfigSecret,
			},
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneAllowedRegistryResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.AllowedRegistryList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.AllowedRegistry{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.AllowedRegistrySpec{
				RegistryPrefix: oldObject.Spec.RegistryPrefix,
			},
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneClusterTemplateResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.ClusterTemplateList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.ClusterTemplate{
			ObjectMeta:             convertObjectMeta(oldObject.ObjectMeta),
			Credential:             oldObject.Credential,
			ClusterLabels:          oldObject.ClusterLabels,
			InheritedClusterLabels: oldObject.InheritedClusterLabels,
			Spec:                   convertClusterSpec(oldObject.Spec),
			UserSSHKeys:            []newv1.ClusterTemplateSSHKey{},
		}

		for _, key := range oldObject.UserSSHKeys {
			newObject.UserSSHKeys = append(newObject.UserSSHKeys, newv1.ClusterTemplateSSHKey(key))
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneClusterTemplateInstanceResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.ClusterTemplateInstanceList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.ClusterTemplateInstance{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.ClusterTemplateInstanceSpec{
				ProjectID:           oldObject.Spec.ProjectID,
				ClusterTemplateID:   oldObject.Spec.ClusterTemplateID,
				ClusterTemplateName: oldObject.Spec.ClusterTemplateName,
				Replicas:            oldObject.Spec.Replicas,
			},
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneConstraintTemplateResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.ConstraintTemplateList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.ConstraintTemplate{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.ConstraintTemplateSpec{
				CRD:      oldObject.Spec.CRD,
				Selector: newv1.ConstraintTemplateSelector(oldObject.Spec.Selector),
				Targets:  oldObject.Spec.Targets,
			},
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneConstraintResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.ConstraintList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.Constraint{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.ConstraintSpec{
				ConstraintType: oldObject.Spec.ConstraintType,
				Disabled:       oldObject.Spec.Disabled,
				Match: newv1.Match{
					Scope:              oldObject.Spec.Match.Scope,
					Namespaces:         oldObject.Spec.Match.Namespaces,
					ExcludedNamespaces: oldObject.Spec.Match.ExcludedNamespaces,
					LabelSelector:      oldObject.Spec.Match.LabelSelector,
					NamespaceSelector:  oldObject.Spec.Match.NamespaceSelector,
				},
				Parameters: newv1.Parameters(oldObject.Spec.Parameters),
				Selector:   newv1.ConstraintSelector(oldObject.Spec.Selector),
			},
		}

		for _, kind := range oldObject.Spec.Match.Kinds {
			newObject.Spec.Match.Kinds = append(newObject.Spec.Match.Kinds, newv1.Kind(kind))
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneEtcdBackupConfigResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.EtcdBackupConfigList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.EtcdBackupConfig{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.EtcdBackupConfigSpec{
				Name:     oldObject.Spec.Name,
				Schedule: oldObject.Spec.Schedule,
				Keep:     oldObject.Spec.Keep,
				Cluster:  migrateObjectReference(oldObject.Spec.Cluster, ""),
			},
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneEtcdRestoreResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.EtcdRestoreList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.EtcdRestore{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.EtcdRestoreSpec{
				Name:                            oldObject.Spec.Name,
				BackupDownloadCredentialsSecret: oldObject.Spec.BackupDownloadCredentialsSecret,
				BackupName:                      oldObject.Spec.BackupName,
				Cluster:                         migrateObjectReference(oldObject.Spec.Cluster, ""),
			},
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneExternalClusterResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.ExternalClusterList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.ExternalCluster{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.ExternalClusterSpec{
				HumanReadableName:   oldObject.Spec.HumanReadableName,
				KubeconfigReference: oldObject.Spec.KubeconfigReference,
			},
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneKubermaticSettingResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.KubermaticSettingList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.KubermaticSetting{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.SettingSpec{
				CustomLinks:                      newv1.CustomLinks{},
				CleanupOptions:                   newv1.CleanupOptions(oldObject.Spec.CleanupOptions),
				DefaultNodeCount:                 oldObject.Spec.DefaultNodeCount,
				DisplayDemoInfo:                  oldObject.Spec.DisplayDemoInfo,
				DisplayAPIDocs:                   oldObject.Spec.DisplayAPIDocs,
				DisplayTermsOfService:            oldObject.Spec.DisplayTermsOfService,
				EnableDashboard:                  oldObject.Spec.EnableDashboard,
				EnableOIDCKubeconfig:             oldObject.Spec.EnableOIDCKubeconfig,
				UserProjectsLimit:                oldObject.Spec.UserProjectsLimit,
				RestrictProjectCreation:          oldObject.Spec.RestrictProjectCreation,
				EnableExternalClusterImport:      oldObject.Spec.EnableExternalClusterImport,
				OpaOptions:                       newv1.OpaOptions(oldObject.Spec.OpaOptions),
				MlaOptions:                       newv1.MlaOptions(oldObject.Spec.MlaOptions),
				MlaAlertmanagerPrefix:            oldObject.Spec.MlaAlertmanagerPrefix,
				MlaGrafanaPrefix:                 oldObject.Spec.MlaGrafanaPrefix,
				MachineDeploymentVMResourceQuota: newv1.MachineDeploymentVMResourceQuota(oldObject.Spec.MachineDeploymentVMResourceQuota),
			},
		}

		for _, link := range oldObject.Spec.CustomLinks {
			newObject.Spec.CustomLinks = append(newObject.Spec.CustomLinks, newv1.CustomLink(link))
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneMLAAdminSettingResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.MLAAdminSettingList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.MLAAdminSetting{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.MLAAdminSettingSpec{
				ClusterName: oldObject.Spec.ClusterName,
				MonitoringRateLimits: &newv1.MonitoringRateLimitSettings{
					IngestionRate:      oldObject.Spec.MonitoringRateLimits.IngestionRate,
					IngestionBurstSize: oldObject.Spec.MonitoringRateLimits.IngestionBurstSize,
					MaxSeriesPerMetric: oldObject.Spec.MonitoringRateLimits.MaxSeriesPerMetric,
					MaxSeriesTotal:     oldObject.Spec.MonitoringRateLimits.MaxSeriesTotal,
					QueryRate:          oldObject.Spec.MonitoringRateLimits.QueryRate,
					QueryBurstSize:     oldObject.Spec.MonitoringRateLimits.QueryBurstSize,
					MaxSamplesPerQuery: oldObject.Spec.MonitoringRateLimits.MaxSamplesPerQuery,
					MaxSeriesPerQuery:  oldObject.Spec.MonitoringRateLimits.MaxSeriesPerQuery,
				},
				LoggingRateLimits: &newv1.LoggingRateLimitSettings{
					IngestionRate:      oldObject.Spec.LoggingRateLimits.IngestionRate,
					IngestionBurstSize: oldObject.Spec.LoggingRateLimits.IngestionBurstSize,
					QueryRate:          oldObject.Spec.LoggingRateLimits.QueryRate,
					QueryBurstSize:     oldObject.Spec.LoggingRateLimits.QueryBurstSize,
				},
			},
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func clonePresetResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.PresetList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.Preset{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.PresetSpec{
				Enabled:        oldObject.Spec.Enabled,
				RequiredEmails: oldObject.Spec.RequiredEmails,
			},
		}

		// Very old KKP versions supported the "RequiredEmailDomain" (singular) field and
		// would transparently translate this into the RequiredEmails (plural) field. This
		// CRD migration we're doing right now is _the_ perfect time to end the deprecation,
		// migrate once and for all and remove the RequiredEmailDomain field.
		if oldObject.Spec.RequiredEmailDomain != "" {
			if oldObject.Spec.RequiredEmails == nil {
				oldObject.Spec.RequiredEmails = []string{}
			}

			oldObject.Spec.RequiredEmails = append(oldObject.Spec.RequiredEmails, oldObject.Spec.RequiredEmailDomain)
		}

		oldSpec := oldObject.Spec

		if oldSpec.AWS != nil {
			newObject.Spec.AWS = &newv1.AWS{
				PresetProvider:      newv1.PresetProvider(oldSpec.AWS.PresetProvider),
				AccessKeyID:         oldSpec.AWS.AccessKeyID,
				SecretAccessKey:     oldSpec.AWS.SecretAccessKey,
				VPCID:               oldSpec.AWS.VPCID,
				RouteTableID:        oldSpec.AWS.RouteTableID,
				InstanceProfileName: oldSpec.AWS.InstanceProfileName,
				SecurityGroupID:     oldSpec.AWS.SecurityGroupID,
				ControlPlaneRoleARN: oldSpec.AWS.ControlPlaneRoleARN,
			}
		}

		if oldSpec.Alibaba != nil {
			newObject.Spec.Alibaba = &newv1.Alibaba{
				PresetProvider:  newv1.PresetProvider(oldSpec.Alibaba.PresetProvider),
				AccessKeyID:     oldSpec.Alibaba.AccessKeyID,
				AccessKeySecret: oldSpec.Alibaba.AccessKeySecret,
			}
		}

		if oldSpec.Anexia != nil {
			newObject.Spec.Anexia = &newv1.Anexia{
				PresetProvider: newv1.PresetProvider(oldSpec.Anexia.PresetProvider),
				Token:          oldSpec.Anexia.Token,
			}
		}

		if oldSpec.Azure != nil {
			newObject.Spec.Azure = &newv1.Azure{
				PresetProvider:    newv1.PresetProvider(oldSpec.Azure.PresetProvider),
				TenantID:          oldSpec.Azure.TenantID,
				SubscriptionID:    oldSpec.Azure.SubscriptionID,
				ClientID:          oldSpec.Azure.ClientID,
				ClientSecret:      oldSpec.Azure.ClientSecret,
				ResourceGroup:     oldSpec.Azure.ResourceGroup,
				VNetResourceGroup: oldSpec.Azure.VNetResourceGroup,
				VNetName:          oldSpec.Azure.VNetName,
				SubnetName:        oldSpec.Azure.SubnetName,
				RouteTableName:    oldSpec.Azure.RouteTableName,
				SecurityGroup:     oldSpec.Azure.SecurityGroup,
				LoadBalancerSKU:   newv1.LBSKU(oldSpec.Azure.LoadBalancerSKU),
			}
		}

		if oldSpec.Digitalocean != nil {
			newObject.Spec.Digitalocean = &newv1.Digitalocean{
				PresetProvider: newv1.PresetProvider(oldSpec.Digitalocean.PresetProvider),
				Token:          oldSpec.Digitalocean.Token,
			}
		}

		if oldSpec.Fake != nil {
			newObject.Spec.Fake = &newv1.Fake{
				PresetProvider: newv1.PresetProvider(oldSpec.Fake.PresetProvider),
				Token:          oldSpec.Fake.Token,
			}
		}

		if oldSpec.GCP != nil {
			newObject.Spec.GCP = &newv1.GCP{
				PresetProvider: newv1.PresetProvider(oldSpec.GCP.PresetProvider),
				Network:        oldSpec.GCP.Network,
				Subnetwork:     oldSpec.GCP.Subnetwork,
				ServiceAccount: oldSpec.GCP.ServiceAccount,
			}
		}

		if oldSpec.Hetzner != nil {
			newObject.Spec.Hetzner = &newv1.Hetzner{
				PresetProvider: newv1.PresetProvider(oldSpec.Hetzner.PresetProvider),
				Token:          oldSpec.Hetzner.Token,
				Network:        oldSpec.Hetzner.Network,
			}
		}

		if oldSpec.Kubevirt != nil {
			newObject.Spec.Kubevirt = &newv1.Kubevirt{
				PresetProvider: newv1.PresetProvider(oldSpec.Kubevirt.PresetProvider),
				Kubeconfig:     oldSpec.Kubevirt.Kubeconfig,
			}
		}

		if oldSpec.Openstack != nil {
			newObject.Spec.Openstack = &newv1.Openstack{
				PresetProvider:              newv1.PresetProvider(oldSpec.Openstack.PresetProvider),
				UseToken:                    oldSpec.Openstack.UseToken,
				ApplicationCredentialID:     oldSpec.Openstack.ApplicationCredentialID,
				ApplicationCredentialSecret: oldSpec.Openstack.ApplicationCredentialSecret,
				Username:                    oldSpec.Openstack.Username,
				Password:                    oldSpec.Openstack.Password,
				Tenant:                      oldSpec.Openstack.Tenant,
				TenantID:                    oldSpec.Openstack.TenantID,
				Domain:                      oldSpec.Openstack.Domain,
				Network:                     oldSpec.Openstack.Network,
				SecurityGroups:              oldSpec.Openstack.SecurityGroups,
				FloatingIPPool:              oldSpec.Openstack.FloatingIPPool,
				RouterID:                    oldSpec.Openstack.RouterID,
				SubnetID:                    oldSpec.Openstack.SubnetID,
			}
		}

		if oldSpec.Packet != nil {
			newObject.Spec.Packet = &newv1.Packet{
				PresetProvider: newv1.PresetProvider(oldSpec.Packet.PresetProvider),
				APIKey:         oldSpec.Packet.APIKey,
				ProjectID:      oldSpec.Packet.ProjectID,
				BillingCycle:   oldSpec.Packet.BillingCycle,
			}
		}

		if oldSpec.VSphere != nil {
			newObject.Spec.VSphere = &newv1.VSphere{
				PresetProvider:   newv1.PresetProvider(oldSpec.VSphere.PresetProvider),
				Username:         oldSpec.VSphere.Username,
				Password:         oldSpec.VSphere.Password,
				VMNetName:        oldSpec.VSphere.VMNetName,
				Datastore:        oldSpec.VSphere.Datastore,
				DatastoreCluster: oldSpec.VSphere.DatastoreCluster,
				ResourcePool:     oldSpec.VSphere.ResourcePool,
			}
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneProjectResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.ProjectList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list Project objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.Project{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.ProjectSpec{
				Name: oldObject.Spec.Name,
			},
			Status: newv1.ProjectStatus{
				Phase: oldObject.Status.Phase,
			},
		}

		if err := ensureObject(ctx, client, &newObject, true); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}

		if err := client.Status().Update(ctx, &newObject); err != nil {
			return 0, fmt.Errorf("failed to update status on %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneRuleGroupResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.RuleGroupList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list RuleGroup objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.RuleGroup{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.RuleGroupSpec{
				RuleGroupType: newv1.RuleGroupType(oldObject.Spec.RuleGroupType),
				Data:          oldObject.Spec.Data,
				Cluster:       migrateObjectReference(oldObject.Spec.Cluster, ""),
			},
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneSeedResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.SeedList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list Seed objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.Seed{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.SeedSpec{
				Country:          oldObject.Spec.Country,
				Location:         oldObject.Spec.Location,
				Kubeconfig:       migrateObjectReference(oldObject.Spec.Kubeconfig, oldObject.Namespace),
				Datacenters:      map[string]newv1.Datacenter{},
				SeedDNSOverwrite: oldObject.Spec.SeedDNSOverwrite,
				NodeportProxy: newv1.NodeportProxyConfig{
					Disable:      oldObject.Spec.NodeportProxy.Disable,
					Annotations:  oldObject.Spec.NodeportProxy.Annotations,
					Envoy:        convertNodeportProxyComponent(oldObject.Spec.NodeportProxy.Envoy),
					EnvoyManager: convertNodeportProxyComponent(oldObject.Spec.NodeportProxy.EnvoyManager),
					Updater:      convertNodeportProxyComponent(oldObject.Spec.NodeportProxy.Updater),
				},
				ExposeStrategy:           newv1.ExposeStrategy(oldObject.Spec.ExposeStrategy),
				DefaultComponentSettings: convertComponentSettings(oldObject.Spec.DefaultComponentSettings),
			},
		}

		for name, oldDC := range oldObject.Spec.Datacenters {
			newObject.Spec.Datacenters[name] = convertDatacenter(oldDC)
		}

		if oldObject.Spec.ProxySettings != nil {
			newObject.Spec.ProxySettings = &newv1.ProxySettings{
				HTTPProxy: (*newv1.ProxyValue)(oldObject.Spec.ProxySettings.HTTPProxy),
				NoProxy:   (*newv1.ProxyValue)(oldObject.Spec.ProxySettings.NoProxy),
			}
		}

		if oldObject.Spec.MLA != nil {
			newObject.Spec.MLA = &newv1.SeedMLASettings{
				UserClusterMLAEnabled: oldObject.Spec.MLA.UserClusterMLAEnabled,
			}
		}

		if oldObject.Spec.Metering != nil {
			newObject.Spec.Metering = &newv1.MeteringConfiguration{
				Enabled:          oldObject.Spec.Metering.Enabled,
				StorageClassName: oldObject.Spec.Metering.StorageClassName,
				StorageSize:      oldObject.Spec.Metering.StorageSize,
			}
		}

		if oldObject.Spec.BackupRestore != nil {
			newObject.Spec.BackupRestore = &newv1.SeedBackupRestoreConfiguration{
				S3Endpoint:   oldObject.Spec.BackupRestore.S3Endpoint,
				S3BucketName: oldObject.Spec.BackupRestore.S3BucketName,
			}
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func convertComponentSettings(oldSettings kubermaticv1.ComponentSettings) newv1.ComponentSettings {
	newSettings := newv1.ComponentSettings{
		Apiserver: newv1.APIServerSettings{
			DeploymentSettings:          convertDeploymentSettings(oldSettings.Apiserver.DeploymentSettings),
			EndpointReconcilingDisabled: oldSettings.Apiserver.EndpointReconcilingDisabled,
			NodePortRange:               oldSettings.Apiserver.NodePortRange,
		},
		ControllerManager: convertControllerSettings(oldSettings.ControllerManager),
		Scheduler:         convertControllerSettings(oldSettings.Scheduler),
		Etcd: newv1.EtcdStatefulSetSettings{
			ClusterSize:  oldSettings.Etcd.ClusterSize,
			StorageClass: oldSettings.Etcd.StorageClass,
			DiskSize:     oldSettings.Etcd.DiskSize,
			Resources:    oldSettings.Etcd.Resources.DeepCopy(),
			Tolerations:  oldSettings.Etcd.Tolerations,
		},
		Prometheus: newv1.StatefulSetSettings{
			Resources: oldSettings.Prometheus.Resources.DeepCopy(),
		},
	}

	return newSettings
}

func convertControllerSettings(oldSettings kubermaticv1.ControllerSettings) newv1.ControllerSettings {
	return newv1.ControllerSettings{
		DeploymentSettings: convertDeploymentSettings(oldSettings.DeploymentSettings),
		LeaderElectionSettings: newv1.LeaderElectionSettings{
			LeaseDurationSeconds: oldSettings.LeaderElectionSettings.LeaseDurationSeconds,
			RenewDeadlineSeconds: oldSettings.LeaderElectionSettings.RenewDeadlineSeconds,
			RetryPeriodSeconds:   oldSettings.LeaderElectionSettings.RetryPeriodSeconds,
		},
	}
}

func convertDeploymentSettings(oldSettings kubermaticv1.DeploymentSettings) newv1.DeploymentSettings {
	return newv1.DeploymentSettings{
		Replicas:    oldSettings.Replicas,
		Resources:   oldSettings.Resources.DeepCopy(),
		Tolerations: oldSettings.Tolerations,
	}
}

func convertNodeportProxyComponent(oldComponent kubermaticv1.NodeportProxyComponent) newv1.NodeportProxyComponent {
	return newv1.NodeportProxyComponent{
		DockerRepository: oldComponent.DockerRepository,
		Resources:        *oldComponent.Resources.DeepCopy(),
	}
}

func convertDatacenter(oldDC kubermaticv1.Datacenter) newv1.Datacenter {
	newDC := newv1.Datacenter{
		Country:  oldDC.Country,
		Location: oldDC.Location,
		Spec: newv1.DatacenterSpec{
			EnforceAuditLogging:      oldDC.Spec.EnforceAuditLogging,
			EnforcePodSecurityPolicy: oldDC.Spec.EnforcePodSecurityPolicy,
			RequiredEmails:           oldDC.Spec.RequiredEmailDomains,
		},
	}

	// migrate from the deprecated flag to finally get rid of RequiredEmailDomain
	if oldDC.Spec.RequiredEmailDomain != "" {
		if newDC.Spec.RequiredEmails == nil {
			newDC.Spec.RequiredEmails = []string{}
		}

		newDC.Spec.RequiredEmails = append(newDC.Spec.RequiredEmails, oldDC.Spec.RequiredEmailDomain)
	}

	if oldDC.Node != nil {
		newDC.Node = &newv1.NodeSettings{
			ProxySettings: newv1.ProxySettings{
				HTTPProxy: (*newv1.ProxyValue)(oldDC.Node.HTTPProxy),
				NoProxy:   (*newv1.ProxyValue)(oldDC.Node.NoProxy),
			},
			InsecureRegistries: oldDC.Node.InsecureRegistries,
			RegistryMirrors:    oldDC.Node.RegistryMirrors,
			PauseImage:         oldDC.Node.PauseImage,
			HyperkubeImage:     oldDC.Node.HyperkubeImage,
		}
	}

	oldSpec := oldDC.Spec

	if oldSpec.AWS != nil {
		newDC.Spec.AWS = &newv1.DatacenterSpecAWS{
			Region: oldSpec.AWS.Region,
			Images: newv1.ImageList(oldSpec.AWS.Images),
		}
	}

	if oldSpec.Alibaba != nil {
		newDC.Spec.Alibaba = &newv1.DatacenterSpecAlibaba{
			Region: oldSpec.Alibaba.Region,
		}
	}

	if oldSpec.Anexia != nil {
		newDC.Spec.Anexia = &newv1.DatacenterSpecAnexia{
			LocationID: oldSpec.Anexia.LocationID,
		}
	}

	if oldSpec.Azure != nil {
		newDC.Spec.Azure = &newv1.DatacenterSpecAzure{
			Location: oldSpec.Azure.Location,
		}
	}

	if oldSpec.Digitalocean != nil {
		newDC.Spec.Digitalocean = &newv1.DatacenterSpecDigitalocean{
			Region: oldSpec.Digitalocean.Region,
		}
	}

	if oldSpec.Fake != nil {
		newDC.Spec.Fake = &newv1.DatacenterSpecFake{
			FakeProperty: oldSpec.Fake.FakeProperty,
		}
	}

	if oldSpec.GCP != nil {
		newDC.Spec.GCP = &newv1.DatacenterSpecGCP{
			Region:       oldSpec.GCP.Region,
			ZoneSuffixes: oldSpec.GCP.ZoneSuffixes,
			Regional:     oldSpec.GCP.Regional,
		}
	}

	if oldSpec.Hetzner != nil {
		newDC.Spec.Hetzner = &newv1.DatacenterSpecHetzner{
			Datacenter: oldSpec.Hetzner.Datacenter,
			Location:   oldSpec.Hetzner.Location,
			Network:    oldSpec.Hetzner.Network,
		}
	}

	if oldSpec.Kubevirt != nil {
		newDC.Spec.Kubevirt = &newv1.DatacenterSpecKubevirt{
			DNSPolicy: oldSpec.Kubevirt.DNSPolicy,
			DNSConfig: oldSpec.Kubevirt.DNSConfig.DeepCopy(),
		}
	}

	if oldSpec.Openstack != nil {
		newDC.Spec.Openstack = &newv1.DatacenterSpecOpenstack{
			AuthURL:              oldSpec.Openstack.AuthURL,
			AvailabilityZone:     oldSpec.Openstack.AvailabilityZone,
			Region:               oldSpec.Openstack.Region,
			IgnoreVolumeAZ:       oldSpec.Openstack.IgnoreVolumeAZ,
			EnforceFloatingIP:    oldSpec.Openstack.EnforceFloatingIP,
			DNSServers:           oldSpec.Openstack.DNSServers,
			Images:               newv1.ImageList(oldSpec.Openstack.Images),
			ManageSecurityGroups: oldSpec.Openstack.ManageSecurityGroups,
			UseOctavia:           oldSpec.Openstack.UseOctavia,
			TrustDevicePath:      oldSpec.Openstack.TrustDevicePath,
			NodeSizeRequirements: newv1.OpenstackNodeSizeRequirements{
				MinimumVCPUs:  oldSpec.Openstack.NodeSizeRequirements.MinimumVCPUs,
				MinimumMemory: oldSpec.Openstack.NodeSizeRequirements.MinimumMemory,
			},
			EnabledFlavors: oldSpec.Openstack.EnabledFlavors,
		}
	}

	if oldSpec.Packet != nil {
		newDC.Spec.Packet = &newv1.DatacenterSpecPacket{
			Facilities: oldSpec.Packet.Facilities,
		}
	}

	if oldSpec.VSphere != nil {
		newDC.Spec.VSphere = &newv1.DatacenterSpecVSphere{
			Endpoint:             oldSpec.VSphere.Endpoint,
			AllowInsecure:        oldSpec.VSphere.AllowInsecure,
			DefaultDatastore:     oldSpec.VSphere.DefaultDatastore,
			Datacenter:           oldSpec.VSphere.Datacenter,
			Cluster:              oldSpec.VSphere.Cluster,
			DefaultStoragePolicy: oldSpec.VSphere.DefaultStoragePolicy,
			RootPath:             oldSpec.VSphere.RootPath,
			Templates:            newv1.ImageList(oldSpec.VSphere.Templates),
		}

		if iu := oldSpec.VSphere.InfraManagementUser; iu != nil {
			newDC.Spec.VSphere.InfraManagementUser = &newv1.VSphereCredentials{
				Username: iu.Username,
				Password: iu.Password,
			}
		}
	}

	return newDC
}

func cloneUserResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.UserList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list User objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.User{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.UserSpec{
				ID:                     oldObject.Spec.ID,
				Name:                   oldObject.Spec.Name,
				Email:                  oldObject.Spec.Email,
				IsAdmin:                oldObject.Spec.IsAdmin,
				InvalidTokensReference: oldObject.Spec.TokenBlackListReference,
			},
		}

		if s := oldObject.Spec.Settings; s != nil {
			newObject.Spec.Settings = &newv1.UserSettings{
				SelectedTheme:              s.SelectedTheme,
				ItemsPerPage:               s.ItemsPerPage,
				SelectedProjectID:          s.SelectedProjectID,
				SelectProjectTableView:     s.SelectProjectTableView,
				CollapseSidenav:            s.CollapseSidenav,
				DisplayAllProjectsForAdmin: s.DisplayAllProjectsForAdmin,
				LastSeenChangelogVersion:   s.LastSeenChangelogVersion,
			}
		}

		if err := ensureObject(ctx, client, &newObject, true); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneUserProjectBindingResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.UserProjectBindingList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list UserProjectBinding objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.UserProjectBinding{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.UserProjectBindingSpec{
				UserEmail: oldObject.Spec.UserEmail,
				ProjectID: oldObject.Spec.ProjectID,
				Group:     oldObject.Spec.Group,
			},
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}

func cloneUserSSHKeyResourcesInCluster(ctx context.Context, logger logrus.FieldLogger, client ctrlruntimeclient.Client) (int, error) {
	oldObjects := &kubermaticv1.UserSSHKeyList{}
	if err := client.List(ctx, oldObjects); err != nil {
		return 0, fmt.Errorf("failed to list UserSSHKey objects: %w", err)
	}

	for _, oldObject := range oldObjects.Items {
		newObject := newv1.UserSSHKey{
			ObjectMeta: convertObjectMeta(oldObject.ObjectMeta),
			Spec: newv1.SSHKeySpec{
				Owner:       oldObject.Spec.Owner,
				Name:        oldObject.Spec.Name,
				Fingerprint: oldObject.Spec.Fingerprint,
				PublicKey:   oldObject.Spec.PublicKey,
				Clusters:    oldObject.Spec.Clusters,
			},
		}

		if err := ensureObject(ctx, client, &newObject, false); err != nil {
			return 0, fmt.Errorf("failed to clone %s: %w", oldObject.Name, err)
		}
	}

	return len(oldObjects.Items), nil
}
