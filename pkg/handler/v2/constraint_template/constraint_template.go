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

package constrainttemplate

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/open-policy-agent/frameworks/constraint/pkg/apis/templates/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/util/errors"
)

func ListEndpoint(constraintTemplateProvider provider.ConstraintTemplateProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		constraintTemplateList, err := constraintTemplateProvider.List()
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		apiCT := make([]*apiv2.ConstraintTemplate, 0)
		for _, ct := range constraintTemplateList.Items {
			apiCT = append(apiCT, convertCTToAPI(&ct))
		}

		return apiCT, nil
	}
}

func GetEndpoint(constraintTemplateProvider provider.ConstraintTemplateProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req := request.(constraintTemplateReq)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		constraintTemplate, err := constraintTemplateProvider.Get(req.Name)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertCTToAPI(constraintTemplate), nil
	}
}

func convertCTToAPI(ct *kubermaticv1.ConstraintTemplate) *apiv2.ConstraintTemplate {
	return &apiv2.ConstraintTemplate{
		Name: ct.Name,
		Spec: ct.Spec,
	}
}

// constraintTemplateReq represents a request for a specific constraintTemplate
// swagger:parameters getConstraintTemplate deleteConstraintTemplate
type constraintTemplateReq struct {
	// in: path
	// required: true
	Name string `json:"ct_name"`
}

func DecodeConstraintTemplateRequest(c context.Context, r *http.Request) (interface{}, error) {
	name := mux.Vars(r)["ct_name"]
	if name == "" {
		return "", fmt.Errorf("'ct_name' parameter is required but was not provided")
	}

	return constraintTemplateReq{
		Name: name,
	}, nil
}

// Validate validates constraintTemplate request
func (req constraintTemplateReq) Validate() error {
	if len(req.Name) == 0 {
		return fmt.Errorf("the constraint template name cannot be empty")
	}
	return nil
}

func CreateEndpoint(userInfoGetter provider.UserInfoGetter, constraintTemplateProvider provider.ConstraintTemplateProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createConstraintTemplateReq)

		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !adminUserInfo.IsAdmin {
			return nil, errors.New(http.StatusForbidden,
				fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", adminUserInfo.Email))
		}

		ct := &kubermaticv1.ConstraintTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name: req.Body.Name,
			},
			Spec: req.Body.ConstraintTemplateSpec,
		}

		if err := validateConstraintTemplate(ct); err != nil {
			return nil, errors.New(http.StatusBadRequest, fmt.Sprintf("create ct validation failed: %v", err))
		}

		ct, err = constraintTemplateProvider.Create(ct)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertCTToAPI(ct), nil
	}
}

// createConstraintTemplateReq represents a request for creating constraint templates
// swagger:parameters createConstraintTemplate
type createConstraintTemplateReq struct {
	// in: body
	Body ctBody
}

type ctBody struct {
	// Name of the constraint template
	Name string `json:"name"`
	// ConstraintTemplateSpec Spec of the constraint template
	ConstraintTemplateSpec v1beta1.ConstraintTemplateSpec `json:"spec"`
}

func DecodeCreateConstraintTemplateRequest(c context.Context, r *http.Request) (interface{}, error) {
	var req createConstraintTemplateReq

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func PatchEndpoint(userInfoGetter provider.UserInfoGetter, constraintTemplateProvider provider.ConstraintTemplateProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(patchConstraintTemplateReq)
		if err := req.Validate(); err != nil {
			return nil, errors.NewBadRequest(err.Error())
		}

		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if !adminUserInfo.IsAdmin {
			return nil, errors.New(http.StatusForbidden,
				fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", adminUserInfo.Email))
		}

		// get CT
		originalCT, err := constraintTemplateProvider.Get(req.Name)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		originalAPICT := convertCTToAPI(originalCT)

		// patch
		originalJSON, err := json.Marshal(originalAPICT)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to convert current ct: %v", err))
		}

		patchedJSON, err := jsonpatch.MergePatch(originalJSON, req.Patch)
		if err != nil {
			return nil, errors.New(http.StatusBadRequest, fmt.Sprintf("failed to merge patch ct: %v", err))
		}

		var patched *apiv2.ConstraintTemplate
		err = json.Unmarshal(patchedJSON, &patched)
		if err != nil {
			return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("failed to unmarshall patch ct: %v", err))
		}

		// validate
		if patched.Name != originalCT.Name {
			return nil, errors.New(http.StatusBadRequest, fmt.Sprintf("Changing ct name is not allowed: %q to %q", originalCT.Name, patched.Name))
		}

		patchedCT := &kubermaticv1.ConstraintTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name: patched.Name,
			},
			Spec: patched.Spec,
		}

		if err := validateConstraintTemplate(patchedCT); err != nil {
			return nil, errors.New(http.StatusBadRequest, fmt.Sprintf("patched ct validation failed: %v", err))
		}

		// apply patch
		patchedCT, err = constraintTemplateProvider.Update(patchedCT)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return convertCTToAPI(patchedCT), nil
	}
}

func validateConstraintTemplate(ct *kubermaticv1.ConstraintTemplate) error {
	if ct.Name != strings.ToLower(ct.Spec.CRD.Spec.Names.Kind) {
		return fmt.Errorf("template's name %s is not equal to the lowercase of CRD's Kind: %s", ct.Name,
			ct.Spec.CRD.Spec.Names.Kind)
	}
	return nil
}

// patchConstraintTemplateReq defines HTTP request for patching constraint templates
// swagger:parameters patchConstraintTemplate
type patchConstraintTemplateReq struct {
	constraintTemplateReq
	// in: body
	Patch json.RawMessage
}

// DecodePatchCTReq decodes http request into patchConstraintTemplateReq
func DecodePatchConstraintTemplateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req patchConstraintTemplateReq

	ctReq, err := DecodeConstraintTemplateRequest(c, r)
	if err != nil {
		return nil, err
	}
	req.constraintTemplateReq = ctReq.(constraintTemplateReq)

	if req.Patch, err = ioutil.ReadAll(r.Body); err != nil {
		return nil, err
	}

	return req, nil
}

func DeleteEndpoint(userInfoGetter provider.UserInfoGetter, constraintTemplateProvider provider.ConstraintTemplateProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(constraintTemplateReq)

		adminUserInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, err
		}
		if !adminUserInfo.IsAdmin {
			return nil, errors.New(http.StatusForbidden,
				fmt.Sprintf("forbidden: \"%s\" doesn't have admin rights", adminUserInfo.Email))
		}

		ct := &kubermaticv1.ConstraintTemplate{
			ObjectMeta: metav1.ObjectMeta{
				Name: req.Name,
			},
		}

		err = constraintTemplateProvider.Delete(ct)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}

		return nil, nil
	}
}
