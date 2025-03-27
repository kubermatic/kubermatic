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

package gcp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider/cloud/gcp"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/resources/cloudconfig/ini"
)

type CloudConfig struct {
	Global GlobalOpts
}

func ForCluster(cluster *kubermaticv1.Cluster, dc *kubermaticv1.Datacenter, credentials resources.Credentials) (*CloudConfig, error) {
	b, err := base64.StdEncoding.DecodeString(credentials.GCP.ServiceAccount)
	if err != nil {
		return nil, fmt.Errorf("error decoding service account: %w", err)
	}

	sam := map[string]string{}
	err = json.Unmarshal(b, &sam)
	if err != nil {
		return nil, fmt.Errorf("failed unmarshalling service account: %w", err)
	}

	projectID := sam["project_id"]
	if projectID == "" {
		return nil, errors.New("empty project_id")
	}

	tag := fmt.Sprintf("kubernetes-cluster-%s", cluster.Name)

	if len(dc.Spec.GCP.ZoneSuffixes) == 0 {
		return nil, errors.New("empty zoneSuffixes")
	}

	localZone := dc.Spec.GCP.Region + "-" + dc.Spec.GCP.ZoneSuffixes[0]

	// By default, all GCP clusters are assumed to be the in the same zone. If the control plane
	// and worker nodes are not it the same zone (localZone), the GCP cloud controller fails
	// to find nodes that are not in the localZone: https://github.com/kubermatic/kubermatic/issues/5025
	// to avoid this, we should enable multizone or regional configuration.
	// It's not easily possible to access the MachineDeployment object from here to compare
	// localZone with the user cluster zone. Additionally, ZoneSuffixes are not used
	// to limit available zones for the user. So, we will just enable multizone support as a workaround.

	// FIXME: Compare localZone to MachineDeployment.Zone and set multizone to true
	// when they differ, or if len(dc.Spec.GCP.ZoneSuffixes) > 1
	multizone := true

	network := cluster.Spec.Cloud.GCP.Network

	if network == "" || network == gcp.DefaultNetwork {
		// NetworkName is used by the gce cloud provider to populate the provider's NetworkURL.
		// This value can be provided in the config as a name or a url. Internally,
		// the gce cloud provider checks it and if it's a name, it will infer the URL from it.
		// However, if the name has a '/', the provider assumes it's a URL and uses it as is.
		// This breaks routes cleanup since the routes are matched against the URL,
		// which would be incorrect in this case.
		// On the provider side, the "global/networks/default" format is the valid
		// one since it's used internally for firewall rules and and network interfaces,
		// so it has to be kept this way.
		// tl;dr: use "default" or a full network URL, not "global/networks/default"
		network = "default"
	}

	return &CloudConfig{
		Global: GlobalOpts{
			ProjectID:      projectID,
			LocalZone:      localZone,
			MultiZone:      multizone,
			Regional:       dc.Spec.GCP.Regional,
			NetworkName:    network,
			SubnetworkName: cluster.Spec.Cloud.GCP.Subnetwork,
			TokenURL:       "nil",
			NodeTags:       []string{tag},
		},
	}, nil
}

func (c *CloudConfig) String() (string, error) {
	out := ini.New()

	global := out.Section("global", "")
	c.Global.toINI(global)

	buf := &bytes.Buffer{}
	if err := out.Render(buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

// GlobalOpts contains the values of the global section of the cloud configuration.
type GlobalOpts struct {
	ProjectID      string
	LocalZone      string
	NetworkName    string
	SubnetworkName string
	TokenURL       string
	MultiZone      bool
	Regional       bool
	NodeTags       []string
}

func (o *GlobalOpts) toINI(section ini.Section) {
	section.AddStringKey("project-id", o.ProjectID)
	section.AddStringKey("local-zone", o.LocalZone)
	section.AddStringKey("network-name", o.NetworkName)
	section.AddStringKey("subnetwork-name", o.SubnetworkName)
	section.AddStringKey("token-url", o.TokenURL)
	section.AddBoolKey("multizone", o.MultiZone)
	section.AddBoolKey("regional", o.Regional)

	for _, tag := range o.NodeTags {
		section.AddStringKey("node-tags", tag)
	}
}
