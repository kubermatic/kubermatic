/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package admin

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/handler/v1/dc"
	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var resourceNameValidator = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)

// CreateSeedEndpoint creates seed object.
func CreateSeedEndpoint(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, seedProvider provider.SeedProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if !userInfo.IsAdmin {
			return nil, utilerrors.New(http.StatusForbidden, fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", userInfo.Email))
		}

		req, ok := request.(createSeedReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}

		err = req.Validate(seedsGetter)
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		newSeed := genSeedFromRequest(req)

		config, err := base64.StdEncoding.DecodeString(req.Body.Spec.Kubeconfig)
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}

		if err := seedProvider.CreateOrUpdateKubeconfigSecretForSeed(ctx, newSeed, config); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		seed, err := seedProvider.CreateUnsecured(ctx, newSeed)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return apiv1.Seed{
			Name:     req.Body.Name,
			SeedSpec: convertSeedSpec(seed.Spec, req.Body.Name),
		}, nil
	}
}

func genSeedFromRequest(req createSeedReq) *kubermaticv1.Seed {
	newSeed := &kubermaticv1.Seed{}
	newSeed.Name = req.Body.Name
	newSeed.Namespace = resources.KubermaticNamespace
	newSeed.Spec = kubermaticv1.SeedSpec{
		Country:                req.Body.Spec.Country,
		Location:               req.Body.Spec.Location,
		Kubeconfig:             kubermaticv1.SeedKubeconfigReference{},
		SeedDNSOverwrite:       req.Body.Spec.SeedDNSOverwrite,
		DefaultClusterTemplate: req.Body.Spec.DefaultClusterTemplate,
		ExposeStrategy:         req.Body.Spec.ExposeStrategy,
	}
	if req.Body.Spec.ProxySettings != nil {
		newSeed.Spec.ProxySettings = &kubermaticv1.ProxySettings{}
		if req.Body.Spec.ProxySettings.NoProxy != "" {
			newSeed.Spec.ProxySettings.NoProxy = kubermaticv1.NewProxyValue(req.Body.Spec.ProxySettings.NoProxy)
		}
		if req.Body.Spec.ProxySettings.HTTPProxy != "" {
			newSeed.Spec.ProxySettings.HTTPProxy = kubermaticv1.NewProxyValue(req.Body.Spec.ProxySettings.HTTPProxy)
		}
	}
	if req.Body.Spec.MLA != nil {
		newSeed.Spec.MLA = &kubermaticv1.SeedMLASettings{
			UserClusterMLAEnabled: req.Body.Spec.MLA.UserClusterMLAEnabled,
		}
	}
	return newSeed
}

// ListSeedsEndpoint returns seed list.
func ListSeedEndpoint(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if !userInfo.IsAdmin {
			return nil, utilerrors.New(http.StatusForbidden, fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", userInfo.Email))
		}
		seedMap, err := seedsGetter()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		var resultList []apiv1.Seed

		for key, value := range seedMap {
			resultList = append(resultList, apiv1.Seed{
				Name:     key,
				SeedSpec: convertSeedSpec(value.Spec, key),
			})
		}

		return resultList, nil
	}
}

// GetSeedEndpoint returns seed element.
func GetSeedEndpoint(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(seedReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		seed, err := getSeed(ctx, req, userInfoGetter, seedsGetter)
		if err != nil {
			return nil, err
		}
		return apiv1.Seed{
			Name:     req.Name,
			SeedSpec: convertSeedSpec(seed.Spec, req.Name),
		}, nil
	}
}

// UpdateSeedEndpoint updates seed element.
func UpdateSeedEndpoint(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, seedProvider provider.SeedProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(updateSeedReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		err := req.Validate()
		if err != nil {
			return nil, utilerrors.NewBadRequest(err.Error())
		}
		seed, err := getSeed(ctx, req.seedReq, userInfoGetter, seedsGetter)
		if err != nil {
			return nil, err
		}

		if req.Body.RawKubeconfig != "" {
			config, err := base64.StdEncoding.DecodeString(req.Body.RawKubeconfig)
			if err != nil {
				return nil, utilerrors.NewBadRequest(err.Error())
			}

			if err := seedProvider.CreateOrUpdateKubeconfigSecretForSeed(ctx, seed, config); err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}
		}

		// ensure it's not possible to randomly override the kubeconfig ref
		req.Body.Spec.Kubeconfig = seed.Spec.Kubeconfig

		originalJSON, err := json.Marshal(seed.Spec)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("failed to convert current seed to JSON: %v", err))
		}
		newJSON, err := json.Marshal(req.Body.Spec)
		if err != nil {
			return nil, utilerrors.New(http.StatusBadRequest, fmt.Sprintf("failed to convert patch seed to JSON: %v", err))
		}

		patchedJSON, err := jsonpatch.MergePatch(originalJSON, newJSON)
		if err != nil {
			return nil, utilerrors.New(http.StatusBadRequest, fmt.Sprintf("failed to merge patch: %v", err))
		}

		var seedSpec *kubermaticv1.SeedSpec
		err = json.Unmarshal(patchedJSON, &seedSpec)
		if err != nil {
			return nil, utilerrors.New(http.StatusInternalServerError, fmt.Sprintf("failed unmarshall patched seed: %v", err))
		}
		seed.Spec = *seedSpec

		updatedSeed, err := seedProvider.UpdateUnsecured(ctx, seed)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return apiv1.Seed{
			Name:     req.Name,
			SeedSpec: convertSeedSpec(updatedSeed.Spec, req.Name),
		}, nil
	}
}

