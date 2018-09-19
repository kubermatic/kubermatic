package vsphere

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

type WorkspaceOpts struct {
	VCenterIP        string `gcfg:"server"`
	Datacenter       string `gcfg:"datacenter"`
	Folder           string `gcfg:"folder"`
	DefaultDatastore string `gcfg:"default-datastore"`
	ResourcePoolPath string `gcfg:"resourcepool-path"`
}

type DiskOpts struct {
	SCSIControllerType string `dcfg:"scsicontrollertype"`
}

type GlobalOpts struct {
	User         string `gcfg:"user"`
	Password     string `gcfg:"password"`
	InsecureFlag bool   `gcfg:"insecure-flag"`
	VCenterPort  string `gcfg:"port"`
}

type VirtualCenterConfig struct {
	User        string `gcfg:"user"`
	Password    string `gcfg:"password"`
	VCenterPort string `gcfg:"port"`
	Datacenters string `gcfg:"datacenters"`
}

// CloudConfig is used to read and store information from the cloud configuration file
type CloudConfig struct {
	Global    GlobalOpts
	Disk      DiskOpts
	Workspace WorkspaceOpts

	VirtualCenter map[string]*VirtualCenterConfig
}

func CloudConfigToString(c *CloudConfig) (string, error) {
	cfg := ini.Empty()

	// Global
	gsec, err := cfg.NewSection("Global")
	if err != nil {
		return "", fmt.Errorf("failed to create the global section in ini: %v", err)
	}
	if _, err := gsec.NewKey("user", escaper.Replace(c.Global.User)); err != nil {
		return "", fmt.Errorf("failed to write global.user: %v", err)
	}
	if _, err := gsec.NewKey("password", escaper.Replace(c.Global.Password)); err != nil {
		return "", fmt.Errorf("failed to write global.password: %v", err)
	}
	if _, err := gsec.NewKey("port", c.Global.VCenterPort); err != nil {
		return "", fmt.Errorf("failed to write global.port: %v", err)
	}
	if _, err := gsec.NewKey("insecure-flag", strconv.FormatBool(c.Global.InsecureFlag)); err != nil {
		return "", fmt.Errorf("failed to write global.insecure-flag: %v", err)
	}

	// Disk
	dsec, err := cfg.NewSection("Disk")
	if err != nil {
		return "", fmt.Errorf("failed to create the Disk section in ini: %v", err)
	}
	if _, err := dsec.NewKey("scsicontrollertype", escaper.Replace(c.Disk.SCSIControllerType)); err != nil {
		return "", fmt.Errorf("failed to write Disk.scsicontrollertype: %v", err)
	}

	// Workspace
	wsec, err := cfg.NewSection("Workspace")
	if err != nil {
		return "", fmt.Errorf("failed to create the Workspace section in ini: %v", err)
	}
	if _, err := wsec.NewKey("server", escaper.Replace(c.Workspace.VCenterIP)); err != nil {
		return "", fmt.Errorf("failed to write Workspace.server: %v", err)
	}
	if _, err := wsec.NewKey("datacenter", escaper.Replace(c.Workspace.Datacenter)); err != nil {
		return "", fmt.Errorf("failed to write Workspace.datacenter: %v", err)
	}
	if _, err := wsec.NewKey("folder", escaper.Replace(c.Workspace.Folder)); err != nil {
		return "", fmt.Errorf("failed to write Workspace.folder: %v", err)
	}
	if _, err := wsec.NewKey("default-datastore", escaper.Replace(c.Workspace.DefaultDatastore)); err != nil {
		return "", fmt.Errorf("failed to write Workspace.default-datastore: %v", err)
	}
	if _, err := wsec.NewKey("resourcepool-path", escaper.Replace(c.Workspace.ResourcePoolPath)); err != nil {
		return "", fmt.Errorf("failed to write Workspace.resourcepool-path: %v", err)
	}

	for ip, vc := range c.VirtualCenter {
		sectionName := fmt.Sprintf("VirtualCenter \"%s\"", ip)
		vcsec, err := cfg.NewSection(sectionName)
		if err != nil {
			return "", fmt.Errorf("failed to create the %s section in ini: %v", sectionName, err)
		}

		if _, err := vcsec.NewKey("user", escaper.Replace(vc.User)); err != nil {
			return "", fmt.Errorf("failed to write %s.user: %v", sectionName, err)
		}
		if _, err := vcsec.NewKey("password", escaper.Replace(vc.Password)); err != nil {
			return "", fmt.Errorf("failed to write %s.password: %v", sectionName, err)
		}
		if _, err := vcsec.NewKey("port", escaper.Replace(vc.VCenterPort)); err != nil {
			return "", fmt.Errorf("failed to write %s.port: %v", sectionName, err)
		}
		if _, err := vcsec.NewKey("datacenters", escaper.Replace(vc.Datacenters)); err != nil {
			return "", fmt.Errorf("failed to write %s.datacenters: %v", sectionName, err)
		}
	}

	b := &bytes.Buffer{}
	if _, err := cfg.WriteTo(b); err != nil {
		return "", fmt.Errorf("failed to write ini to buffer: %v", err)
	}
	return b.String(), nil
}
