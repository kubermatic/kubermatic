package hetzner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	"github.com/hetznercloud/hcloud-go/hcloud"

	"github.com/kubermatic/machine-controller/pkg/cloudprovider/cloud"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/common/ssh"
	cloudprovidererrors "github.com/kubermatic/machine-controller/pkg/cloudprovider/errors"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/instance"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"

	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/cluster-api/pkg/apis/cluster/common"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	machineUIDLabelKey = "machine-uid"
)

type provider struct {
	configVarResolver *providerconfig.ConfigVarResolver
}

// New returns a Hetzner provider
func New(configVarResolver *providerconfig.ConfigVarResolver) cloud.Provider {
	return &provider{configVarResolver: configVarResolver}
}

type RawConfig struct {
	Token      providerconfig.ConfigVarString `json:"token"`
	ServerType providerconfig.ConfigVarString `json:"serverType"`
	Datacenter providerconfig.ConfigVarString `json:"datacenter"`
	Location   providerconfig.ConfigVarString `json:"location"`
}

type Config struct {
	Token      string
	ServerType string
	Datacenter string
	Location   string
}

func getNameForOS(os providerconfig.OperatingSystem) (string, error) {
	switch os {
	case providerconfig.OperatingSystemUbuntu:
		return "ubuntu-18.04", nil
	case providerconfig.OperatingSystemCentOS:
		return "centos-7", nil
	}
	return "", providerconfig.ErrOSNotSupported
}

func getClient(token string) *hcloud.Client {
	return hcloud.NewClient(hcloud.WithToken(token))
}

func (p *provider) getConfig(s v1alpha1.ProviderSpec) (*Config, *providerconfig.Config, error) {
	if s.Value == nil {
		return nil, nil, fmt.Errorf("machine.spec.providerconfig.value is nil")
	}
	pconfig := providerconfig.Config{}
	err := json.Unmarshal(s.Value.Raw, &pconfig)
	if err != nil {
		return nil, nil, err
	}

	rawConfig := RawConfig{}
	if err = json.Unmarshal(pconfig.CloudProviderSpec.Raw, &rawConfig); err != nil {
		return nil, nil, err
	}

	c := Config{}
	c.Token, err = p.configVarResolver.GetConfigVarStringValueOrEnv(rawConfig.Token, "HZ_TOKEN")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the value of \"token\" field, error = %v", err)
	}
	c.ServerType, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.ServerType)
	if err != nil {
		return nil, nil, err
	}
	c.Datacenter, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Datacenter)
	if err != nil {
		return nil, nil, err
	}
	c.Location, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Location)
	if err != nil {
		return nil, nil, err
	}
	return &c, &pconfig, err
}

func (p *provider) Validate(spec v1alpha1.MachineSpec) error {
	c, pc, err := p.getConfig(spec.ProviderSpec)
	if err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	if c.Token == "" {
		return errors.New("token is missing")
	}

	_, err = getNameForOS(pc.OperatingSystem)
	if err != nil {
		return fmt.Errorf("invalid/not supported operating system specified %q: %v", pc.OperatingSystem, err)
	}

	ctx := context.TODO()
	client := getClient(c.Token)

	if c.Location != "" && c.Datacenter != "" {
		return fmt.Errorf("location and datacenter must not be set at the same time")
	}

	if c.Location != "" {
		if _, _, err = client.Location.Get(ctx, c.Location); err != nil {
			return fmt.Errorf("failed to get location: %v", err)
		}
	}

	if c.Datacenter != "" {
		if _, _, err = client.Datacenter.Get(ctx, c.Datacenter); err != nil {
			return fmt.Errorf("failed to get datacenter: %v", err)
		}
	}

	if _, _, err = client.ServerType.Get(ctx, c.ServerType); err != nil {
		return fmt.Errorf("failed to get server type: %v", err)
	}

	return nil
}

