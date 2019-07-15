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
func ListEndpoint(dcs map[string]*kubermaticv1.SeedDatacenter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		adcs := map[string]apiv1.Datacenter{}
		var keys []string
		for _, seed := range dcs {
			for nodeLocationName, nodeLocation := range seed.Spec.NodeLocations {
				spec, err := apiSpec(nodeLocation.DeepCopy())
				if err != nil {
					log.Logger.Errorf("api spec error in dc %q: %v", nodeLocationName, err)
					continue
				}

				adc := apiv1.Datacenter{
					Metadata: apiv1.LegacyObjectMeta{
						Name:            nodeLocationName,
						ResourceVersion: "1",
					},
					Spec: *spec,
				}
				adcs[nodeLocationName] = adc
				keys = append(keys, nodeLocationName)
			}
		}

		// Maintain a stable order
		sort.Strings(keys)
		var sortedAPIDCs []apiv1.Datacenter
		for _, dcName := range keys {
			sortedAPIDCs = append(sortedAPIDCs, adcs[dcName])
		}

		return sortedAPIDCs, nil
	}
}

// GetEndpoint an HTTP endpoint that returns a single apiv1.Datacenter object
func GetEndpoint(dcs map[string]*kubermaticv1.SeedDatacenter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(LegacyDCReq)
		return GetDatacenter(dcs, req.DC)
	}
}

// GetDatacenter a function that gives you a single apiv1.Datacenter object
func GetDatacenter(dcs map[string]*kubermaticv1.SeedDatacenter, datacenterToGet string) (apiv1.Datacenter, error) {
	dc, err := provider.NodeLocationFromSeedMap(dcs, datacenterToGet)
	if err != nil {
		return apiv1.Datacenter{}, errors.NewNotFound("datacenter", datacenterToGet)
	}

	spec, err := apiSpec(dc)
	if err != nil {
		return apiv1.Datacenter{}, fmt.Errorf("api spec error in dc %q: %v", datacenterToGet, err)
	}

	return apiv1.Datacenter{
		Metadata: apiv1.LegacyObjectMeta{
			Name:            datacenterToGet,
			ResourceVersion: "1",
		},
		Spec: *spec,
	}, nil
}

func imagesMap(images kubermaticv1.ImageList) map[string]string {
	m := map[string]string{}
	for os, image := range images {
		m[string(os)] = image
	}
	return m
}

func apiSpec(dc *kubermaticv1.NodeLocation) (*apiv1.DatacenterSpec, error) {
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
