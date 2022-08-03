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

package gke

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	semverlib "github.com/Masterminds/semver/v3"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	googleapi "google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gcp"
	"k8c.io/kubermatic/v2/pkg/resources"
	ksemver "k8c.io/kubermatic/v2/pkg/semver"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/clientcmd/api"
)

const allZones = "-"

func GetClusterConfig(ctx context.Context, sa, clusterName, zone string) (*api.Config, error) {
	svc, project, err := ConnectToContainerService(ctx, sa)
	if err != nil {
		return nil, err
	}
	req := svc.Projects.Zones.Clusters.Get(project, zone, clusterName)
	resp, err := req.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("cannot get cluster for project=%s: %w", project, DecodeError(err))
	}
	config := api.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters:   map[string]*api.Cluster{},  // Clusters is a map of referencable names to cluster configs
		AuthInfos:  map[string]*api.AuthInfo{}, // AuthInfos is a map of referencable names to user configs
		Contexts:   map[string]*api.Context{},  // Contexts is a map of referencable names to context configs
	}

	cred, err := getCredentials(ctx, sa)
	if err != nil {
		return nil, fmt.Errorf("can't get credentials %w", err)
	}
	token, err := cred.TokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("can't get token %w", err)
	}
	name := fmt.Sprintf("gke_%s_%s_%s", project, resp.Zone, resp.Name)
	cert, err := base64.StdEncoding.DecodeString(resp.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return nil, fmt.Errorf("invalid certificate cluster=%s cert=%s: %w", name, resp.MasterAuth.ClusterCaCertificate, err)
	}
	// example: gke_my-project_us-central1-b_cluster-1 => https://XX.XX.XX.XX
	config.Clusters[name] = &api.Cluster{
		CertificateAuthorityData: cert,
		Server:                   "https://" + resp.Endpoint,
	}
	config.CurrentContext = name
	// Just reuse the context name as an auth name.
	config.Contexts[name] = &api.Context{
		Cluster:  name,
		AuthInfo: name,
	}
	// GCP specific configation; use cloud platform scope.
	config.AuthInfos[name] = &api.AuthInfo{
		Token: token.AccessToken,
	}
	return &config, nil
}

// ConnectToContainerService establishes a service connection to the Container Engine.
func ConnectToContainerService(ctx context.Context, serviceAccount string) (*container.Service, string, error) {
	client, projectID, err := createClient(ctx, serviceAccount, container.CloudPlatformScope)
	if err != nil {
		return nil, "", fmt.Errorf("cannot create Google Cloud client: %w", err)
	}
	svc, err := container.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, "", fmt.Errorf("cannot connect to Google Cloud: %w", err)
	}
	return svc, projectID, nil
}

