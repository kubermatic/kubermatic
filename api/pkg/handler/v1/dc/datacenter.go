package dc

import (
	"context"
	"encoding/json"
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

		// Get the DCs and immediately filter out the ones restricted by e-mail domain if user is not admin
		dcs := getAPIDCsFromSeedMap(seeds)
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

	// Get the DCs and immediately filter out the ones restricted by e-mail domain if user is not admin
	dcs := getAPIDCsFromSeedMap(seeds)
	if !userInfo.IsAdmin {
		dcs, err = filterDCsByEmail(userInfo, dcs)
		if err != nil {
			return apiv1.Datacenter{}, errors.New(http.StatusInternalServerError,
				fmt.Sprintf("failed to filter datacenters by email: %v", err))
		}
	}

	return filterDCsByName(dcs, datacenterToGet)
}

// GetEndpointForProvider an HTTP endpoint that returns a specified apiv1.Datacenter for a specified provider
func GetEndpointForProvider(seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(forProviderDCGetReq)
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

		return filterDCsByName(dcs, req.Datacenter)
	}
}

func filterDCsByName(dcs []apiv1.Datacenter, dcName string) (apiv1.Datacenter, error) {
	var foundDCs []apiv1.Datacenter
	for _, unfilteredDC := range dcs {
		if unfilteredDC.Metadata.Name == dcName {
			foundDCs = append(foundDCs, unfilteredDC)
		}
	}

	if n := len(foundDCs); n > 1 {
		return apiv1.Datacenter{}, fmt.Errorf("did not find one but %d datacenters for name %q", n, dcName)
	}
	if len(foundDCs) == 0 {
		return apiv1.Datacenter{}, errors.NewNotFound("datacenter", dcName)
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

// CreateEndpoint an HTTP endpoint that creates a specified apiv1.Datacenter
func CreateEndpoint(seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter, seedsClientGetter provider.SeedClientGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(createDCReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}

		err := req.validate()
		if err != nil {
			return nil, errors.New(http.StatusBadRequest, fmt.Sprintf("Validation error: %v", err))
		}

		if err := validateUser(ctx, userInfoGetter); err != nil {
			return nil, err
		}

		// Get the seed in which the dc should be created
		seeds, err := seedsGetter()
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
		}
		seed, ok := seeds[req.Body.Spec.Seed]
		if !ok {
			return nil, errors.New(http.StatusBadRequest,
				fmt.Sprintf("Bad request: seed %q does not exist", req.Body.Spec.Seed))
		}

		// Check if dc already exists
		if _, ok = seed.Spec.Datacenters[req.Body.Name]; ok {
			return nil, errors.New(http.StatusBadRequest,
				fmt.Sprintf("Bad request: datacenter %q already exists", req.Body.Name))
		}

		// Add DC, update seed
		seed.Spec.Datacenters[req.Body.Name] = apiToKubermatic(&req.Body.Spec)

		seedClient, err := seedsClientGetter(seed)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to get seed client: %v", err))
		}

		if err = seedClient.Update(ctx, seed); err != nil {
			return nil, errors.New(http.StatusInternalServerError,
				fmt.Sprintf("failed to update seed %q datacenter %q: %v", seed.Name, req.Body.Name, err))
		}

		return &apiv1.Datacenter{
			Metadata: apiv1.LegacyObjectMeta{
				Name: req.Body.Name,
			},
			Spec: req.Body.Spec,
		}, nil
	}
}

// UpdateEndpoint an HTTP endpoint that updates a specified apiv1.Datacenter
func UpdateEndpoint(seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter,
	seedsClientGetter provider.SeedClientGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(updateDCReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}

		err := req.validate()
		if err != nil {
			return nil, errors.New(http.StatusBadRequest, fmt.Sprintf("Validation error: %v", err))
		}

		if err := validateUser(ctx, userInfoGetter); err != nil {
			return nil, err
		}

		// Get the seed in which the dc should be updated
		seeds, err := seedsGetter()
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
		}
		seed, ok := seeds[req.Body.Spec.Seed]
		if !ok {
			return nil, errors.New(http.StatusBadRequest,
				fmt.Sprintf("Bad request: seed %q does not exist", req.Body.Spec.Seed))
		}

		// get the dc to update
		if _, ok := seed.Spec.Datacenters[req.DCToUpdate]; !ok {
			return nil, errors.New(http.StatusBadRequest,
				fmt.Sprintf("Bad request: datacenter %q does not exists", req.DCToUpdate))
		}

		// Do an extra check if name changed and remove old dc
		if !strings.EqualFold(req.DCToUpdate, req.Body.Name) {
			if _, ok := seed.Spec.Datacenters[req.Body.Name]; ok {
				return nil, errors.New(http.StatusBadRequest,
					fmt.Sprintf("Bad request: cannot change %q datacenter name to %q as it already exists",
						req.DCToUpdate, req.Body.Name))
			}
			delete(seed.Spec.Datacenters, req.DCToUpdate)
		}
		seed.Spec.Datacenters[req.Body.Name] = apiToKubermatic(&req.Body.Spec)

		seedClient, err := seedsClientGetter(seed)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to get seed client: %v", err))
		}

		if err = seedClient.Update(ctx, seed); err != nil {
			return nil, errors.New(http.StatusInternalServerError,
				fmt.Sprintf("failed to update seed %q datacenter %q: %v", seed.Name, req.DCToUpdate, err))
		}

		return &apiv1.Datacenter{
			Metadata: apiv1.LegacyObjectMeta{
				Name: req.Body.Name,
			},
			Spec: req.Body.Spec,
		}, nil
	}
}

