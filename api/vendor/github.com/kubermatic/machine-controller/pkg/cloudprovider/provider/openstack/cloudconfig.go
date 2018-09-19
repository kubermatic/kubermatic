package openstack

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/ini.v1"
)

// Allowed escaping characters by gopkg.in/gcfg.v1 - the lib kubernetes uses
var escaper = strings.NewReplacer(
	`\`, `\\`,
	`"`, `\"`,
)

type LoadBalancerOpts struct {
	ManageSecurityGroups bool `gcfg:"manage-security-groups"`
}

type BlockStorageOpts struct {
	BSVersion       string `gcfg:"bs-version"`
	TrustDevicePath bool   `gcfg:"trust-device-path"`
	IgnoreVolumeAZ  bool   `gcfg:"ignore-volume-az"`
}

type GlobalOpts struct {
	AuthURL    string `gcfg:"auth-url"`
	Username   string
	Password   string
	TenantName string `gcfg:"tenant-name"`
	DomainName string `gcfg:"domain-name"`
	Region     string
}

// CloudConfig is used to read and store information from the cloud configuration file
type CloudConfig struct {
	Global       GlobalOpts
	LoadBalancer LoadBalancerOpts
	BlockStorage BlockStorageOpts
}

func CloudConfigToString(c *CloudConfig) (string, error) {
	cfg := ini.Empty()

	// Global
	gsec, err := cfg.NewSection("Global")
	if err != nil {
		return "", fmt.Errorf("failed to create the global section in ini: %v", err)
	}
	if _, err := gsec.NewKey("auth-url", escaper.Replace(c.Global.AuthURL)); err != nil {
		return "", fmt.Errorf("failed to write global.auth-url: %v", err)
	}
	if _, err := gsec.NewKey("username", escaper.Replace(c.Global.Username)); err != nil {
		return "", fmt.Errorf("failed to write global.username: %v", err)
	}
	if _, err := gsec.NewKey("password", escaper.Replace(c.Global.Password)); err != nil {
		return "", fmt.Errorf("failed to write global.password: %v", err)
	}
	if _, err := gsec.NewKey("tenant-name", escaper.Replace(c.Global.TenantName)); err != nil {
		return "", fmt.Errorf("failed to write global.tenant-name: %v", err)
	}
	if _, err := gsec.NewKey("domain-name", escaper.Replace(c.Global.DomainName)); err != nil {
		return "", fmt.Errorf("failed to write global.domain-name: %v", err)
	}
	if _, err := gsec.NewKey("region", escaper.Replace(c.Global.Region)); err != nil {
		return "", fmt.Errorf("failed to write global.region: %v", err)
	}

	// LoadBalancer
	lbsec, err := cfg.NewSection("LoadBalancer")
	if err != nil {
		return "", fmt.Errorf("failed to create the LoadBalancer section in ini: %v", err)
	}
	if _, err := lbsec.NewKey("manage-security-groups", strconv.FormatBool(c.LoadBalancer.ManageSecurityGroups)); err != nil {
		return "", fmt.Errorf("failed to write LoadBalancer.manage-security-groups: %v", err)
	}

	// BlockStorage
	bssec, err := cfg.NewSection("BlockStorage")
	if err != nil {
		return "", fmt.Errorf("failed to create the BlockStorage section in ini: %v", err)
	}
	if _, err := bssec.NewKey("ignore-volume-az", strconv.FormatBool(c.BlockStorage.IgnoreVolumeAZ)); err != nil {
		return "", fmt.Errorf("failed to write BlockStorage.ignore-volume-az: %v", err)
	}
	if _, err := bssec.NewKey("trust-device-path", strconv.FormatBool(c.BlockStorage.TrustDevicePath)); err != nil {
		return "", fmt.Errorf("failed to write BlockStorage.trust-device-path: %v", err)
	}
	if _, err := bssec.NewKey("bs-version", c.BlockStorage.BSVersion); err != nil {
		return "", fmt.Errorf("failed to write BlockStorage.bs-version: %v", err)
	}

	b := &bytes.Buffer{}
	if _, err := cfg.WriteTo(b); err != nil {
		return "", fmt.Errorf("failed to write ini to buffer: %v", err)
	}
	return b.String(), nil
}
