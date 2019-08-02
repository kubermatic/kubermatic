package network

// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Code generated by Microsoft (R) AutoRest Code Generator.
// Changes may cause incorrect behavior and will be lost if the code is regenerated.

import (
	"context"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/validation"
	"github.com/Azure/go-autorest/tracing"
	"net/http"
)

// InboundNatRulesClient is the network Client
type InboundNatRulesClient struct {
	BaseClient
}

// NewInboundNatRulesClient creates an instance of the InboundNatRulesClient client.
func NewInboundNatRulesClient(subscriptionID string) InboundNatRulesClient {
	return NewInboundNatRulesClientWithBaseURI(DefaultBaseURI, subscriptionID)
}

// NewInboundNatRulesClientWithBaseURI creates an instance of the InboundNatRulesClient client.
func NewInboundNatRulesClientWithBaseURI(baseURI string, subscriptionID string) InboundNatRulesClient {
	return InboundNatRulesClient{NewWithBaseURI(baseURI, subscriptionID)}
}

// CreateOrUpdate creates or updates a load balancer inbound nat rule.
// Parameters:
// resourceGroupName - the name of the resource group.
// loadBalancerName - the name of the load balancer.
// inboundNatRuleName - the name of the inbound nat rule.
// inboundNatRuleParameters - parameters supplied to the create or update inbound nat rule operation.
func (client InboundNatRulesClient) CreateOrUpdate(ctx context.Context, resourceGroupName string, loadBalancerName string, inboundNatRuleName string, inboundNatRuleParameters InboundNatRule) (result InboundNatRulesCreateOrUpdateFuture, err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/InboundNatRulesClient.CreateOrUpdate")
		defer func() {
			sc := -1
			if result.Response() != nil {
				sc = result.Response().StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	if err := validation.Validate([]validation.Validation{
		{TargetValue: inboundNatRuleParameters,
			Constraints: []validation.Constraint{{Target: "inboundNatRuleParameters.InboundNatRulePropertiesFormat", Name: validation.Null, Rule: false,
				Chain: []validation.Constraint{{Target: "inboundNatRuleParameters.InboundNatRulePropertiesFormat.BackendIPConfiguration", Name: validation.Null, Rule: false,
					Chain: []validation.Constraint{{Target: "inboundNatRuleParameters.InboundNatRulePropertiesFormat.BackendIPConfiguration.InterfaceIPConfigurationPropertiesFormat", Name: validation.Null, Rule: false,
						Chain: []validation.Constraint{{Target: "inboundNatRuleParameters.InboundNatRulePropertiesFormat.BackendIPConfiguration.InterfaceIPConfigurationPropertiesFormat.PublicIPAddress", Name: validation.Null, Rule: false,
							Chain: []validation.Constraint{{Target: "inboundNatRuleParameters.InboundNatRulePropertiesFormat.BackendIPConfiguration.InterfaceIPConfigurationPropertiesFormat.PublicIPAddress.PublicIPAddressPropertiesFormat", Name: validation.Null, Rule: false,
								Chain: []validation.Constraint{{Target: "inboundNatRuleParameters.InboundNatRulePropertiesFormat.BackendIPConfiguration.InterfaceIPConfigurationPropertiesFormat.PublicIPAddress.PublicIPAddressPropertiesFormat.IPConfiguration", Name: validation.Null, Rule: false,
									Chain: []validation.Constraint{{Target: "inboundNatRuleParameters.InboundNatRulePropertiesFormat.BackendIPConfiguration.InterfaceIPConfigurationPropertiesFormat.PublicIPAddress.PublicIPAddressPropertiesFormat.IPConfiguration.IPConfigurationPropertiesFormat", Name: validation.Null, Rule: false,
										Chain: []validation.Constraint{{Target: "inboundNatRuleParameters.InboundNatRulePropertiesFormat.BackendIPConfiguration.InterfaceIPConfigurationPropertiesFormat.PublicIPAddress.PublicIPAddressPropertiesFormat.IPConfiguration.IPConfigurationPropertiesFormat.PublicIPAddress", Name: validation.Null, Rule: false, Chain: nil}}},
									}},
								}},
							}},
						}},
					}},
				}}}}}); err != nil {
		return result, validation.NewError("network.InboundNatRulesClient", "CreateOrUpdate", err.Error())
	}

	req, err := client.CreateOrUpdatePreparer(ctx, resourceGroupName, loadBalancerName, inboundNatRuleName, inboundNatRuleParameters)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InboundNatRulesClient", "CreateOrUpdate", nil, "Failure preparing request")
		return
	}

	result, err = client.CreateOrUpdateSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InboundNatRulesClient", "CreateOrUpdate", result.Response(), "Failure sending request")
		return
	}

	return
}

