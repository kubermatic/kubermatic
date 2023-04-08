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
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"

	kubermaticv1 "k8c.io/api/v3/pkg/apis/kubermatic/v1"
)

const (
	tenantHeaderName               = "X-Scope-OrgID"
	defaultNamespace               = "default"
	alertmanagerAlertsEndpoint     = "/api/v1/alerts"
	metricsRuleGroupConfigEndpoint = "/api/v1/rules"
	logRuleGroupConfigEndpoint     = "/loki/api/v1/rules"
)

type ClientProvider func() Client

type Client interface {
	GetAlertmanagerConfiguration(ctx context.Context, tenant string) ([]byte, error)
	SetAlertmanagerConfiguration(ctx context.Context, tenant string, config []byte) error
	DeleteAlertmanagerConfiguration(ctx context.Context, tenant string) error

	GetRuleGroupConfiguration(ctx context.Context, tenant string, ruleGroupType kubermaticv1.RuleGroupType, groupName string) ([]byte, error)
	SetRuleGroupConfiguration(ctx context.Context, tenant string, ruleGroupType kubermaticv1.RuleGroupType, groupName string, config []byte) error
	DeleteRuleGroupConfiguration(ctx context.Context, tenant string, ruleGroupType kubermaticv1.RuleGroupType, groupName string) error
}

type cortexClient struct {
	httpClient      *http.Client
	alertmanagerURL string
	rulerURL        string
	lokiURL         string
}

var _ Client = &cortexClient{}

func NewClient(httpClient *http.Client, alertmanagerURL string, rulerURL string, lokiURL string) Client {
	return &cortexClient{
		httpClient:      httpClient,
		alertmanagerURL: alertmanagerURL,
		rulerURL:        rulerURL,
		lokiURL:         lokiURL,
	}
}

// https://cortexmetrics.io/docs/api/#get-alertmanager-configuration
func (c *cortexClient) GetAlertmanagerConfiguration(ctx context.Context, tenant string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.alertmanagerURL+alertmanagerAlertsEndpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(tenantHeaderName, tenant)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d, response body: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// https://cortexmetrics.io/docs/api/#set-alertmanager-configuration
func (c *cortexClient) SetAlertmanagerConfiguration(ctx context.Context, tenant string, config []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.alertmanagerURL+alertmanagerAlertsEndpoint, bytes.NewReader(config))
	if err != nil {
		return err
	}
	req.Header.Add(tenantHeaderName, tenant)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("status code: %d, error: %w", resp.StatusCode, err)
		}

		return fmt.Errorf("status code: %d, response body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// https://cortexmetrics.io/docs/api/#delete-alertmanager-configuration
func (c *cortexClient) DeleteAlertmanagerConfiguration(ctx context.Context, tenant string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.alertmanagerURL+alertmanagerAlertsEndpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Add(tenantHeaderName, tenant)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("status code: %d, error: %w", resp.StatusCode, err)
		}

		return fmt.Errorf("status code: %d, response body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// https://cortexmetrics.io/docs/api/#get-rule-group
func (c *cortexClient) GetRuleGroupConfiguration(ctx context.Context, tenant string, ruleGroupType kubermaticv1.RuleGroupType, groupName string) ([]byte, error) {
	url, err := c.getCortexRequestURL(ruleGroupType, groupName)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add(tenantHeaderName, tenant)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status code: %d, response body: %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// https://cortexmetrics.io/docs/api/#set-rule-group
// The group name parameter is only in the interface to make the mock client work, the
// actual client here does not need it for the POST request.
func (c *cortexClient) SetRuleGroupConfiguration(ctx context.Context, tenant string, ruleGroupType kubermaticv1.RuleGroupType, _ string, config []byte) error {
	url, err := c.getBaseCortexRequestURL(ruleGroupType)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(config))
	if err != nil {
		return err
	}
	req.Header.Add(tenantHeaderName, tenant)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	if resp.StatusCode != http.StatusAccepted {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		return fmt.Errorf("status code: %d, response body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// https://cortexmetrics.io/docs/api/#delete-rule-group
func (c *cortexClient) DeleteRuleGroupConfiguration(ctx context.Context, tenant string, ruleGroupType kubermaticv1.RuleGroupType, groupName string) error {
	url, err := c.getCortexRequestURL(ruleGroupType, groupName)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Add(tenantHeaderName, tenant)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil
	}

	if resp.StatusCode != http.StatusAccepted {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("failed to read response body: %w", err)
		}

		return fmt.Errorf("status code: %d, response body: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (c *cortexClient) getBaseCortexRequestURL(ruleGroupType kubermaticv1.RuleGroupType) (string, error) {
	switch ruleGroupType {
	case kubermaticv1.RuleGroupTypeLogs:
		return fmt.Sprintf("%s%s/%s", c.lokiURL, logRuleGroupConfigEndpoint, defaultNamespace), nil
	case kubermaticv1.RuleGroupTypeMetrics:
		return fmt.Sprintf("%s%s/%s", c.rulerURL, metricsRuleGroupConfigEndpoint, defaultNamespace), nil
	default:
		return "", fmt.Errorf("unknown rule group type %q", ruleGroupType)
	}
}

func (c *cortexClient) getCortexRequestURL(ruleGroupType kubermaticv1.RuleGroupType, groupName string) (string, error) {
	base, err := c.getBaseCortexRequestURL(ruleGroupType)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", base, groupName), nil
}
