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

package externalcluster

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/go-kit/kit/endpoint"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	gcpprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/gcp"
	gkeprovider "k8c.io/kubermatic/v2/pkg/provider/cloud/gke"
	"k8c.io/kubermatic/v2/pkg/resources"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	ManualMode = "Manual"
	AutoMode   = "Auto"
)

func GKEImagesWithClusterCredentialsEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(ctx, settingsProvider) {
			return nil, utilerrors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req, ok := request.(GetClusterReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())
		sa, err := secretKeySelector(cluster.Spec.CloudSpec.GKE.CredentialsReference, resources.GCPServiceAccount)
		if err != nil {
			return nil, err
		}

		images := apiv2.GKEImageList{}
		svc, gcpProject, err := gkeprovider.ConnectToContainerService(ctx, sa)
		if err != nil {
			return nil, err
		}

		config, err := svc.Projects.Zones.GetServerconfig(gcpProject, cluster.Spec.CloudSpec.GKE.Zone).Context(ctx).Do()
		if err != nil {
			return nil, gkeprovider.DecodeError(err)
		}

		for _, imageType := range config.ValidImageTypes {
			images = append(images, apiv2.GKEImage{
				Name:      imageType,
				IsDefault: imageType == config.DefaultImageType,
			})
		}

		return images, nil
	}
}

func GKEZonesWithClusterCredentialsEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(ctx, settingsProvider) {
			return nil, utilerrors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req, ok := request.(GetClusterReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())
		sa, err := secretKeySelector(cluster.Spec.CloudSpec.GKE.CredentialsReference, resources.GCPServiceAccount)
		if err != nil {
			return nil, err
		}
		computeService, gcpProject, err := gcpprovider.ConnectToComputeService(ctx, sa)
		if err != nil {
			return nil, err
		}
		clusterZone := cluster.Spec.CloudSpec.GKE.Zone
		// Zone always contains continent-region-suffix.
		// To find other zones we construct new continent-region string.
		locationList := strings.Split(clusterZone, "-")
		if len(locationList) != 3 {
			return nil, fmt.Errorf("incorrect zone format, %s", clusterZone)
		}
		continentRegion := fmt.Sprintf("%s-%s", locationList[0], locationList[1])

		zones := apiv2.GKEZoneList{}
		zoneReq := computeService.Zones.List(gcpProject)
		err = zoneReq.Pages(ctx, func(page *compute.ZoneList) error {
			for _, zone := range page.Items {
				if strings.HasPrefix(zone.Name, continentRegion) {
					apiZone := apiv2.GKEZone{
						Name:      zone.Name,
						IsDefault: zone.Name == clusterZone,
					}
					zones = append(zones, apiZone)
				}
			}
			return nil
		})

		return zones, err
	}
}

func GKESizesWithClusterCredentialsEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(ctx, settingsProvider) {
			return nil, utilerrors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req, ok := request.(GetClusterReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())
		sa, err := secretKeySelector(cluster.Spec.CloudSpec.GKE.CredentialsReference, resources.GCPServiceAccount)
		if err != nil {
			return nil, err
		}

		return gkeprovider.ListGKESizes(ctx, sa, cluster.Spec.CloudSpec.GKE.Zone)
	}
}

func GKEDiskTypesWithClusterCredentialsEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(ctx, settingsProvider) {
			return nil, utilerrors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req, ok := request.(GetClusterReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		cluster, err := getCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, project.Name, req.ClusterID)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, privilegedClusterProvider.GetMasterClient())
		sa, err := secretKeySelector(cluster.Spec.CloudSpec.GKE.CredentialsReference, resources.GCPServiceAccount)
		if err != nil {
			return nil, err
		}

		return gkeprovider.ListGKEDiskTypes(ctx, sa, cluster.Spec.CloudSpec.GKE.Zone)
	}
}

