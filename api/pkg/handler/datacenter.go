package handler

import (
	"context"
	"fmt"
	"sort"

	"github.com/go-kit/kit/endpoint"
	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

func datacentersEndpoint(dcs map[string]provider.DatacenterMeta) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)

		adcs := []api.Datacenter{}
		var keys []string
		for k := range dcs {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, dcName := range keys {
			dc := dcs[dcName]

			if dc.Private && !user.IsAdmin() {
				glog.V(7).Infof("Hiding dc %q for non-admin user %q", dcName, user.ID)
				continue
			}

			spec, err := apiSpec(&dc)
			if err != nil {
				glog.Errorf("api spec error in dc %q: %v", dcName, err)
				continue
			}

			adc := api.Datacenter{
				Metadata: api.Metadata{
					Name:     dcName,
					Revision: "1",
				},
				Spec: *spec,
				Seed: dc.IsSeed,
			}
			adcs = append(adcs, adc)
		}

		return adcs, nil
	}
}

func datacenterEndpoint(dcs map[string]provider.DatacenterMeta) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		req := request.(DCReq)

		dc, found := dcs[req.DC]
		if !found {
			return nil, errors.NewNotFound("datacenter", req.DC)
		}

		if dc.Private && !user.IsAdmin() {
			return nil, errors.NewNotFound("datacenter", req.DC)
		}

		spec, err := apiSpec(&dc)
		if err != nil {
			return nil, fmt.Errorf("api spec error in dc %q: %v", req.DC, err)
		}

		return &api.Datacenter{
			Metadata: api.Metadata{
				Name:     req.DC,
				Revision: "1",
			},
			Spec: *spec,
			Seed: dc.IsSeed,
		}, nil
	}
}

func apiSpec(dc *provider.DatacenterMeta) (*api.DatacenterSpec, error) {
	p, err := provider.DatacenterCloudProviderName(&dc.Spec)
	if err != nil {
		return nil, err
	}
	spec := &api.DatacenterSpec{
		Location: dc.Location,
		Country:  dc.Country,
		Provider: p,
	}

	switch {
	case dc.Spec.Digitalocean != nil:
		spec.Digitalocean = &api.DigitialoceanDatacenterSpec{
			Region: dc.Spec.Digitalocean.Region,
		}
	case dc.Spec.AWS != nil:
		spec.AWS = &api.AWSDatacenterSpec{
			Region: dc.Spec.AWS.Region,
		}
	case dc.Spec.BringYourOwn != nil:
		spec.BringYourOwn = &api.BringYourOwnDatacenterSpec{}
	case dc.Spec.BareMetal != nil:
		spec.BareMetal = &api.BareMetalDatacenterSpec{}
	case dc.Spec.Openstack != nil:
		spec.Openstack = &api.OpenstackDatacenterSpec{
			AuthURL:          dc.Spec.Openstack.AuthURL,
			AvailabilityZone: dc.Spec.Openstack.AvailabilityZone,
		}
	}

	return spec, nil
}