// CreateOrUpdatePreparer prepares the CreateOrUpdate request.
func (client InboundNatRulesClient) CreateOrUpdatePreparer(ctx context.Context, resourceGroupName string, loadBalancerName string, inboundNatRuleName string, inboundNatRuleParameters InboundNatRule) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"inboundNatRuleName": autorest.Encode("path", inboundNatRuleName),
		"loadBalancerName":   autorest.Encode("path", loadBalancerName),
		"resourceGroupName":  autorest.Encode("path", resourceGroupName),
		"subscriptionId":     autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-06-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsPut(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/loadBalancers/{loadBalancerName}/inboundNatRules/{inboundNatRuleName}", pathParameters),
		autorest.WithJSON(inboundNatRuleParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// CreateOrUpdateSender sends the CreateOrUpdate request. The method will close the
// http.Response Body if it receives an error.
func (client InboundNatRulesClient) CreateOrUpdateSender(req *http.Request) (future InboundNatRulesCreateOrUpdateFuture, err error) {
	var resp *http.Response
	resp, err = autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
	if err != nil {
		return
	}
	future.Future, err = azure.NewFutureFromResponse(resp)
	return
}

// CreateOrUpdateResponder handles the response to the CreateOrUpdate request. The method always
// closes the http.Response Body.
func (client InboundNatRulesClient) CreateOrUpdateResponder(resp *http.Response) (result InboundNatRule, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusCreated),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// Delete deletes the specified load balancer inbound nat rule.
// Parameters:
// resourceGroupName - the name of the resource group.
// loadBalancerName - the name of the load balancer.
// inboundNatRuleName - the name of the inbound nat rule.
func (client InboundNatRulesClient) Delete(ctx context.Context, resourceGroupName string, loadBalancerName string, inboundNatRuleName string) (result InboundNatRulesDeleteFuture, err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/InboundNatRulesClient.Delete")
		defer func() {
			sc := -1
			if result.Response() != nil {
				sc = result.Response().StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	req, err := client.DeletePreparer(ctx, resourceGroupName, loadBalancerName, inboundNatRuleName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InboundNatRulesClient", "Delete", nil, "Failure preparing request")
		return
	}

	result, err = client.DeleteSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InboundNatRulesClient", "Delete", result.Response(), "Failure sending request")
		return
	}

	return
}

// DeletePreparer prepares the Delete request.
func (client InboundNatRulesClient) DeletePreparer(ctx context.Context, resourceGroupName string, loadBalancerName string, inboundNatRuleName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"inboundNatRuleName": autorest.Encode("path", inboundNatRuleName),
		"loadBalancerName":   autorest.Encode("path", loadBalancerName),
		"resourceGroupName":  autorest.Encode("path", resourceGroupName),
		"subscriptionId":     autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-06-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsDelete(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/loadBalancers/{loadBalancerName}/inboundNatRules/{inboundNatRuleName}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// DeleteSender sends the Delete request. The method will close the
// http.Response Body if it receives an error.
func (client InboundNatRulesClient) DeleteSender(req *http.Request) (future InboundNatRulesDeleteFuture, err error) {
	var resp *http.Response
	resp, err = autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
	if err != nil {
		return
	}
	future.Future, err = azure.NewFutureFromResponse(resp)
	return
}

// DeleteResponder handles the response to the Delete request. The method always
// closes the http.Response Body.
func (client InboundNatRulesClient) DeleteResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted, http.StatusNoContent),
		autorest.ByClosing())
	result.Response = resp
	return
}

// Get gets the specified load balancer inbound nat rule.
// Parameters:
// resourceGroupName - the name of the resource group.
// loadBalancerName - the name of the load balancer.
// inboundNatRuleName - the name of the inbound nat rule.
// expand - expands referenced resources.
func (client InboundNatRulesClient) Get(ctx context.Context, resourceGroupName string, loadBalancerName string, inboundNatRuleName string, expand string) (result InboundNatRule, err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/InboundNatRulesClient.Get")
		defer func() {
			sc := -1
			if result.Response.Response != nil {
				sc = result.Response.Response.StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	req, err := client.GetPreparer(ctx, resourceGroupName, loadBalancerName, inboundNatRuleName, expand)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InboundNatRulesClient", "Get", nil, "Failure preparing request")
		return
	}

	resp, err := client.GetSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "network.InboundNatRulesClient", "Get", resp, "Failure sending request")
		return
	}

	result, err = client.GetResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InboundNatRulesClient", "Get", resp, "Failure responding to request")
	}

	return
}

