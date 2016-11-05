package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/go-kit/kit/endpoint"
	"github.com/golang/glog"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"
)

func datacentersEndpoint(
	dcs map[string]provider.DatacenterMeta,
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(dcsReq)

		adcs := make([]api.Datacenter, 0, len(kps))
		for dcName := range dcs {
			_, kpFound := kps[dcName]
			dc := dcs[dcName]

			if _, isAdmin := req.user.Roles["admin"]; dc.Private && !isAdmin {
				glog.V(7).Infof("Hiding dc %q for non-admin user", dcName, req.user.Name)
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

func decodeDcKeyListRequest(r *http.Request) (interface{}, error) {
	var req dcKeyListReq

	dr, err := decodeDcReq(r)
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
			return nil, NewNotFound("datacenter", req.dc)
		}

		//TODO: Make more generic.
		// also for digitalocean ... or libmachine?
		if dc.Spec.AWS == nil {
			return nil, NewBadRequest("not aws", req.dc)
		}

		// Create aws ec2 client.
		config := aws.NewConfig()
		config = config.WithRegion(dc.Spec.AWS.Region())
		// TODO(realfake): Implement token for AWS.
		config = config.WithCredentials(credentials.NewStaticCredentials(req.Username, req.Password, ""))
		sess := ec2.New(session.New(config))

		// Begin describing key pairs.
		return sess.DescribeKeyPairs(&ec2.DescribeKeyPairsInput{})
	}
}

func datacenterEndpoint(
	dcs map[string]provider.DatacenterMeta,
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(dcReq)

		dc, found := dcs[req.dc]
		if !found {
			return nil, NewNotFound("datacenter", req.dc)
		}

		if _, isAdmin := req.user.Roles["admin"]; dc.Private && !isAdmin {
			return nil, NewNotFound("datacenter", req.dc)
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
	userReq
}

func decodeDatacentersReq(r *http.Request) (interface{}, error) {
	var req dcsReq

	ur, err := decodeUserReq(r)
	if err != nil {
		return nil, err
	}
	req.userReq = ur.(userReq)

	return req, nil
}

type dcReq struct {
	userReq
	dc string
}

func decodeDcReq(r *http.Request) (interface{}, error) {
	var req dcReq

	dr, err := decodeUserReq(r)
	if err != nil {
		return nil, err
	}
	req.userReq = dr.(userReq)

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
	case dc.Spec.BringYourOwn != nil:
		spec.BringYourOwn = &api.BringYourOwnDatacenterSpec{}
	case dc.Spec.AWS != nil:
		spec.AWS = &api.AWSDatacenterSpec{
			AvailabilityZone: dc.Spec.AWS.AvailabilityZone,
		}
	}

	return spec, nil
}
