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

package openstack

import (
	"bytes"
	"maps"
	"slices"
	"strconv"
	"time"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/openstack"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig/ini"
)

const (
	defaultLBMethod  = "ROUND_ROBIN"
	defaultBSVersion = "auto"
)

// The structs in this file mimic the original types for the
// CCM @ https://github.com/kubernetes/cloud-provider-openstack/blob/release-1.30/pkg/openstack/openstack.go
// CSI @ https://github.com/kubernetes/cloud-provider-openstack/blob/release-1.30/pkg/csi/cinder/openstack/openstack.go
// but were trimmed down to what KKP needs and to avoid a heavy dependency
// on the Openstack CCM Go module.
type CloudConfig struct {
	Global            GlobalOpts
	LoadBalancer      LoadBalancerOpts
	BlockStorage      BlockStorageOpts
	LoadBalancerClass LBClassOpts
}

func ForCluster(cluster *kubermaticv1.Cluster, dc *kubermaticv1.Datacenter, credentials resources.Credentials) CloudConfig {
	manageSecurityGroups := dc.Spec.Openstack.ManageSecurityGroups

	lbProvider := ""
	if dc.Spec.Openstack.LoadBalancerProvider != nil {
		lbProvider = *dc.Spec.Openstack.LoadBalancerProvider
	}

	lbMethod := ""
	if dc.Spec.Openstack.LoadBalancerMethod != nil {
		lbMethod = *dc.Spec.Openstack.LoadBalancerMethod
	}

	trustDevicePath := dc.Spec.Openstack.TrustDevicePath
	useOctavia := dc.Spec.Openstack.UseOctavia
	if cluster.Spec.Cloud.Openstack.UseOctavia != nil {
		useOctavia = cluster.Spec.Cloud.Openstack.UseOctavia
	}

	dcLbClasses := dc.Spec.Openstack.LoadBalancerClasses
	clusterLbClasses := cluster.Spec.Cloud.Openstack.LoadBalancerClasses

	// Preallocate for DC + Cluster classes
	lbClassOpts := make(LBClassOpts, len(dcLbClasses)+len(clusterLbClasses))

	addClasses := func(lbClasses []kubermaticv1.LoadBalancerClass) {
		for _, lbClass := range lbClasses {
			lbClassOpts[lbClass.Name] = &LBClass{
				FloatingNetworkID:  lbClass.Config.FloatingNetworkID,
				FloatingSubnetID:   lbClass.Config.FloatingSubnetID,
				FloatingSubnet:     lbClass.Config.FloatingSubnet,
				FloatingSubnetTags: lbClass.Config.FloatingSubnetTags,
				NetworkID:          lbClass.Config.NetworkID,
				SubnetID:           lbClass.Config.SubnetID,
				MemberSubnetID:     lbClass.Config.MemberSubnetID,
			}
		}
	}

	// DC first, then Cluster to allow overrides.
	if len(dcLbClasses) > 0 {
		addClasses(dcLbClasses)
	}
	if len(clusterLbClasses) > 0 {
		addClasses(clusterLbClasses)
	}

	cc := CloudConfig{
		Global: GlobalOpts{
			AuthURL:                     dc.Spec.Openstack.AuthURL,
			Username:                    credentials.Openstack.Username,
			Password:                    credentials.Openstack.Password,
			DomainName:                  credentials.Openstack.Domain,
			TenantName:                  credentials.Openstack.Project,
			TenantID:                    credentials.Openstack.ProjectID,
			Region:                      dc.Spec.Openstack.Region,
			ApplicationCredentialSecret: credentials.Openstack.ApplicationCredentialSecret,
			ApplicationCredentialID:     credentials.Openstack.ApplicationCredentialID,
		},
		BlockStorage: BlockStorageOpts{
			TrustDevicePath: trustDevicePath != nil && *trustDevicePath,
			IgnoreVolumeAZ:  dc.Spec.Openstack.IgnoreVolumeAZ,
		},
		LoadBalancer: LoadBalancerOpts{
			ManageSecurityGroups: manageSecurityGroups == nil || *manageSecurityGroups,
			LBMethod:             lbMethod,
			LBProvider:           lbProvider,
			UseOctavia:           useOctavia,
		},
		LoadBalancerClass: lbClassOpts,
	}

	// Set NodeVolumeAttachLimit if specified in the DC or Cluster spec
	if dc.Spec.Openstack.NodeVolumeAttachLimit != nil {
		cc.BlockStorage.NodeVolumeAttachLimit = *dc.Spec.Openstack.NodeVolumeAttachLimit
	}
	if cluster.Spec.Cloud.Openstack.NodeVolumeAttachLimit != nil {
		cc.BlockStorage.NodeVolumeAttachLimit = *cluster.Spec.Cloud.Openstack.NodeVolumeAttachLimit
	}

	// Ingress hostname settings
	if cluster.Spec.Cloud.Openstack.EnableIngressHostname != nil {
		cc.LoadBalancer.EnableIngressHostname = cluster.Spec.Cloud.Openstack.EnableIngressHostname
	}

	if cluster.Spec.Cloud.Openstack.IngressHostnameSuffix != nil {
		cc.LoadBalancer.IngressHostnameSuffix = cluster.Spec.Cloud.Openstack.IngressHostnameSuffix
	}

	if cluster.Annotations != nil {
		// prefer LoadBalancer-specific floating IP pool if set, otherwise fallback to the default one.
		if cluster.Annotations[openstack.LoadBalancerFloatingIPPoolIDAnnotation] != "" {
			cc.LoadBalancer.FloatingNetworkID = cluster.Annotations[openstack.LoadBalancerFloatingIPPoolIDAnnotation]
		} else if cluster.Annotations[openstack.FloatingIPPoolIDAnnotation] != "" {
			cc.LoadBalancer.FloatingNetworkID = cluster.Annotations[openstack.FloatingIPPoolIDAnnotation]
		}
	}

	return cc
}