func GetClusterStatus(ctx context.Context, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticv1.ExternalClusterCloudSpec) (*apiv2.ExternalClusterStatus, error) {
	sa, err := secretKeySelector(cloudSpec.GKE.CredentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := ConnectToContainerService(ctx, sa)
	if err != nil {
		return nil, err
	}

	req := svc.Projects.Zones.Clusters.Get(project, cloudSpec.GKE.Zone, cloudSpec.GKE.Name)
	gkeCluster, err := req.Context(ctx).Do()
	if err != nil {
		return nil, DecodeError(err)
	}

	return &apiv2.ExternalClusterStatus{
		State:         ConvertStatus(gkeCluster.Status),
		StatusMessage: GetStatusMessage(gkeCluster),
	}, nil
}

func ListClusters(ctx context.Context, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ExternalClusterProvider, projectID, sa string) (apiv2.GKEClusterList, error) {
	clusters := apiv2.GKEClusterList{}

	project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, projectID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	clusterList, err := clusterProvider.List(ctx, project)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	gkeExternalClusterNames := sets.NewString()
	for _, externalCluster := range clusterList.Items {
		cloud := externalCluster.Spec.CloudSpec
		if cloud != nil && cloud.GKE != nil {
			gkeExternalClusterNames.Insert(cloud.GKE.Name)
		}
	}

	gkeExternalCluster := make(map[string]sets.String)
	for _, externalCluster := range clusterList.Items {
		cloud := externalCluster.Spec.CloudSpec
		if cloud != nil && cloud.GKE != nil {
			zone := cloud.GKE.Zone
			if _, ok := gkeExternalCluster[zone]; !ok {
				gkeExternalCluster[zone] = make(sets.String)
			}
			gkeExternalCluster[zone] = gkeExternalCluster[zone].Insert(cloud.GKE.Name)
		}
	}

	svc, gkeProject, err := ConnectToContainerService(ctx, sa)
	if err != nil {
		return clusters, err
	}

	req := svc.Projects.Zones.Clusters.List(gkeProject, allZones)
	resp, err := req.Context(ctx).Do()
	if err != nil {
		return clusters, fmt.Errorf("clusters list project=%v: %w", project, DecodeError(err))
	}
	for _, f := range resp.Clusters {
		var imported bool
		if clusterSet, ok := gkeExternalCluster[f.Zone]; ok {
			if clusterSet.Has(f.Name) {
				imported = true
			}
		}
		clusters = append(clusters, apiv2.GKECluster{Name: f.Name, Zone: f.Zone, IsImported: imported})
	}
	return clusters, nil
}

func ListUpgrades(ctx context.Context, sa, zone, name string) ([]*apiv1.MasterVersion, error) {
	upgrades := make([]*apiv1.MasterVersion, 0)
	svc, project, err := ConnectToContainerService(ctx, sa)
	if err != nil {
		return nil, err
	}

	clusterReq := svc.Projects.Zones.Clusters.Get(project, zone, name)
	cluster, err := clusterReq.Context(ctx).Do()
	if err != nil {
		return nil, DecodeError(err)
	}

	currentClusterVer, err := semverlib.NewVersion(cluster.CurrentMasterVersion)
	if err != nil {
		return nil, err
	}
	releaseChannel := ""
	if cluster.ReleaseChannel != nil {
		releaseChannel = cluster.ReleaseChannel.Channel
	}

	req := svc.Projects.Zones.GetServerconfig(project, zone)
	resp, err := req.Context(ctx).Do()
	if err != nil {
		return nil, DecodeError(err)
	}
	upgradesMap := map[string]bool{}
	for _, channel := range resp.Channels {
		// select versions from the current channel
		if releaseChannel == channel.Channel {
			for _, v := range channel.ValidVersions {
				validVersion, err := semverlib.NewVersion(v)
				if err != nil {
					return nil, err
				}
				// select the correct version from the channel
				if isValidVersion(currentClusterVer, validVersion) {
					upgradesMap[v] = v == channel.DefaultVersion
				}
			}
		}
	}
	for version, isDefault := range upgradesMap {
		v, err := ksemver.NewSemver(version)
		if err != nil {
			return nil, err
		}
		upgrades = append(upgrades, &apiv1.MasterVersion{
			Version: v.Semver(),
			Default: isDefault,
		})
	}

	return upgrades, nil
}

func ListMachineDeploymentUpgrades(ctx context.Context, sa, zone, clusterName, machineDeployment string) ([]*apiv1.MasterVersion, error) {
	upgrades := make([]*apiv1.MasterVersion, 0)
	svc, project, err := ConnectToContainerService(ctx, sa)
	if err != nil {
		return nil, err
	}

	clusterReq := svc.Projects.Zones.Clusters.Get(project, zone, clusterName)
	cluster, err := clusterReq.Context(ctx).Do()
	if err != nil {
		return nil, DecodeError(err)
	}

	currentClusterVer, err := semverlib.NewVersion(cluster.CurrentMasterVersion)
	if err != nil {
		return nil, err
	}

	req := svc.Projects.Zones.Clusters.NodePools.Get(project, zone, clusterName, machineDeployment)
	np, err := req.Context(ctx).Do()
	if err != nil {
		return nil, DecodeError(err)
	}

	currentMachineDeploymentVer, err := semverlib.NewVersion(np.Version)
	if err != nil {
		return nil, err
	}

	// return control plane version
	if currentClusterVer.GreaterThan(currentMachineDeploymentVer) {
		upgrades = append(upgrades, &apiv1.MasterVersion{Version: currentClusterVer})
	}

	return upgrades, nil
}

func isValidVersion(currentVersion, newVersion *semverlib.Version) bool {
	// new version has to be different from current version
	if currentVersion.Equal(newVersion) {
		return false
	}
	// major version can not be changed
	if currentVersion.Major() != newVersion.Major() {
		return false
	}
	// only one minor version can be updated at once
	if (currentVersion.Minor() + 1) < newVersion.Minor() {
		return false
	}
	// new version can not be lower than current one
	if currentVersion.Minor() > newVersion.Minor() || (currentVersion.Minor() == newVersion.Minor() && currentVersion.Patch() > newVersion.Patch()) {
		return false
	}

	return true
}

func ListImages(ctx context.Context, sa, zone string) (apiv2.GKEImageList, error) {
	images := apiv2.GKEImageList{}
	svc, project, err := ConnectToContainerService(ctx, sa)
	if err != nil {
		return nil, err
	}

	config, err := svc.Projects.Zones.GetServerconfig(project, zone).Context(ctx).Do()
	if err != nil {
		return nil, DecodeError(err)
	}

	for _, imageType := range config.ValidImageTypes {
		images = append(images, apiv2.GKEImage{
			Name:      imageType,
			IsDefault: imageType == config.DefaultImageType,
		})
	}

	return images, nil
}

func ListZones(ctx context.Context, sa string) (apiv2.GKEZoneList, error) {
	computeService, gcpProject, err := gcp.ConnectToComputeService(ctx, sa)
	if err != nil {
		return nil, err
	}

	zones := apiv2.GKEZoneList{}
	zoneReq := computeService.Zones.List(gcpProject)
	err = zoneReq.Pages(ctx, func(page *compute.ZoneList) error {
		for _, zone := range page.Items {
			zones = append(zones, apiv2.GKEZone{Name: zone.Name})
		}
		return nil
	})

	return zones, err
}

func ValidateCredentials(ctx context.Context, sa string) error {
	svc, project, err := ConnectToContainerService(ctx, sa)
	if err != nil {
		return err
	}
	_, err = svc.Projects.Zones.Clusters.List(project, allZones).Context(ctx).Do()

	return DecodeError(err)
}

func ConvertStatus(status string) apiv2.ExternalClusterState {
	switch status {
	// The PROVISIONING state indicates the cluster is being created.
	case string(resources.ProvisioningGKEState):
		return apiv2.ProvisioningExternalClusterState
	// The RUNNING state indicates the cluster has been created and is fully usable.
	case string(resources.RunningGKEState):
		return apiv2.RunningExternalClusterState
	// The RECONCILING state indicates that some work is
	// actively being done on the cluster, such as upgrading the master or
	// node software.
	case string(resources.ReconcilingGKEState):
		return apiv2.ReconcilingExternalClusterState
	// The STOPPING state indicates the cluster is being deleted.
	case string(resources.StoppingGKEState):
		return apiv2.DeletingExternalClusterState
	// The ERROR state indicates the cluster is unusable. It
	// will be automatically deleted.
	case string(resources.ErrorGKEState):
		return apiv2.ErrorExternalClusterState
	// The DEGRADED state indicates the cluster requires user
	// action to restore full functionality.
	case string(resources.DegradedGKEState):
		return apiv2.ErrorExternalClusterState
	// "STATUS_UNSPECIFIED" - Not set.
	case string(resources.UnspecifiedGKEState):
		return apiv2.UnknownExternalClusterState
	default:
		return apiv2.UnknownExternalClusterState
	}
}

func GetMDStatusMessage(np *container.NodePool) string {
	var statusMessage string
	if np == nil {
		return statusMessage
	}
	statusMessage = np.StatusMessage
	if statusMessage == "" {
		if np.Conditions != nil && len(np.Conditions) > 0 {
			statusMessage = np.Conditions[1].Message
		}
	}
	return statusMessage
}

func ConvertMDStatus(status string) apiv2.ExternalClusterMDState {
	switch status {
	// The PROVISIONING state indicates the node pool is being created.
	case string(resources.ProvisioningGKEMDState):
		return apiv2.ProvisioningExternalClusterMDState
	// The RUNNING state indicates the node pool has been
	// created and is fully usable.
	case string(resources.RunningGKEMDState):
		return apiv2.RunningExternalClusterMDState
	// "RECONCILING" - The RECONCILING state indicates that some work is
	// actively being done on the node pool, such as upgrading node
	// software.
	case string(resources.ReconcilingGKEMDState):
		return apiv2.ReconcilingExternalClusterMDState
	// "STOPPING" - The STOPPING state indicates the node pool is being deleted.
	case string(resources.StoppingGKEMDState):
		return apiv2.DeletingExternalClusterMDState
	// The ERROR state indicates the node pool may be unusable.
	case string(resources.ErrorGKEMDState):
		return apiv2.ErrorExternalClusterMDState
	// The RUNNING_WITH_ERROR state indicates the
	// node pool has been created and is partially usable. Some error state
	// has occurred and some functionality may be impaired.
	case string(resources.RunningWithErrorGKEMDState):
		return apiv2.ErrorExternalClusterMDState
	// "STATUS_UNSPECIFIED" - Not set.
	case string(resources.UnspecifiedGKEMDState):
		return apiv2.UnknownExternalClusterMDState
	default:
		return apiv2.UnknownExternalClusterMDState
	}
}

func GetStatusMessage(gkeCluster *container.Cluster) string {
	var statusMessage string
	if gkeCluster == nil {
		return statusMessage
	}
	statusMessage = gkeCluster.StatusMessage
	if statusMessage == "" {
		if gkeCluster.Conditions != nil && len(gkeCluster.Conditions) > 0 {
			statusMessage = gkeCluster.Conditions[1].Message
		}
	}
	return statusMessage
}

func getCredentials(ctx context.Context, serviceAccount string) (*google.Credentials, error) {
	b, err := base64.StdEncoding.DecodeString(serviceAccount)
	if err != nil {
		return nil, fmt.Errorf("error decoding service account: %w", err)
	}
	sam := map[string]string{}
	err = json.Unmarshal(b, &sam)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshaling service account: %w", err)
	}
	return google.CredentialsFromJSON(ctx, b, container.CloudPlatformScope)
}

func createClient(ctx context.Context, serviceAccount string, scope string) (*http.Client, string, error) {
	b, err := base64.StdEncoding.DecodeString(serviceAccount)
	if err != nil {
		return nil, "", fmt.Errorf("error decoding service account: %w", err)
	}
	sam := map[string]string{}
	err = json.Unmarshal(b, &sam)
	if err != nil {
		return nil, "", fmt.Errorf("failed unmarshaling service account: %w", err)
	}

	projectID := sam["project_id"]
	if projectID == "" {
		return nil, "", errors.New("empty project_id")
	}
	conf, err := google.JWTConfigFromJSON(b, scope)
	if err != nil {
		return nil, "", err
	}

	client := conf.Client(ctx)

	return client, projectID, nil
}

func DecodeError(err error) error {
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return utilerrors.New(apiErr.Code, apiErr.Message)
	}

	return err
}
