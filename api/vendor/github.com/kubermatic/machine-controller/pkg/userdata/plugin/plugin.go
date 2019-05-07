/*
Copyright 2019 The Machine Controller Authors.

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

//
// Core UserData plugin.
//

// Package plugin provides the plugin side of the plugin mechanism.
// Individual plugins have to implement the provider interface,
// pass it to a new plugin instance, and call run.
package plugin

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"

	"github.com/kubermatic/machine-controller/pkg/apis/plugin"
)

// Provider defines the interface each plugin has to implement
// for the retrieval of the userdata based on the given arguments.
type Provider interface {
	UserData(
		spec clusterv1alpha1.MachineSpec,
		kubeconfig *clientcmdapi.Config,
		cloudConfig string,
		cloudProviderName string,
		clusterDNSIPs []net.IP,
		externalCloudProvider bool,
	) (string, error)
}

// Plugin implements a convenient helper to map the request to the given
// provider and return the response.
type Plugin struct {
	provider Provider
	debug    bool
}

// New creates a new plugin.
func New(provider Provider, debug bool) *Plugin {
	return &Plugin{
		provider: provider,
		debug:    debug,
	}
}

// Run looks for the given request and executes it.
func (p *Plugin) Run() error {
	reqEnv := os.Getenv(plugin.EnvUserDataRequest)
	if reqEnv == "" {
		resp := plugin.ErrorResponse{
			Err: fmt.Sprintf("environment variable '%s' not set", plugin.EnvUserDataRequest),
		}
		return p.printResponse(resp)
	}
	// Handle the request for user data.
	var req plugin.UserDataRequest
	err := json.Unmarshal([]byte(reqEnv), &req)
	if err != nil {
		return err
	}
	userData, err := p.provider.UserData(
		req.MachineSpec,
		req.KubeConfig,
		req.CloudConfig,
		req.CloudProviderName,
		req.DNSIPs,
		req.ExternalCloudProvider,
	)
	var resp plugin.UserDataResponse
	if err != nil {
		resp.Err = err.Error()
	} else {
		resp.UserData = userData
	}
	return p.printResponse(resp)
}

// printResponse marshals the respons and prints it to stdout.
func (p *Plugin) printResponse(resp interface{}) error {
	bs, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	_, err = fmt.Printf("%s", string(bs))
	return err
}