// DeleteEndpoint an HTTP endpoint that deletes a specified apiv1.Datacenter
func DeleteEndpoint(seedsGetter provider.SeedsGetter, userInfoGetter provider.UserInfoGetter,
	seedsClientGetter provider.SeedClientGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(deleteDCReq)
		if !ok {
			return nil, errors.NewBadRequest("invalid request")
		}

		if err := validateUser(ctx, userInfoGetter); err != nil {
			return nil, err
		}

		// Get the seed in which the dc should be deleted
		seeds, err := seedsGetter()
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
		}
		seed, ok := seeds[req.Seed]
		if !ok {
			return nil, errors.New(http.StatusBadRequest,
				fmt.Sprintf("Bad request: seed %q does not exist", req.Seed))
		}

		// get the dc to delete
		if _, ok := seed.Spec.Datacenters[req.DC]; !ok {
			return nil, errors.New(http.StatusBadRequest,
				fmt.Sprintf("Bad request: datacenter %q does not exists", req.DC))
		}
		delete(seed.Spec.Datacenters, req.DC)

		seedClient, err := seedsClientGetter(seed)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to get seed client: %v", err))
		}

		if err = seedClient.Update(ctx, seed); err != nil {
			return nil, errors.New(http.StatusInternalServerError,
				fmt.Sprintf("failed to delete seed %q datacenter %q: %v", seed.Name, req.DC, err))
		}

		return nil, nil
	}
}

func validateUser(ctx context.Context, userInfoGetter provider.UserInfoGetter) error {
	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return common.KubernetesErrorToHTTPError(err)
	}

	if !userInfo.IsAdmin {
		return errors.New(http.StatusForbidden,
			fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", userInfo.Email))
	}
	return nil
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

func apiToKubermatic(datacenter *apiv1.DatacenterSpec) kubermaticv1.Datacenter {
	return kubermaticv1.Datacenter{
		Country:  datacenter.Country,
		Location: datacenter.Location,
		Node:     datacenter.Node,
		Spec: kubermaticv1.DatacenterSpec{
			Digitalocean:             datacenter.Digitalocean,
			BringYourOwn:             datacenter.BringYourOwn,
			AWS:                      datacenter.AWS,
			Azure:                    datacenter.Azure,
			Openstack:                datacenter.Openstack,
			Packet:                   datacenter.Packet,
			Hetzner:                  datacenter.Hetzner,
			VSphere:                  datacenter.VSphere,
			GCP:                      datacenter.GCP,
			Kubevirt:                 datacenter.Kubevirt,
			Alibaba:                  datacenter.Alibaba,
			RequiredEmailDomain:      datacenter.RequiredEmailDomain,
			RequiredEmailDomains:     datacenter.RequiredEmailDomains,
			EnforceAuditLogging:      datacenter.EnforceAuditLogging,
			EnforcePodSecurityPolicy: datacenter.EnforcePodSecurityPolicy,
		},
	}
}

// LegacyDCReq represent a request for datacenter specific data
// swagger:parameters getDatacenter
type LegacyDCReq struct {
	// in: path
	// required: true
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

// DecodeForProviderDCListReq decodes http request into forProviderDCListReq
func DecodeForProviderDCListReq(c context.Context, r *http.Request) (interface{}, error) {
	var req forProviderDCListReq

	req.Provider = mux.Vars(r)["provider_name"]
	if req.Provider == "" {
		return nil, fmt.Errorf("'provider_name' parameter is required but was not provided")
	}
	return req, nil
}

// forProviderDCGetReq defines HTTP request for GetDCForProvider
// swagger:parameters getDCForProvider
type forProviderDCGetReq struct {
	// in: path
	// required: true
	Provider string `json:"provider_name"`
	// in: path
	// required: true
	Datacenter string `json:"dc"`
}

// DecodeForProviderDCGetReq decodes http request into forProviderDCGetReq
func DecodeForProviderDCGetReq(c context.Context, r *http.Request) (interface{}, error) {
	var req forProviderDCGetReq

	req.Provider = mux.Vars(r)["provider_name"]
	if req.Provider == "" {
		return nil, fmt.Errorf("'provider_name' parameter is required but was not provided")
	}

	req.Datacenter = mux.Vars(r)["dc"]
	if req.Datacenter == "" {
		return nil, fmt.Errorf("'dc' parameter is required but was not provided")
	}
	return req, nil
}

// createDCReq defines HTTP request for CreateDC
// swagger:parameters createDC
type createDCReq struct {
	// in: path
	// required: true
	Seed string `json:"seed_name"`
	// in: body
	Body struct {
		Name string               `json:"name"`
		Spec apiv1.DatacenterSpec `json:"spec"`
	}
}

func (req createDCReq) validate() error {
	if err := validateProvider(&req.Body.Spec); err != nil {
		return err
	}

	if !strings.EqualFold(req.Seed, req.Body.Spec.Seed) {
		return fmt.Errorf("path seed %q and request seed %q not equal", req.Seed, req.Body.Spec.Seed)
	}

	return nil
}

// DecodeCreateDCReq decodes http request into createDCReq
func DecodeCreateDCReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createDCReq

	req.Seed = mux.Vars(r)["seed_name"]
	if req.Seed == "" {
		return nil, fmt.Errorf("'seed_name' parameter is required but was not provided")
	}

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}
	return req, nil
}

