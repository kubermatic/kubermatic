package dc

import (
	"context"
	"fmt"
	"net/http"
	"sort"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/log"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// ListEndpoint an HTTP endpoint that returns a list of apiv1.Datacenter
func ListEndpoint(seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		seeds, err := seedsGetter()
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
		}
		// Maintain a stable order. We do not check for duplicate names here
		dcs := getAPIDCsFromSeedMap(seeds)
		sort.SliceStable(dcs, func(i, j int) bool {
			return dcs[i].Metadata.Name < dcs[j].Metadata.Name
		})

		return dcs, nil
	}
}

// GetEndpoint an HTTP endpoint that returns a single apiv1.Datacenter object
func GetEndpoint(seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(LegacyDCReq)
		return GetDatacenter(seedsGetter, req.DC)
	}
}

// GetDatacenter a function that gives you a single apiv1.Datacenter object
func GetDatacenter(seedsGetter provider.SeedsGetter, datacenterToGet string) (apiv1.Datacenter, error) {
	seeds, err := seedsGetter()
	if err != nil {
		return apiv1.Datacenter{}, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to list seeds: %v", err))
	}
	// The datacenter endpoints return both node and seed dcs, so we have to iterate through
	// everything
	var foundDCs []apiv1.Datacenter
	for _, unfilteredDC := range getAPIDCsFromSeedMap(seeds) {
		if unfilteredDC.Metadata.Name == datacenterToGet {
			foundDCs = append(foundDCs, unfilteredDC)
		}
	}

	if n := len(foundDCs); n > 1 {
		return apiv1.Datacenter{}, fmt.Errorf("Did not find one but %d datacenters for name %q", n, datacenterToGet)
	}
	if len(foundDCs) == 0 {
		return apiv1.Datacenter{}, errors.NewNotFound("datacenter", datacenterToGet)
	}

	return foundDCs[0], nil
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

func imagesMap(images kubermaticv1.ImageList) map[string]string {
	m := map[string]string{}
	for os, image := range images {
		m[string(os)] = image
	}
	return m
}

func apiSpec(dc *kubermaticv1.Datacenter) (*apiv1.DatacenterSpec, error) {
	p, err := provider.DatacenterCloudProviderName(dc.Spec.DeepCopy())
	if err != nil {
		return nil, err
	}
	spec := &apiv1.DatacenterSpec{
		Location: dc.Location,
		Country:  dc.Country,
		Provider: p,
	}

	switch {
	case dc.Spec.Digitalocean != nil:
		spec.Digitalocean = &apiv1.DigitialoceanDatacenterSpec{
			Region: dc.Spec.Digitalocean.Region,
		}
	case dc.Spec.AWS != nil:
		spec.AWS = &apiv1.AWSDatacenterSpec{
			Region: dc.Spec.AWS.Region,
		}
	case dc.Spec.BringYourOwn != nil:
		spec.BringYourOwn = &apiv1.BringYourOwnDatacenterSpec{}
	case dc.Spec.Openstack != nil:
		spec.Openstack = &apiv1.OpenstackDatacenterSpec{
			AuthURL:           dc.Spec.Openstack.AuthURL,
			AvailabilityZone:  dc.Spec.Openstack.AvailabilityZone,
			Region:            dc.Spec.Openstack.Region,
			Images:            imagesMap(dc.Spec.Openstack.Images),
			EnforceFloatingIP: dc.Spec.Openstack.EnforceFloatingIP,
		}
	case dc.Spec.Hetzner != nil:
		spec.Hetzner = &apiv1.HetznerDatacenterSpec{
			Datacenter: dc.Spec.Hetzner.Datacenter,
			Location:   dc.Spec.Hetzner.Location,
		}
	case dc.Spec.VSphere != nil:
		spec.VSphere = &apiv1.VSphereDatacenterSpec{
			Endpoint:   dc.Spec.VSphere.Endpoint,
			Datacenter: dc.Spec.VSphere.Datacenter,
			Datastore:  dc.Spec.VSphere.Datastore,
			Cluster:    dc.Spec.VSphere.Cluster,
			Templates:  imagesMap(dc.Spec.VSphere.Templates),
		}
	case dc.Spec.Azure != nil:
		spec.Azure = &apiv1.AzureDatacenterSpec{
			Location: dc.Spec.Azure.Location,
		}
	case dc.Spec.Packet != nil:
		spec.Packet = &apiv1.PacketDatacenterSpec{
			Facilities: dc.Spec.Packet.Facilities,
		}
	case dc.Spec.GCP != nil:
		spec.GCP = &apiv1.GCPDatacenterSpec{
			Region:       dc.Spec.GCP.Region,
			ZoneSuffixes: dc.Spec.GCP.ZoneSuffixes,
			Regional:     dc.Spec.GCP.Regional,
		}
	case dc.Spec.Kubevirt != nil:
		spec.Kubevirt = &apiv1.KubevirtDatacenterSpec{}
	}

	return spec, nil
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
