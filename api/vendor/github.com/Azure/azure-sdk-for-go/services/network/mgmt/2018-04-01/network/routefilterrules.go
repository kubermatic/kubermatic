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
	"net/http"
)

// RouteFilterRulesClient is the network Client
type RouteFilterRulesClient struct {
	BaseClient
}

// NewRouteFilterRulesClient creates an instance of the RouteFilterRulesClient client.
func NewRouteFilterRulesClient(subscriptionID string) RouteFilterRulesClient {
	return NewRouteFilterRulesClientWithBaseURI(DefaultBaseURI, subscriptionID)
}

// NewRouteFilterRulesClientWithBaseURI creates an instance of the RouteFilterRulesClient client.
func NewRouteFilterRulesClientWithBaseURI(baseURI string, subscriptionID string) RouteFilterRulesClient {
	return RouteFilterRulesClient{NewWithBaseURI(baseURI, subscriptionID)}
}

// CreateOrUpdate creates or updates a route in the specified route filter.
// Parameters:
// resourceGroupName - the name of the resource group.
// routeFilterName - the name of the route filter.
// ruleName - the name of the route filter rule.
// routeFilterRuleParameters - parameters supplied to the create or update route filter rule operation.
func (client RouteFilterRulesClient) CreateOrUpdate(ctx context.Context, resourceGroupName string, routeFilterName string, ruleName string, routeFilterRuleParameters RouteFilterRule) (result RouteFilterRulesCreateOrUpdateFuture, err error) {
	if err := validation.Validate([]validation.Validation{
		{TargetValue: routeFilterRuleParameters,
			Constraints: []validation.Constraint{{Target: "routeFilterRuleParameters.RouteFilterRulePropertiesFormat", Name: validation.Null, Rule: false,
				Chain: []validation.Constraint{{Target: "routeFilterRuleParameters.RouteFilterRulePropertiesFormat.RouteFilterRuleType", Name: validation.Null, Rule: true, Chain: nil},
					{Target: "routeFilterRuleParameters.RouteFilterRulePropertiesFormat.Communities", Name: validation.Null, Rule: true, Chain: nil},
				}}}}}); err != nil {
		return result, validation.NewError("network.RouteFilterRulesClient", "CreateOrUpdate", err.Error())
	}

	req, err := client.CreateOrUpdatePreparer(ctx, resourceGroupName, routeFilterName, ruleName, routeFilterRuleParameters)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "CreateOrUpdate", nil, "Failure preparing request")
		return
	}

	result, err = client.CreateOrUpdateSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "CreateOrUpdate", result.Response(), "Failure sending request")
		return
	}

	return
}

// CreateOrUpdatePreparer prepares the CreateOrUpdate request.
func (client RouteFilterRulesClient) CreateOrUpdatePreparer(ctx context.Context, resourceGroupName string, routeFilterName string, ruleName string, routeFilterRuleParameters RouteFilterRule) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"routeFilterName":   autorest.Encode("path", routeFilterName),
		"ruleName":          autorest.Encode("path", ruleName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-04-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsPut(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters/{routeFilterName}/routeFilterRules/{ruleName}", pathParameters),
		autorest.WithJSON(routeFilterRuleParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// CreateOrUpdateSender sends the CreateOrUpdate request. The method will close the
// http.Response Body if it receives an error.
func (client RouteFilterRulesClient) CreateOrUpdateSender(req *http.Request) (future RouteFilterRulesCreateOrUpdateFuture, err error) {
	sender := autorest.DecorateSender(client, azure.DoRetryWithRegistration(client.Client))
	future.Future = azure.NewFuture(req)
	future.req = req
	_, err = future.Done(sender)
	if err != nil {
		return
	}
	err = autorest.Respond(future.Response(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusCreated))
	return
}

// CreateOrUpdateResponder handles the response to the CreateOrUpdate request. The method always
// closes the http.Response Body.
func (client RouteFilterRulesClient) CreateOrUpdateResponder(resp *http.Response) (result RouteFilterRule, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusCreated),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// Delete deletes the specified rule from a route filter.
// Parameters:
// resourceGroupName - the name of the resource group.
// routeFilterName - the name of the route filter.
// ruleName - the name of the rule.
func (client RouteFilterRulesClient) Delete(ctx context.Context, resourceGroupName string, routeFilterName string, ruleName string) (result RouteFilterRulesDeleteFuture, err error) {
	req, err := client.DeletePreparer(ctx, resourceGroupName, routeFilterName, ruleName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "Delete", nil, "Failure preparing request")
		return
	}

	result, err = client.DeleteSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "Delete", result.Response(), "Failure sending request")
		return
	}

	return
}

