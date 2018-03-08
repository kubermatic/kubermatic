package digitalocean

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/digitalocean/godo"
	"github.com/golang/glog"
	"golang.org/x/crypto/ssh"
	"golang.org/x/oauth2"

	"github.com/kubermatic/machine-controller/pkg/cloudprovider/cloud"
	cloudprovidererrors "github.com/kubermatic/machine-controller/pkg/cloudprovider/errors"
	"github.com/kubermatic/machine-controller/pkg/cloudprovider/instance"
	"github.com/kubermatic/machine-controller/pkg/machines/v1alpha1"
	"github.com/kubermatic/machine-controller/pkg/providerconfig"
	"github.com/pborman/uuid"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
)

const privateRSAKeyBitSize = 4096

type provider struct {
	configVarResolver *providerconfig.ConfigVarResolver
}

// New returns a digitalocean provider
func New(configVarResolver *providerconfig.ConfigVarResolver) cloud.Provider {
	return &provider{configVarResolver: configVarResolver}
}

type RawConfig struct {
	Token             providerconfig.ConfigVarString   `json:"token"`
	Region            providerconfig.ConfigVarString   `json:"region"`
	Size              providerconfig.ConfigVarString   `json:"size"`
	Backups           providerconfig.ConfigVarBool     `json:"backups"`
	IPv6              providerconfig.ConfigVarBool     `json:"ipv6"`
	PrivateNetworking providerconfig.ConfigVarBool     `json:"private_networking"`
	Monitoring        providerconfig.ConfigVarBool     `json:"monitoring"`
	Tags              []providerconfig.ConfigVarString `json:"tags"`
}

type Config struct {
	Token             string
	Region            string
	Size              string
	Backups           bool
	IPv6              bool
	PrivateNetworking bool
	Monitoring        bool
	Tags              []string
}

const (
	createCheckPeriod           = 10 * time.Second
	createCheckTimeout          = 5 * time.Minute
	createCheckFailedWaitPeriod = 10 * time.Second
)

type TokenSource struct {
	AccessToken string
}

func (t *TokenSource) Token() (*oauth2.Token, error) {
	token := &oauth2.Token{
		AccessToken: t.AccessToken,
	}
	return token, nil
}

func getSlugForOS(os providerconfig.OperatingSystem) (string, error) {
	switch os {
	case providerconfig.OperatingSystemUbuntu:
		return "ubuntu-16-04-x64", nil
	case providerconfig.OperatingSystemCoreos:
		return "coreos-stable", nil
	}
	return "", providerconfig.ErrOSNotSupported
}

func getClient(token string) *godo.Client {
	tokenSource := &TokenSource{
		AccessToken: token,
	}

	oauthClient := oauth2.NewClient(context.Background(), tokenSource)
	return godo.NewClient(oauthClient)
}

func (p *provider) getConfig(s runtime.RawExtension) (*Config, *providerconfig.Config, error) {
	pconfig := providerconfig.Config{}
	err := json.Unmarshal(s.Raw, &pconfig)
	if err != nil {
		return nil, nil, err
	}
	rawConfig := RawConfig{}
	err = json.Unmarshal(pconfig.CloudProviderSpec.Raw, &rawConfig)
	if err != nil {
		return nil, nil, err
	}

	c := Config{}
	c.Token, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Token)
	if err != nil {
		return nil, nil, err
	}
	c.Region, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Region)
	if err != nil {
		return nil, nil, err
	}
	c.Size, err = p.configVarResolver.GetConfigVarStringValue(rawConfig.Size)
	if err != nil {
		return nil, nil, err
	}
	c.Backups, err = p.configVarResolver.GetConfigVarBoolValue(rawConfig.Backups)
	if err != nil {
		return nil, nil, err
	}
	c.IPv6, err = p.configVarResolver.GetConfigVarBoolValue(rawConfig.IPv6)
	if err != nil {
		return nil, nil, err
	}
	c.PrivateNetworking, err = p.configVarResolver.GetConfigVarBoolValue(rawConfig.PrivateNetworking)
	if err != nil {
		return nil, nil, err
	}
	c.Monitoring, err = p.configVarResolver.GetConfigVarBoolValue(rawConfig.Monitoring)
	if err != nil {
		return nil, nil, err
	}
	for _, tag := range rawConfig.Tags {
		tagVal, err := p.configVarResolver.GetConfigVarStringValue(tag)
		if err != nil {
			return nil, nil, err
		}
		c.Tags = append(c.Tags, tagVal)
	}

	return &c, &pconfig, err
}