// GetPreparer prepares the Get request.
func (client InboundNatRulesClient) GetPreparer(ctx context.Context, resourceGroupName string, loadBalancerName string, inboundNatRuleName string, expand string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"inboundNatRuleName": autorest.Encode("path", inboundNatRuleName),
		"loadBalancerName":   autorest.Encode("path", loadBalancerName),
		"resourceGroupName":  autorest.Encode("path", resourceGroupName),
		"subscriptionId":     autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-06-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}
	if len(expand) > 0 {
		queryParameters["$expand"] = autorest.Encode("query", expand)
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/loadBalancers/{loadBalancerName}/inboundNatRules/{inboundNatRuleName}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// GetSender sends the Get request. The method will close the
// http.Response Body if it receives an error.
func (client InboundNatRulesClient) GetSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// GetResponder handles the response to the Get request. The method always
// closes the http.Response Body.
func (client InboundNatRulesClient) GetResponder(resp *http.Response) (result InboundNatRule, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// List gets all the inbound nat rules in a load balancer.
// Parameters:
// resourceGroupName - the name of the resource group.
// loadBalancerName - the name of the load balancer.
func (client InboundNatRulesClient) List(ctx context.Context, resourceGroupName string, loadBalancerName string) (result InboundNatRuleListResultPage, err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/InboundNatRulesClient.List")
		defer func() {
			sc := -1
			if result.inrlr.Response.Response != nil {
				sc = result.inrlr.Response.Response.StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	result.fn = client.listNextResults
	req, err := client.ListPreparer(ctx, resourceGroupName, loadBalancerName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InboundNatRulesClient", "List", nil, "Failure preparing request")
		return
	}

	resp, err := client.ListSender(req)
	if err != nil {
		result.inrlr.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "network.InboundNatRulesClient", "List", resp, "Failure sending request")
		return
	}

	result.inrlr, err = client.ListResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InboundNatRulesClient", "List", resp, "Failure responding to request")
	}

	return
}

// ListPreparer prepares the List request.
func (client InboundNatRulesClient) ListPreparer(ctx context.Context, resourceGroupName string, loadBalancerName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"loadBalancerName":  autorest.Encode("path", loadBalancerName),
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-06-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/loadBalancers/{loadBalancerName}/inboundNatRules", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// ListSender sends the List request. The method will close the
// http.Response Body if it receives an error.
func (client InboundNatRulesClient) ListSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// ListResponder handles the response to the List request. The method always
// closes the http.Response Body.
func (client InboundNatRulesClient) ListResponder(resp *http.Response) (result InboundNatRuleListResult, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// listNextResults retrieves the next set of results, if any.
func (client InboundNatRulesClient) listNextResults(ctx context.Context, lastResults InboundNatRuleListResult) (result InboundNatRuleListResult, err error) {
	req, err := lastResults.inboundNatRuleListResultPreparer(ctx)
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.InboundNatRulesClient", "listNextResults", nil, "Failure preparing next results request")
	}
	if req == nil {
		return
	}
	resp, err := client.ListSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		return result, autorest.NewErrorWithError(err, "network.InboundNatRulesClient", "listNextResults", resp, "Failure sending next results request")
	}
	result, err = client.ListResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.InboundNatRulesClient", "listNextResults", resp, "Failure responding to next results request")
	}
	return
}

// ListComplete enumerates all values, automatically crossing page boundaries as required.
func (client InboundNatRulesClient) ListComplete(ctx context.Context, resourceGroupName string, loadBalancerName string) (result InboundNatRuleListResultIterator, err error) {
	if tracing.IsEnabled() {
		ctx = tracing.StartSpan(ctx, fqdn+"/InboundNatRulesClient.List")
		defer func() {
			sc := -1
			if result.Response().Response.Response != nil {
				sc = result.page.Response().Response.Response.StatusCode
			}
			tracing.EndSpan(ctx, sc, err)
		}()
	}
	result.page, err = client.List(ctx, resourceGroupName, loadBalancerName)
	return
}