func createOrImportGKECluster(ctx context.Context, name string, userInfoGetter provider.UserInfoGetter, project *kubermaticv1.Project, spec *apiv2.ExternalClusterSpec, cloud *apiv2.ExternalClusterCloudSpec, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) (*kubermaticv1.ExternalCluster, error) {
	isImported := resources.ExternalClusterIsImportedTrue

	if cloud.GKE.Name == "" || cloud.GKE.Zone == "" || cloud.GKE.ServiceAccount == "" {
		return nil, utilerrors.NewBadRequest("the GKE cluster name, zone or service account can not be empty")
	}

	if spec != nil && spec.GKEClusterSpec != nil {
		if err := createNewGKECluster(ctx, spec.GKEClusterSpec, cloud.GKE); err != nil {
			return nil, err
		}
		isImported = resources.ExternalClusterIsImportedFalse
	}

	newCluster := genExternalCluster(name, project.Name, isImported)
	newCluster.Spec.CloudSpec = kubermaticv1.ExternalClusterCloudSpec{
		GKE: &kubermaticv1.ExternalClusterGKECloudSpec{
			Name: cloud.GKE.Name,
			Zone: cloud.GKE.Zone,
		},
	}
	keyRef, err := clusterProvider.CreateOrUpdateCredentialSecretForCluster(ctx, cloud, project.Name, newCluster.Name)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	newCluster.Spec.CloudSpec.GKE.CredentialsReference = keyRef

	return createNewCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, newCluster, project)
}

func patchGKECluster(ctx context.Context, oldCluster, newCluster *apiv2.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector) (*apiv2.ExternalCluster, error) {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := gkeprovider.ConnectToContainerService(ctx, sa)
	if err != nil {
		return nil, err
	}

	updateclusterrequest := &container.UpdateClusterRequest{}
	newVersion := newCluster.Spec.Version.Semver()
	if !oldCluster.Spec.Version.Semver().Equal(newVersion) {
		updateclusterrequest.Update = &container.ClusterUpdate{
			DesiredMasterVersion: newVersion.String(),
		}
	}

	req := svc.Projects.Zones.Clusters.Update(project, oldCluster.Cloud.GKE.Zone, oldCluster.Cloud.GKE.Name, updateclusterrequest)
	_, err = req.Context(ctx).Do()

	return newCluster, gkeprovider.DecodeError(err)
}

func getGKENodePools(ctx context.Context, cluster *kubermaticv1.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterMachineDeployment, error) {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := gkeprovider.ConnectToContainerService(ctx, sa)
	if err != nil {
		return nil, err
	}

	req := svc.Projects.Zones.Clusters.NodePools.List(project, cluster.Spec.CloudSpec.GKE.Zone, cluster.Spec.CloudSpec.GKE.Name)
	resp, err := req.Context(ctx).Do()
	if err != nil {
		return nil, gkeprovider.DecodeError(err)
	}

	machineDeployments := make([]apiv2.ExternalClusterMachineDeployment, 0, len(resp.NodePools))

	nodes, err := clusterProvider.ListNodes(ctx, cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	for _, md := range resp.NodePools {
		var readyReplicasCount int32
		for _, node := range nodes.Items {
			if node.Labels[resources.GKENodepoolNameLabel] == md.Name && kuberneteshelper.IsNodeReady(&node) {
				readyReplicasCount++
			}
		}

		machineDeployments = append(machineDeployments, createMachineDeploymentFromGKENodePoll(md, readyReplicasCount))
	}

	return machineDeployments, err
}

func createMachineDeploymentFromGKENodePoll(np *container.NodePool, readyReplicas int32) apiv2.ExternalClusterMachineDeployment {
	md := apiv2.ExternalClusterMachineDeployment{
		NodeDeployment: apiv1.NodeDeployment{
			ObjectMeta: apiv1.ObjectMeta{
				ID:   np.Name,
				Name: np.Name,
			},
			Spec: apiv1.NodeDeploymentSpec{
				Replicas: int32(np.InitialNodeCount),
				Template: apiv1.NodeSpec{
					Versions: apiv1.NodeVersionInfo{
						Kubelet: np.Version,
					},
				},
			},
			Status: clusterv1alpha1.MachineDeploymentStatus{
				Replicas:      int32(np.InitialNodeCount),
				ReadyReplicas: readyReplicas,
			},
		},
		Cloud: &apiv2.ExternalClusterMachineDeploymentCloudSpec{
			GKE: &apiv2.GKEMachineDeploymentCloudSpec{},
		},
	}

	if np.Autoscaling != nil {
		md.Cloud.GKE.Autoscaling = &apiv2.GKENodePoolAutoscaling{
			Autoprovisioned: np.Autoscaling.Autoprovisioned,
			Enabled:         np.Autoscaling.Enabled,
			MaxNodeCount:    np.Autoscaling.MaxNodeCount,
			MinNodeCount:    np.Autoscaling.MinNodeCount,
		}
	}
	if np.Config != nil {
		md.Cloud.GKE.Config = &apiv2.GKENodeConfig{
			DiskSizeGb:    np.Config.DiskSizeGb,
			DiskType:      np.Config.DiskType,
			ImageType:     np.Config.ImageType,
			LocalSsdCount: np.Config.LocalSsdCount,
			MachineType:   np.Config.MachineType,
		}
	}
	if np.Management != nil {
		md.Cloud.GKE.Management = &apiv2.GKENodeManagement{
			AutoRepair:  np.Management.AutoRepair,
			AutoUpgrade: np.Management.AutoUpgrade,
		}
	}

	md.Phase = apiv2.ExternalClusterMDPhase{
		State:         gkeprovider.ConvertMDStatus(np.Status),
		StatusMessage: gkeprovider.GetMDStatusMessage(np),
	}

	return md
}

func getGKENodePool(ctx context.Context, cluster *kubermaticv1.ExternalCluster, nodeGroupName string, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) (*apiv2.ExternalClusterMachineDeployment, error) {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := gkeprovider.ConnectToContainerService(ctx, sa)
	if err != nil {
		return nil, err
	}

	return getGKEMachineDeployment(ctx, svc, project, cluster, nodeGroupName, clusterProvider)
}

func deleteGKENodePool(ctx context.Context, cluster *kubermaticv1.ExternalCluster, nodePoolID string, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) error {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return err
	}
	svc, project, err := gkeprovider.ConnectToContainerService(ctx, sa)
	if err != nil {
		return err
	}

	req := svc.Projects.Zones.Clusters.NodePools.Delete(project, cluster.Spec.CloudSpec.GKE.Zone, cluster.Spec.CloudSpec.GKE.Name, nodePoolID)
	_, err = req.Context(ctx).Do()
	return gkeprovider.DecodeError(err)
}