func (p *provider) AddDefaults(spec v1alpha1.MachineSpec) (v1alpha1.MachineSpec, bool, error) {
	return spec, false, nil
}

func (p *provider) Validate(spec v1alpha1.MachineSpec) error {
	c, pc, err := p.getConfig(spec.ProviderConfig)
	if err != nil {
		return fmt.Errorf("failed to parse config: %v", err)
	}

	if c.Token == "" {
		return errors.New("token is missing")
	}

	if c.Region == "" {
		return errors.New("region is missing")
	}

	if c.Size == "" {
		return errors.New("size is missing")
	}

	_, err = getSlugForOS(pc.OperatingSystem)
	if err != nil {
		return fmt.Errorf("invalid operating system specified %q: %v", pc.OperatingSystem, err)
	}

	ctx := context.TODO()
	client := getClient(c.Token)

	regions, _, err := client.Regions.List(ctx, &godo.ListOptions{PerPage: 1000})
	if err != nil {
		return err
	}
	var foundRegion bool
	for _, region := range regions {
		if region.Slug == c.Region {
			foundRegion = true
			break
		}
	}
	if !foundRegion {
		return fmt.Errorf("region %q not found", c.Region)
	}

	sizes, _, err := client.Sizes.List(ctx, &godo.ListOptions{PerPage: 1000})
	if err != nil {
		return err
	}
	var foundSize bool
	for _, size := range sizes {
		if size.Slug == c.Size {
			if !size.Available {
				return fmt.Errorf("size is not available")
			}

			var regionAvailable bool
			for _, region := range size.Regions {
				if region == c.Region {
					regionAvailable = true
					break
				}
			}

			if !regionAvailable {
				return fmt.Errorf("size %q is not available in region %q", c.Size, c.Region)
			}

			foundSize = true
			break
		}
	}
	if !foundSize {
		return fmt.Errorf("size %q not found", c.Size)
	}

	return nil
}