func (p *provider) Create(machine *v1alpha1.Machine, _ *cloud.MachineCreateDeleteData, userdata string) (instance.Instance, error) {
	c, pc, err := p.getConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}

	ctx := context.TODO()
	client := getClient(c.Token)

	imageName, err := getNameForOS(pc.OperatingSystem)
	if err != nil {
		return nil, cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Invalid operating system specified %q, details = %v", pc.OperatingSystem, err),
		}
	}

	serverCreateOpts := hcloud.ServerCreateOpts{
		Name:     machine.Spec.Name,
		UserData: userdata,
		Labels: map[string]string{
			machineUIDLabelKey: string(machine.UID),
		},
	}

	if c.Datacenter != "" {
		serverCreateOpts.Datacenter, _, err = client.Datacenter.Get(ctx, c.Datacenter)
		if err != nil {
			return nil, hzErrorToTerminalError(err, "failed to get datacenter")
		}
	}

	if c.Location != "" {
		serverCreateOpts.Location, _, err = client.Location.Get(ctx, c.Location)
		if err != nil {
			return nil, hzErrorToTerminalError(err, "failed to get location")
		}
	}

	serverCreateOpts.Image, _, err = client.Image.Get(ctx, imageName)
	if err != nil {
		return nil, hzErrorToTerminalError(err, "failed to get image")
	}

	serverCreateOpts.ServerType, _, err = client.ServerType.Get(ctx, c.ServerType)
	if err != nil {
		return nil, hzErrorToTerminalError(err, "failed to get server type")
	}

	sshkey, err := ssh.NewKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate ssh key: %v", err)
	}

	hkey, res, err := client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      sshkey.Name,
		PublicKey: sshkey.PublicKey,
	})
	if err != nil {
		return nil, fmt.Errorf("creating temporary ssh key failed with error %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("got invalid http status code when creating ssh key: expected=%d, god=%d", http.StatusCreated, res.StatusCode)
	}
	defer func() {
		_, err := client.SSHKey.Delete(ctx, hkey)
		if err != nil {
			glog.Errorf("Failed to delete temporary ssh key: %v", err)
		}
	}()
	serverCreateOpts.SSHKeys = []*hcloud.SSHKey{hkey}

	serverCreateRes, res, err := client.Server.Create(ctx, serverCreateOpts)
	if err != nil {
		return nil, hzErrorToTerminalError(err, "failed to create server")
	}
	if res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create server invalid status code returned. expected=%d got %d", http.StatusCreated, res.StatusCode)
	}

	return &hetznerServer{server: serverCreateRes.Server}, nil
}

func (p *provider) Cleanup(machine *v1alpha1.Machine, _ *cloud.MachineCreateDeleteData) (bool, error) {
	instance, err := p.Get(machine)
	if err != nil {
		if err == cloudprovidererrors.ErrInstanceNotFound {
			return true, nil
		}
		return false, err
	}

	c, _, err := p.getConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return false, cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}

	ctx := context.TODO()
	client := getClient(c.Token)

	res, err := client.Server.Delete(ctx, instance.(*hetznerServer).server)
	if err != nil {
		return false, hzErrorToTerminalError(err, "failed to delete the server")
	}
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {
		return false, fmt.Errorf("invalid status code returned. expected=%d got=%d", http.StatusOK, res.StatusCode)
	}
	return false, nil
}

func (p *provider) AddDefaults(spec v1alpha1.MachineSpec) (v1alpha1.MachineSpec, error) {
	return spec, nil
}

func (p *provider) Get(machine *v1alpha1.Machine) (instance.Instance, error) {
	c, _, err := p.getConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return nil, cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}

	ctx := context.TODO()
	client := getClient(c.Token)

	servers, _, err := client.Server.List(ctx, hcloud.ServerListOpts{ListOpts: hcloud.ListOpts{
		LabelSelector: machineUIDLabelKey + "==" + string(machine.UID),
	}})
	if err != nil {
		return nil, hzErrorToTerminalError(err, "failed to list servers")
	}

	for _, server := range servers {
		if server.Labels[machineUIDLabelKey] == string(machine.UID) {
			return &hetznerServer{server: server}, nil
		}
	}

	return nil, cloudprovidererrors.ErrInstanceNotFound
}

