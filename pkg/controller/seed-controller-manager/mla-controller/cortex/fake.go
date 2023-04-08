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

package cortex

import (
	"context"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
)

type FakeCortex struct {
	alertmanagers map[string][]byte
	ruleGroups    map[string]map[kubermaticv1.RuleGroupType]map[string][]byte
}

var _ Client = &FakeCortex{}

func NewFakeClient() *FakeCortex {
	return &FakeCortex{
		alertmanagers: map[string][]byte{},
		ruleGroups:    map[string]map[kubermaticv1.RuleGroupType]map[string][]byte{},
	}
}

func (c *FakeCortex) GetAlertmanagerConfiguration(ctx context.Context, tenant string) ([]byte, error) {
	return c.alertmanagers[tenant], nil
}

func (c *FakeCortex) SetAlertmanagerConfiguration(ctx context.Context, tenant string, config []byte) error {
	c.alertmanagers[tenant] = config

	return nil
}

func (c *FakeCortex) DeleteAlertmanagerConfiguration(ctx context.Context, tenant string) error {
	delete(c.alertmanagers, tenant)

	return nil
}

func (c *FakeCortex) GetRuleGroupConfiguration(ctx context.Context, tenant string, ruleGroupType kubermaticv1.RuleGroupType, groupName string) ([]byte, error) {
	if err := c.validateRuleGroupType(ruleGroupType); err != nil {
		return nil, err
	}

	return c.ruleGroups[tenant][ruleGroupType][groupName], nil
}

func (c *FakeCortex) SetRuleGroupConfiguration(ctx context.Context, tenant string, ruleGroupType kubermaticv1.RuleGroupType, groupName string, config []byte) error {
	if err := c.validateRuleGroupType(ruleGroupType); err != nil {
		return err
	}

	if _, ok := c.ruleGroups[tenant]; !ok {
		c.ruleGroups[tenant] = map[kubermaticv1.RuleGroupType]map[string][]byte{}
	}

	if _, ok := c.ruleGroups[tenant][ruleGroupType]; !ok {
		c.ruleGroups[tenant][ruleGroupType] = map[string][]byte{}
	}

	c.ruleGroups[tenant][ruleGroupType][groupName] = config

	return nil
}

func (c *FakeCortex) DeleteRuleGroupConfiguration(ctx context.Context, tenant string, ruleGroupType kubermaticv1.RuleGroupType, groupName string) error {
	if err := c.validateRuleGroupType(ruleGroupType); err != nil {
		return err
	}

	groups, ok := c.ruleGroups[tenant][ruleGroupType]
	if ok {
		delete(groups, groupName)
		c.ruleGroups[tenant][ruleGroupType] = groups
	}

	return nil
}

func (c *FakeCortex) validateRuleGroupType(ruleGroupType kubermaticv1.RuleGroupType) (err error) {
	_, err = (&cortexClient{}).getBaseCortexRequestURL(ruleGroupType)
	return
}