func patchGKEMachineDeployment(ctx context.Context, oldMD, newMD *apiv2.ExternalClusterMachineDeployment, cluster *kubermaticv1.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector) (*apiv2.ExternalClusterMachineDeployment, error) {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := gkeprovider.ConnectToContainerService(ctx, sa)
	if err != nil {
		return nil, err
	}

	// The GKE can update Node Pool size or NodePoll object. Can't change both because first update blocks the second one.
	// It's required to update Node Poll size separately.

	// only when size was updates otherwise change NodePoll object with other parameters
	if oldMD.Spec.Replicas != newMD.Spec.Replicas {
		sizeRequest := &container.SetNodePoolSizeRequest{
			NodeCount: int64(newMD.Spec.Replicas),
		}
		sizeReq := svc.Projects.Zones.Clusters.NodePools.SetSize(project, cluster.Spec.CloudSpec.GKE.Zone, cluster.Spec.CloudSpec.GKE.Name, oldMD.Name, sizeRequest)
		_, err = sizeReq.Context(ctx).Do()
		if err != nil {
			return nil, gkeprovider.DecodeError(err)
		}
		return newMD, nil
	}

	updateRequest := &container.UpdateNodePoolRequest{
		NodeVersion: newMD.Spec.Template.Versions.Kubelet,
	}
	updateReq := svc.Projects.Zones.Clusters.NodePools.Update(project, cluster.Spec.CloudSpec.GKE.Zone, cluster.Spec.CloudSpec.GKE.Name, oldMD.Name, updateRequest)
	_, err = updateReq.Context(ctx).Do()
	if err != nil {
		return nil, gkeprovider.DecodeError(err)
	}

	return newMD, nil
}

