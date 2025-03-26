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

	apiv1 "k8c.io/kubermatic/sdk/v2/api/v1"
	apiv2 "k8c.io/kubermatic/sdk/v2/api/v2"
	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	ksemver "k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gcp"
	"k8c.io/kubermatic/v2/pkg/resources"
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
		Clusters:   map[string]*api.Cluster{},  // Clusters is a map of referenceable names to cluster configs
		AuthInfos:  map[string]*api.AuthInfo{}, // AuthInfos is a map of referenceable names to user configs
		Contexts:   map[string]*api.Context{},  // Contexts is a map of referenceable names to context configs
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
	// GCP specific configuration; use cloud platform scope.
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

func GetClusterStatus(ctx context.Context, secretKeySelector provider.SecretKeySelectorValueFunc, cloudSpec *kubermaticv1.ExternalClusterGKECloudSpec) (*kubermaticv1.ExternalClusterCondition, error) {
	sa, err := secretKeySelector(cloudSpec.CredentialsReference, resources.GCPServiceAccount)
	if err != nil {
		return nil, err
	}
	svc, project, err := ConnectToContainerService(ctx, sa)
	if err != nil {
		return nil, err
	}

	req := svc.Projects.Zones.Clusters.Get(project, cloudSpec.Zone, cloudSpec.Name)
	gkeCluster, err := req.Context(ctx).Do()
	if err != nil {
		return nil, DecodeError(err)
	}

	return &kubermaticv1.ExternalClusterCondition{
		Phase:   ConvertStatus(gkeCluster.Status),
		Message: GetStatusMessage(gkeCluster),
	}, nil
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

	req := svc.Projects.Zones.GetServerconfig(project, zone)
	resp, err := req.Context(ctx).Do()
	if err != nil {
		return nil, DecodeError(err)
	}
	upgradesMap := map[string]bool{}

	if cluster.ReleaseChannel != nil && len(cluster.ReleaseChannel.Channel) > 0 && cluster.ReleaseChannel.Channel != resources.GKEUnspecifiedReleaseChannel {
		releaseChannel := cluster.ReleaseChannel.Channel
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
	} else {
		for _, v := range resp.ValidMasterVersions {
			validVersion, err := semverlib.NewVersion(v)
			if err != nil {
				return nil, err
			}
			if isValidVersion(currentClusterVer, validVersion) {
				upgradesMap[v] = v == resp.DefaultClusterVersion
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
		return DecodeError(err)
	}
	_, err = svc.Projects.Zones.Clusters.List(project, allZones).Context(ctx).Do()

	return DecodeError(err)
}

func ConvertStatus(status string) kubermaticv1.ExternalClusterPhase {
	switch status {
	// The PROVISIONING state indicates the cluster is being created.
	case string(resources.ProvisioningGKEState):
		return kubermaticv1.ExternalClusterPhaseProvisioning
	// The RUNNING state indicates the cluster has been created and is fully usable.
	case string(resources.RunningGKEState):
		return kubermaticv1.ExternalClusterPhaseRunning
	// The RECONCILING state indicates that some work is
	// actively being done on the cluster, such as upgrading the master or
	// node software.
	case string(resources.ReconcilingGKEState):
		return kubermaticv1.ExternalClusterPhaseReconciling
	// The STOPPING state indicates the cluster is being deleted.
	case string(resources.StoppingGKEState):
		return kubermaticv1.ExternalClusterPhaseDeleting
	// The ERROR state indicates the cluster is unusable. It
	// will be automatically deleted.
	case string(resources.ErrorGKEState):
		return kubermaticv1.ExternalClusterPhaseError
	// The DEGRADED state indicates the cluster requires user
	// action to restore full functionality.
	case string(resources.DegradedGKEState):
		return kubermaticv1.ExternalClusterPhaseError
	// "STATUS_UNSPECIFIED" - Not set.
	case string(resources.UnspecifiedGKEState):
		return kubermaticv1.ExternalClusterPhaseUnknown
	default:
		return kubermaticv1.ExternalClusterPhaseUnknown
	}
}

func GetMDStatusMessage(np *container.NodePool) string {
	var statusMessage string
	if np == nil {
		return statusMessage
	}
	statusMessage = np.StatusMessage
	if statusMessage == "" {
		if len(np.Conditions) > 1 {
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
		if len(gkeCluster.Conditions) > 1 {
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
		return nil, fmt.Errorf("failed unmarshalling service account: %w", err)
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
		return nil, "", fmt.Errorf("failed unmarshalling service account: %w", err)
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

func ListGKESizes(ctx context.Context, sa, zone string) (apiv1.GCPMachineSizeList, error) {
	sizes := apiv1.GCPMachineSizeList{}

	computeService, project, err := gcp.ConnectToComputeService(ctx, sa)
	if err != nil {
		return sizes, err
	}

	req := computeService.MachineTypes.List(project, zone)
	err = req.Pages(ctx, func(page *compute.MachineTypeList) error {
		for _, machineType := range page.Items {
			mt := apiv1.GCPMachineSize{
				Name:        machineType.Name,
				Description: machineType.Description,
				Memory:      machineType.MemoryMb,
				VCPUs:       machineType.GuestCpus,
			}
			sizes = append(sizes, mt)
		}
		return nil
	})

	return sizes, err
}

func ListGKEDiskTypes(ctx context.Context, sa string, zone string) (apiv2.GKEDiskTypeList, error) {
	diskTypes := apiv2.GKEDiskTypeList{}
	// Currently accepted values: 'pd-standard', 'pd-ssd' or 'pd-balanced'
	// Reference: https://pkg.go.dev/google.golang.org/api/container/v1#NodeConfig

	excludedDiskTypes := sets.New("local-ssd", "pd-extreme")
	computeService, project, err := gcp.ConnectToComputeService(ctx, sa)
	if err != nil {
		return diskTypes, err
	}

	req := computeService.DiskTypes.List(project, zone)
	err = req.Pages(ctx, func(page *compute.DiskTypeList) error {
		for _, diskType := range page.Items {
			if !excludedDiskTypes.Has(diskType.Name) {
				dt := apiv2.GKEDiskType{
					Name:              diskType.Name,
					Description:       diskType.Description,
					DefaultDiskSizeGb: diskType.DefaultDiskSizeGb,
					Kind:              diskType.Kind,
				}
				diskTypes = append(diskTypes, dt)
			}
		}
		return nil
	})

	return diskTypes, err
}

func DecodeError(err error) error {
	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return utilerrors.New(apiErr.Code, apiErr.Message)
	}

	return err
}
