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

package rulegroup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticcrdv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v2/cluster"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	"sigs.k8s.io/yaml"
)

// getReq defines HTTP request for getting ruleGroup
// swagger:parameters getRuleGroup
type getReq struct {
	cluster.GetClusterReq
	// in: path
	// required: true
	RuleGroupID string `json:"rulegroup_id"`
}

// listReq defines HTTP request for listing ruleGroups
// swagger:parameters listRuleGroups
type listReq struct {
	cluster.GetClusterReq
	// in: query
	Type string `json:"type,omitempty"`
}

// createReq defines HTTP request for creating ruleGroup
// swagger:parameters createRuleGroup
type createReq struct {
	cluster.GetClusterReq
	// in: body
	// required: true
	Body apiv2.RuleGroup
}

func (req *createReq) validate() (ruleGroupName string, err error) {
	return getRuleGroupNameInData(req.Body.Data)
}

// updateReq defines HTTP request for updating ruleGroup
// swagger:parameters updateRuleGroup
type updateReq struct {
	cluster.GetClusterReq
	// in: path
	// required: true
	RuleGroupID string `json:"rulegroup_id"`
	// in: body
	// required: true
	Body apiv2.RuleGroup
}

func (req *updateReq) validate() error {
	ruleGroupNameInData, err := getRuleGroupNameInData(req.Body.Data)
	if err != nil {
		return err
	}
	if req.RuleGroupID != ruleGroupNameInData {
		return fmt.Errorf("cannot update rule group name")
	}
	return nil
}

// deleteReq defines HTTP request for deleting ruleGroup
// swagger:parameters deleteRuleGroup
type deleteReq struct {
	cluster.GetClusterReq
	// in: path
	// required: true
	RuleGroupID string `json:"rulegroup_id"`
}

func DecodeGetReq(c context.Context, r *http.Request) (interface{}, error) {
	var req getReq
	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = cr.(cluster.GetClusterReq)

	ruleGroupID, err := decodeRuleGroupID(r)
	if err != nil {
		return nil, err
	}
	req.RuleGroupID = ruleGroupID
	return req, nil
}

func DecodeListReq(c context.Context, r *http.Request) (interface{}, error) {
	var req listReq
	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = cr.(cluster.GetClusterReq)
	req.Type = r.URL.Query().Get("type")
	if len(req.Type) > 0 {
		if req.Type == string(kubermaticcrdv1.RuleGroupTypeMetrics) {
			return req, nil
		}
		return nil, utilerrors.NewBadRequest("wrong query parameter, unsupported type: %s, supported value: %s", req.Type, kubermaticcrdv1.RuleGroupTypeMetrics)
	}
	return req, nil
}

func DecodeCreateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createReq
	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = cr.(cluster.GetClusterReq)

	if err = json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}
	return req, nil
}

func DecodeUpdateReq(c context.Context, r *http.Request) (interface{}, error) {
	var req updateReq
	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = cr.(cluster.GetClusterReq)
	ruleGroupID, err := decodeRuleGroupID(r)
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
	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = cr.(cluster.GetClusterReq)

	ruleGroupID, err := decodeRuleGroupID(r)
	if err != nil {
		return nil, err
	}
	req.RuleGroupID = ruleGroupID
	return req, nil
}

func decodeRuleGroupID(r *http.Request) (string, error) {
	ruleGroupID := mux.Vars(r)["rulegroup_id"]
	if ruleGroupID == "" {
		return "", utilerrors.NewBadRequest("rulegroup_id parameter is required but was not provided")
	}
	return ruleGroupID, nil
}

func getRuleGroupNameInData(data []byte) (string, error) {
	bodyMap := map[string]interface{}{}
	if err := yaml.Unmarshal(data, &bodyMap); err != nil {
		return "", fmt.Errorf("cannot unmarshal rule group data in yaml: %w", err)
	}
	ruleGroupName, ok := bodyMap["name"].(string)
	if !ok {
		return "", fmt.Errorf("rule group name cannot be parsed in the data")
	}
	if ruleGroupName == "" {
		return "", fmt.Errorf("rule group name cannot be empty in the data")
	}
	return ruleGroupName, nil
}
