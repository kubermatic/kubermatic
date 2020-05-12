package dc

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// ListEndpoint an HTTP endpoint that returns a list of apiv1.Datacenter
func ListEndpoint(seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		seeds, err := seedsGetter()
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// Get the DCs and immediately filter out the ones restricted by e-mail domain.
		dcs, err := filterDCsByEmail(userInfo, getAPIDCsFromSeedMap(seeds))
		if err != nil {
			return apiv1.Datacenter{}, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list datacenters: %v", err))
		}
		// Maintain a stable order. We do not check for duplicate names here
		sort.SliceStable(dcs, func(i, j int) bool {
			return dcs[i].Metadata.Name < dcs[j].Metadata.Name
		})

		return dcs, nil
	}
}

// ListEndpoint an HTTP endpoint that returns a list of apiv1.Datacenter for a specified provider
func ListEndpointForProvider(seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(forProviderDCListReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}

		seeds, err := seedsGetter()
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
		}

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		// Get the DCs and immediately filter them out for the provider.
		dcs := filterDCsByProvider(req.Provider, getAPIDCsFromSeedMap(seeds))

		// Filter out dc restricted by email if user is not admin
		if !userInfo.IsAdmin {
			dcs, err = filterDCsByEmail(userInfo, dcs)
			if err != nil {
				return apiv1.Datacenter{}, errors.New(http.StatusInternalServerError,
					fmt.Sprintf("failed to filter datacenters by email: %v", err))
			}
		}

		// Maintain a stable order. We do not check for duplicate names here
		sort.SliceStable(dcs, func(i, j int) bool {
			return dcs[i].Metadata.Name < dcs[j].Metadata.Name
		})

		return dcs, nil
	}
}

func filterDCsByProvider(providerName string, list []apiv1.Datacenter) []apiv1.Datacenter {
	var dcList []apiv1.Datacenter

	for _, dc := range list {
		if dc.Spec.Provider == providerName {
			dcList = append(dcList, dc)
		}
	}
	return dcList
}

// GetEndpoint an HTTP endpoint that returns a single apiv1.Datacenter object
func GetEndpoint(seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(LegacyDCReq)

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return GetDatacenter(userInfo, seedsGetter, req.DC)
	}
}

// GetDatacenter a function that gives you a single apiv1.Datacenter object
func GetDatacenter(userInfo *provider.UserInfo, seedsGetter provider.SeedsGetter, datacenterToGet string) (apiv1.Datacenter, error) {
	seeds, err := seedsGetter()
	if err != nil {
		return apiv1.Datacenter{}, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}

	// Get the DCs and immediately filter out the ones restricted by e-mail domain.
	dcs, err := filterDCsByEmail(userInfo, getAPIDCsFromSeedMap(seeds))
	if err != nil {
		return apiv1.Datacenter{}, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list datacenters: %v", err))
	}

	// The datacenter endpoints return both node and seed dcs, so we have to iterate through
	// everything
	var foundDCs []apiv1.Datacenter
	for _, unfilteredDC := range dcs {
		if unfilteredDC.Metadata.Name == datacenterToGet {
			foundDCs = append(foundDCs, unfilteredDC)
		}
	}

	if n := len(foundDCs); n > 1 {
		return apiv1.Datacenter{}, fmt.Errorf("did not find one but %d datacenters for name %q", n, datacenterToGet)
	}
	if len(foundDCs) == 0 {
		return apiv1.Datacenter{}, errors.NewNotFound("datacenter", datacenterToGet)
	}

	return foundDCs[0], nil
}

func filterDCsByEmail(userInfo *provider.UserInfo, list []apiv1.Datacenter) ([]apiv1.Datacenter, error) {
	var dcList []apiv1.Datacenter

iterateOverDCs:
	for _, dc := range list {
		requiredEmailDomain := dc.Spec.RequiredEmailDomain
		requiredEmailDomainsList := dc.Spec.RequiredEmailDomains

		if requiredEmailDomain == "" && len(requiredEmailDomainsList) == 0 {
			// find datacenter for "all" without RequiredEmailDomain(s) field
			dcList = append(dcList, dc)
		} else {
			// find datacenter for specific email domain
			split := strings.Split(userInfo.Email, "@")
			if len(split) != 2 {
				return nil, fmt.Errorf("invalid email address")
			}
			userDomain := split[1]

			if requiredEmailDomain != "" && strings.EqualFold(userDomain, requiredEmailDomain) {
				dcList = append(dcList, dc)
				continue iterateOverDCs
			}

			for _, whitelistedDomain := range requiredEmailDomainsList {
				if whitelistedDomain != "" && strings.EqualFold(userDomain, whitelistedDomain) {
					dcList = append(dcList, dc)
					continue iterateOverDCs
				}
			}
		}
	}
	return dcList, nil
}

