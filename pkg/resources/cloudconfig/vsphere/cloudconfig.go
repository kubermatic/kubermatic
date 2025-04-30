/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package vsphere

import (
	"bytes"
	"fmt"
	"net/url"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig/ini"

	"k8s.io/apimachinery/pkg/util/sets"
)

type CloudConfig struct {
	Global    GlobalOpts
	Disk      DiskOpts
	Workspace WorkspaceOpts

	VirtualCenter map[string]VirtualCenterConfig
}

func ForCluster(cluster *kubermaticv1.Cluster, dc *kubermaticv1.Datacenter, credentials resources.Credentials) (*CloudConfig, error) {
	vsphereURL, err := url.Parse(dc.Spec.VSphere.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse vsphere endpoint: %w", err)
	}
	port := "443"
	if urlPort := vsphereURL.Port(); urlPort != "" {
		port = urlPort
	}
	datastore := dc.Spec.VSphere.DefaultDatastore
	// if a datastore is provided at cluster level override the default
	// datastore provided at datacenter level.
	if cluster.Spec.Cloud.VSphere.Datastore != "" {
		datastore = cluster.Spec.Cloud.VSphere.Datastore
	} else if cluster.Spec.Cloud.VSphere.DatastoreCluster != "" {
		datastore = cluster.Spec.Cloud.VSphere.DatastoreCluster
	}

	// Originally, we have been setting cluster-id to the vSphere Compute Cluster name
	// (provided via the Datacenter object), however, this is supposed to identify the
	// Kubernetes cluster, therefore it must be unique. This feature flag is enabled by
	// default for new vSphere clusters, while existing vSphere clusters must be
	// migrated manually (preferably by following advice here:
	// https://kb.vmware.com/s/article/84446).
	clusterID := dc.Spec.VSphere.Cluster
	if cluster.Spec.Features[kubermaticv1.ClusterFeatureVsphereCSIClusterID] {
		clusterID = cluster.Name
	}

	cc := &CloudConfig{
		Global: GlobalOpts{
			User:             credentials.VSphere.Username,
			Password:         credentials.VSphere.Password,
			VCenterIP:        vsphereURL.Hostname(),
			VCenterPort:      port,
			InsecureFlag:     dc.Spec.VSphere.AllowInsecure,
			Datacenter:       dc.Spec.VSphere.Datacenter,
			DefaultDatastore: datastore,
			WorkingDir:       cluster.Name,
			ClusterID:        clusterID,
		},
		Workspace: WorkspaceOpts{
			// This is redundant with what the Vsphere cloud provider itself does:
			// https://github.com/kubernetes/kubernetes/blob/9d80e7522ab7fc977e40dd6f3b5b16d8ebfdc435/pkg/cloudprovider/providers/vsphere/vsphere.go#L346
			// We do it here because the fields in the "Global" object
			// are marked as deprecated even thought the code checks
			// if they are set and will make the controller-manager crash
			// if they are not - But maybe that will change at some point
			VCenterIP:        vsphereURL.Hostname(),
			Datacenter:       dc.Spec.VSphere.Datacenter,
			Folder:           cluster.Spec.Cloud.VSphere.Folder,
			DefaultDatastore: datastore,
		},
		Disk: DiskOpts{
			SCSIControllerType: "pvscsi",
		},
		VirtualCenter: map[string]VirtualCenterConfig{
			vsphereURL.Hostname(): {
				User:        credentials.VSphere.Username,
				Password:    credentials.VSphere.Password,
				VCenterPort: port,
				Datacenters: dc.Spec.VSphere.Datacenter,
			},
		},
	}

	if cluster.IsDualStack() && resources.ExternalCloudProviderEnabled(cluster) {
		cc.Global.IPFamily = "ipv4,ipv6"
	}

	return cc, nil
}

func (c *CloudConfig) String() (string, error) {
	out := ini.New()

	global := out.Section("Global", "")
	c.Global.toINI(global)

	disk := out.Section("Disk", "")
	c.Disk.toINI(disk)

	workspace := out.Section("Workspace", "")
	c.Workspace.toINI(workspace)

	// ensure a stable iteration order
	vcNames := sets.List(sets.KeySet(c.VirtualCenter))

	for _, name := range vcNames {
		section := out.Section("VirtualCenter", name)
		c.VirtualCenter[name].toINI(section)
	}

	buf := &bytes.Buffer{}
	if err := out.Render(buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

type WorkspaceOpts struct {
	VCenterIP        string
	Datacenter       string
	Folder           string
	DefaultDatastore string
	ResourcePoolPath string
}

func (o *WorkspaceOpts) toINI(section ini.Section) {
	section.AddStringKey("server", o.VCenterIP)
	section.AddStringKey("datacenter", o.Datacenter)
	section.AddStringKey("folder", o.Folder)
	section.AddStringKey("default-datastore", o.DefaultDatastore)
	section.AddStringKey("resourcepool-path", o.ResourcePoolPath)
}

type DiskOpts struct {
	SCSIControllerType string
}

func (o *DiskOpts) toINI(section ini.Section) {
	section.AddStringKey("scsicontrollertype", o.SCSIControllerType)
}

type GlobalOpts struct {
	User             string
	Password         string
	InsecureFlag     bool
	VCenterPort      string
	WorkingDir       string
	Datacenter       string
	DefaultDatastore string
	VCenterIP        string
	ClusterID        string
	IPFamily         string
}

func (o *GlobalOpts) toINI(section ini.Section) {
	section.AddStringKey("user", o.User)
	section.AddStringKey("password", o.Password)
	section.AddStringKey("port", o.VCenterPort)
	section.AddBoolKey("insecure-flag", o.InsecureFlag)
	section.AddStringKey("working-dir", o.WorkingDir)
	section.AddStringKey("datacenter", o.Datacenter)
	section.AddStringKey("datastore", o.DefaultDatastore)
	section.AddStringKey("server", o.VCenterIP)
	section.AddStringKey("cluster-id", o.ClusterID)

	if o.IPFamily != "" {
		section.AddStringKey("ip-family", o.IPFamily)
	}
}

type VirtualCenterConfig struct {
	User        string
	Password    string
	VCenterPort string
	Datacenters string
	IPFamily    string
}

func (o VirtualCenterConfig) toINI(section ini.Section) {
	section.AddStringKey("user", o.User)
	section.AddStringKey("password", o.Password)
	section.AddStringKey("port", o.VCenterPort)
	section.AddStringKey("datacenters", o.Datacenters)

	if o.IPFamily != "" {
		section.AddStringKey("ip-family", o.IPFamily)
	}
}