func getGKEMachineDeployment(ctx context.Context, svc *container.Service, projectID string, cluster *kubermaticv1.ExternalCluster, nodeGroupName string, clusterProvider provider.ExternalClusterProvider) (*apiv2.ExternalClusterMachineDeployment, error) {
	req := svc.Projects.Zones.Clusters.NodePools.Get(projectID, cluster.Spec.CloudSpec.GKE.Zone, cluster.Spec.CloudSpec.GKE.Name, nodeGroupName)
	np, err := req.Context(ctx).Do()
	if err != nil {
		return nil, gkeprovider.DecodeError(err)
	}

	nodes, err := clusterProvider.ListNodes(ctx, cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	var readyReplicas int32
	for _, n := range nodes.Items {
		if n.Labels != nil {
			if n.Labels[resources.GKENodepoolNameLabel] == np.Name {
				readyReplicas++
			}
		}
	}
	md := createMachineDeploymentFromGKENodePoll(np, readyReplicas)

	return &md, nil
}

func createGKENodePool(ctx context.Context, cloudSpec *kubermaticv1.ExternalClusterGKECloudSpec, machineDeployment apiv2.ExternalClusterMachineDeployment, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector) (*apiv2.ExternalClusterMachineDeployment, error) {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := gkeprovider.ConnectToContainerService(ctx, sa)
	if err != nil {
		return nil, err
	}

	if machineDeployment.Cloud.GKE == nil {
		return nil, fmt.Errorf("GKE cloud spec cannot be empty")
	}

	gke := machineDeployment.Cloud.GKE

	nodePool := &container.NodePool{
		Config:            nil,
		InitialNodeCount:  int64(machineDeployment.Spec.Replicas),
		InstanceGroupUrls: nil,
		Locations:         nil,
		Management:        nil,
		MaxPodsConstraint: nil,
		Name:              machineDeployment.Name,
		Version:           machineDeployment.Spec.Template.Versions.Kubelet,
	}

	if gke.Config != nil {
		nodePool.Config = &container.NodeConfig{
			DiskSizeGb:    gke.Config.DiskSizeGb,
			DiskType:      gke.Config.DiskType,
			ImageType:     gke.Config.ImageType,
			Labels:        gke.Config.Labels,
			LocalSsdCount: gke.Config.LocalSsdCount,
			MachineType:   gke.Config.MachineType,
		}
	}
	if gke.Autoscaling != nil {
		nodePool.Autoscaling = &container.NodePoolAutoscaling{
			Autoprovisioned: gke.Autoscaling.Autoprovisioned,
			Enabled:         gke.Autoscaling.Enabled,
			MaxNodeCount:    gke.Autoscaling.MaxNodeCount,
			MinNodeCount:    gke.Autoscaling.MinNodeCount,
		}
	}
	if gke.Management != nil {
		nodePool.Management = &container.NodeManagement{
			AutoRepair:  gke.Management.AutoRepair,
			AutoUpgrade: gke.Management.AutoUpgrade,
		}
	}

	createRequest := &container.CreateNodePoolRequest{
		NodePool: nodePool,
	}
	req := svc.Projects.Zones.Clusters.NodePools.Create(project, cloudSpec.Zone, cloudSpec.Name, createRequest)
	_, err = req.Context(ctx).Do()
	if err != nil {
		return nil, gkeprovider.DecodeError(err)
	}

	machineDeployment.Phase = apiv2.ExternalClusterMDPhase{
		State: apiv2.ProvisioningExternalClusterMDState,
	}
	return &machineDeployment, nil
}

func createNewGKECluster(ctx context.Context, gkeClusterSpec *apiv2.GKEClusterSpec, gkeCloudSpec *apiv2.GKECloudSpec) error {
	svc, project, err := gkeprovider.ConnectToContainerService(ctx, gkeCloudSpec.ServiceAccount)
	if err != nil {
		return err
	}

	createRequest := &container.CreateClusterRequest{
		Cluster: genGKECluster(gkeClusterSpec, gkeCloudSpec),
	}

	req := svc.Projects.Zones.Clusters.Create(project, gkeCloudSpec.Zone, createRequest)
	_, err = req.Context(ctx).Do()

	return gkeprovider.DecodeError(err)
}

func genGKECluster(gkeClusterSpec *apiv2.GKEClusterSpec, gkeCloudSpec *apiv2.GKECloudSpec) *container.Cluster {
	newCluster := &container.Cluster{
		ClusterIpv4Cidr:       gkeClusterSpec.ClusterIpv4Cidr,
		InitialClusterVersion: gkeClusterSpec.InitialClusterVersion,
		InitialNodeCount:      gkeClusterSpec.InitialNodeCount,
		Locations:             gkeClusterSpec.Locations,
		Name:                  gkeCloudSpec.Name,
		Network:               gkeClusterSpec.Network,
		Subnetwork:            gkeClusterSpec.Subnetwork,
		TpuIpv4CidrBlock:      gkeClusterSpec.TpuIpv4CidrBlock,
		EnableKubernetesAlpha: gkeClusterSpec.EnableKubernetesAlpha,
		EnableTpu:             gkeClusterSpec.EnableTpu,
		Autopilot: &container.Autopilot{
			Enabled: gkeClusterSpec.Autopilot,
		},
		VerticalPodAutoscaling: &container.VerticalPodAutoscaling{
			Enabled: gkeClusterSpec.VerticalPodAutoscaling,
		},
	}
	if gkeClusterSpec.NodeConfig != nil {
		newCluster.NodeConfig = &container.NodeConfig{
			DiskSizeGb:    gkeClusterSpec.NodeConfig.DiskSizeGb,
			DiskType:      gkeClusterSpec.NodeConfig.DiskType,
			ImageType:     gkeClusterSpec.NodeConfig.ImageType,
			Labels:        gkeClusterSpec.NodeConfig.Labels,
			LocalSsdCount: gkeClusterSpec.NodeConfig.LocalSsdCount,
			MachineType:   gkeClusterSpec.NodeConfig.MachineType,
			Preemptible:   gkeClusterSpec.NodeConfig.Preemptible,
		}
	}
	if gkeClusterSpec.Autoscaling != nil {
		newCluster.Autoscaling = &container.ClusterAutoscaling{
			AutoprovisioningLocations:  gkeClusterSpec.Autoscaling.AutoprovisioningLocations,
			EnableNodeAutoprovisioning: gkeClusterSpec.Autoscaling.EnableNodeAutoprovisioning,
		}
		if gkeClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults != nil {
			newCluster.Autoscaling.AutoprovisioningNodePoolDefaults = &container.AutoprovisioningNodePoolDefaults{
				BootDiskKmsKey: gkeClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.BootDiskKmsKey,
				DiskSizeGb:     gkeClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.DiskSizeGb,
				DiskType:       gkeClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.DiskType,
				MinCpuPlatform: gkeClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.MinCpuPlatform,
				OauthScopes:    gkeClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.OauthScopes,
				ServiceAccount: gkeClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.ServiceAccount,
			}
			if gkeClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.Management != nil {
				newCluster.Autoscaling.AutoprovisioningNodePoolDefaults.Management = &container.NodeManagement{
					AutoRepair:  gkeClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.Management.AutoRepair,
					AutoUpgrade: gkeClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.Management.AutoUpgrade,
				}
			}
			if gkeClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.UpgradeSettings != nil {
				newCluster.Autoscaling.AutoprovisioningNodePoolDefaults.UpgradeSettings = &container.UpgradeSettings{
					MaxSurge:       gkeClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.UpgradeSettings.MaxSurge,
					MaxUnavailable: gkeClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.UpgradeSettings.MaxUnavailable,
				}
			}
			if gkeClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.ShieldedInstanceConfig != nil {
				newCluster.Autoscaling.AutoprovisioningNodePoolDefaults.ShieldedInstanceConfig = &container.ShieldedInstanceConfig{
					EnableIntegrityMonitoring: gkeClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.ShieldedInstanceConfig.EnableIntegrityMonitoring,
					EnableSecureBoot:          gkeClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.ShieldedInstanceConfig.EnableSecureBoot,
				}
			}
		}
		if len(gkeClusterSpec.Autoscaling.ResourceLimits) != 0 {
			newCluster.Autoscaling.ResourceLimits = make([]*container.ResourceLimit, len(gkeClusterSpec.Autoscaling.ResourceLimits))
			for _, rl := range gkeClusterSpec.Autoscaling.ResourceLimits {
				newCluster.Autoscaling.ResourceLimits = append(newCluster.Autoscaling.ResourceLimits, &container.ResourceLimit{
					Maximum:      rl.Maximum,
					Minimum:      rl.Minimum,
					ResourceType: rl.ResourceType,
				})
			}
		}
	}

	return newCluster
}

func getGKEClusterDetails(ctx context.Context, apiCluster *apiv2.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticv1.ExternalClusterGKECloudSpec) (*apiv2.ExternalCluster, error) {
	sa, err := secretKeySelector(cloudSpec.CredentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := gkeprovider.ConnectToContainerService(ctx, sa)
	if err != nil {
		return nil, err
	}

	req := svc.Projects.Zones.Clusters.Get(project, cloudSpec.Zone, cloudSpec.Name)
	resp, err := req.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("cannot get cluster for project=%s: %w", project, gkeprovider.DecodeError(err))
	}

	clusterSpec := &apiv2.GKEClusterSpec{
		ClusterIpv4Cidr:       resp.ClusterIpv4Cidr,
		EnableKubernetesAlpha: resp.EnableKubernetesAlpha,
		EnableTpu:             resp.EnableTpu,
		InitialClusterVersion: resp.InitialClusterVersion,
		InitialNodeCount:      resp.InitialNodeCount,
		Locations:             resp.Locations,
		Network:               resp.Network,
		Subnetwork:            resp.Subnetwork,
		TpuIpv4CidrBlock:      resp.TpuIpv4CidrBlock,
	}

	clusterSpec.CreateTime = resp.CreateTime

	if resp.DefaultMaxPodsConstraint != nil {
		clusterSpec.DefaultMaxPodsConstraint = &resp.DefaultMaxPodsConstraint.MaxPodsPerNode
	}
	if resp.Autopilot != nil {
		clusterSpec.Autopilot = resp.Autopilot.Enabled
	}
	if resp.VerticalPodAutoscaling != nil {
		clusterSpec.VerticalPodAutoscaling = resp.VerticalPodAutoscaling.Enabled
	}
	if resp.Autoscaling != nil {
		clusterSpec.Autoscaling = &apiv2.GKEClusterAutoscaling{
			AutoprovisioningLocations:  resp.Autoscaling.AutoprovisioningLocations,
			EnableNodeAutoprovisioning: resp.Autoscaling.EnableNodeAutoprovisioning,
		}
		if resp.Autoscaling.AutoprovisioningNodePoolDefaults != nil {
			clusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults = &apiv2.GKEAutoprovisioningNodePoolDefaults{
				BootDiskKmsKey: resp.Autoscaling.AutoprovisioningNodePoolDefaults.BootDiskKmsKey,
				DiskSizeGb:     resp.Autoscaling.AutoprovisioningNodePoolDefaults.DiskSizeGb,
				DiskType:       resp.Autoscaling.AutoprovisioningNodePoolDefaults.DiskType,
				MinCpuPlatform: resp.Autoscaling.AutoprovisioningNodePoolDefaults.MinCpuPlatform,
				OauthScopes:    resp.Autoscaling.AutoprovisioningNodePoolDefaults.OauthScopes,
				ServiceAccount: resp.Autoscaling.AutoprovisioningNodePoolDefaults.ServiceAccount,
			}
			if resp.Autoscaling.AutoprovisioningNodePoolDefaults.Management != nil {
				clusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.Management = &apiv2.GKENodeManagement{
					AutoRepair:  resp.Autoscaling.AutoprovisioningNodePoolDefaults.Management.AutoRepair,
					AutoUpgrade: resp.Autoscaling.AutoprovisioningNodePoolDefaults.Management.AutoUpgrade,
				}
			}
			if resp.Autoscaling.AutoprovisioningNodePoolDefaults.ShieldedInstanceConfig != nil {
				clusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.ShieldedInstanceConfig = &apiv2.GKEShieldedInstanceConfig{
					EnableIntegrityMonitoring: resp.Autoscaling.AutoprovisioningNodePoolDefaults.ShieldedInstanceConfig.EnableIntegrityMonitoring,
					EnableSecureBoot:          resp.Autoscaling.AutoprovisioningNodePoolDefaults.ShieldedInstanceConfig.EnableSecureBoot,
				}
			}
			if resp.Autoscaling.AutoprovisioningNodePoolDefaults.UpgradeSettings != nil {
				clusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.UpgradeSettings = &apiv2.GKEUpgradeSettings{
					MaxSurge:       resp.Autoscaling.AutoprovisioningNodePoolDefaults.UpgradeSettings.MaxSurge,
					MaxUnavailable: resp.Autoscaling.AutoprovisioningNodePoolDefaults.UpgradeSettings.MaxUnavailable,
				}
			}
		}

		if resp.Autoscaling.ResourceLimits != nil {
			clusterSpec.Autoscaling.ResourceLimits = make([]*apiv2.GKEResourceLimit, 0)
			for _, limit := range resp.Autoscaling.ResourceLimits {
				clusterSpec.Autoscaling.ResourceLimits = append(clusterSpec.Autoscaling.ResourceLimits, &apiv2.GKEResourceLimit{
					Maximum:      limit.Maximum,
					Minimum:      limit.Minimum,
					ResourceType: limit.ResourceType,
				})
			}
		}
	}

	apiCluster.Spec.GKEClusterSpec = clusterSpec

	return apiCluster, nil
}

func deleteGKECluster(ctx context.Context, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticv1.ExternalClusterGKECloudSpec) error {
	sa, err := secretKeySelector(cloudSpec.CredentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return err
	}
	svc, project, err := gkeprovider.ConnectToContainerService(ctx, sa)
	if err != nil {
		return err
	}

	req := svc.Projects.Zones.Clusters.Delete(project, cloudSpec.Zone, cloudSpec.Name)
	_, err = req.Context(ctx).Do()
	if err != nil {
		return gkeprovider.DecodeError(err)
	}

	return nil
}

// GKEVMReq represent a request for GKE VM.
// swagger:parameters listGKEImages listGKEVMSizes
type GKEVMReq struct {
	GKECommonReq
	// The zone name
	// in: header
	// name: Zone
	Zone string
}

// GKEVersionsReq represent a request for GKE versions.
// swagger:parameters listGKEVersions
type GKEVersionsReq struct {
	GKECommonReq
	// The zone name
	// in: header
	// name: Zone
	Zone string
	// The Mode is how you want GKE Control plane version to be managed.
	// Manual: Manually manage the version upgrades.
	// Auto: automatically manage the cluster's control plane version.
	// in: header
	// name: Mode
	Mode string
	// The ReleaseChannel
	// in: header
	// name: ReleaseChannel
	ReleaseChannel string
}

func GKEVersionsEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(GKEVersionsReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		sa := req.ServiceAccount
		var err error
		if len(req.Credential) > 0 {
			sa, err = getSAFromPreset(ctx, userInfoGetter, presetProvider, req.Credential)
			if err != nil {
				return nil, err
			}
		}
		return ListGKEVersions(ctx, sa, req.Zone, req.Mode, req.ReleaseChannel)
	}
}