// DeletePreparer prepares the Delete request.
func (client RouteFilterRulesClient) DeletePreparer(ctx context.Context, resourceGroupName string, routeFilterName string, ruleName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"routeFilterName":   autorest.Encode("path", routeFilterName),
		"ruleName":          autorest.Encode("path", ruleName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-04-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsDelete(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters/{routeFilterName}/routeFilterRules/{ruleName}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// DeleteSender sends the Delete request. The method will close the
// http.Response Body if it receives an error.
func (client RouteFilterRulesClient) DeleteSender(req *http.Request) (future RouteFilterRulesDeleteFuture, err error) {
	sender := autorest.DecorateSender(client, azure.DoRetryWithRegistration(client.Client))
	future.Future = azure.NewFuture(req)
	future.req = req
	_, err = future.Done(sender)
	if err != nil {
		return
	}
	err = autorest.Respond(future.Response(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted, http.StatusNoContent))
	return
}

// DeleteResponder handles the response to the Delete request. The method always
// closes the http.Response Body.
func (client RouteFilterRulesClient) DeleteResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted, http.StatusNoContent),
		autorest.ByClosing())
	result.Response = resp
	return
}

// Get gets the specified rule from a route filter.
// Parameters:
// resourceGroupName - the name of the resource group.
// routeFilterName - the name of the route filter.
// ruleName - the name of the rule.
func (client RouteFilterRulesClient) Get(ctx context.Context, resourceGroupName string, routeFilterName string, ruleName string) (result RouteFilterRule, err error) {
	req, err := client.GetPreparer(ctx, resourceGroupName, routeFilterName, ruleName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "Get", nil, "Failure preparing request")
		return
	}

	resp, err := client.GetSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "Get", resp, "Failure sending request")
		return
	}

	result, err = client.GetResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "Get", resp, "Failure responding to request")
	}

	return
}

// GetPreparer prepares the Get request.
func (client RouteFilterRulesClient) GetPreparer(ctx context.Context, resourceGroupName string, routeFilterName string, ruleName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"routeFilterName":   autorest.Encode("path", routeFilterName),
		"ruleName":          autorest.Encode("path", ruleName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-04-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters/{routeFilterName}/routeFilterRules/{ruleName}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// GetSender sends the Get request. The method will close the
// http.Response Body if it receives an error.
func (client RouteFilterRulesClient) GetSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// GetResponder handles the response to the Get request. The method always
// closes the http.Response Body.
func (client RouteFilterRulesClient) GetResponder(resp *http.Response) (result RouteFilterRule, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// ListByRouteFilter gets all RouteFilterRules in a route filter.
// Parameters:
// resourceGroupName - the name of the resource group.
// routeFilterName - the name of the route filter.
func (client RouteFilterRulesClient) ListByRouteFilter(ctx context.Context, resourceGroupName string, routeFilterName string) (result RouteFilterRuleListResultPage, err error) {
	result.fn = client.listByRouteFilterNextResults
	req, err := client.ListByRouteFilterPreparer(ctx, resourceGroupName, routeFilterName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "ListByRouteFilter", nil, "Failure preparing request")
		return
	}

	resp, err := client.ListByRouteFilterSender(req)
	if err != nil {
		result.rfrlr.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "ListByRouteFilter", resp, "Failure sending request")
		return
	}

	result.rfrlr, err = client.ListByRouteFilterResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "ListByRouteFilter", resp, "Failure responding to request")
	}

	return
}

