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
	"net/http"
)

// DdosProtectionPlansClient is the network Client
type DdosProtectionPlansClient struct {
	BaseClient
}

// NewDdosProtectionPlansClient creates an instance of the DdosProtectionPlansClient client.
func NewDdosProtectionPlansClient(subscriptionID string) DdosProtectionPlansClient {
	return NewDdosProtectionPlansClientWithBaseURI(DefaultBaseURI, subscriptionID)
}

// NewDdosProtectionPlansClientWithBaseURI creates an instance of the DdosProtectionPlansClient client.
func NewDdosProtectionPlansClientWithBaseURI(baseURI string, subscriptionID string) DdosProtectionPlansClient {
	return DdosProtectionPlansClient{NewWithBaseURI(baseURI, subscriptionID)}
}

// CreateOrUpdate creates or updates a DDoS protection plan.
// Parameters:
// resourceGroupName - the name of the resource group.
// ddosProtectionPlanName - the name of the DDoS protection plan.
// parameters - parameters supplied to the create or update operation.
func (client DdosProtectionPlansClient) CreateOrUpdate(ctx context.Context, resourceGroupName string, ddosProtectionPlanName string, parameters DdosProtectionPlan) (result DdosProtectionPlansCreateOrUpdateFuture, err error) {
	req, err := client.CreateOrUpdatePreparer(ctx, resourceGroupName, ddosProtectionPlanName, parameters)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "CreateOrUpdate", nil, "Failure preparing request")
		return
	}

	result, err = client.CreateOrUpdateSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "CreateOrUpdate", result.Response(), "Failure sending request")
		return
	}

	return
}

// CreateOrUpdatePreparer prepares the CreateOrUpdate request.
func (client DdosProtectionPlansClient) CreateOrUpdatePreparer(ctx context.Context, resourceGroupName string, ddosProtectionPlanName string, parameters DdosProtectionPlan) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"ddosProtectionPlanName": autorest.Encode("path", ddosProtectionPlanName),
		"resourceGroupName":      autorest.Encode("path", resourceGroupName),
		"subscriptionId":         autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-04-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsPut(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/ddosProtectionPlans/{ddosProtectionPlanName}", pathParameters),
		autorest.WithJSON(parameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// CreateOrUpdateSender sends the CreateOrUpdate request. The method will close the
// http.Response Body if it receives an error.
func (client DdosProtectionPlansClient) CreateOrUpdateSender(req *http.Request) (future DdosProtectionPlansCreateOrUpdateFuture, err error) {
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
func (client DdosProtectionPlansClient) CreateOrUpdateResponder(resp *http.Response) (result DdosProtectionPlan, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusCreated),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// Delete deletes the specified DDoS protection plan.
// Parameters:
// resourceGroupName - the name of the resource group.
// ddosProtectionPlanName - the name of the DDoS protection plan.
func (client DdosProtectionPlansClient) Delete(ctx context.Context, resourceGroupName string, ddosProtectionPlanName string) (result DdosProtectionPlansDeleteFuture, err error) {
	req, err := client.DeletePreparer(ctx, resourceGroupName, ddosProtectionPlanName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "Delete", nil, "Failure preparing request")
		return
	}

	result, err = client.DeleteSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "Delete", result.Response(), "Failure sending request")
		return
	}

	return
}

// DeletePreparer prepares the Delete request.
func (client DdosProtectionPlansClient) DeletePreparer(ctx context.Context, resourceGroupName string, ddosProtectionPlanName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"ddosProtectionPlanName": autorest.Encode("path", ddosProtectionPlanName),
		"resourceGroupName":      autorest.Encode("path", resourceGroupName),
		"subscriptionId":         autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-04-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsDelete(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/ddosProtectionPlans/{ddosProtectionPlanName}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// DeleteSender sends the Delete request. The method will close the
// http.Response Body if it receives an error.
func (client DdosProtectionPlansClient) DeleteSender(req *http.Request) (future DdosProtectionPlansDeleteFuture, err error) {
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
func (client DdosProtectionPlansClient) DeleteResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted, http.StatusNoContent),
		autorest.ByClosing())
	result.Response = resp
	return
}

// Get gets information about the specified DDoS protection plan.
// Parameters:
// resourceGroupName - the name of the resource group.
// ddosProtectionPlanName - the name of the DDoS protection plan.
func (client DdosProtectionPlansClient) Get(ctx context.Context, resourceGroupName string, ddosProtectionPlanName string) (result DdosProtectionPlan, err error) {
	req, err := client.GetPreparer(ctx, resourceGroupName, ddosProtectionPlanName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "Get", nil, "Failure preparing request")
		return
	}

	resp, err := client.GetSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "Get", resp, "Failure sending request")
		return
	}

	result, err = client.GetResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "Get", resp, "Failure responding to request")
	}

	return
}