func (c *CloudConfig) String() (string, error) {
	out := ini.New()

	global := out.Section("Global", "")
	c.Global.toINI(global)

	lb := out.Section("LoadBalancer", "")
	c.LoadBalancer.toINI(lb)

	bs := out.Section("BlockStorage", "")
	c.BlockStorage.toINI(bs)

	for _, name := range slices.Sorted(maps.Keys(c.LoadBalancerClass)) {
		if name == "" {
			continue
		}

		lbClass := out.Section("LoadBalancerClass", name)
		opts, ok := c.LoadBalancerClass[name]
		if !ok {
			continue
		}

		opts.toINI(lbClass)
	}

	buf := &bytes.Buffer{}
	if err := out.Render(buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

type LoadBalancerOpts struct {
	UseOctavia           *bool
	SubnetID             string
	FloatingNetworkID    string
	LBMethod             string
	LBProvider           string
	CreateMonitor        bool
	MonitorDelay         time.Duration
	MonitorTimeout       time.Duration
	MonitorMaxRetries    uint
	ManageSecurityGroups bool

	EnableIngressHostname *bool
	IngressHostnameSuffix *string
}

func (o *LoadBalancerOpts) toINI(section ini.Section) {
	section.AddBoolKey("manage-security-groups", o.ManageSecurityGroups)
	section.AddStringKey("lb-version", "v2")
	section.AddStringKey("lb-provider", o.LBProvider)
	section.AddStringKey("subnet-id", o.SubnetID)
	section.AddStringKey("floating-network-id", o.FloatingNetworkID)

	method := o.LBMethod
	if method == "" {
		method = defaultLBMethod
	}
	section.AddStringKey("lb-method", method)

	if val := o.UseOctavia; val != nil {
		section.AddBoolKey("use-octavia", *val)
	}

	if enable := o.EnableIngressHostname; enable != nil {
		section.AddBoolKey("enable-ingress-hostname", *enable)

		if suffix := o.IngressHostnameSuffix; suffix != nil {
			section.AddStringKey("ingress-hostname-suffix", *suffix)
		}
	}

	if o.CreateMonitor {
		section.AddBoolKey("create-monitor", true)
		section.AddStringKey("monitor-delay", o.MonitorDelay.String())
		section.AddStringKey("monitor-timeout", o.MonitorTimeout.String())
		section.AddStringKey("monitor-max-retries", strconv.FormatUint(uint64(o.MonitorMaxRetries), 10))
	}
}

type BlockStorageOpts struct {
	BSVersion             string
	TrustDevicePath       bool
	IgnoreVolumeAZ        bool
	NodeVolumeAttachLimit uint
}

func (o *BlockStorageOpts) toINI(section ini.Section) {
	section.AddBoolKey("ignore-volume-az", o.IgnoreVolumeAZ)
	section.AddBoolKey("trust-device-path", o.TrustDevicePath)

	version := o.BSVersion
	if version == "" {
		version = defaultBSVersion
	}
	section.AddStringKey("bs-version", version)

	if limit := o.NodeVolumeAttachLimit; limit != 0 {
		section.AddStringKey("node-volume-attach-limit", strconv.FormatUint(uint64(limit), 10))
	}
}

type GlobalOpts struct {
	AuthURL                     string
	Username                    string
	Password                    string
	ApplicationCredentialID     string
	ApplicationCredentialSecret string
	TenantName                  string
	TenantID                    string
	DomainName                  string
	Region                      string
}

func (o *GlobalOpts) toINI(section ini.Section) {
	section.AddStringKey("auth-url", o.AuthURL)
	section.AddStringKey("region", o.Region)

	if o.ApplicationCredentialID != "" {
		section.AddStringKey("application-credential-id", o.ApplicationCredentialID)
		section.AddStringKey("application-credential-secret", o.ApplicationCredentialSecret)
	} else {
		section.AddStringKey("username", o.Username)
		section.AddStringKey("password", o.Password)
		section.AddStringKey("tenant-name", o.TenantName)
		section.AddStringKey("tenant-id", o.TenantID)
		section.AddStringKey("domain-name", o.DomainName)
	}
}

// LBClassOpts is a map of LoadBalancerClass names to their corresponding options.
// The key is the name of the LoadBalancerClass, and the value is the corresponding options.
type LBClassOpts map[string]*LBClass

// LBClass defines the corresponding floating network, floating subnet or internal subnet ID
// for the LoadBalancerClass section of the cloud.cfg.
type LBClass struct {
	FloatingNetworkID  string
	FloatingSubnetID   string
	FloatingSubnet     string
	FloatingSubnetTags string
	NetworkID          string
	SubnetID           string
	MemberSubnetID     string
}

func (o *LBClass) toINI(section ini.Section) {
	addIfNotEmpty := func(key, value string) {
		if value != "" {
			section.AddStringKey(key, value)
		}
	}

	addIfNotEmpty("floating-network-id", o.FloatingNetworkID)
	addIfNotEmpty("floating-subnet-id", o.FloatingSubnetID)
	addIfNotEmpty("floating-subnet", o.FloatingSubnet)
	addIfNotEmpty("floating-subnet-tags", o.FloatingSubnetTags)
	addIfNotEmpty("network-id", o.NetworkID)
	addIfNotEmpty("subnet-id", o.SubnetID)
	addIfNotEmpty("member-subnet-id", o.MemberSubnetID)
}
