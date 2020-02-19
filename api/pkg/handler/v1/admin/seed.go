package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	k8cerrors "github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

// ListSeedsEndpoint returns seed list
func ListSeedEndpoint(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if !userInfo.IsAdmin {
			return nil, k8cerrors.New(http.StatusForbidden, fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", userInfo.Email))
		}
		seedMap, err := seedsGetter()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		var resultList []apiv1.Seed

		for key, value := range seedMap {
			resultList = append(resultList, apiv1.Seed{
				Name:     key,
				SeedSpec: convertSeedSpec(value.Spec),
			})
		}

		return resultList, nil
	}
}

// GetSeedEndpoint returns seed element
func GetSeedEndpoint(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(seedReq)
		if !ok {
			return nil, k8cerrors.NewBadRequest("invalid request")
		}
		seed, err := getSeed(ctx, req, userInfoGetter, seedsGetter)
		if err != nil {
			return nil, err
		}
		return apiv1.Seed{
			Name:     req.Name,
			SeedSpec: convertSeedSpec(seed.Spec),
		}, nil
	}
}

// UpdateSeedEndpoint updates seed element
func UpdateSeedEndpoint(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(updateSeedReq)
		if !ok {
			return nil, k8cerrors.NewBadRequest("invalid request")
		}
		err := req.Validate()
		if err != nil {
			return nil, k8cerrors.NewBadRequest(err.Error())
		}
		seed, err := getSeed(ctx, req.seedReq, userInfoGetter, seedsGetter)
		if err != nil {
			return nil, err
		}
		seedClient, err := seedClientGetter(seed)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		oldSeed := seed.DeepCopy()
		seed.Spec = req.Body.Spec

		if err := seedClient.Patch(ctx, seed, ctrlruntimeclient.MergeFrom(oldSeed)); err != nil {
			return nil, fmt.Errorf("failed to update Seed: %v", err)
		}

		return apiv1.Seed{
			Name:     req.Name,
			SeedSpec: convertSeedSpec(req.Body.Spec),
		}, nil
	}
}

// DeleteSeedEndpoint deletes seed CRD element with the given name from the Kubermatic
func DeleteSeedEndpoint(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, seedClientGetter provider.SeedClientGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(seedReq)
		if !ok {
			return nil, k8cerrors.NewBadRequest("invalid request")
		}
		seed, err := getSeed(ctx, req, userInfoGetter, seedsGetter)
		if err != nil {
			return nil, err
		}
		seedClient, err := seedClientGetter(seed)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if err := seedClient.Delete(ctx, seed); err != nil {
			return nil, fmt.Errorf("failed to delete seed: %v", err)
		}

		return nil, nil
	}
}

func getSeed(ctx context.Context, req seedReq, userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter) (*kubermaticv1.Seed, error) {
	userInfo, err := userInfoGetter(ctx, "")
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}
	if !userInfo.IsAdmin {
		return nil, k8cerrors.New(http.StatusForbidden, fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", userInfo.Email))
	}
	seedMap, err := seedsGetter()
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	result, ok := seedMap[req.Name]
	if !ok {
		return nil, k8cerrors.NewNotFound("Seed", req.Name)
	}
	return result, nil
}

// seedReq defines HTTP request for getSeed
// swagger:parameters getSeed deleteSeed
type seedReq struct {
	// in: path
	// required: true
	Name string `json:"seed_name"`
}

// updateSeedReq defines HTTP request for updateSeed
// swagger:parameters updateSeed
type updateSeedReq struct {
	seedReq
	// in: body
	Body struct {
		Name string `json:"name"`

		Spec kubermaticv1.SeedSpec `json:"spec"`
	}
}

func DecodeUpdateSeedReq(c context.Context, r *http.Request) (interface{}, error) {
	var req updateSeedReq
	seedName, err := DecodeSeedReq(c, r)
	if err != nil {
		return nil, err
	}
	req.seedReq = seedName.(seedReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func DecodeSeedReq(c context.Context, r *http.Request) (interface{}, error) {
	var req seedReq
	name := mux.Vars(r)["seed_name"]
	if name == "" {
		return nil, fmt.Errorf("'seed_name' parameter is required but was not provided")
	}
	req.Name = name

	return req, nil
}

// Validate validates UpdateAdmissionPluginEndpoint request
func (r updateSeedReq) Validate() error {
	if r.Name != r.Body.Name {
		return fmt.Errorf("seed name mismatch, you requested to update Seed = %s but body contains Seed = %s", r.Name, r.Body.Name)
	}
	return nil
}

func convertSeedSpec(seedSpec kubermaticv1.SeedSpec) apiv1.SeedSpec {
	resultSeedSpec := apiv1.SeedSpec{
		Country:  seedSpec.Country,
		Location: seedSpec.Location,
		Kubeconfig: corev1.ObjectReference{
			Kind:            seedSpec.Kubeconfig.Kind,
			Namespace:       seedSpec.Kubeconfig.Namespace,
			Name:            seedSpec.Kubeconfig.Name,
			UID:             seedSpec.Kubeconfig.UID,
			APIVersion:      seedSpec.Kubeconfig.APIVersion,
			ResourceVersion: seedSpec.Kubeconfig.ResourceVersion,
			FieldPath:       seedSpec.Kubeconfig.FieldPath,
		},
		SeedDNSOverwrite: seedSpec.SeedDNSOverwrite,
		ProxySettings:    seedSpec.ProxySettings,
		ExposeStrategy:   seedSpec.ExposeStrategy,
	}
	if seedSpec.Datacenters != nil {
		resultSeedSpec.SeedDatacenters = make(map[string]apiv1.SeedDatacenter)
		for name, datacenter := range seedSpec.Datacenters {
			resultSeedSpec.SeedDatacenters[name] = apiv1.SeedDatacenter{
				Country:  datacenter.Country,
				Location: datacenter.Location,
				Node:     datacenter.Node,
				Spec: apiv1.SeedDatacenterSpec{
					Digitalocean:         datacenter.Spec.Digitalocean,
					BringYourOwn:         datacenter.Spec.BringYourOwn,
					AWS:                  datacenter.Spec.AWS,
					Azure:                datacenter.Spec.Azure,
					Openstack:            datacenter.Spec.Openstack,
					Packet:               datacenter.Spec.Packet,
					Hetzner:              datacenter.Spec.Hetzner,
					VSphere:              datacenter.Spec.VSphere,
					GCP:                  datacenter.Spec.GCP,
					Kubevirt:             datacenter.Spec.Kubevirt,
					Fake:                 datacenter.Spec.Fake,
					RequiredEmailDomain:  datacenter.Spec.RequiredEmailDomain,
					RequiredEmailDomains: datacenter.Spec.RequiredEmailDomains,
					EnforceAuditLogging:  datacenter.Spec.EnforceAuditLogging,
				},
			}
		}
	}

	return resultSeedSpec
}
