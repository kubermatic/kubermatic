package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/packethost/packngo"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// PacketSizesReq represent a request for Packet sizes.
// swagger:parameters listPacketSizes
type PacketSizesReq struct {
	// in: header
	// name: APIKey
	APIKey string `json:"apiKey"`
	// in: header
	// name: ProjectID
	ProjectID string `json:"projectID"`
	// in: header
	// name: Credential
	Credential string `json:"credential"`
}

// PacketSizesNoCredentialsReq represent a request for Packet sizes EP
// swagger:parameters listPacketSizesNoCredentials
type PacketSizesNoCredentialsReq struct {
	common.GetClusterReq
}

// Used to decode response object
type plansRoot struct {
	Plans []packngo.Plan `json:"plans"`
}

func DecodePacketSizesReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req PacketSizesReq

	req.APIKey = r.Header.Get("apiKey")
	req.ProjectID = r.Header.Get("projectID")
	req.Credential = r.Header.Get("credential")

	return req, nil
}

func DecodePacketSizesNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req PacketSizesNoCredentialsReq

	commonReq, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = commonReq.(common.GetClusterReq)

	return req, nil
}

func PacketSizesEndpoint(credentialManager common.PresetsManager) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(PacketSizesReq)

		projectID := req.ProjectID
		apiKey := req.APIKey

		if len(req.Credential) > 0 && credentialManager.GetPresets().Packet.Credentials != nil {
			for _, credential := range credentialManager.GetPresets().Packet.Credentials {
				if credential.Name == req.Credential {
					projectID = credential.ProjectID
					apiKey = credential.APIKey
					break
				}
			}
		}
		return sizes(ctx, apiKey, projectID)
	}
}

func PacketSizesNoCredentialsEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(PacketSizesNoCredentialsReq)
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
		if cluster.Spec.Cloud.Packet == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
		}

		return sizes(ctx, cluster.Spec.Cloud.Packet.APIKey, cluster.Spec.Cloud.Packet.ProjectID)
	}
}

func sizes(_ context.Context, apiKey, projectID string) (apiv1.PacketSizeList, error) {
	sizes := apiv1.PacketSizeList{}
	root := new(plansRoot)

	if len(apiKey) == 0 {
		return sizes, fmt.Errorf("missing required parameter: apiKey")
	}

	if len(projectID) == 0 {
		return sizes, fmt.Errorf("missing required parameter: projectID")
	}

	client := packngo.NewClientWithAuth("kubermatic", apiKey, nil)
	req, err := client.NewRequest("GET", "/projects/"+projectID+"/plans", nil)
	if err != nil {
		return sizes, err
	}

	_, err = client.Do(req, root)
	if err != nil {
		return sizes, err
	}

	plans := root.Plans
	for _, plan := range plans {
		sizes = append(sizes, toPacketSize(plan))
	}

	return sizes, nil
}

func toPacketSize(plan packngo.Plan) apiv1.PacketSize {
	drives := make([]apiv1.PacketDrive, 0)
	for _, drive := range plan.Specs.Drives {
		drives = append(drives, apiv1.PacketDrive{
			Count: drive.Count,
			Size:  drive.Size,
			Type:  drive.Type,
		})
	}

	memory := "N/A"
	if plan.Specs.Memory != nil {
		memory = plan.Specs.Memory.Total
	}

	cpus := make([]apiv1.PacketCPU, 0)
	for _, cpu := range plan.Specs.Cpus {
		cpus = append(cpus, apiv1.PacketCPU{
			Count: cpu.Count,
			Type:  cpu.Type,
		})
	}

	return apiv1.PacketSize{
		Name:   plan.Name,
		CPUs:   cpus,
		Memory: memory,
		Drives: drives,
	}
}