// uploadSSHPublicKey uploads public part of the key to digital ocean
// this method returns an error if the key already exists
func uploadSSHPublicKey(ctx context.Context, service godo.KeysService, key *rsa.PublicKey) (string, error) {
	pk, err := ssh.NewPublicKey(key)
	if err != nil {
		return "", fmt.Errorf("failed to parse publickey: %v", err)
	}

	fingerprint := ssh.FingerprintLegacyMD5(pk)
	existingkey, res, err := service.GetByFingerprint(ctx, fingerprint)
	if err == nil && existingkey != nil && res.StatusCode >= http.StatusOK && res.StatusCode <= http.StatusAccepted {
		return "", fmt.Errorf("failed to create ssh public key, the key already exists")
	}

	newDoKey, _, err := service.Create(ctx, &godo.KeyCreateRequest{
		PublicKey: string(ssh.MarshalAuthorizedKey(pk)),
		Name:      string(uuid.NewUUID()),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create ssh public key on digitalocean: %v", err)
	}

	return newDoKey.Fingerprint, nil
}

func (p *provider) Create(machine *v1alpha1.Machine, userdata string) (instance.Instance, error) {
	c, pc, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	ctx := context.TODO()
	client := getClient(c.Token)

	tmpRSAKeyPair, err := rsa.GenerateKey(rand.Reader, privateRSAKeyBitSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create private RSA key: %v", err)
	}

	if err := tmpRSAKeyPair.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate private RSA key: %v", err)
	}
	fingerprint, err := uploadSSHPublicKey(ctx, client.Keys, &tmpRSAKeyPair.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed upload the ssh key, error = %v", err)
	}
	defer func() {
		_, err := client.Keys.DeleteByFingerprint(ctx, fingerprint)
		if err != nil {
			glog.Errorf("failed to remove a temporary ssh key with fingerprint = %v, due to = %v", fingerprint, err)
		}
	}()

	slug, err := getSlugForOS(pc.OperatingSystem)
	if err != nil {
		return nil, fmt.Errorf("invalid operating system specified %q: %v", pc.OperatingSystem, err)
	}
	createRequest := &godo.DropletCreateRequest{
		Image:             godo.DropletCreateImage{Slug: slug},
		Name:              machine.Spec.Name,
		Region:            c.Region,
		Size:              c.Size,
		IPv6:              c.IPv6,
		PrivateNetworking: c.PrivateNetworking,
		Backups:           c.Backups,
		Monitoring:        c.Monitoring,
		UserData:          userdata,
		SSHKeys:           []godo.DropletCreateSSHKey{{Fingerprint: fingerprint}},
		Tags:              append(c.Tags, string(machine.UID)),
	}

	droplet, _, err := client.Droplets.Create(ctx, createRequest)
	if err != nil {
		return nil, err
	}

	//We need to wait until the droplet really got created as tags will be only applied when the droplet is running
	err = wait.Poll(createCheckPeriod, createCheckTimeout, func() (done bool, err error) {
		newDroplet, _, err := client.Droplets.Get(ctx, droplet.ID)
		if err != nil {
			//Well just wait 10 sec and hope the droplet got started by then...
			time.Sleep(createCheckFailedWaitPeriod)
			return false, fmt.Errorf("droplet (id='%d') got created but we failed to fetch its status", droplet.ID)
		}
		if sets.NewString(newDroplet.Tags...).Has(string(machine.UID)) {
			glog.V(6).Infof("droplet (id='%d') got fully created", droplet.ID)
			return true, nil
		}
		glog.V(6).Infof("waiting until droplet (id='%d') got fully created...", droplet.ID)
		return false, nil
	})

	return &doInstance{droplet: droplet}, nil
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
	doID, err := strconv.Atoi(i.ID())
	if err != nil {
		return fmt.Errorf("failed to convert instance id %s to int: %v", i.ID(), err)
	}
	_, err = client.Droplets.Delete(ctx, doID)
	return err
}

func (p *provider) Get(machine *v1alpha1.Machine) (instance.Instance, error) {
	c, _, err := p.getConfig(machine.Spec.ProviderConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %v", err)
	}

	ctx := context.TODO()
	client := getClient(c.Token)
	droplets, _, err := client.Droplets.List(ctx, &godo.ListOptions{PerPage: 1000})
	if err != nil {
		return nil, fmt.Errorf("failed to get droplets: %v", err)
	}

	for i, droplet := range droplets {
		if droplet.Name == machine.Spec.Name && sets.NewString(droplet.Tags...).Has(string(machine.UID)) {
			return &doInstance{droplet: &droplets[i]}, nil
		}
	}

	return nil, cloudprovidererrors.ErrInstanceNotFound
}

func (p *provider) GetCloudConfig(spec v1alpha1.MachineSpec) (config string, name string, err error) {
	return "", "", nil
}

type doInstance struct {
	droplet *godo.Droplet
}

func (d *doInstance) Name() string {
	return d.droplet.Name
}

func (d *doInstance) ID() string {
	return strconv.Itoa(d.droplet.ID)
}

func (d *doInstance) Addresses() []string {
	var addresses []string
	for _, n := range d.droplet.Networks.V4 {
		addresses = append(addresses, n.IPAddress)
	}
	for _, n := range d.droplet.Networks.V6 {
		addresses = append(addresses, n.IPAddress)
	}
	return addresses
}

func (d *doInstance) Status() instance.Status {
	switch d.droplet.Status {
	case "new":
		return instance.StatusCreating
	case "active":
		return instance.StatusRunning
	default:
		return instance.StatusUnknown
	}
}