// Validate validates GKECommonReq request.
func (req GKECommonReq) Validate() error {
	if len(req.ServiceAccount) == 0 && len(req.Credential) == 0 {
		return fmt.Errorf("GKE credentials cannot be empty")
	}
	return nil
}

// Validate validates GKEVMReq request.
func (req GKEVMReq) Validate() error {
	if err := req.GKECommonReq.Validate(); err != nil {
		return err
	}
	if len(req.Zone) == 0 {
		return fmt.Errorf("GKE Zone cannot be empty")
	}
	return nil
}

// Validate validates GKEVersionsReq request.
func (req GKEVersionsReq) Validate() error {
	if err := req.GKECommonReq.Validate(); err != nil {
		return err
	}
	if len(req.Zone) == 0 {
		return fmt.Errorf("GKE Zone cannot be empty")
	}
	if len(req.Mode) == 0 {
		return fmt.Errorf("GKE \"Mode\" cannot be empty")
	}
	if !sets.NewString(ManualMode, AutoMode).Has(req.Mode) {
		return fmt.Errorf("provide valid GKE Mode: %s", req.Mode)
	}
	if req.Mode == AutoMode {
		if len(req.ReleaseChannel) == 0 {
			return fmt.Errorf("GKE \"ReleaseChannel\" cannot be empty")
		}
		if !sets.NewString(resources.GKERapidReleaseChannel, resources.GKERegularReleaseChannel, resources.GKEStableReleaseChannel).Has(req.ReleaseChannel) {
			return fmt.Errorf("provide valid GKE ReleaseChannel: %s", req.ReleaseChannel)
		}
	}

	return nil
}

func DecodeGKEVMReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GKEVMReq

	commonReq, err := DecodeGKECommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GKECommonReq = commonReq.(GKECommonReq)

	req.Zone = r.Header.Get("Zone")

	return req, nil
}

func DecodeGKEVersionsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GKEVersionsReq

	commonReq, err := DecodeGKECommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GKECommonReq = commonReq.(GKECommonReq)

	req.Zone = r.Header.Get("Zone")
	req.ReleaseChannel = r.Header.Get("ReleaseChannel")
	req.Mode = r.Header.Get("Mode")

	return req, nil
}

func DecodeGKEClusterListReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GKEClusterListReq

	commonReq, err := DecodeGKECommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GKECommonReq = commonReq.(GKECommonReq)
	pr, err := common.DecodeProjectRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.ProjectReq = pr.(common.ProjectReq)

	return req, nil
}

// GKETypesReq represent a request for GKE types.
// swagger:parameters validateGKECredentials
type GKETypesReq struct {
	GKECommonReq
}

func DecodeGKECommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GKECommonReq

	req.ServiceAccount = r.Header.Get("ServiceAccount")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

func DecodeGKETypesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GKETypesReq

	commonReq, err := DecodeGKECommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GKECommonReq = commonReq.(GKECommonReq)

	return req, nil
}

func GKEClustersEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, presetProvider provider.PresetProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(GKEClusterListReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		sa := req.ServiceAccount
		var err error
		if len(req.Credential) > 0 {
			sa, err = getSAFromPreset(ctx, userInfoGetter, presetProvider, req.Credential)
			if err != nil {
				return nil, err
			}
		}
		return gkeprovider.ListClusters(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, clusterProvider, req.ProjectID, sa)
	}
}

func GKEImagesEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(GKEVMReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		sa := req.ServiceAccount
		var err error
		if len(req.Credential) > 0 {
			sa, err = getSAFromPreset(ctx, userInfoGetter, presetProvider, req.Credential)
			if err != nil {
				return nil, err
			}
		}
		return gkeprovider.ListImages(ctx, sa, req.Zone)
	}
}

func GKEZonesEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(GKECommonReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		sa := req.ServiceAccount
		var err error
		if len(req.Credential) > 0 {
			sa, err = getSAFromPreset(ctx, userInfoGetter, presetProvider, req.Credential)
			if err != nil {
				return nil, err
			}
		}
		return gkeprovider.ListZones(ctx, sa)
	}
}

// GKECommonReq represent a request with common parameters for GKE.
type GKECommonReq struct {
	// The plain GCP service account
	// in: header
	// name: ServiceAccount
	ServiceAccount string
	// The credential name used in the preset for the GCP provider
	// in: header
	// name: Credential
	Credential string
}

