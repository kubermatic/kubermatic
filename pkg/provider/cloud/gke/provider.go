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
	"google.golang.org/api/container/v1"
	"google.golang.org/api/option"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	ksemver "k8c.io/kubermatic/v2/pkg/semver"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/tools/clientcmd/api"
)

const allZones = "-"

func GetCLusterConfig(ctx context.Context, sa, clusterName, zone string) (*api.Config, error) {
	svc, project, err := ConnectToContainerService(sa)
	if err != nil {
		return nil, err
	}
	req := svc.Projects.Zones.Clusters.Get(project, zone, clusterName)
	resp, err := req.Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("cannot get cluster for project=%s: %w", project, err)
	}
	config := api.Config{
		APIVersion: "v1",
		Kind:       "Config",
		Clusters:   map[string]*api.Cluster{},  // Clusters is a map of referencable names to cluster configs
		AuthInfos:  map[string]*api.AuthInfo{}, // AuthInfos is a map of referencable names to user configs
		Contexts:   map[string]*api.Context{},  // Contexts is a map of referencable names to context configs
	}

	cred, err := getCredentials(sa)
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
func ConnectToContainerService(serviceAccount string) (*container.Service, string, error) {
	ctx := context.Background()
	client, projectID, err := createClient(ctx, serviceAccount, container.CloudPlatformScope)
	if err != nil {
		return nil, "", fmt.Errorf("cannot create Google Cloud client: %v", err)
	}
	svc, err := container.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, "", fmt.Errorf("cannot connect to Google Cloud: %v", err)
	}
	return svc, projectID, nil
}

func GetGKEClusterStatus(ctx context.Context, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticapiv1.ExternalClusterCloudSpec) (*apiv2.ExternalClusterStatus, error) {
	sa, err := secretKeySelector(cloudSpec.GKE.CredentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := ConnectToContainerService(sa)
	if err != nil {
		return nil, err
	}

	req := svc.Projects.Zones.Clusters.Get(project, cloudSpec.GKE.Zone, cloudSpec.GKE.Name)
	gkeCluster, err := req.Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	return &apiv2.ExternalClusterStatus{
		State:         convertGKEStatus(gkeCluster.Status),
		StatusMessage: gkeCluster.StatusMessage,
	}, nil

}

func ListGKEClusters(ctx context.Context, projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter, clusterProvider provider.ExternalClusterProvider, projectID, sa string) (apiv2.GKEClusterList, error) {
	clusters := apiv2.GKEClusterList{}

	project, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, projectID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	clusterList, err := clusterProvider.List(project)
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

	svc, gkeProject, err := ConnectToContainerService(sa)
	if err != nil {
		return clusters, err
	}

	req := svc.Projects.Zones.Clusters.List(gkeProject, allZones)
	resp, err := req.Context(ctx).Do()
	if err != nil {
		return clusters, fmt.Errorf("clusters list project=%v: %w", project, err)
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

func ListGKEUpgrades(ctx context.Context, sa, zone, name string) ([]*apiv1.MasterVersion, error) {
	upgrades := make([]*apiv1.MasterVersion, 0)
	svc, project, err := ConnectToContainerService(sa)
	if err != nil {
		return nil, err
	}

	clusterReq := svc.Projects.Zones.Clusters.Get(project, zone, name)
	cluster, err := clusterReq.Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	currentClusterVer, err := semverlib.NewVersion(cluster.CurrentMasterVersion)
	if err != nil {
		return nil, err
	}

	req := svc.Projects.Zones.GetServerconfig(project, zone)
	resp, err := req.Context(ctx).Do()
	if err != nil {
		return nil, err
	}
	upgradesMap := map[string]bool{}
	for _, channel := range resp.Channels {

		defaultChannelVersion, err := semverlib.NewVersion(channel.DefaultVersion)
		if err != nil {
			return nil, err
		}
		// select correct channel
		if isValidVersion(currentClusterVer, defaultChannelVersion) {
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

func ListGKEMachineDeploymentUpgrades(ctx context.Context, sa, zone, clusterName, machineDeployment string) ([]*apiv1.MasterVersion, error) {
	upgrades := make([]*apiv1.MasterVersion, 0)
	svc, project, err := ConnectToContainerService(sa)
	if err != nil {
		return nil, err
	}

	clusterReq := svc.Projects.Zones.Clusters.Get(project, zone, clusterName)
	cluster, err := clusterReq.Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	currentClusterVer, err := semverlib.NewVersion(cluster.CurrentMasterVersion)
	if err != nil {
		return nil, err
	}

	req := svc.Projects.Zones.Clusters.NodePools.Get(project, zone, clusterName, machineDeployment)
	np, err := req.Context(ctx).Do()
	if err != nil {
		return nil, err
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
	if currentVersion.Major() == newVersion.Major() && (currentVersion.Minor()+1) == newVersion.Minor() {
		return true
	}
	return false
}

func ListGKEImages(ctx context.Context, sa, zone string) (apiv2.GKEImageList, error) {
	images := apiv2.GKEImageList{}
	svc, project, err := ConnectToContainerService(sa)
	if err != nil {
		return nil, err
	}

	config, err := svc.Projects.Zones.GetServerconfig(project, zone).Context(ctx).Do()
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

func ValidateGKECredentials(ctx context.Context, sa string) error {
	svc, project, err := ConnectToContainerService(sa)
	if err != nil {
		return err
	}
	_, err = svc.Projects.Zones.Clusters.List(project, allZones).Context(ctx).Do()

	return err
}

func convertGKEStatus(status string) apiv2.ExternalClusterState {
	switch status {
	case "PROVISIONING":
		return apiv2.PROVISIONING
	case "RUNNING":
		return apiv2.RUNNING
	case "RECONCILING":
		return apiv2.RECONCILING
	case "STOPPING":
		return apiv2.DELETING
	case "ERROR":
		return apiv2.ERROR
	default:
		return apiv2.UNKNOWN
	}
}

func getCredentials(serviceAccount string) (*google.Credentials, error) {
	ctx := context.Background()
	b, err := base64.StdEncoding.DecodeString(serviceAccount)
	if err != nil {
		return nil, fmt.Errorf("error decoding service account: %v", err)
	}
	sam := map[string]string{}
	err = json.Unmarshal(b, &sam)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshaling service account: %v", err)
	}
	return google.CredentialsFromJSON(ctx, b, container.CloudPlatformScope)
}

func createClient(ctx context.Context, serviceAccount string, scope string) (*http.Client, string, error) {
	b, err := base64.StdEncoding.DecodeString(serviceAccount)
	if err != nil {
		return nil, "", fmt.Errorf("error decoding service account: %v", err)
	}
	sam := map[string]string{}
	err = json.Unmarshal(b, &sam)
	if err != nil {
		return nil, "", fmt.Errorf("failed unmarshaling service account: %v", err)
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
