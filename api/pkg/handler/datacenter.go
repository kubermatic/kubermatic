package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/go-kit/kit/endpoint"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/kubermatic/kubermatic/api"
	"github.com/kubermatic/kubermatic/api/pkg/handler/errors"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
)

func datacentersEndpoint(
	dcs map[string]provider.DatacenterMeta,
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	// TODO: Move out static function (range over dcs)
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user, err := GetUser(ctx)
		if err != nil {
			return nil, err
		}

		adcs := make([]api.Datacenter, 0, len(kps))
		var keys []string
		for k := range dcs {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, dcName := range keys {
			_, kpFound := kps[dcName]
			dc := dcs[dcName]

			if _, isAdmin := user.Roles[AdminRoleKey]; dc.Private && !isAdmin {
				glog.V(7).Infof("Hiding dc %q for non-admin user %q", dcName, user.Name)
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
				Seed: kpFound,
			}
			adcs = append(adcs, adc)
		}

		return adcs, nil
	}
}

type dcKeyListReq struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
	dcReq
}

func decodeDcKeyListRequest(c context.Context, r *http.Request) (interface{}, error) {
	var req dcKeyListReq

	dr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.dcReq = dr.(dcReq)

	err = json.NewDecoder(r.Body).Decode(&req)
	return req, err
}

func datacenterKeyEndpoint(
	dcs map[string]provider.DatacenterMeta,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(dcKeyListReq)

		dc, found := dcs[req.dc]
		if !found {
			return nil, errors.NewNotFound("datacenter", req.dc)
		}

		//TODO: Make more generic.
		// also for digitalocean ... or libmachine?
		if dc.Spec.AWS == nil {
			return nil, errors.NewBadRequest("not aws", req.dc)
		}

		config := aws.NewConfig().
			WithMaxRetries(10).
			WithRegion(dc.Spec.AWS.Region).
			WithCredentials(credentials.NewStaticCredentials(req.Username, req.Password, ""))
		s := session.New(config)
		client := ec2.New(s)
		keys, err := client.DescribeKeyPairs(&ec2.DescribeKeyPairsInput{})

		//Empty slices are getting marshaled to null...
		//We always want to return an array to the frontend!
		if len(keys.KeyPairs) == 0 {
			keys.KeyPairs = make([]*ec2.KeyPairInfo, 0)
		}

		return keys, err
	}
}

func datacenterEndpoint(
	dcs map[string]provider.DatacenterMeta,
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user, err := GetUser(ctx)
		if err != nil {
			return nil, err
		}
		req := request.(dcReq)

		dc, found := dcs[req.dc]
		if !found {
			return nil, errors.NewNotFound("datacenter", req.dc)
		}

		if _, isAdmin := user.Roles[AdminRoleKey]; dc.Private && !isAdmin {
			return nil, errors.NewNotFound("datacenter", req.dc)
		}

		_, kpFound := kps[req.dc]

		spec, err := apiSpec(&dc)
		if err != nil {
			return nil, fmt.Errorf("api spec error in dc %q: %v", req.dc, err)
		}

		return &api.Datacenter{
			Metadata: api.Metadata{
				Name:     req.dc,
				Revision: "1",
			},
			Spec: *spec,
			Seed: kpFound,
		}, nil
	}
}

type dcsReq struct {
}

func decodeDatacentersReq(c context.Context, r *http.Request) (interface{}, error) {
	var req dcsReq

	return req, nil
}

type dcReq struct {
	dc string
}

func decodeDcReq(c context.Context, r *http.Request) (interface{}, error) {
	var req dcReq

	req.dc = mux.Vars(r)["dc"]
	return req, nil
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