// GKEClusterListReq represent a request for GKE cluster list.
// swagger:parameters listGKEClusters
type GKEClusterListReq struct {
	common.ProjectReq
	GKECommonReq
}

func GKEValidateCredentialsEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(GKETypesReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		var err error
		sa := req.ServiceAccount
		if len(req.Credential) > 0 {
			sa, err = getSAFromPreset(ctx, userInfoGetter, presetProvider, req.Credential)
			if err != nil {
				return nil, err
			}
		}

		err = gkeprovider.ValidateCredentials(ctx, sa)
		if err != nil {
			err = fmt.Errorf("invalid credentials!: %w", err)
		}
		return nil, err
	}
}
func getSAFromPreset(ctx context.Context,
	userInfoGetter provider.UserInfoGetter,
	presetProvider provider.PresetProvider,
	presetName string,
) (string, error) {
	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return "", common.KubernetesErrorToHTTPError(err)
	}
	preset, err := presetProvider.GetPreset(ctx, userInfo, presetName)
	if err != nil {
		return "", utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", presetName, userInfo.Email))
	}
	credentials := preset.Spec.GKE
	if credentials == nil {
		return "", fmt.Errorf("gke credentials not present in the preset %s", presetName)
	}
	return credentials.ServiceAccount, nil
}

func ListGKEVersions(ctx context.Context, sa, zone, mode, releaseChannel string) ([]*apiv1.MasterVersion, error) {
	svc, project, err := gkeprovider.ConnectToContainerService(ctx, sa)
	if err != nil {
		return nil, err
	}

	req := svc.Projects.Zones.GetServerconfig(project, zone)
	resp, err := req.Context(ctx).Do()
	if err != nil {
		return nil, gkeprovider.DecodeError(err)
	}

	versions := make([]*apiv1.MasterVersion, 0)
	if mode == ManualMode {
		for _, v := range resp.ValidMasterVersions {
			validVersion, err := semverlib.NewVersion(v)
			if err != nil {
				return nil, err
			}
			versions = append(versions, &apiv1.MasterVersion{
				Version: validVersion,
				Default: v == resp.DefaultClusterVersion,
			})
		}
	} else if mode == AutoMode {
		for _, channel := range resp.Channels {
			// select versions from the current channel
			if channel.Channel == releaseChannel {
				for _, v := range channel.ValidVersions {
					validVersion, err := semverlib.NewVersion(v)
					if err != nil {
						return nil, err
					}
					versions = append(versions, &apiv1.MasterVersion{
						Version: validVersion,
						Default: v == channel.DefaultVersion,
					})
				}
				break
			}
		}
	}

	return versions, err
}

func GKEVMSizesEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(GKEVMReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		sa := req.ServiceAccount
		var err error
		if len(req.Credential) > 0 {
			sa, err = getSAFromPreset(ctx, userInfoGetter, presetProvider, req.Credential)
			if err != nil {
				return nil, err
			}
		}
		return gkeprovider.ListGKESizes(ctx, sa, req.Zone)
	}
}

func GKEDiskTypesEndpoint(presetProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(GKEVMReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		if err := req.Validate(); err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		sa := req.ServiceAccount
		var err error
		if len(req.Credential) > 0 {
			sa, err = getSAFromPreset(ctx, userInfoGetter, presetProvider, req.Credential)
			if err != nil {
				return nil, err
			}
		}
		return gkeprovider.ListGKEDiskTypes(ctx, sa, req.Zone)
	}
}
