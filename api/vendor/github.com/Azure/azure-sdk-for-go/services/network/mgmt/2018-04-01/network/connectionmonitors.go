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

// ConnectionMonitorsClient is the network Client
type ConnectionMonitorsClient struct {
	BaseClient
}

// NewConnectionMonitorsClient creates an instance of the ConnectionMonitorsClient client.
func NewConnectionMonitorsClient(subscriptionID string) ConnectionMonitorsClient {
	return NewConnectionMonitorsClientWithBaseURI(DefaultBaseURI, subscriptionID)
}

// NewConnectionMonitorsClientWithBaseURI creates an instance of the ConnectionMonitorsClient client.
func NewConnectionMonitorsClientWithBaseURI(baseURI string, subscriptionID string) ConnectionMonitorsClient {
	return ConnectionMonitorsClient{NewWithBaseURI(baseURI, subscriptionID)}
}

// CreateOrUpdate create or update a connection monitor.
// Parameters:
// resourceGroupName - the name of the resource group containing Network Watcher.
// networkWatcherName - the name of the Network Watcher resource.
// connectionMonitorName - the name of the connection monitor.
// parameters - parameters that define the operation to create a connection monitor.
func (client ConnectionMonitorsClient) CreateOrUpdate(ctx context.Context, resourceGroupName string, networkWatcherName string, connectionMonitorName string, parameters ConnectionMonitor) (result ConnectionMonitorsCreateOrUpdateFuture, err error) {
	if err := validation.Validate([]validation.Validation{
		{TargetValue: parameters,
			Constraints: []validation.Constraint{{Target: "parameters.ConnectionMonitorParameters", Name: validation.Null, Rule: true,
				Chain: []validation.Constraint{{Target: "parameters.ConnectionMonitorParameters.Source", Name: validation.Null, Rule: true,
					Chain: []validation.Constraint{{Target: "parameters.ConnectionMonitorParameters.Source.ResourceID", Name: validation.Null, Rule: true, Chain: nil}}},
					{Target: "parameters.ConnectionMonitorParameters.Destination", Name: validation.Null, Rule: true, Chain: nil},
				}}}}}); err != nil {
		return result, validation.NewError("network.ConnectionMonitorsClient", "CreateOrUpdate", err.Error())
	}

	req, err := client.CreateOrUpdatePreparer(ctx, resourceGroupName, networkWatcherName, connectionMonitorName, parameters)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.ConnectionMonitorsClient", "CreateOrUpdate", nil, "Failure preparing request")
		return
	}

	result, err = client.CreateOrUpdateSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.ConnectionMonitorsClient", "CreateOrUpdate", result.Response(), "Failure sending request")
		return
	}

	return
}