// DeleteSeedEndpoint deletes seed CRD element with the given name from the Kubermatic.
func DeleteSeedEndpoint(userInfoGetter provider.UserInfoGetter, seedsGetter provider.SeedsGetter, masterClient ctrlruntimeclient.Client) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(seedReq)
		if !ok {
			return nil, utilerrors.NewBadRequest("invalid request")
		}
		seed, err := getSeed(ctx, req, userInfoGetter, seedsGetter)
		if err != nil {
			return nil, err
		}

		if err := masterClient.Delete(ctx, seed); err != nil {
			return nil, fmt.Errorf("failed to delete seed: %w", err)
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
		return nil, utilerrors.New(http.StatusForbidden, fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", userInfo.Email))
	}
	seedMap, err := seedsGetter()
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	result, ok := seedMap[req.Name]
	if !ok {
		return nil, utilerrors.NewNotFound("Seed", req.Name)
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
		Name string                `json:"name"`
		Spec kubermaticv1.SeedSpec `json:"spec"`
		// RawKubeconfig raw kubeconfig decoded to base64
		RawKubeconfig string `json:"raw_kubeconfig,omitempty"`
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

// Validate validates UpdateAdmissionPluginEndpoint request.
func (r updateSeedReq) Validate() error {
	if r.Name != r.Body.Name {
		return fmt.Errorf("seed name mismatch, you requested to update Seed %q but body contains Seed %q", r.Name, r.Body.Name)
	}

	if r.Body.Spec.EtcdBackupRestore == nil {
		return nil
	}

	defaultDestination := r.Body.Spec.EtcdBackupRestore.DefaultDestination
	if len(defaultDestination) > 0 {
		if !resourceNameValidator.MatchString(defaultDestination) {
			return fmt.Errorf("default destination name is invalid, must match %s", resourceNameValidator.String())
		}
	}

	for k := range r.Body.Spec.EtcdBackupRestore.Destinations {
		if len(k) > 0 {
			if !resourceNameValidator.MatchString(k) {
				return fmt.Errorf("destination name is invalid, must match %s", resourceNameValidator.String())
			}
		}
	}
	return nil
}

// createSeedReq defines HTTP request for createSeed
// swagger:parameters createSeed
type createSeedReq struct {
	// in: body
	Body struct {
		Name string               `json:"name"`
		Spec apiv1.CreateSeedSpec `json:"spec"`
	}
}

func DecodeCreateSeedReq(_ context.Context, r *http.Request) (interface{}, error) {
	var req createSeedReq
	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func (r createSeedReq) Validate(seedsGetter provider.SeedsGetter) error {
	if r.Body.Name == "" {
		return fmt.Errorf("the seed name cannot be empty")
	}
	if r.Body.Spec.Kubeconfig == "" {
		return fmt.Errorf("the kubeconfig cannot be empty")
	}
	seedMap, err := seedsGetter()
	if err != nil {
		return common.KubernetesErrorToHTTPError(err)
	}

	_, ok := seedMap[r.Body.Name]
	if ok {
		return fmt.Errorf("seed with the name %s already exists", r.Body.Name)
	}

	return nil
}

func convertSeedSpec(seedSpec kubermaticv1.SeedSpec, seedName string) apiv1.SeedSpec {
	resultSeedSpec := apiv1.SeedSpec{
		Country:  seedSpec.Country,
		Location: seedSpec.Location,
		Kubeconfig: corev1.ObjectReference{
			Name:      seedSpec.Kubeconfig.Name,
			FieldPath: seedSpec.Kubeconfig.FieldPath,
		},
		SeedDNSOverwrite:  seedSpec.SeedDNSOverwrite,
		ProxySettings:     seedSpec.ProxySettings,
		ExposeStrategy:    seedSpec.ExposeStrategy,
		MLA:               seedSpec.MLA,
		EtcdBackupRestore: seedSpec.EtcdBackupRestore,
	}
	if seedSpec.Datacenters != nil {
		resultSeedSpec.SeedDatacenters = make(map[string]apiv1.Datacenter)
		for name, datacenter := range seedSpec.Datacenters {
			dcSpec, err := dc.ConvertInternalDCToExternalSpec(&datacenter, seedName)
			if err != nil {
				log.Logger.Errorf("api spec error in dc %q: %v", name, err)
				continue
			}
			resultSeedSpec.SeedDatacenters[name] = apiv1.Datacenter{
				Metadata: apiv1.DatacenterMeta{
					Name: name,
				},
				Spec: *dcSpec,
			}
		}
	}

	return resultSeedSpec
}
