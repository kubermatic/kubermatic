/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package rulegroupadmin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticcrdv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v2/rulegroup"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"
)

// getReq defines HTTP request for getting ruleGroup
// swagger:parameters getAdminRuleGroup
type getReq struct {
	// in: path
	// required: true
	SeedName string `json:"seed_name"`
	// in: path
	// required: true
	RuleGroupID string `json:"rulegroup_id"`
}

func (req getReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		SeedName: req.SeedName,
	}
}

// listReq defines HTTP request for listing ruleGroups
// swagger:parameters listAdminRuleGroups
type listReq struct {
	// in: path
	// required: true
	SeedName string `json:"seed_name"`
	// in: query
	Type string `json:"type,omitempty"`
}

func (req listReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		SeedName: req.SeedName,
	}
}

// createReq defines HTTP request for creating ruleGroup
// swagger:parameters createAdminRuleGroup
type createReq struct {
	// in: path
	// required: true
	SeedName string `json:"seed_name"`
	// in: body
	// required: true
	Body apiv2.RuleGroup
}

func (req createReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		SeedName: req.SeedName,
	}
}

func (req createReq) validate() (ruleGroupName string, err error) {
	return rulegroup.GetRuleGroupNameInData(req.Body.Data)
}

// updateReq defines HTTP request for updating ruleGroup
// swagger:parameters updateAdminRuleGroup
type updateReq struct {
	// in: path
	// required: true
	SeedName string `json:"seed_name"`
	// in: path
	// required: true
	RuleGroupID string `json:"rulegroup_id"`
	// in: body
	// required: true
	Body apiv2.RuleGroup
}

func (req updateReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		SeedName: req.SeedName,
	}
}

func (req updateReq) validate() error {
	ruleGroupNameInData, err := rulegroup.GetRuleGroupNameInData(req.Body.Data)
	if err != nil {
		return err
	}
	if req.RuleGroupID != ruleGroupNameInData {
		return fmt.Errorf("cannot update rule group name")
	}
	return nil
}

// deleteReq defines HTTP request for deleting ruleGroup
// swagger:parameters deleteAdminRuleGroup
type deleteReq struct {
	// in: path
	// required: true
	SeedName string `json:"seed_name"`
	// in: path
	// required: true
	RuleGroupID string `json:"rulegroup_id"`
}

func (req deleteReq) GetSeedCluster() apiv1.SeedCluster {
	return apiv1.SeedCluster{
		SeedName: req.SeedName,
	}
}

func DecodeGetReq(c context.Context, r *http.Request) (interface{}, error) {
	var req getReq
	seedName, err := getSeedName(r)
	if err != nil {
		return nil, err
	}
	req.SeedName = seedName
	ruleGroupID, err := rulegroup.DecodeRuleGroupID(r)
	if err != nil {
		return nil, err
	}
	req.RuleGroupID = ruleGroupID
	return req, nil
}

func DecodeListReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listReq
	seedName, err := getSeedName(r)
	if err != nil {
		return nil, err
	}
	req.SeedName = seedName
	req.Type = r.URL.Query().Get("type")
	if len(req.Type) > 0 {
		if (req.Type == string(kubermaticcrdv1.RuleGroupTypeMetrics)) ||
			(req.Type == string(kubermaticcrdv1.RuleGroupTypeLogs)) {
			return req, nil
		}
		return nil, utilerrors.NewBadRequest("wrong query parameter, unsupported type: %s, supported value: %s, %s", req.Type, kubermaticcrdv1.RuleGroupTypeMetrics, kubermaticcrdv1.RuleGroupTypeLogs)
	}
	return req, nil
}

func DecodeCreateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createReq
	seedName, err := getSeedName(r)
	if err != nil {
		return nil, err
	}
	req.SeedName = seedName
	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}
	return req, nil
}

func DecodeUpdateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req updateReq
	seedName, err := getSeedName(r)
	if err != nil {
		return nil, err
	}
	req.SeedName = seedName
	ruleGroupID, err := rulegroup.DecodeRuleGroupID(r)
	if err != nil {
		return nil, err
	}
	req.RuleGroupID = ruleGroupID

	if err = json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}
	return req, nil
}

func DecodeDeleteReq(c context.Context, r *http.Request) (interface{}, error) {
	var req deleteReq
	seedName, err := getSeedName(r)
	if err != nil {
		return nil, err
	}
	req.SeedName = seedName
	ruleGroupID, err := rulegroup.DecodeRuleGroupID(r)
	if err != nil {
		return nil, err
	}
	req.RuleGroupID = ruleGroupID
	return req, nil
}

func getSeedName(r *http.Request) (string, error) {
	seedName := mux.Vars(r)["seed_name"]
	if seedName == "" {
		return "", fmt.Errorf("'seed_name' parameter is required but was not provided")
	}
	return seedName, nil
}