func (p *provider) MigrateUID(machine *v1alpha1.Machine, new types.UID) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	c, _, err := p.getConfig(machine.Spec.ProviderSpec)
	if err != nil {
		return cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}
	client := getClient(c.Token)

	// We didn't use the UID for Hetzner before
	server, _, err := client.Server.Get(ctx, machine.Spec.Name)
	if err != nil {
		return fmt.Errorf("failed to get server: %v", err)
	}
	if server == nil {
		glog.Infof("No instance exists for machine %s", machine.Name)
		return nil
	}

	glog.Infof("Setting UID label for machine %s", machine.Name)
	_, response, err := client.Server.Update(ctx, server, hcloud.ServerUpdateOpts{
		Labels: map[string]string{machineUIDLabelKey: string(new)},
	})
	if err != nil {
		return fmt.Errorf("failed to update UID label: %v", err)
	}
	if response.Response.StatusCode != http.StatusOK {
		return fmt.Errorf("got unexpected response code %v, expected %v", response.Response.Status, http.StatusOK)
	}
	// This succeeds, but does not result in a label on the server, seems to be a bug
	// on Hetzner side
	glog.Infof("Successfully set UID label for machine %s", machine.Name)

	return nil
}

func (p *provider) GetCloudConfig(spec v1alpha1.MachineSpec) (config string, name string, err error) {
	return "", "", nil
}

func (p *provider) MachineMetricsLabels(machine *v1alpha1.Machine) (map[string]string, error) {
	labels := make(map[string]string)

	c, _, err := p.getConfig(machine.Spec.ProviderSpec)
	if err == nil {
		labels["size"] = c.ServerType
		labels["dc"] = c.Datacenter
		labels["location"] = c.Location
	}

	return labels, err
}

type hetznerServer struct {
	server *hcloud.Server
}

func (s *hetznerServer) Name() string {
	return s.server.Name
}

func (s *hetznerServer) ID() string {
	return strconv.Itoa(s.server.ID)
}

func (s *hetznerServer) Addresses() []string {
	var addresses []string
	for _, fips := range s.server.PublicNet.FloatingIPs {
		addresses = append(addresses, fips.IP.String())
	}

	return append(addresses, s.server.PublicNet.IPv4.IP.String(), s.server.PublicNet.IPv6.IP.String())
}

func (s *hetznerServer) Status() instance.Status {
	switch s.server.Status {
	case hcloud.ServerStatusInitializing:
		return instance.StatusCreating
	case hcloud.ServerStatusRunning:
		return instance.StatusRunning
	default:
		return instance.StatusUnknown
	}
}

// hzErrorToTerminalError judges if the given error
// can be qualified as a "terminal" error, for more info see v1alpha1.MachineStatus
//
// if the given error doesn't qualify the error passed as an argument will be returned
func hzErrorToTerminalError(err error, msg string) error {
	prepareAndReturnError := func() error {
		return fmt.Errorf("%s, due to %s", msg, err)
	}

	if err != nil {
		if hcloud.IsError(err, hcloud.ErrorCode("unauthorized")) {
			// authorization primitives come from MachineSpec
			// thus we are setting InvalidConfigurationMachineError
			return cloudprovidererrors.TerminalError{
				Reason:  common.InvalidConfigurationMachineError,
				Message: "A request has been rejected due to invalid credentials which were taken from the MachineSpec",
			}
		}

		return prepareAndReturnError()
	}

	return err
}

func (p *provider) SetMetricsForMachines(machines v1alpha1.MachineList) error {
	return nil
}