// CreateOrUpdatePreparer prepares the CreateOrUpdate request.
func (client ConnectionMonitorsClient) CreateOrUpdatePreparer(ctx context.Context, resourceGroupName string, networkWatcherName string, connectionMonitorName string, parameters ConnectionMonitor) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"connectionMonitorName": autorest.Encode("path", connectionMonitorName),
		"networkWatcherName":    autorest.Encode("path", networkWatcherName),
		"resourceGroupName":     autorest.Encode("path", resourceGroupName),
		"subscriptionId":        autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-04-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsContentType("application/json; charset=utf-8"),
		autorest.AsPut(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/networkWatchers/{networkWatcherName}/connectionMonitors/{connectionMonitorName}", pathParameters),
		autorest.WithJSON(parameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// CreateOrUpdateSender sends the CreateOrUpdate request. The method will close the
// http.Response Body if it receives an error.
func (client ConnectionMonitorsClient) CreateOrUpdateSender(req *http.Request) (future ConnectionMonitorsCreateOrUpdateFuture, err error) {
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
func (client ConnectionMonitorsClient) CreateOrUpdateResponder(resp *http.Response) (result ConnectionMonitorResult, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusCreated),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// Delete deletes the specified connection monitor.
// Parameters:
// resourceGroupName - the name of the resource group containing Network Watcher.
// networkWatcherName - the name of the Network Watcher resource.
// connectionMonitorName - the name of the connection monitor.
func (client ConnectionMonitorsClient) Delete(ctx context.Context, resourceGroupName string, networkWatcherName string, connectionMonitorName string) (result ConnectionMonitorsDeleteFuture, err error) {
	req, err := client.DeletePreparer(ctx, resourceGroupName, networkWatcherName, connectionMonitorName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.ConnectionMonitorsClient", "Delete", nil, "Failure preparing request")
		return
	}

	result, err = client.DeleteSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.ConnectionMonitorsClient", "Delete", result.Response(), "Failure sending request")
		return
	}

	return
}

// DeletePreparer prepares the Delete request.
func (client ConnectionMonitorsClient) DeletePreparer(ctx context.Context, resourceGroupName string, networkWatcherName string, connectionMonitorName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"connectionMonitorName": autorest.Encode("path", connectionMonitorName),
		"networkWatcherName":    autorest.Encode("path", networkWatcherName),
		"resourceGroupName":     autorest.Encode("path", resourceGroupName),
		"subscriptionId":        autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-04-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsDelete(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/networkWatchers/{networkWatcherName}/connectionMonitors/{connectionMonitorName}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// DeleteSender sends the Delete request. The method will close the
// http.Response Body if it receives an error.
func (client ConnectionMonitorsClient) DeleteSender(req *http.Request) (future ConnectionMonitorsDeleteFuture, err error) {
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
func (client ConnectionMonitorsClient) DeleteResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted, http.StatusNoContent),
		autorest.ByClosing())
	result.Response = resp
	return
}

// Get gets a connection monitor by name.
// Parameters:
// resourceGroupName - the name of the resource group containing Network Watcher.
// networkWatcherName - the name of the Network Watcher resource.
// connectionMonitorName - the name of the connection monitor.
func (client ConnectionMonitorsClient) Get(ctx context.Context, resourceGroupName string, networkWatcherName string, connectionMonitorName string) (result ConnectionMonitorResult, err error) {
	req, err := client.GetPreparer(ctx, resourceGroupName, networkWatcherName, connectionMonitorName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.ConnectionMonitorsClient", "Get", nil, "Failure preparing request")
		return
	}

	resp, err := client.GetSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "network.ConnectionMonitorsClient", "Get", resp, "Failure sending request")
		return
	}

	result, err = client.GetResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.ConnectionMonitorsClient", "Get", resp, "Failure responding to request")
	}

	return
}

