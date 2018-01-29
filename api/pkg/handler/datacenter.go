package handler

import (
	"context"
	"fmt"
	"sort"

	"github.com/go-kit/kit/endpoint"
	"github.com/golang/glog"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

func datacentersEndpoint(dcs map[string]provider.DatacenterMeta) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)

		adcs := []apiv1.Datacenter{}
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

			adc := apiv1.Datacenter{
				Metadata: apiv1.Metadata{
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

		return &apiv1.Datacenter{
			Metadata: apiv1.Metadata{
				Name:     req.DC,
				Revision: "1",
			},
			Spec: *spec,
			Seed: dc.IsSeed,
		}, nil
	}
}

func apiSpec(dc *provider.DatacenterMeta) (*apiv1.DatacenterSpec, error) {
	p, err := provider.DatacenterCloudProviderName(&dc.Spec)
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
	case dc.Spec.BareMetal != nil:
		spec.BareMetal = &apiv1.BareMetalDatacenterSpec{}
	case dc.Spec.Openstack != nil:
		spec.Openstack = &apiv1.OpenstackDatacenterSpec{
			AuthURL:          dc.Spec.Openstack.AuthURL,
			AvailabilityZone: dc.Spec.Openstack.AvailabilityZone,
		}
	}

	return spec, nil
}
