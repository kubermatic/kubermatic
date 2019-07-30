package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	awsProvider "github.com/kubermatic/kubermatic/api/pkg/provider/cloud/aws"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// AWSCommonReq represent a request with common parameters for GCP.
type AWSCommonReq struct {
	// in: header
	// name: AccessKeyID
	AccessKeyID string
	// in: header
	// name: SecretAccessKey
	SecretAccessKey string
	// in: header
	// name: Credential
	Credential string
}

// AWSZoneReq represent a request for AWS zones.
// swagger:parameters listAWSZones
type AWSZoneReq struct {
	AWSCommonReq
	// in: path
	// required: true
	DC string `json:"dc"`
}

// DecodeAWSCommonReq decodes the base type for a AWS special endpoint request
func DecodeAWSCommonReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AWSCommonReq

	req.AccessKeyID = r.Header.Get("AccessKeyID")
	req.SecretAccessKey = r.Header.Get("SecretAccessKey")
	req.Credential = r.Header.Get("Credential")

	return req, nil
}

// DecodeAWSZoneReq decodes a request for a list of AWS zones
func DecodeAWSZoneReq(c context.Context, r *http.Request) (interface{}, error) {
	var req AWSZoneReq

	commonReq, err := DecodeAWSCommonReq(c, r)
	if err != nil {
		return nil, err
	}
	req.AWSCommonReq = commonReq.(AWSCommonReq)

	dc, ok := mux.Vars(r)["dc"]
	if !ok {
		return req, fmt.Errorf("'dc' parameter is required")
	}
	req.DC = dc

	return req, nil
}

// AWSZoneEndpoint handles the request to list AWS availability zones in a given region, using provided credentials
func AWSZoneEndpoint(credentialManager common.PresetsManager, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(AWSZoneReq)

		keyID := req.AccessKeyID
		keySecret := req.SecretAccessKey

		if len(req.Credential) > 0 && credentialManager.GetPresets().AWS.Credentials != nil {
			for _, credential := range credentialManager.GetPresets().AWS.Credentials {
				if credential.Name == req.Credential {
					keyID = credential.AccessKeyID
					keySecret = credential.SecretAccessKey
					break
				}
			}
		}

		return listAWSZones(ctx, keyID, keySecret, req.DC, seedsGetter)
	}
}

// AWSZoneNoCredentialsEndpoint handles the request to list AWS availability zones in a given region, using credentials from a given datacenter
func AWSZoneNoCredentialsEndpoint(projectProvider provider.ProjectProvider, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(common.GetClusterReq)
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
		if cluster.Spec.Cloud.AWS == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
		}

		keyID := cluster.Spec.Cloud.AWS.AccessKeyID
		keySecret := cluster.Spec.Cloud.AWS.SecretAccessKey
		return listAWSZones(ctx, keyID, keySecret, cluster.Spec.Cloud.DatacenterName, seedsGetter)
	}
}

func listAWSZones(ctx context.Context, keyID, keySecret, datacenterName string, seedsGetter provider.SeedsGetter) (apiv1.AWSZoneList, error) {
	zones := apiv1.AWSZoneList{}

	seeds, err := seedsGetter()
	if err != nil {
		return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}
	datacenter, err := provider.DatacenterFromSeedMap(seeds, datacenterName)
	if err != nil {
		return nil, errors.NewBadRequest("%v", err)
	}

	if datacenter.Spec.AWS == nil {
		return nil, errors.NewBadRequest("the %s is not AWS datacenter", datacenterName)
	}

	ec2, err := awsProvider.NewCloudProvider(datacenter)
	if err != nil {
		return nil, err
	}

	zoneResults, err := ec2.GetAvailabilityZonesInRegion(kubermaticv1.CloudSpec{
		DatacenterName: datacenterName,
		AWS: &kubermaticv1.AWSCloudSpec{
			AccessKeyID:     keyID,
			SecretAccessKey: keySecret,
		},
	}, datacenter.Spec.AWS.Region)
	if err != nil {
		return nil, err
	}

	for _, z := range zoneResults {
		zones = append(zones, apiv1.AWSZone{Name: *z.ZoneName})
	}

	return zones, err
}
