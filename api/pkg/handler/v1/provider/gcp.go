package provider

import (
	"context"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"google.golang.org/api/compute/v1"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/gcp"
)

// GCPMachineTypesReq represent a request for GCP machine or disk types.
type GCPMachineTypesReq struct {
	ServiceAccount string
	Zone           string
	Credential     string
}

func DecodeGCPTypesReqReq(c context.Context, r *http.Request) (interface{}, error) {
	var req GCPMachineTypesReq

	req.ServiceAccount = r.Header.Get("ServiceAccount")
	req.Zone = r.Header.Get("Zone")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

func GCPDiskTypesEndpoint(credentialManager common.CredentialManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPMachineTypesReq)

		zone := req.Zone
		sa := req.ServiceAccount

		if len(req.Credential) > 0 && credentialManager.GetCredentials().GCP != nil {
			for _, credential := range credentialManager.GetCredentials().GCP {
				if credential.Name == req.Credential {
					sa = credential.ServiceAccount
					break
				}
			}
		}

		return listGCPDiskTypes(ctx, sa, zone)
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

func GCPSizeEndpoint(credentialManager common.CredentialManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(GCPMachineTypesReq)

		zone := req.Zone
		sa := req.ServiceAccount

		if len(req.Credential) > 0 && credentialManager.GetCredentials().GCP != nil {
			for _, credential := range credentialManager.GetCredentials().GCP {
				if credential.Name == req.Credential {
					sa = credential.ServiceAccount
					break
				}
			}
		}

		return listGCPSizes(ctx, sa, zone)
	}
}

func listGCPSizes(ctx context.Context, sa string, zone string) (apiv1.GCPMachineSizeList, error) {
	machineTypes := apiv1.GCPMachineSizeList{}

	computeService, project, err := gcp.ConnectToComputeService(sa)
	if err != nil {
		return machineTypes, err
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
				}

				machineTypes = append(machineTypes, mt)
			}
		}
		return nil
	})

	return machineTypes, err
}