// GetPreparer prepares the Get request.
func (client ConnectionMonitorsClient) GetPreparer(ctx context.Context, resourceGroupName string, networkWatcherName string, connectionMonitorName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"connectionMonitorName": autorest.Encode("path", connectionMonitorName),
		"networkWatcherName":    autorest.Encode("path", networkWatcherName),
		"resourceGroupName":     autorest.Encode("path", resourceGroupName),
		"subscriptionId":        autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-04-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/networkWatchers/{networkWatcherName}/connectionMonitors/{connectionMonitorName}", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// GetSender sends the Get request. The method will close the
// http.Response Body if it receives an error.
func (client ConnectionMonitorsClient) GetSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// GetResponder handles the response to the Get request. The method always
// closes the http.Response Body.
func (client ConnectionMonitorsClient) GetResponder(resp *http.Response) (result ConnectionMonitorResult, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// List lists all connection monitors for the specified Network Watcher.
// Parameters:
// resourceGroupName - the name of the resource group containing Network Watcher.
// networkWatcherName - the name of the Network Watcher resource.
func (client ConnectionMonitorsClient) List(ctx context.Context, resourceGroupName string, networkWatcherName string) (result ConnectionMonitorListResult, err error) {
	req, err := client.ListPreparer(ctx, resourceGroupName, networkWatcherName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.ConnectionMonitorsClient", "List", nil, "Failure preparing request")
		return
	}

	resp, err := client.ListSender(req)
	if err != nil {
		result.Response = autorest.Response{Response: resp}
		err = autorest.NewErrorWithError(err, "network.ConnectionMonitorsClient", "List", resp, "Failure sending request")
		return
	}

	result, err = client.ListResponder(resp)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.ConnectionMonitorsClient", "List", resp, "Failure responding to request")
	}

	return
}

// ListPreparer prepares the List request.
func (client ConnectionMonitorsClient) ListPreparer(ctx context.Context, resourceGroupName string, networkWatcherName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"networkWatcherName": autorest.Encode("path", networkWatcherName),
		"resourceGroupName":  autorest.Encode("path", resourceGroupName),
		"subscriptionId":     autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-04-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsGet(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/networkWatchers/{networkWatcherName}/connectionMonitors", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// ListSender sends the List request. The method will close the
// http.Response Body if it receives an error.
func (client ConnectionMonitorsClient) ListSender(req *http.Request) (*http.Response, error) {
	return autorest.SendWithSender(client, req,
		azure.DoRetryWithRegistration(client.Client))
}

// ListResponder handles the response to the List request. The method always
// closes the http.Response Body.
func (client ConnectionMonitorsClient) ListResponder(resp *http.Response) (result ConnectionMonitorListResult, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// Query query a snapshot of the most recent connection states.
// Parameters:
// resourceGroupName - the name of the resource group containing Network Watcher.
// networkWatcherName - the name of the Network Watcher resource.
// connectionMonitorName - the name given to the connection monitor.
func (client ConnectionMonitorsClient) Query(ctx context.Context, resourceGroupName string, networkWatcherName string, connectionMonitorName string) (result ConnectionMonitorsQueryFuture, err error) {
	req, err := client.QueryPreparer(ctx, resourceGroupName, networkWatcherName, connectionMonitorName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.ConnectionMonitorsClient", "Query", nil, "Failure preparing request")
		return
	}

	result, err = client.QuerySender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.ConnectionMonitorsClient", "Query", result.Response(), "Failure sending request")
		return
	}

	return
}

// QueryPreparer prepares the Query request.
func (client ConnectionMonitorsClient) QueryPreparer(ctx context.Context, resourceGroupName string, networkWatcherName string, connectionMonitorName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"connectionMonitorName": autorest.Encode("path", connectionMonitorName),
		"networkWatcherName":    autorest.Encode("path", networkWatcherName),
		"resourceGroupName":     autorest.Encode("path", resourceGroupName),
		"subscriptionId":        autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-04-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsPost(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/networkWatchers/{networkWatcherName}/connectionMonitors/{connectionMonitorName}/query", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// QuerySender sends the Query request. The method will close the
// http.Response Body if it receives an error.
func (client ConnectionMonitorsClient) QuerySender(req *http.Request) (future ConnectionMonitorsQueryFuture, err error) {
	sender := autorest.DecorateSender(client, azure.DoRetryWithRegistration(client.Client))
	future.Future = azure.NewFuture(req)
	future.req = req
	_, err = future.Done(sender)
	if err != nil {
		return
	}
	err = autorest.Respond(future.Response(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted))
	return
}

// QueryResponder handles the response to the Query request. The method always
// closes the http.Response Body.
func (client ConnectionMonitorsClient) QueryResponder(resp *http.Response) (result ConnectionMonitorQueryResult, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted),
		autorest.ByUnmarshallingJSON(&result),
		autorest.ByClosing())
	result.Response = autorest.Response{Response: resp}
	return
}

// Start starts the specified connection monitor.
// Parameters:
// resourceGroupName - the name of the resource group containing Network Watcher.
// networkWatcherName - the name of the Network Watcher resource.
// connectionMonitorName - the name of the connection monitor.
func (client ConnectionMonitorsClient) Start(ctx context.Context, resourceGroupName string, networkWatcherName string, connectionMonitorName string) (result ConnectionMonitorsStartFuture, err error) {
	req, err := client.StartPreparer(ctx, resourceGroupName, networkWatcherName, connectionMonitorName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.ConnectionMonitorsClient", "Start", nil, "Failure preparing request")
		return
	}

	result, err = client.StartSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.ConnectionMonitorsClient", "Start", result.Response(), "Failure sending request")
		return
	}

	return
}

// StartPreparer prepares the Start request.
func (client ConnectionMonitorsClient) StartPreparer(ctx context.Context, resourceGroupName string, networkWatcherName string, connectionMonitorName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"connectionMonitorName": autorest.Encode("path", connectionMonitorName),
		"networkWatcherName":    autorest.Encode("path", networkWatcherName),
		"resourceGroupName":     autorest.Encode("path", resourceGroupName),
		"subscriptionId":        autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-04-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsPost(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/networkWatchers/{networkWatcherName}/connectionMonitors/{connectionMonitorName}/start", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// StartSender sends the Start request. The method will close the
// http.Response Body if it receives an error.
func (client ConnectionMonitorsClient) StartSender(req *http.Request) (future ConnectionMonitorsStartFuture, err error) {
	sender := autorest.DecorateSender(client, azure.DoRetryWithRegistration(client.Client))
	future.Future = azure.NewFuture(req)
	future.req = req
	_, err = future.Done(sender)
	if err != nil {
		return
	}
	err = autorest.Respond(future.Response(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted))
	return
}

// StartResponder handles the response to the Start request. The method always
// closes the http.Response Body.
func (client ConnectionMonitorsClient) StartResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted),
		autorest.ByClosing())
	result.Response = resp
	return
}

// Stop stops the specified connection monitor.
// Parameters:
// resourceGroupName - the name of the resource group containing Network Watcher.
// networkWatcherName - the name of the Network Watcher resource.
// connectionMonitorName - the name of the connection monitor.
func (client ConnectionMonitorsClient) Stop(ctx context.Context, resourceGroupName string, networkWatcherName string, connectionMonitorName string) (result ConnectionMonitorsStopFuture, err error) {
	req, err := client.StopPreparer(ctx, resourceGroupName, networkWatcherName, connectionMonitorName)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.ConnectionMonitorsClient", "Stop", nil, "Failure preparing request")
		return
	}

	result, err = client.StopSender(req)
	if err != nil {
		err = autorest.NewErrorWithError(err, "network.ConnectionMonitorsClient", "Stop", result.Response(), "Failure sending request")
		return
	}

	return
}

