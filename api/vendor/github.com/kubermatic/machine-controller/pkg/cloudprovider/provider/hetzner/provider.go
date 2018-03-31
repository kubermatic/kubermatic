package hetzner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/golang/glog"
	"github.com/hetznercloud/hcloud-go/hcloud"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/kubermatic/machine-controller/pkg/cloudprovider/cloud"
	cloudprovidererrors "github.com/kubermatic/machine-controller/pkg/cloudprovider/errors"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/instance"
	"github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
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

// Protects creation of public key
var publicKeyCreationLock = &sync.Mutex{}

func getNameForOS(os providerconfig.OperatingSystem) (string, error) {
	switch os {
	case providerconfig.OperatingSystemUbuntu:
		return "ubuntu-16.04", nil
	}
	return "", providerconfig.ErrOSNotSupported
}

func getClient(token string) *hcloud.Client {
	return hcloud.NewClient(hcloud.WithToken(token))
}

func (p *provider) getConfig(s runtime.RawExtension) (*Config, *providerconfig.Config, error) {
	pconfig := providerconfig.Config{}
	err := json.Unmarshal(s.Raw, &pconfig)
	if err != nil {
		return nil, nil, err
	}

	rawConfig := RawConfig{}
	err = json.Unmarshal(pconfig.CloudProviderSpec.Raw, &rawConfig)

	c := Config{}
	c.Token, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Token)
	if err != nil {
		return nil, nil, err
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

func (p *provider) Create(machine *v1alpha1.Machine, userdata string) (instance.Instance, error) {
	c, pc, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	ctx := context.TODO()
	client := getClient(c.Token)

	imageName, err := getNameForOS(pc.OperatingSystem)
	if err != nil {
		return nil, fmt.Errorf("invalid operating system specified %q: %v", pc.OperatingSystem, err)
	}

	serverCreateOpts := hcloud.ServerCreateOpts{
		Name:     machine.Spec.Name,
		UserData: userdata,
	}

	if c.Datacenter != "" {
		serverCreateOpts.Datacenter, _, err = client.Datacenter.Get(ctx, c.Datacenter)
		if err != nil {
			return nil, fmt.Errorf("failed to get datacenter: %v", err)
		}
	}

	if c.Location != "" {
		serverCreateOpts.Location, _, err = client.Location.Get(ctx, c.Location)
		if err != nil {
			return nil, fmt.Errorf("failed to get location: %v", err)
		}
	}

	serverCreateOpts.Image, _, err = client.Image.Get(ctx, imageName)
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %v", err)
	}

	serverCreateOpts.ServerType, _, err = client.ServerType.Get(ctx, c.ServerType)
	if err != nil {
		return nil, fmt.Errorf("failed to get server type: %v", err)
	}

	serverCreateRes, res, err := client.Server.Create(ctx, serverCreateOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %v", err)
	}
	if res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create server invalid status code returned. expected=%d got %d", http.StatusCreated, res.StatusCode)
	}

	return &hetznerServer{server: serverCreateRes.Server}, nil
}

func (p *provider) Delete(machine *v1alpha1.Machine) error {
	c, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	ctx := context.TODO()
	client := getClient(c.Token)
	i, err := p.Get(machine)
	if err != nil {
		if err == cloudprovidererrors.ErrInstanceNotFound {
			glog.V(4).Info("instance already deleted")
			return nil
		}
		return err
	}

	res, err := client.Server.Delete(ctx, i.(*hetznerServer).server)
	if err != nil {
		return fmt.Errorf("failed to delete the server: %v", err)
	}
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNotFound {
		return fmt.Errorf("invalid status code returned. expected=%d got=%d", http.StatusOK, res.StatusCode)
	}
	return err
}

func (p *provider) AddDefaults(spec v1alpha1.MachineSpec) (v1alpha1.MachineSpec, bool, error) {
	return spec, false, nil
}

func (p *provider) Get(machine *v1alpha1.Machine) (instance.Instance, error) {
	c, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	ctx := context.TODO()
	client := getClient(c.Token)

	server, _, err := client.Server.Get(ctx, machine.Spec.Name)
	if err != nil {
		return nil, err
	}
	if server == nil {
		return nil, cloudprovidererrors.ErrInstanceNotFound
	}

	return &hetznerServer{server: server}, nil
}

func (p *provider) GetCloudConfig(spec v1alpha1.MachineSpec) (config string, name string, err error) {
	return "", "", nil
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

func (d *hetznerServer) Status() instance.Status {
	switch d.server.Status {
	case hcloud.ServerStatusInitializing:
		return instance.StatusCreating
	case hcloud.ServerStatusRunning:
		return instance.StatusRunning
	default:
		return instance.StatusUnknown
	}
}
