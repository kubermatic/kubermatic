package provider

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"google.golang.org/api/compute/v1"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/dc"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/gcp"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// GCPZoneReq represent a request for GCP zones.
// swagger:parameters listGCPZones
type GCPZoneReq struct {
	GCPCommonReq
	// in: path
	// required: true
	DC string `json:"dc"`
}

// GCPTypesReq represent a request for GCP machine or disk types.
// swagger:parameters listGCPDiskTypes listGCPSizes
type GCPTypesReq struct {
	GCPCommonReq
	// in: header
	// name: Zone
	Zone string
}

// GCPCommonReq represent a request with common parameters for GCP.
type GCPCommonReq struct {
	// in: header
	// name: ServiceAccount
	ServiceAccount string
	// in: header
	// name: Credential
	Credential string
}

// GCPTypesNoCredentialReq represent a request for GCP machine or disk types.
// swagger:parameters listGCPSizesNoCredentials listGCPDiskTypesNoCredentials
type GCPTypesNoCredentialReq struct {
	common.GetClusterReq
	// in: header
	// name: Zone
	Zone string
}

func DecodeGCPTypesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GCPTypesReq

	commonReq, err := DecodeGCPCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GCPCommonReq = commonReq.(GCPCommonReq)
	req.Zone = r.Header.Get("Zone")

	return req, nil
}

func DecodeGCPZoneReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GCPZoneReq

	commonReq, err := DecodeGCPCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GCPCommonReq = commonReq.(GCPCommonReq)

	dc, ok := mux.Vars(r)["dc"]
	if !ok {
		return req, fmt.Errorf("'dc' parameter is required")
	}
	req.DC = dc

	return req, nil
}

func DecodeGCPCommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GCPCommonReq

	req.ServiceAccount = r.Header.Get("ServiceAccount")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

func DecodeGCPTypesNoCredentialReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GCPTypesNoCredentialReq

	commonReq, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = commonReq.(common.GetClusterReq)
	req.Zone = r.Header.Get("Zone")

	return req, nil
}

func GCPDiskTypesEndpoint(credentialManager common.PresetsManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPTypesReq)

		zone := req.Zone
		sa := req.ServiceAccount

		if len(req.Credential) > 0 && credentialManager.GetPresets().GCP.Credentials != nil {
			for _, credential := range credentialManager.GetPresets().GCP.Credentials {
				if credential.Name == req.Credential {
					sa = credential.ServiceAccount
					break
				}
			}
		}

		return listGCPDiskTypes(ctx, sa, zone)
	}
}

func GCPDiskTypesNoCredentialsEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPTypesNoCredentialReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if cluster.Spec.Cloud.GCP == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
		}

		sa := cluster.Spec.Cloud.GCP.ServiceAccount
		return listGCPDiskTypes(ctx, sa, req.Zone)
	}
}

func listGCPDiskTypes(ctx context.Context, sa string, zone string) (apiv1.GCPDiskTypeList, error) {
	diskTypes := apiv1.GCPDiskTypeList{}

	computeService, project, err := gcp.ConnectToComputeService(sa)
	if err != nil {
		return diskTypes, err
	}

	req := computeService.DiskTypes.List(project, zone)
	err = req.Pages(ctx, func(page *compute.DiskTypeList) error {
		for _, diskType := range page.Items {
			if diskType.Name != "local-ssd" {
				// TODO: There are some issues at the moment with local-ssd, that's why it is disabled at the moment.
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

func GCPSizeEndpoint(credentialManager common.PresetsManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPTypesReq)

		zone := req.Zone
		sa := req.ServiceAccount

		if len(req.Credential) > 0 && credentialManager.GetPresets().GCP.Credentials != nil {
			for _, credential := range credentialManager.GetPresets().GCP.Credentials {
				if credential.Name == req.Credential {
					sa = credential.ServiceAccount
					break
				}
			}
		}

		return listGCPSizes(ctx, sa, zone)
	}
}

func GCPSizeNoCredentialsEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPTypesNoCredentialReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if cluster.Spec.Cloud.GCP == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
		}

		sa := cluster.Spec.Cloud.GCP.ServiceAccount
		return listGCPSizes(ctx, sa, req.Zone)
	}
}

func listGCPSizes(ctx context.Context, sa string, zone string) (apiv1.GCPMachineSizeList, error) {
	sizes := apiv1.GCPMachineSizeList{}

	computeService, project, err := gcp.ConnectToComputeService(sa)
	if err != nil {
		return sizes, err
	}

	req := computeService.MachineTypes.List(project, zone)
	err = req.Pages(ctx, func(page *compute.MachineTypeList) error {
		for _, machineType := range page.Items {
			// TODO: Make the check below more generic, working for all the providers. It is needed as the pods
			//  with memory under 2 GB will be full with required pods like kube-proxy, CNI etc.
			if machineType.MemoryMb > 2048 {
				mt := apiv1.GCPMachineSize{
					Name:        machineType.Name,
					Description: machineType.Description,
					Memory:      machineType.MemoryMb,
					VCPUs:       machineType.GuestCpus,
				}

				sizes = append(sizes, mt)
			}
		}
		return nil
	})

	return sizes, err
}

func GCPZoneEndpoint(credentialManager common.PresetsManager, dcs map[string]provider.DatacenterMeta) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPZoneReq)

		sa := req.ServiceAccount

		if len(req.Credential) > 0 && credentialManager.GetPresets().GCP.Credentials != nil {
			for _, credential := range credentialManager.GetPresets().GCP.Credentials {
				if credential.Name == req.Credential {
					sa = credential.ServiceAccount
					break
				}
			}
		}

		return listGCPZones(ctx, sa, req.DC, dcs)
	}
}

func GCPZoneNoCredentialsEndpoint(projectProvider provider.ProjectProvider, dcs map[string]provider.DatacenterMeta) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPTypesNoCredentialReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if cluster.Spec.Cloud.GCP == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
		}

		sa := cluster.Spec.Cloud.GCP.ServiceAccount
		return listGCPZones(ctx, sa, req.DC, dcs)
	}
}

func listGCPZones(ctx context.Context, sa, datacenterName string, dcs map[string]provider.DatacenterMeta) (apiv1.GCPZoneList, error) {
	zones := apiv1.GCPZoneList{}

	datacenter, err := dc.GetDatacenter(dcs, datacenterName)
	if err != nil {
		return nil, errors.NewBadRequest("%v", err)
	}

	if datacenter.Spec.GCP == nil {
		return nil, errors.NewBadRequest("the %s is not GCP datacenter", datacenterName)
	}

	computeService, project, err := gcp.ConnectToComputeService(sa)
	if err != nil {
		return nil, err
	}

	req := computeService.Zones.List(project)
	err = req.Pages(ctx, func(page *compute.ZoneList) error {
		for _, zone := range page.Items {

			if strings.HasPrefix(zone.Name, datacenter.Spec.GCP.Region) {
				apiZone := apiv1.GCPZone{Name: zone.Name}
				zones = append(zones, apiZone)
			}
		}
		return nil
	})

	return zones, err
}
