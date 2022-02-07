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

	"github.com/go-kit/kit/endpoint"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	providercommon "k8c.io/kubermatic/v2/pkg/handler/common/provider"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gcp"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gke"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/util/errors"

	"k8s.io/apimachinery/pkg/util/sets"
)

const GKENodepoolNameLabel = "cloud.google.com/gke-nodepool"

func GKEImagesWithClusterCredentialsEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(settingsProvider) {
			return nil, errors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(getClusterReq)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest(err.Error())
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
		svc, gcpProject, err := gke.ConnectToContainerService(sa)
		if err != nil {
			return nil, err
		}

		config, err := svc.Projects.Zones.GetServerconfig(gcpProject, cluster.Spec.CloudSpec.GKE.Zone).Context(ctx).Do()
		if err != nil {
			return nil, err
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
		if !AreExternalClustersEnabled(settingsProvider) {
			return nil, errors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(getClusterReq)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest(err.Error())
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
		computeService, gcpProject, err := gcp.ConnectToComputeService(sa)
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
		if !AreExternalClustersEnabled(settingsProvider) {
			return nil, errors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(getClusterReq)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest(err.Error())
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

		settings, err := settingsProvider.GetGlobalSettings()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return providercommon.ListGCPSizes(ctx, settings.Spec.MachineDeploymentVMResourceQuota, sa, cluster.Spec.CloudSpec.GKE.Zone)
	}
}

func GKEDiskTypesWithClusterCredentialsEndpoint(userInfoGetter provider.UserInfoGetter, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider, settingsProvider provider.SettingsProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		if !AreExternalClustersEnabled(settingsProvider) {
			return nil, errors.New(http.StatusForbidden, "external cluster functionality is disabled")
		}

		req := request.(getClusterReq)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest(err.Error())
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

		return listGKEDiskTypes(ctx, sa, cluster.Spec.CloudSpec.GKE.Zone)
	}
}

func listGKEDiskTypes(ctx context.Context, sa string, zone string) (apiv1.GCPDiskTypeList, error) {
	diskTypes := apiv1.GCPDiskTypeList{}
	// TODO: There are some issues at the moment with local-ssd and pd-extreme, that's why it is disabled at the moment.
	excludedDiskTypes := sets.NewString("local-ssd", "pd-extreme")
	computeService, project, err := gcp.ConnectToComputeService(sa)
	if err != nil {
		return diskTypes, err
	}

	req := computeService.DiskTypes.List(project, zone)
	err = req.Pages(ctx, func(page *compute.DiskTypeList) error {
		for _, diskType := range page.Items {
			if !excludedDiskTypes.Has(diskType.Name) {
				dt := apiv1.GCPDiskType{
					Name:        diskType.Name,
					Description: diskType.Description,
				}
				diskTypes = append(diskTypes, dt)
			}
		}
		return nil
	})

	return diskTypes, err
}

func createOrImportGKECluster(ctx context.Context, name string, userInfoGetter provider.UserInfoGetter, project *kubermaticv1.Project, cloud *apiv2.ExternalClusterCloudSpec, clusterProvider provider.ExternalClusterProvider, privilegedClusterProvider provider.PrivilegedExternalClusterProvider) (*kubermaticv1.ExternalCluster, error) {
	if cloud.GKE.Name == "" || cloud.GKE.Zone == "" || cloud.GKE.ServiceAccount == "" {
		return nil, errors.NewBadRequest("the GKE cluster name, zone or service account can not be empty")
	}

	if cloud.GKE.ClusterSpec != nil {
		if err := createNewGKECluster(ctx, cloud.GKE); err != nil {
			return nil, err
		}
	}

	newCluster := genExternalCluster(name, project.Name)
	newCluster.Spec.CloudSpec = &kubermaticv1.ExternalClusterCloudSpec{
		GKE: &kubermaticv1.ExternalClusterGKECloudSpec{
			Name: cloud.GKE.Name,
			Zone: cloud.GKE.Zone,
		},
	}
	keyRef, err := clusterProvider.CreateOrUpdateCredentialSecretForCluster(ctx, cloud, project.Name, newCluster.Name)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	kuberneteshelper.AddFinalizer(newCluster, apiv1.CredentialsSecretsCleanupFinalizer)
	newCluster.Spec.CloudSpec.GKE.CredentialsReference = keyRef

	return createNewCluster(ctx, userInfoGetter, clusterProvider, privilegedClusterProvider, newCluster, project)
}

func patchGKECluster(ctx context.Context, oldCluster, newCluster *apiv2.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector) (*apiv2.ExternalCluster, error) {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := gke.ConnectToContainerService(sa)
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

	return newCluster, err
}

func getGKENodePools(ctx context.Context, cluster *kubermaticv1.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterMachineDeployment, error) {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := gke.ConnectToContainerService(sa)
	if err != nil {
		return nil, err
	}

	req := svc.Projects.Zones.Clusters.NodePools.List(project, cluster.Spec.CloudSpec.GKE.Zone, cluster.Spec.CloudSpec.GKE.Name)
	resp, err := req.Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	machineDeployments := make([]apiv2.ExternalClusterMachineDeployment, 0, len(resp.NodePools))

	nodes, err := clusterProvider.ListNodes(cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	for _, md := range resp.NodePools {
		var readyReplicas int32
		for _, n := range nodes.Items {
			if n.Labels != nil {
				if n.Labels[GKENodepoolNameLabel] == md.Name {
					readyReplicas++
				}
			}
		}

		machineDeployments = append(machineDeployments, createMachineDeploymentFromGKENodePoll(md, readyReplicas))
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
	return md
}

func getGKENodePool(ctx context.Context, cluster *kubermaticv1.ExternalCluster, nodeGroupName string, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) (*apiv2.ExternalClusterMachineDeployment, error) {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := gke.ConnectToContainerService(sa)
	if err != nil {
		return nil, err
	}

	return getGKEMachineDeployment(ctx, svc, project, cluster, nodeGroupName, clusterProvider)
}

func getGKENodes(ctx context.Context, cluster *kubermaticv1.ExternalCluster, nodePoolID string, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) ([]apiv2.ExternalClusterNode, error) {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := gke.ConnectToContainerService(sa)
	if err != nil {
		return nil, err
	}

	req := svc.Projects.Zones.Clusters.NodePools.Get(project, cluster.Spec.CloudSpec.GKE.Zone, cluster.Spec.CloudSpec.GKE.Name, nodePoolID)
	resp, err := req.Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	var nodesV1 []apiv2.ExternalClusterNode

	nodes, err := clusterProvider.ListNodes(cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	for _, n := range nodes.Items {
		if n.Labels != nil {
			if n.Labels[GKENodepoolNameLabel] == resp.Name {
				outNode, err := outputNode(n)
				if err != nil {
					return nil, fmt.Errorf("failed to output node %s: %w", n.Name, err)
				}
				nodesV1 = append(nodesV1, *outNode)
			}
		}
	}

	return nodesV1, err
}

func deleteGKENodePool(ctx context.Context, cluster *kubermaticv1.ExternalCluster, nodePoolID string, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector, clusterProvider provider.ExternalClusterProvider) error {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return err
	}
	svc, project, err := gke.ConnectToContainerService(sa)
	if err != nil {
		return err
	}

	req := svc.Projects.Zones.Clusters.NodePools.Delete(project, cluster.Spec.CloudSpec.GKE.Zone, cluster.Spec.CloudSpec.GKE.Name, nodePoolID)
	_, err = req.Context(ctx).Do()
	return err
}

func patchGKEMachineDeployment(ctx context.Context, oldMD, newMD *apiv2.ExternalClusterMachineDeployment, cluster *kubermaticv1.ExternalCluster, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector) (*apiv2.ExternalClusterMachineDeployment, error) {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := gke.ConnectToContainerService(sa)
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
			return nil, err
		}
		return newMD, nil
	}

	updateRequest := &container.UpdateNodePoolRequest{
		NodeVersion: newMD.Spec.Template.Versions.Kubelet,
	}
	updateReq := svc.Projects.Zones.Clusters.NodePools.Update(project, cluster.Spec.CloudSpec.GKE.Zone, cluster.Spec.CloudSpec.GKE.Name, oldMD.Name, updateRequest)
	_, err = updateReq.Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	return newMD, nil
}

func getGKEMachineDeployment(ctx context.Context, svc *container.Service, projectID string, cluster *kubermaticv1.ExternalCluster, nodeGroupName string, clusterProvider provider.ExternalClusterProvider) (*apiv2.ExternalClusterMachineDeployment, error) {
	req := svc.Projects.Zones.Clusters.NodePools.Get(projectID, cluster.Spec.CloudSpec.GKE.Zone, cluster.Spec.CloudSpec.GKE.Name, nodeGroupName)
	np, err := req.Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	nodes, err := clusterProvider.ListNodes(cluster)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	var readyReplicas int32
	for _, n := range nodes.Items {
		if n.Labels != nil {
			if n.Labels[GKENodepoolNameLabel] == np.Name {
				readyReplicas++
			}
		}
	}
	md := createMachineDeploymentFromGKENodePoll(np, readyReplicas)
	return &md, nil
}

func createGKENodePool(ctx context.Context, cluster *kubermaticv1.ExternalCluster, machineDeployment apiv2.ExternalClusterMachineDeployment, secretKeySelector provider.SecretKeySelectorValueFunc, credentialsReference *providerconfig.GlobalSecretKeySelector) (*apiv2.ExternalClusterMachineDeployment, error) {
	sa, err := secretKeySelector(credentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := gke.ConnectToContainerService(sa)
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
	req := svc.Projects.Zones.Clusters.NodePools.Create(project, cluster.Spec.CloudSpec.GKE.Zone, cluster.Spec.CloudSpec.GKE.Name, createRequest)
	_, err = req.Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return &machineDeployment, nil
}

func createNewGKECluster(ctx context.Context, gkeCloudSpec *apiv2.GKECloudSpec) error {
	svc, project, err := gke.ConnectToContainerService(gkeCloudSpec.ServiceAccount)
	if err != nil {
		return err
	}

	createRequest := &container.CreateClusterRequest{
		Cluster: genGKECluster(gkeCloudSpec),
	}

	req := svc.Projects.Zones.Clusters.Create(project, gkeCloudSpec.Zone, createRequest)
	_, err = req.Context(ctx).Do()
	if err != nil {
		return err
	}

	return nil
}

func genGKECluster(gkeCloudSpec *apiv2.GKECloudSpec) *container.Cluster {
	newCluster := &container.Cluster{
		ClusterIpv4Cidr:       gkeCloudSpec.ClusterSpec.ClusterIpv4Cidr,
		InitialClusterVersion: gkeCloudSpec.ClusterSpec.InitialClusterVersion,
		InitialNodeCount:      gkeCloudSpec.ClusterSpec.InitialNodeCount,
		Locations:             gkeCloudSpec.ClusterSpec.Locations,
		Name:                  gkeCloudSpec.Name,
		Network:               gkeCloudSpec.ClusterSpec.Network,
		Subnetwork:            gkeCloudSpec.ClusterSpec.Subnetwork,
		TpuIpv4CidrBlock:      gkeCloudSpec.ClusterSpec.TpuIpv4CidrBlock,
		EnableKubernetesAlpha: gkeCloudSpec.ClusterSpec.EnableKubernetesAlpha,
		EnableTpu:             gkeCloudSpec.ClusterSpec.EnableTpu,
		Autopilot: &container.Autopilot{
			Enabled: gkeCloudSpec.ClusterSpec.Autopilot,
		},
		VerticalPodAutoscaling: &container.VerticalPodAutoscaling{
			Enabled: gkeCloudSpec.ClusterSpec.VerticalPodAutoscaling,
		},
	}
	if gkeCloudSpec.ClusterSpec.NodeConfig != nil {
		newCluster.NodeConfig = &container.NodeConfig{
			DiskSizeGb:    gkeCloudSpec.ClusterSpec.NodeConfig.DiskSizeGb,
			DiskType:      gkeCloudSpec.ClusterSpec.NodeConfig.DiskType,
			ImageType:     gkeCloudSpec.ClusterSpec.NodeConfig.ImageType,
			Labels:        gkeCloudSpec.ClusterSpec.NodeConfig.Labels,
			LocalSsdCount: gkeCloudSpec.ClusterSpec.NodeConfig.LocalSsdCount,
			MachineType:   gkeCloudSpec.ClusterSpec.NodeConfig.MachineType,
			Preemptible:   gkeCloudSpec.ClusterSpec.NodeConfig.Preemptible,
		}
	}
	if gkeCloudSpec.ClusterSpec.Autoscaling != nil {
		newCluster.Autoscaling = &container.ClusterAutoscaling{
			AutoprovisioningLocations:  gkeCloudSpec.ClusterSpec.Autoscaling.AutoprovisioningLocations,
			EnableNodeAutoprovisioning: gkeCloudSpec.ClusterSpec.Autoscaling.EnableNodeAutoprovisioning,
		}
		if gkeCloudSpec.ClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults != nil {
			newCluster.Autoscaling.AutoprovisioningNodePoolDefaults = &container.AutoprovisioningNodePoolDefaults{
				BootDiskKmsKey: gkeCloudSpec.ClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.BootDiskKmsKey,
				DiskSizeGb:     gkeCloudSpec.ClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.DiskSizeGb,
				DiskType:       gkeCloudSpec.ClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.DiskType,
				MinCpuPlatform: gkeCloudSpec.ClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.MinCpuPlatform,
				OauthScopes:    gkeCloudSpec.ClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.OauthScopes,
				ServiceAccount: gkeCloudSpec.ClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.ServiceAccount,
			}
			if gkeCloudSpec.ClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.Management != nil {
				newCluster.Autoscaling.AutoprovisioningNodePoolDefaults.Management = &container.NodeManagement{
					AutoRepair:  gkeCloudSpec.ClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.Management.AutoRepair,
					AutoUpgrade: gkeCloudSpec.ClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.Management.AutoUpgrade,
				}
			}
			if gkeCloudSpec.ClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.UpgradeSettings != nil {
				newCluster.Autoscaling.AutoprovisioningNodePoolDefaults.UpgradeSettings = &container.UpgradeSettings{
					MaxSurge:       gkeCloudSpec.ClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.UpgradeSettings.MaxSurge,
					MaxUnavailable: gkeCloudSpec.ClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.UpgradeSettings.MaxUnavailable,
				}
			}
			if gkeCloudSpec.ClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.ShieldedInstanceConfig != nil {
				newCluster.Autoscaling.AutoprovisioningNodePoolDefaults.ShieldedInstanceConfig = &container.ShieldedInstanceConfig{
					EnableIntegrityMonitoring: gkeCloudSpec.ClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.ShieldedInstanceConfig.EnableIntegrityMonitoring,
					EnableSecureBoot:          gkeCloudSpec.ClusterSpec.Autoscaling.AutoprovisioningNodePoolDefaults.ShieldedInstanceConfig.EnableSecureBoot,
				}
			}
		}
		if len(gkeCloudSpec.ClusterSpec.Autoscaling.ResourceLimits) != 0 {
			newCluster.Autoscaling.ResourceLimits = make([]*container.ResourceLimit, len(gkeCloudSpec.ClusterSpec.Autoscaling.ResourceLimits))
			for _, rl := range gkeCloudSpec.ClusterSpec.Autoscaling.ResourceLimits {
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