// GetPreparer prepares the Get request.
func (client DdosProtectionPlansClient) GetPreparer(ctx context.Context, resourceGroupName string, ddosProtectionPlanName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"ddosProtectionPlanName": autorest.Encode("path", ddosProtectionPlanName),
		"resourceGroupName":      autorest.Encode("path", resourceGroupName),
		"subscriptionId":         autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-04-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/ddosProtectionPlans/{ddosProtectionPlanName}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// GetSender sends the Get request. The method will close the
// http.Response Body if it receives an error.
func (client DdosProtectionPlansClient) GetSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// GetResponder handles the response to the Get request. The method always
// closes the http.Response Body.
func (client DdosProtectionPlansClient) GetResponder(resp *http.Response) (result DdosProtectionPlan, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// List gets all DDoS protection plans in a subscription.
func (client DdosProtectionPlansClient) List(ctx context.Context) (result DdosProtectionPlanListResultPage, err error) {
	result.fn = client.listNextResults
	req, err := client.ListPreparer(ctx)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "List", nil, "Failure preparing request")
		return
	}

	resp, err := client.ListSender(req)
	if err != nil {
		result.dpplr.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "List", resp, "Failure sending request")
		return
	}

	result.dpplr, err = client.ListResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "List", resp, "Failure responding to request")
	}

	return
}

// ListPreparer prepares the List request.
func (client DdosProtectionPlansClient) ListPreparer(ctx context.Context) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"subscriptionId": autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-04-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/providers/Microsoft.Network/ddosProtectionPlans", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// ListSender sends the List request. The method will close the
// http.Response Body if it receives an error.
func (client DdosProtectionPlansClient) ListSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// ListResponder handles the response to the List request. The method always
// closes the http.Response Body.
func (client DdosProtectionPlansClient) ListResponder(resp *http.Response) (result DdosProtectionPlanListResult, err error) {
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
func (client DdosProtectionPlansClient) listNextResults(lastResults DdosProtectionPlanListResult) (result DdosProtectionPlanListResult, err error) {
	req, err := lastResults.ddosProtectionPlanListResultPreparer()
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "listNextResults", nil, "Failure preparing next results request")
	}
	if req == nil {
		return
	}
	resp, err := client.ListSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		return result, autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "listNextResults", resp, "Failure sending next results request")
	}
	result, err = client.ListResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "listNextResults", resp, "Failure responding to next results request")
	}
	return
}

// ListComplete enumerates all values, automatically crossing page boundaries as required.
func (client DdosProtectionPlansClient) ListComplete(ctx context.Context) (result DdosProtectionPlanListResultIterator, err error) {
	result.page, err = client.List(ctx)
	return
}

// ListByResourceGroup gets all the DDoS protection plans in a resource group.
// Parameters:
// resourceGroupName - the name of the resource group.
func (client DdosProtectionPlansClient) ListByResourceGroup(ctx context.Context, resourceGroupName string) (result DdosProtectionPlanListResultPage, err error) {
	result.fn = client.listByResourceGroupNextResults
	req, err := client.ListByResourceGroupPreparer(ctx, resourceGroupName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "ListByResourceGroup", nil, "Failure preparing request")
		return
	}

	resp, err := client.ListByResourceGroupSender(req)
	if err != nil {
		result.dpplr.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "ListByResourceGroup", resp, "Failure sending request")
		return
	}

	result.dpplr, err = client.ListByResourceGroupResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "ListByResourceGroup", resp, "Failure responding to request")
	}

	return
}

// ListByResourceGroupPreparer prepares the ListByResourceGroup request.
func (client DdosProtectionPlansClient) ListByResourceGroupPreparer(ctx context.Context, resourceGroupName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"resourceGroupName": autorest.Encode("path", resourceGroupName),
		"subscriptionId":    autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-04-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/ddosProtectionPlans", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// ListByResourceGroupSender sends the ListByResourceGroup request. The method will close the
// http.Response Body if it receives an error.
func (client DdosProtectionPlansClient) ListByResourceGroupSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// ListByResourceGroupResponder handles the response to the ListByResourceGroup request. The method always
// closes the http.Response Body.
func (client DdosProtectionPlansClient) ListByResourceGroupResponder(resp *http.Response) (result DdosProtectionPlanListResult, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// listByResourceGroupNextResults retrieves the next set of results, if any.
func (client DdosProtectionPlansClient) listByResourceGroupNextResults(lastResults DdosProtectionPlanListResult) (result DdosProtectionPlanListResult, err error) {
	req, err := lastResults.ddosProtectionPlanListResultPreparer()
	if err != nil {
		return result, autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "listByResourceGroupNextResults", nil, "Failure preparing next results request")
	}
	if req == nil {
		return
	}
	resp, err := client.ListByResourceGroupSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		return result, autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "listByResourceGroupNextResults", resp, "Failure sending next results request")
	}
	result, err = client.ListByResourceGroupResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.DdosProtectionPlansClient", "listByResourceGroupNextResults", resp, "Failure responding to next results request")
	}
	return
}

// ListByResourceGroupComplete enumerates all values, automatically crossing page boundaries as required.
func (client DdosProtectionPlansClient) ListByResourceGroupComplete(ctx context.Context, resourceGroupName string) (result DdosProtectionPlanListResultIterator, err error) {
	result.page, err = client.ListByResourceGroup(ctx, resourceGroupName)
	return
}
