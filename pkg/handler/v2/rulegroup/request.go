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

	"sigs.k8s.io/yaml"
)

// getReq defines HTTP request for getting ruleGroup
// swagger:parameters getRuleGroup
type getReq struct {
	cluster.GetClusterReq
	// in: path
	// required: true
	Name string `json:"rule_group_name"`
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

// updateReq defines HTTP request for updating ruleGroup
// swagger:parameters updateRuleGroup
type updateReq struct {
	cluster.GetClusterReq
	// in: path
	// required: true
	Name string `json:"rule_group_name"`
	// in: body
	// required: true
	Body apiv2.RuleGroup
}

func (req *updateReq) validateUpdateReq() error {
	ruleGroupNameInData, err := getRuleGroupNameInData(req.Body.Data)
	if err != nil {
		return err
	}
	if req.Name != ruleGroupNameInData {
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
	Name string `json:"rule_group_name"`
}

func DecodeGetReq(c context.Context, r *http.Request) (interface{}, error) {
	var req getReq
	cr, err := cluster.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.GetClusterReq = cr.(cluster.GetClusterReq)

	ruleGroupName, err := decodeRuleGroupName(r)
	if err != nil {
		return nil, err
	}
	req.Name = ruleGroupName
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
		return nil, fmt.Errorf("wrong query parametr, unsupported type: %s, supported value: %s", req.Type, kubermaticcrdv1.RuleGroupTypeMetrics)
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
	ruleGroupName, err := decodeRuleGroupName(r)
	if err != nil {
		return nil, err
	}
	req.Name = ruleGroupName

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

	ruleGroupName, err := decodeRuleGroupName(r)
	if err != nil {
		return nil, err
	}
	req.Name = ruleGroupName
	return req, nil
}

func decodeRuleGroupName(r *http.Request) (string, error) {
	ruleGroupName := mux.Vars(r)["rule_group_name"]
	if ruleGroupName == "" {
		return "", fmt.Errorf("rule_group_name parameter is required but was not provided")
	}
	return ruleGroupName, nil
}

func getRuleGroupNameInData(data []byte) (string, error) {
	bodyMap := map[string]interface{}{}
	if err := yaml.Unmarshal(data, &bodyMap); err != nil {
		return "", fmt.Errorf("cannot unmarshal rule group data in yaml: %w", err)
	}
	ruleGroupName := bodyMap["name"].(string)
	if ruleGroupName == "" {
		return "", fmt.Errorf("rule group name cannot be empty in the data")
	}
	return ruleGroupName, nil
}
