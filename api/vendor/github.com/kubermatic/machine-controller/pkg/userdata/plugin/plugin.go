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

	"github.com/kubermatic/machine-controller/pkg/userdata/cloud"
)

// Provider defines the interface each plugin has to implement
// for the retrieval of the userdata based on the given arguments.
type Provider interface {
	UserData(
		spec clusterv1alpha1.MachineSpec,
		kubeconfig *clientcmdapi.Config,
		ccProvider cloud.ConfigProvider,
		clusterDNSIPs []net.IP,
		externalCloudProvider bool,
	) (string, error)
}

// Handler cares dispatching of the RPC calls to the given Provider.
type Handler struct {
	provider Provider
}

// UserData receives the UserData RPC message and calls the provider.
func (h *Handler) UserData(req *UserDataRequest, resp *UserDataResponse) error {
	userData, err := h.provider.UserData(
		req.MachineSpec,
		req.KubeConfig,
		req.CloudConfig,
		req.DNSIPs,
		req.ExternalCloudProvider,
	)
	resp.UserData = userData
	if err != nil {
		resp.Err = err.Error()
	}
	return nil
}

// Plugin implements a convenient helper to map the request to the given
// provider and return the response.
type Plugin struct {
	provider Provider
	debug    bool
}

// New creates a new plugin. Debug flag is not yet handled.
func New(provider Provider, debug bool) *Plugin {
	return &Plugin{
		provider: provider,
		debug:    debug,
	}
}

// Run looks for the given request and executes it.
func (p *Plugin) Run() error {
	reqCmd := os.Getenv(EnvRequest)
	switch reqCmd {
	case EnvUserDataRequest:
		return p.handleUserDataRequest()
	default:
		return p.handleUnknownRequest(reqCmd)
	}
}

// handleUserDataRequest handles the request for user data.
func (p *Plugin) handleUserDataRequest() error {
	reqEnv := os.Getenv(EnvUserDataRequest)
	var req UserDataRequest
	err := json.Unmarshal([]byte(reqEnv), &req)
	if err != nil {
		return err
	}
	userData, err := p.provider.UserData(
		req.MachineSpec,
		req.KubeConfig,
		req.CloudConfig,
		req.DNSIPs,
		req.ExternalCloudProvider,
	)
	var resp UserDataResponse
	if err != nil {
		resp.Err = err.Error()
	} else {
		resp.UserData = userData
	}
	return p.printResponse(resp)
}

// handleUnknownRequest handles unknown requests.
func (p *Plugin) handleUnknownRequest(reqCmd string) error {
	var resp ErrorResponse
	if reqCmd == "" {
		resp.Err = "no request command given"
	} else {
		resp.Err = fmt.Sprintf("unknown request command '%s'", reqCmd)
	}
	return p.printResponse(resp)
}

func (p *Plugin) printResponse(resp interface{}) error {
	bs, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	fmt.Printf("%s", string(bs))
	return nil
}