// StopPreparer prepares the Stop request.
func (client ConnectionMonitorsClient) StopPreparer(ctx context.Context, resourceGroupName string, networkWatcherName string, connectionMonitorName string) (*http.Request, error) {
	pathParameters := map[string]interface{}{
		"connectionMonitorName": autorest.Encode("path", connectionMonitorName),
		"networkWatcherName":    autorest.Encode("path", networkWatcherName),
		"resourceGroupName":     autorest.Encode("path", resourceGroupName),
		"subscriptionId":        autorest.Encode("path", client.SubscriptionID),
	}

	const APIVersion = "2018-04-01"
	queryParameters := map[string]interface{}{
		"api-version": APIVersion,
	}

	preparer := autorest.CreatePreparer(
		autorest.AsPost(),
		autorest.WithBaseURL(client.BaseURI),
		autorest.WithPathParameters("/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Network/networkWatchers/{networkWatcherName}/connectionMonitors/{connectionMonitorName}/stop", pathParameters),
		autorest.WithQueryParameters(queryParameters))
	return preparer.Prepare((&http.Request{}).WithContext(ctx))
}

// StopSender sends the Stop request. The method will close the
// http.Response Body if it receives an error.
func (client ConnectionMonitorsClient) StopSender(req *http.Request) (future ConnectionMonitorsStopFuture, err error) {
	sender := autorest.DecorateSender(client, azure.DoRetryWithRegistration(client.Client))
	future.Future = azure.NewFuture(req)
	future.req = req
	_, err = future.Done(sender)
	if err != nil {
		return
	}
	err = autorest.Respond(future.Response(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted))
	return
}

// StopResponder handles the response to the Stop request. The method always
// closes the http.Response Body.
func (client ConnectionMonitorsClient) StopResponder(resp *http.Response) (result autorest.Response, err error) {
	err = autorest.Respond(
		resp,
		client.ByInspecting(),
		azure.WithErrorUnlessStatusCode(http.StatusOK, http.StatusAccepted),
		autorest.ByClosing())
	result.Response = resp
	return
}