func getAPIDCsFromSeedMap(seeds map[string]*kubermaticv1.Seed) []apiv1.Datacenter {
	var foundDCs []apiv1.Datacenter
	for _, seed := range seeds {
		foundDCs = append(foundDCs, apiv1.Datacenter{
			Metadata: apiv1.LegacyObjectMeta{
				Name:            seed.Name,
				ResourceVersion: "1",
			},
			Seed: true,
		})

		for datacenterName, datacenter := range seed.Spec.Datacenters {
			spec, err := apiSpec(datacenter.DeepCopy())
			if err != nil {
				log.Logger.Errorf("api spec error in dc %q: %v", datacenterName, err)
				continue
			}
			spec.Seed = seed.Name
			foundDCs = append(foundDCs, apiv1.Datacenter{
				Metadata: apiv1.LegacyObjectMeta{
					Name:            datacenterName,
					ResourceVersion: "1",
				},
				Spec: *spec,
			})
		}
	}

	return foundDCs
}

func apiSpec(dc *kubermaticv1.Datacenter) (*apiv1.DatacenterSpec, error) {
	p, err := provider.DatacenterCloudProviderName(dc.Spec.DeepCopy())
	if err != nil {
		return nil, err
	}
	return &apiv1.DatacenterSpec{
		Location:                 dc.Location,
		Country:                  dc.Country,
		Provider:                 p,
		Node:                     dc.Node,
		Digitalocean:             dc.Spec.Digitalocean,
		AWS:                      dc.Spec.AWS,
		BringYourOwn:             dc.Spec.BringYourOwn,
		Openstack:                dc.Spec.Openstack,
		Hetzner:                  dc.Spec.Hetzner,
		VSphere:                  dc.Spec.VSphere,
		Azure:                    dc.Spec.Azure,
		Packet:                   dc.Spec.Packet,
		GCP:                      dc.Spec.GCP,
		Kubevirt:                 dc.Spec.Kubevirt,
		Alibaba:                  dc.Spec.Alibaba,
		RequiredEmailDomain:      dc.Spec.RequiredEmailDomain,
		RequiredEmailDomains:     dc.Spec.RequiredEmailDomains,
		EnforceAuditLogging:      dc.Spec.EnforceAuditLogging,
		EnforcePodSecurityPolicy: dc.Spec.EnforcePodSecurityPolicy,
	}, nil
}

// DCsReq represent a request for datacenters specific data
type DCsReq struct{}

// DecodeDatacentersReq decodes HTTP request into DCsReq
func DecodeDatacentersReq(c context.Context, r *http.Request) (interface{}, error) {
	var req DCsReq

	return req, nil
}

// LegacyDCReq represent a request for datacenter specific data
// swagger:parameters getDatacenter
type LegacyDCReq struct {
	// in: path
	DC string `json:"dc"`
}

// GetDC returns the name of the datacenter in the request
func (req LegacyDCReq) GetDC() string {
	return req.DC
}

// DecodeLegacyDcReq decodes http request into LegacyDCReq
func DecodeLegacyDcReq(c context.Context, r *http.Request) (interface{}, error) {
	var req LegacyDCReq

	req.DC = mux.Vars(r)["dc"]
	return req, nil
}

// forProviderDCListReq defines HTTP request for ListDCForProvider
// swagger:parameters listDCForProvider
type forProviderDCListReq struct {
	// in: path
	// required: true
	Provider string `json:"provider_name"`
}

// DecodeForProviderDCListReq decodes http request into ForProviderDCListReq
func DecodeForProviderDCListReq(c context.Context, r *http.Request) (interface{}, error) {
	var req forProviderDCListReq

	req.Provider = mux.Vars(r)["provider_name"]
	if req.Provider == "" {
		return nil, fmt.Errorf("'provider_name' parameter is required but was not provided")
	}
	return req, nil
}