// ListByRouteFilterPreparer prepares the ListByRouteFilter request.
func (client RouteFilterRulesClient) ListByRouteFilterPreparer(ctx context.Context, resourceGroupName string, routeFilterName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"routeFilterName":   autorest.Encode("path", routeFilterName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-04-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters/{routeFilterName}/routeFilterRules", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// ListByRouteFilterSender sends the ListByRouteFilter request. The method will close the
// http.Response Body if it receives an error.
func (client RouteFilterRulesClient) ListByRouteFilterSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// ListByRouteFilterResponder handles the response to the ListByRouteFilter request. The method always
// closes the http.Response Body.
func (client RouteFilterRulesClient) ListByRouteFilterResponder(resp *http.Response) (result RouteFilterRuleListResult, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// listByRouteFilterNextResults retrieves the next set of results, if any.
func (client RouteFilterRulesClient) listByRouteFilterNextResults(lastResults RouteFilterRuleListResult) (result RouteFilterRuleListResult, err error) {
	req, err := lastResults.routeFilterRuleListResultPreparer()
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "listByRouteFilterNextResults", nil, "Failure preparing next results request")
	}
	if req == nil {
		return
	}
	resp, err := client.ListByRouteFilterSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		return result, autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "listByRouteFilterNextResults", resp, "Failure sending next results request")
	}
	result, err = client.ListByRouteFilterResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "listByRouteFilterNextResults", resp, "Failure responding to next results request")
	}
	return
}

// ListByRouteFilterComplete enumerates all values, automatically crossing page boundaries as required.
func (client RouteFilterRulesClient) ListByRouteFilterComplete(ctx context.Context, resourceGroupName string, routeFilterName string) (result RouteFilterRuleListResultIterator, err error) {
	result.page, err = client.ListByRouteFilter(ctx, resourceGroupName, routeFilterName)
	return
}

// Update updates a route in the specified route filter.
// Parameters:
// resourceGroupName - the name of the resource group.
// routeFilterName - the name of the route filter.
// ruleName - the name of the route filter rule.
// routeFilterRuleParameters - parameters supplied to the update route filter rule operation.
func (client RouteFilterRulesClient) Update(ctx context.Context, resourceGroupName string, routeFilterName string, ruleName string, routeFilterRuleParameters PatchRouteFilterRule) (result RouteFilterRulesUpdateFuture, err error) {
	req, err := client.UpdatePreparer(ctx, resourceGroupName, routeFilterName, ruleName, routeFilterRuleParameters)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "Update", nil, "Failure preparing request")
		return
	}

	result, err = client.UpdateSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.RouteFilterRulesClient", "Update", result.Response(), "Failure sending request")
		return
	}

	return
}

// UpdatePreparer prepares the Update request.
func (client RouteFilterRulesClient) UpdatePreparer(ctx context.Context, resourceGroupName string, routeFilterName string, ruleName string, routeFilterRuleParameters PatchRouteFilterRule) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"routeFilterName":   autorest.Encode("path", routeFilterName),
		"ruleName":          autorest.Encode("path", ruleName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-04-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsPatch(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/routeFilters/{routeFilterName}/routeFilterRules/{ruleName}", pathParameters),
		autorest.WithJSON(routeFilterRuleParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// UpdateSender sends the Update request. The method will close the
// http.Response Body if it receives an error.
func (client RouteFilterRulesClient) UpdateSender(req *http.Request) (future RouteFilterRulesUpdateFuture, err error) {
	sender := autorest.DecorateSender(client, azure.DoRetryWithRegistration(client.Client))
	future.Future = azure.NewFuture(req)
	future.req = req
	_, err = future.Done(sender)
	if err != nil {
		return
	}
	err = autorest.Respond(future.Response(),
		azure.WithErrorUnlessStatusCode(http.StatusOK))
	return
}

// UpdateResponder handles the response to the Update request. The method always
// closes the http.Response Body.
func (client RouteFilterRulesClient) UpdateResponder(resp *http.Response) (result RouteFilterRule, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}