// updateDCReq defines HTTP request for UpdateDC
// swagger:parameters updateDC
type updateDCReq struct {
	createDCReq
	// in: path
	// required: true
	DCToUpdate string `json:"dc"`
}

// DecodeUpdateDCReq decodes http request into updateDCReq
func DecodeUpdateDCReq(c context.Context, r *http.Request) (interface{}, error) {
	var req updateDCReq

	createReq, err := DecodeCreateDCReq(c, r)
	if err != nil {
		return nil, err
	}
	req.createDCReq = createReq.(createDCReq)

	req.DCToUpdate = mux.Vars(r)["dc"]
	if req.DCToUpdate == "" {
		return nil, fmt.Errorf("'dc' parameter is required but was not provided")
	}

	return req, nil
}

func validateProvider(dcSpec *apiv1.DatacenterSpec) error {
	var providerNames []string

	if dcSpec.Alibaba != nil {
		providerNames = append(providerNames, provider.AlibabaCloudProvider)
	}
	if dcSpec.BringYourOwn != nil {
		providerNames = append(providerNames, provider.BringYourOwnCloudProvider)
	}
	if dcSpec.Digitalocean != nil {
		providerNames = append(providerNames, provider.DigitaloceanCloudProvider)
	}
	if dcSpec.AWS != nil {
		providerNames = append(providerNames, provider.AWSCloudProvider)
	}
	if dcSpec.Openstack != nil {
		providerNames = append(providerNames, provider.OpenstackCloudProvider)
	}
	if dcSpec.Packet != nil {
		providerNames = append(providerNames, provider.PacketCloudProvider)
	}
	if dcSpec.Hetzner != nil {
		providerNames = append(providerNames, provider.HetznerCloudProvider)
	}
	if dcSpec.VSphere != nil {
		providerNames = append(providerNames, provider.VSphereCloudProvider)
	}
	if dcSpec.Azure != nil {
		providerNames = append(providerNames, provider.AzureCloudProvider)
	}
	if dcSpec.GCP != nil {
		providerNames = append(providerNames, provider.GCPCloudProvider)
	}
	if dcSpec.Kubevirt != nil {
		providerNames = append(providerNames, provider.KubevirtCloudProvider)
	}

	if len(providerNames) != 1 {
		return fmt.Errorf("one DC provider should be specified, got: %v", providerNames)
	}
	return nil
}

// deleteDCReq defines HTTP request for DeleteDC
// swagger:parameters deleteDC
type deleteDCReq struct {
	// in: path
	// required: true
	Seed string `json:"seed_name"`
	// in: path
	// required: true
	DC string `json:"dc"`
}

// DecodeDeleteDCReq decodes http request into deleteDCReq
func DecodeDeleteDCReq(c context.Context, r *http.Request) (interface{}, error) {
	var req deleteDCReq

	req.Seed = mux.Vars(r)["seed_name"]
	if req.Seed == "" {
		return nil, fmt.Errorf("'seed_name' parameter is required but was not provided")
	}

	req.DC = mux.Vars(r)["dc"]
	if req.DC == "" {
		return nil, fmt.Errorf("'dc' parameter is required but was not provided")
	}

	return req, nil
}
