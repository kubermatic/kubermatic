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

	common "sigs.k8s.io/cluster-api/pkg/apis/cluster/common"
	"sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
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

func (p *provider) getConfig(s v1alpha1.ProviderConfig) (*Config, *providerconfig.Config, error) {
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
	c, pc, err := p.getConfig(spec.ProviderConfig)
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

func (p *provider) Create(machine *v1alpha1.Machine, _ cloud.MachineUpdater, userdata string) (instance.Instance, error) {
	c, pc, err := p.getConfig(machine.Spec.ProviderConfig)
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
	if err != nil || res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("creating temporary ssh key failed with http status %d and error %v", res.StatusCode, err)
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

func (p *provider) Delete(machine *v1alpha1.Machine, _ cloud.MachineUpdater) error {
	instance, err := p.Get(machine)
	if err != nil {
		if err == cloudprovidererrors.ErrInstanceNotFound {
			return nil
		}
		return err
	}

	c, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}

	ctx := context.TODO()
	client := getClient(c.Token)

	res, err := client.Server.Delete(ctx, instance.(*hetznerServer).server)
	if err != nil {
		return hzErrorToTerminalError(err, "failed to delete the server")
	}
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {
		return fmt.Errorf("invalid status code returned. expected=%d got=%d", http.StatusOK, res.StatusCode)
	}
	return nil
}

func (p *provider) AddDefaults(spec v1alpha1.MachineSpec) (v1alpha1.MachineSpec, bool, error) {
	return spec, false, nil
}

func (p *provider) Get(machine *v1alpha1.Machine) (instance.Instance, error) {
	c, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, cloudprovidererrors.TerminalError{
			Reason:  common.InvalidConfigurationMachineError,
			Message: fmt.Sprintf("Failed to parse MachineSpec, due to %v", err),
		}
	}

	ctx := context.TODO()
	client := getClient(c.Token)

	server, _, err := client.Server.Get(ctx, machine.Spec.Name)
	if err != nil {
		return nil, hzErrorToTerminalError(err, "failed to get the server object")
	}
	if server == nil {
		return nil, cloudprovidererrors.ErrInstanceNotFound
	}

	return &hetznerServer{server: server}, nil
}

func (p *provider) GetCloudConfig(spec v1alpha1.MachineSpec) (config string, name string, err error) {
	return "", "", nil
}

func (p *provider) MachineMetricsLabels(machine *v1alpha1.Machine) (map[string]string, error) {
	labels := make(map[string]string)

	c, _, err := p.getConfig(machine.Spec.ProviderConfig)
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
