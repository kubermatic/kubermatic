/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package scenarios

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/config"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	"k8c.io/kubermatic/v2/pkg/machine/provider"
	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	"k8c.io/machine-controller/sdk/cloudprovider/kubevirt"
	providerconfig "k8c.io/machine-controller/sdk/providerconfig"

	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/utils/ptr"
	corev1 "k8s.io/api/core/v1"
	kubevirtcorev1 "kubevirt.io/api/core/v1"

	"gopkg.in/yaml.v3"
)

const (
	kubevirtImageHTTPServerSvc = "http://image-repo.kube-system.svc/images"
)

type kubevirtScenario struct {
	baseScenario
	vcpus            string
	memory           string
	diskSize         string
	storageClassName string
	image            string
	secrets          types.Secrets
	flavor           *kubevirtFlavorTemplate
}

func (s *kubevirtScenario) SetFlavor(flavor *config.Flavor) error {
	if flavor == nil || flavor.Value == nil {
		return fmt.Errorf("invalid flavor provided")
	}

	providerConfig, ok := flavor.Value.(map[string]interface{})
	if !ok {
		return fmt.Errorf("flavor.Value has invalid type %T", flavor.Value)
	}

	// Decode providerConfig into a typed struct for easier access
	var spec kubevirtFlavorTemplate
	if err := spec.FromMap(providerConfig); err != nil {
		return err
	}

	// Backward compatibility: allow top-level osImage
	if spec.VirtualMachine.Template.PrimaryDisk.OSImage == "" {
		if v, ok := providerConfig["osImage"].(string); ok && v != "" {
			spec.VirtualMachine.Template.PrimaryDisk.OSImage = v
		}
	}

	if err := spec.Validate(true); err != nil {
		return err
	}

	// cache on scenario and fill legacy fields
	s.flavor = &spec
	tpl := spec.VirtualMachine.Template
	// Prefer explicit vcpus over cpus if provided; otherwise keep prior behavior
	if tpl.VCPUs != "" {
		s.vcpus = tpl.VCPUs
	} else {
		s.vcpus = tpl.CPUs
	}
	s.memory = tpl.Memory
	s.diskSize = tpl.PrimaryDisk.Size
	s.storageClassName = tpl.PrimaryDisk.StorageClassName
	s.image = tpl.PrimaryDisk.OSImage

	return nil
}

func (s *kubevirtScenario) compatibleOperatingSystems() sets.Set[providerconfig.OperatingSystem] {
	return sets.New(
		providerconfig.OperatingSystemUbuntu,
		providerconfig.OperatingSystemRHEL,
		providerconfig.OperatingSystemFlatcar,
		providerconfig.OperatingSystemRockyLinux,
	)
}

func (s *kubevirtScenario) IsValid() error {
	if err := s.baseScenario.IsValid(); err != nil {
		return err
	}

	if compat := s.compatibleOperatingSystems(); !compat.Has(s.operatingSystem) {
		return fmt.Errorf("provider supports only %v", sets.List(compat))
	}

	return nil
}

func (s *kubevirtScenario) ProviderSpec() (*kubevirt.RawConfig, error) {
	// Prefer building from parsed flavor when available
	if s.flavor != nil {
		cfg, err := s.flavor.ToProviderConfig(s.secrets.Kubevirt.Kubeconfig)
		if err != nil {
			return nil, err
		}
		return cfg, nil
	}

	// build from individual fields
	config := provider.NewKubevirtConfig().
		WithKubeconfig(s.secrets.Kubevirt.Kubeconfig).
		WithVCPUs(s.vcpus).
		WithMemory(s.memory).
		WithPrimaryDiskOSImage(s.image).
		WithPrimaryDiskSize(s.diskSize).
		WithPrimaryDiskStorageClassName(s.storageClassName).
		Build()
	return &config, nil
}

func (s *kubevirtScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	return &kubermaticv1.ClusterSpec{
		Cloud: kubermaticv1.CloudSpec{
			DatacenterName: secrets.Kubevirt.KKPDatacenter,
			Kubevirt: &kubermaticv1.KubevirtCloudSpec{
				Kubeconfig: secrets.Kubevirt.Kubeconfig,
				StorageClasses: []kubermaticv1.KubeVirtInfraStorageClass{{
					Name:           s.storageClassName,
					IsDefaultClass: ptr.To(true),
				}},
			},
		},
		Version: s.clusterVersion,
	}
}

func (s *kubevirtScenario) MachineDeployments(_ context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster, sshPubKeys []string) ([]clusterv1alpha1.MachineDeployment, error) {
	spec, err := s.ProviderSpec()
	if err != nil {
		return nil, fmt.Errorf("failed to create provider spec: %w", err)
	}

	md, err := s.createMachineDeployment(cluster, num, spec, sshPubKeys, secrets)
	if err != nil {
		return nil, err
	}

	return []clusterv1alpha1.MachineDeployment{md}, nil
}

// KubevirtFlavorMutator injects the osImage under
// virtualMachine.template.primaryDisk.osImage for generated flavors.
var KubevirtFlavorMutator FlavorMutator = func(combo map[string]interface{}, image string) (map[string]interface{}, error) {
	if image == "" {
		// still normalize DNS config if present
	}

	vmNode, _ := combo["virtualMachine"].(map[string]interface{})
	if vmNode == nil {
		vmNode = make(map[string]interface{})
		combo["virtualMachine"] = vmNode
	}

	templateNode, _ := vmNode["template"].(map[string]interface{})
	if templateNode == nil {
		templateNode = make(map[string]interface{})
		vmNode["template"] = templateNode
	}

	diskNode, _ := templateNode["primaryDisk"].(map[string]interface{})
	if diskNode == nil {
		diskNode = make(map[string]interface{})
		templateNode["primaryDisk"] = diskNode
	}

	if image != "" {
		diskNode["osImage"] = image
	}

	// Normalize dnsConfig fields to always be slices of strings in the output
	if dc, ok := vmNode["dnsConfig"].(map[string]interface{}); ok {
		// helper to coerce value to []interface{} of strings
		toIfaceSlice := func(v interface{}) []interface{} {
			switch vv := v.(type) {
			case string:
				if vv == "" {
					return nil
				}
				return []interface{}{vv}
			case []string:
				res := make([]interface{}, 0, len(vv))
				for _, s := range vv {
					if s != "" {
						res = append(res, s)
					}
				}
				return res
			case []interface{}:
				// ensure only non-empty strings
				res := make([]interface{}, 0, len(vv))
				for _, it := range vv {
					if s, ok := it.(string); ok && s != "" {
						res = append(res, s)
					}
				}
				return res
			default:
				return nil
			}
		}

		// nameservers
		if v, ok := dc["nameservers"]; ok {
			if list := toIfaceSlice(v); len(list) > 0 {
				dc["nameservers"] = list
			} else {
				delete(dc, "nameservers")
			}
		}
		// search domains: accept "searchDomains" or "searches", prefer "searchDomains"
		if v, ok := dc["searchDomains"]; ok {
			if list := toIfaceSlice(v); len(list) > 0 {
				dc["searchDomains"] = list
			} else {
				delete(dc, "searchDomains")
			}
		} else if v, ok := dc["searches"]; ok {
			if list := toIfaceSlice(v); len(list) > 0 {
				dc["searchDomains"] = list
				delete(dc, "searches")
			} else {
				delete(dc, "searches")
			}
		}
	}

	return combo, nil
}

type kubevirtFlavorTemplate struct {
	Affinity struct {
		PodAffinityPreset     string `yaml:"podAffinityPreset" json:"podAffinityPreset"`
		PodAntiAffinityPreset string `yaml:"podAntiAffinityPreset" json:"podAntiAffinityPreset"`
		NodeAffinityPreset    struct {
			Type   string   `yaml:"type" json:"type"`
			Key    string   `yaml:"key" json:"key"`
			Values []string `yaml:"values" json:"values"`
		} `yaml:"nodeAffinityPreset" json:"nodeAffinityPreset"`
	} `yaml:"affinity" json:"affinity"`
	TopologySpreadConstraints []struct {
		MaxSkew           string `yaml:"maxSkew" json:"maxSkew"`
		TopologyKey       string `yaml:"topologyKey" json:"topologyKey"`
		WhenUnsatisfiable string `yaml:"whenUnsatisfiable" json:"whenUnsatisfiable"`
	} `yaml:"topologySpreadConstraints" json:"topologySpreadConstraints"`
	VirtualMachine struct {
		// Optional instancetype/preference
		Instancetype struct {
			Name                         string `yaml:"name" json:"name"`
			Kind                         string `yaml:"kind" json:"kind"`
			RevisionName                 string `yaml:"revisionName" json:"revisionName"`
			InferFromVolume              string `yaml:"inferFromVolume" json:"inferFromVolume"`
			InferFromVolumeFailurePolicy string `yaml:"inferFromVolumeFailurePolicy" json:"inferFromVolumeFailurePolicy"`
		} `yaml:"instancetype" json:"instancetype"`
		Preference struct {
			Name                         string `yaml:"name" json:"name"`
			Kind                         string `yaml:"kind" json:"kind"`
			RevisionName                 string `yaml:"revisionName" json:"revisionName"`
			InferFromVolume              string `yaml:"inferFromVolume" json:"inferFromVolume"`
			InferFromVolumeFailurePolicy string `yaml:"inferFromVolumeFailurePolicy" json:"inferFromVolumeFailurePolicy"`
		} `yaml:"preference" json:"preference"`
		Template struct {
			// Either CPUs or VCPUs can be used
			CPUs        string `yaml:"cpus" json:"cpus"`
			VCPUs       string `yaml:"vcpus" json:"vcpus"`
			Memory      string `yaml:"memory" json:"memory"`
			PrimaryDisk struct {
				Size                  string   `yaml:"size" json:"size"`
				StorageClassName      string   `yaml:"storageClassName" json:"storageClassName"`
				StorageAccessType     string   `yaml:"storageAccessType" json:"storageAccessType"`
				OSImage               string   `yaml:"osImage" json:"osImage"`
				Source                string   `yaml:"source" json:"source"`
				PullMethod            string   `yaml:"pullMethod" json:"pullMethod"`
				DataVolumeSecretRef   string   `yaml:"dataVolumeSecretRef" json:"dataVolumeSecretRef"`
				ExtraHeaders          []string `yaml:"extraHeaders" json:"extraHeaders"`
				ExtraHeadersSecretRef string   `yaml:"extraHeadersSecretRef" json:"extraHeadersSecretRef"`
				StorageTarget         string   `yaml:"storageTarget" json:"storageTarget"`
			} `yaml:"primaryDisk" json:"primaryDisk"`
			SecondaryDisks []struct {
				Size              string `yaml:"size" json:"size"`
				StorageClassName  string `yaml:"storageClassName" json:"storageClassName"`
				StorageAccessType string `yaml:"storageAccessType" json:"storageAccessType"`
			} `yaml:"secondaryDisks" json:"secondaryDisks"`
		} `yaml:"template" json:"template"`
		DNSPolicy               string `yaml:"dnsPolicy" json:"dnsPolicy"`
		DNSConfig               struct {
			Nameservers []string `yaml:"nameservers" json:"nameservers"`
			Searches    []string `yaml:"searches" json:"searches"`
			Options     []struct {
				Name  string `yaml:"name" json:"name"`
				Value string `yaml:"value" json:"value"`
			} `yaml:"options" json:"options"`
		} `yaml:"dnsConfig" json:"dnsConfig"`
		EnableNetworkMultiQueue bool   `yaml:"enableNetworkMultiQueue" json:"enableNetworkMultiQueue"`
		EvictionStrategy        string `yaml:"evictionStrategy" json:"evictionStrategy"`
		Location                struct {
			Region string `yaml:"region" json:"region"`
			Zone   string `yaml:"zone" json:"zone"`
		} `yaml:"location" json:"location"`
		ProviderNetwork struct {
			Name string `yaml:"name" json:"name"`
			VPC  struct {
				Name   string `yaml:"name" json:"name"`
				Subnet struct {
					Name string `yaml:"name" json:"name"`
				} `yaml:"subnet" json:"subnet"`
			} `yaml:"vpc" json:"vpc"`
		} `yaml:"providerNetwork" json:"providerNetwork"`
		// Deprecated but still supported
		Flavor struct {
			Name    string `yaml:"name" json:"name"`
			Profile string `yaml:"profile" json:"profile"`
		} `yaml:"flavor" json:"flavor"`
	} `yaml:"virtualMachine" json:"virtualMachine"`
}

// FromMap decodes a generic map into the typed flavor template.
func (t *kubevirtFlavorTemplate) FromMap(m map[string]interface{}) error {
	// Normalize dnsConfig: accept scalar or list for nameservers/search domains, and accept
	// both "searchDomains" (generator output) and "searches" (internal field name).
	if vm, ok := m["virtualMachine"].(map[string]interface{}); ok {
		if dc, ok := vm["dnsConfig"].(map[string]interface{}); ok {
			// nameservers: string -> []string, []interface{} -> []string
			if ns, ok := dc["nameservers"]; ok {
				switch v := ns.(type) {
				case string:
					if v != "" {
						dc["nameservers"] = []string{v}
					} else {
						delete(dc, "nameservers")
					}
				case []interface{}:
					arr := make([]string, 0, len(v))
					for _, it := range v {
						if s, ok := it.(string); ok && s != "" {
							arr = append(arr, s)
						}
					}
					dc["nameservers"] = arr
				}
			}
			// search domains can come as "searchDomains" or "searches"; normalize to "searches" as []string
			if sd, ok := dc["searchDomains"]; ok {
				switch v := sd.(type) {
				case string:
					if v != "" {
						dc["searches"] = []string{v}
					}
				case []interface{}:
					arr := make([]string, 0, len(v))
					for _, it := range v {
						if s, ok := it.(string); ok && s != "" {
							arr = append(arr, s)
						}
					}
					dc["searches"] = arr
				}
				delete(dc, "searchDomains")
			} else if sd, ok := dc["searches"]; ok {
				switch v := sd.(type) {
				case string:
					if v != "" {
						dc["searches"] = []string{v}
					} else {
						delete(dc, "searches")
					}
				case []interface{}:
					arr := make([]string, 0, len(v))
					for _, it := range v {
						if s, ok := it.(string); ok && s != "" {
							arr = append(arr, s)
						}
					}
					dc["searches"] = arr
				}
			}
		}
	}

	raw, err := yaml.Marshal(m)
	if err != nil {
		return fmt.Errorf("failed to marshal flavor value: %w", err)
	}
	if err := yaml.Unmarshal(raw, t); err != nil {
		return fmt.Errorf("failed to unmarshal flavor into struct: %w", err)
	}
	return nil
}

// Validate validates required fields. If requireImage is true, OSImage must be set.
func (t *kubevirtFlavorTemplate) Validate(requireImage bool) error {
	tpl := t.VirtualMachine.Template
	// consider instancetype present
	hasIT := t.VirtualMachine.Instancetype.Name != "" || t.VirtualMachine.Instancetype.Kind != "" || t.VirtualMachine.Instancetype.RevisionName != "" || t.VirtualMachine.Instancetype.InferFromVolume != "" || t.VirtualMachine.Instancetype.InferFromVolumeFailurePolicy != ""
	// Allow either CPUs or VCPUs or Instancetype
	if tpl.CPUs == "" && tpl.VCPUs == "" && !hasIT {
		return fmt.Errorf("failed to find 'cpus' in flavor")
	}
	// Memory may come from Instancetype
	if tpl.Memory == "" && !hasIT {
		return fmt.Errorf("failed to find 'memory' in flavor")
	}
	if tpl.PrimaryDisk.Size == "" {
		return fmt.Errorf("failed to find 'diskSize' in flavor")
	}
	if tpl.PrimaryDisk.StorageClassName == "" {
		return fmt.Errorf("failed to find 'storageClassName' in flavor")
	}
	if requireImage && tpl.PrimaryDisk.OSImage == "" {
		return fmt.Errorf("failed to find 'osImage' in flavor")
	}
	return nil
}

// ToMap encodes the flavor into a generic nested map suitable for scenarios.yaml.
func (t *kubevirtFlavorTemplate) ToMap() map[string]interface{} {
	m := map[string]interface{}{}
	vm := map[string]interface{}{}
	tpl := map[string]interface{}{}
	disk := map[string]interface{}{}

	if v := t.VirtualMachine.Template.CPUs; v != "" {
		tpl["cpus"] = v
	}
	if v := t.VirtualMachine.Template.VCPUs; v != "" {
		tpl["vcpus"] = v
	}
	if v := t.VirtualMachine.Template.Memory; v != "" {
		tpl["memory"] = v
	}
	if v := t.VirtualMachine.Template.PrimaryDisk.Size; v != "" {
		disk["size"] = v
	}
	if v := t.VirtualMachine.Template.PrimaryDisk.StorageClassName; v != "" {
		disk["storageClassName"] = v
	}
	if v := t.VirtualMachine.Template.PrimaryDisk.StorageAccessType; v != "" {
		disk["storageAccessType"] = v
	}
	if v := t.VirtualMachine.Template.PrimaryDisk.OSImage; v != "" {
		disk["osImage"] = v
	}
	if v := t.VirtualMachine.Template.PrimaryDisk.Source; v != "" {
		disk["source"] = v
	}
	if v := t.VirtualMachine.Template.PrimaryDisk.PullMethod; v != "" {
		disk["pullMethod"] = v
	}
	if v := t.VirtualMachine.Template.PrimaryDisk.DataVolumeSecretRef; v != "" {
		disk["dataVolumeSecretRef"] = v
	}
	if arr := t.VirtualMachine.Template.PrimaryDisk.ExtraHeaders; len(arr) > 0 {
		disk["extraHeaders"] = arr
	}
	if v := t.VirtualMachine.Template.PrimaryDisk.ExtraHeadersSecretRef; v != "" {
		disk["extraHeadersSecretRef"] = v
	}
	if v := t.VirtualMachine.Template.PrimaryDisk.StorageTarget; v != "" {
		disk["storageTarget"] = v
	}
	if len(disk) > 0 {
		tpl["primaryDisk"] = disk
	}
	if sds := t.VirtualMachine.Template.SecondaryDisks; len(sds) > 0 {
		var out []map[string]interface{}
		for _, d := range sds {
			m := map[string]interface{}{}
			if v := d.Size; v != "" {
				m["size"] = v
			}
			if v := d.StorageClassName; v != "" {
				m["storageClassName"] = v
			}
			if v := d.StorageAccessType; v != "" {
				m["storageAccessType"] = v
			}
			if len(m) > 0 {
				out = append(out, m)
			}
		}
		if len(out) > 0 {
			tpl["secondaryDisks"] = out
		}
	}
	if len(tpl) > 0 {
		vm["template"] = tpl
	}
	if v := t.VirtualMachine.DNSPolicy; v != "" {
		vm["dnsPolicy"] = v
	}
	// DNSConfig
	if dc := t.VirtualMachine.DNSConfig; len(dc.Nameservers) > 0 || len(dc.Searches) > 0 || len(dc.Options) > 0 {
		dco := map[string]interface{}{}
		if len(dc.Nameservers) > 0 {
			dco["nameservers"] = append([]string(nil), dc.Nameservers...)
		}
		if len(dc.Searches) > 0 {
			// Emit using "searchDomains" for compatibility with generated names and existing files
			dco["searchDomains"] = append([]string(nil), dc.Searches...)
		}
		if len(dc.Options) > 0 {
			opts := make([]map[string]interface{}, 0, len(dc.Options))
			for _, o := range dc.Options {
				om := map[string]interface{}{"name": o.Name}
				if o.Value != "" {
					om["value"] = o.Value
				}
				opts = append(opts, om)
			}
			dco["options"] = opts
		}
		vm["dnsConfig"] = dco
	}
	if t.VirtualMachine.EnableNetworkMultiQueue {
		vm["enableNetworkMultiQueue"] = t.VirtualMachine.EnableNetworkMultiQueue
	}
	if v := t.VirtualMachine.EvictionStrategy; v != "" {
		vm["evictionStrategy"] = v
	}
	if loc := t.VirtualMachine.Location; loc.Region != "" || loc.Zone != "" {
		vm["location"] = map[string]interface{}{
			"region": loc.Region,
			"zone":   loc.Zone,
		}
	}
	if pn := t.VirtualMachine.ProviderNetwork; pn.Name != "" || pn.VPC.Name != "" || pn.VPC.Subnet.Name != "" {
		vm["providerNetwork"] = map[string]interface{}{
			"name": pn.Name,
			"vpc": map[string]interface{}{
				"name": pn.VPC.Name,
				"subnet": map[string]interface{}{
					"name": pn.VPC.Subnet.Name,
				},
			},
		}
	}
	// Instancetype
	if it := t.VirtualMachine.Instancetype; it.Name != "" || it.Kind != "" || it.RevisionName != "" || it.InferFromVolume != "" || it.InferFromVolumeFailurePolicy != "" {
		itMap := map[string]interface{}{}
		if it.Name != "" {
			itMap["name"] = it.Name
		}
		if it.Kind != "" {
			itMap["kind"] = it.Kind
		}
		if it.RevisionName != "" {
			itMap["revisionName"] = it.RevisionName
		}
		if it.InferFromVolume != "" {
			itMap["inferFromVolume"] = it.InferFromVolume
		}
		if it.InferFromVolumeFailurePolicy != "" {
			itMap["inferFromVolumeFailurePolicy"] = it.InferFromVolumeFailurePolicy
		}
		vm["instancetype"] = itMap
	}
	// Preference
	if pr := t.VirtualMachine.Preference; pr.Name != "" || pr.Kind != "" || pr.RevisionName != "" || pr.InferFromVolume != "" || pr.InferFromVolumeFailurePolicy != "" {
		prMap := map[string]interface{}{}
		if pr.Name != "" {
			prMap["name"] = pr.Name
		}
		if pr.Kind != "" {
			prMap["kind"] = pr.Kind
		}
		if pr.RevisionName != "" {
			prMap["revisionName"] = pr.RevisionName
		}
		if pr.InferFromVolume != "" {
			prMap["inferFromVolume"] = pr.InferFromVolume
		}
		if pr.InferFromVolumeFailurePolicy != "" {
			prMap["inferFromVolumeFailurePolicy"] = pr.InferFromVolumeFailurePolicy
		}
		vm["preference"] = prMap
	}
	// Deprecated Flavor
	if fl := t.VirtualMachine.Flavor; fl.Name != "" || fl.Profile != "" {
		vm["flavor"] = map[string]interface{}{
			"name":    fl.Name,
			"profile": fl.Profile,
		}
	}

	if len(vm) > 0 {
		m["virtualMachine"] = vm
	}
	// Affinity
	if a := t.Affinity; a.PodAffinityPreset != "" || a.PodAntiAffinityPreset != "" || a.NodeAffinityPreset.Type != "" || a.NodeAffinityPreset.Key != "" || len(a.NodeAffinityPreset.Values) > 0 {
		af := map[string]interface{}{}
		if v := a.PodAffinityPreset; v != "" {
			af["podAffinityPreset"] = v
		}
		if v := a.PodAntiAffinityPreset; v != "" {
			af["podAntiAffinityPreset"] = v
		}
		nap := map[string]interface{}{}
		if v := a.NodeAffinityPreset.Type; v != "" {
			nap["type"] = v
		}
		if v := a.NodeAffinityPreset.Key; v != "" {
			nap["key"] = v
		}
		if vs := a.NodeAffinityPreset.Values; len(vs) > 0 {
			nap["values"] = vs
		}
		if len(nap) > 0 {
			af["nodeAffinityPreset"] = nap
		}
		if len(af) > 0 {
			m["affinity"] = af
		}
	}
	// TopologySpreadConstraints
	if tscs := t.TopologySpreadConstraints; len(tscs) > 0 {
		arr := make([]map[string]interface{}, 0, len(tscs))
		for _, c := range tscs {
			cm := map[string]interface{}{}
			if v := c.MaxSkew; v != "" {
				cm["maxSkew"] = v
			}
			if v := c.TopologyKey; v != "" {
				cm["topologyKey"] = v
			}
			if v := c.WhenUnsatisfiable; v != "" {
				cm["whenUnsatisfiable"] = v
			}
			if len(cm) > 0 {
				arr = append(arr, cm)
			}
		}
		if len(arr) > 0 {
			m["topologySpreadConstraints"] = arr
		}
	}
	return m
}

// ToProviderConfig builds a kubevirt.RawConfig from the flavor and kubeconfig.
func (t *kubevirtFlavorTemplate) ToProviderConfig(kubeconfig string) (*kubevirt.RawConfig, error) {
	if err := t.Validate(true); err != nil {
		return nil, err
	}
	cfgBuilder := provider.NewKubevirtConfig().
		WithKubeconfig(kubeconfig).
		WithMemory(t.VirtualMachine.Template.Memory).
		WithPrimaryDiskOSImage(t.VirtualMachine.Template.PrimaryDisk.OSImage).
		WithPrimaryDiskSize(t.VirtualMachine.Template.PrimaryDisk.Size).
		WithPrimaryDiskStorageClassName(t.VirtualMachine.Template.PrimaryDisk.StorageClassName)
	// Only set VCPUs if provided
	if t.VirtualMachine.Template.VCPUs != "" {
		cfgBuilder = cfgBuilder.WithVCPUs(t.VirtualMachine.Template.VCPUs)
	}
	cfg := cfgBuilder.Build()

	// Set optional fields directly on the RawConfig
	cfg.VirtualMachine.Template.CPUs = providerconfig.ConfigVarString{Value: t.VirtualMachine.Template.CPUs}

	// Instancetype
	if it := t.VirtualMachine.Instancetype; it.Name != "" || it.Kind != "" || it.RevisionName != "" || it.InferFromVolume != "" || it.InferFromVolumeFailurePolicy != "" {
		matcher := &kubevirtcorev1.InstancetypeMatcher{
			Name:           it.Name,
			Kind:           it.Kind,
			RevisionName:   it.RevisionName,
			InferFromVolume: it.InferFromVolume,
		}
		if it.InferFromVolumeFailurePolicy != "" {
			p := kubevirtcorev1.InferFromVolumeFailurePolicy(it.InferFromVolumeFailurePolicy)
			matcher.InferFromVolumeFailurePolicy = &p
		}
		cfg.VirtualMachine.Instancetype = matcher
	}
	// Preference
	if pr := t.VirtualMachine.Preference; pr.Name != "" || pr.Kind != "" || pr.RevisionName != "" || pr.InferFromVolume != "" || pr.InferFromVolumeFailurePolicy != "" {
		matcher := &kubevirtcorev1.PreferenceMatcher{
			Name:           pr.Name,
			Kind:           pr.Kind,
			RevisionName:   pr.RevisionName,
			InferFromVolume: pr.InferFromVolume,
		}
		if pr.InferFromVolumeFailurePolicy != "" {
			p := kubevirtcorev1.InferFromVolumeFailurePolicy(pr.InferFromVolumeFailurePolicy)
			matcher.InferFromVolumeFailurePolicy = &p
		}
		cfg.VirtualMachine.Preference = matcher
	}

	// PrimaryDisk advanced options
	cfg.VirtualMachine.Template.PrimaryDisk.StorageAccessType = providerconfig.ConfigVarString{Value: t.VirtualMachine.Template.PrimaryDisk.StorageAccessType}
	cfg.VirtualMachine.Template.PrimaryDisk.Source = providerconfig.ConfigVarString{Value: t.VirtualMachine.Template.PrimaryDisk.Source}
	cfg.VirtualMachine.Template.PrimaryDisk.PullMethod = providerconfig.ConfigVarString{Value: t.VirtualMachine.Template.PrimaryDisk.PullMethod}
	cfg.VirtualMachine.Template.PrimaryDisk.DataVolumeSecretRef = providerconfig.ConfigVarString{Value: t.VirtualMachine.Template.PrimaryDisk.DataVolumeSecretRef}
	cfg.VirtualMachine.Template.PrimaryDisk.ExtraHeaders = append([]string(nil), t.VirtualMachine.Template.PrimaryDisk.ExtraHeaders...)
	cfg.VirtualMachine.Template.PrimaryDisk.ExtraHeadersSecretRef = providerconfig.ConfigVarString{Value: t.VirtualMachine.Template.PrimaryDisk.ExtraHeadersSecretRef}
	cfg.VirtualMachine.Template.PrimaryDisk.StorageTarget = providerconfig.ConfigVarString{Value: t.VirtualMachine.Template.PrimaryDisk.StorageTarget}

	// SecondaryDisks
	if len(t.VirtualMachine.Template.SecondaryDisks) > 0 {
		cfg.VirtualMachine.Template.SecondaryDisks = nil
		for _, d := range t.VirtualMachine.Template.SecondaryDisks {
			cfg.VirtualMachine.Template.SecondaryDisks = append(cfg.VirtualMachine.Template.SecondaryDisks, kubevirt.SecondaryDisks{
				Disk: kubevirt.Disk{
					Size:              providerconfig.ConfigVarString{Value: d.Size},
					StorageClassName:  providerconfig.ConfigVarString{Value: d.StorageClassName},
					StorageAccessType: providerconfig.ConfigVarString{Value: d.StorageAccessType},
				},
			})
		}
	}

	// VM-level options
	if v := t.VirtualMachine.DNSPolicy; v != "" {
		cfg.VirtualMachine.DNSPolicy = providerconfig.ConfigVarString{Value: v}
	}
	// DNSConfig
	if dc := t.VirtualMachine.DNSConfig; len(dc.Nameservers) > 0 || len(dc.Searches) > 0 || len(dc.Options) > 0 {
		opts := make([]corev1.PodDNSConfigOption, 0, len(dc.Options))
		for _, o := range dc.Options {
			ov := o // copy
			var valPtr *string
			if ov.Value != "" {
				val := ov.Value
				valPtr = &val
			}
			opts = append(opts, corev1.PodDNSConfigOption{Name: ov.Name, Value: valPtr})
		}
		cfg.VirtualMachine.DNSConfig = &corev1.PodDNSConfig{
			Nameservers: append([]string(nil), dc.Nameservers...),
			Searches:    append([]string(nil), dc.Searches...),
			Options:     opts,
		}
	}
	cfg.VirtualMachine.EnableNetworkMultiQueue = providerconfig.ConfigVarBool{Value: ptr.To(t.VirtualMachine.EnableNetworkMultiQueue)}
	if v := t.VirtualMachine.EvictionStrategy; v != "" {
		cfg.VirtualMachine.EvictionStrategy = v
	}
	if loc := t.VirtualMachine.Location; loc.Region != "" || loc.Zone != "" {
		cfg.VirtualMachine.Location = &kubevirt.Location{Region: loc.Region, Zone: loc.Zone}
	}
	if pn := t.VirtualMachine.ProviderNetwork; pn.Name != "" || pn.VPC.Name != "" || pn.VPC.Subnet.Name != "" {
		cfg.VirtualMachine.ProviderNetwork = &kubevirt.ProviderNetwork{
			Name: pn.Name,
			VPC: kubevirt.VPC{
				Name: pn.VPC.Name,
				Subnet: &kubevirt.Subnet{Name: pn.VPC.Subnet.Name},
			},
		}
	}
	// Deprecated Flavor
	if fl := t.VirtualMachine.Flavor; fl.Name != "" || fl.Profile != "" {
		cfg.VirtualMachine.Flavor.Name = providerconfig.ConfigVarString{Value: fl.Name}
		cfg.VirtualMachine.Flavor.Profile = providerconfig.ConfigVarString{Value: fl.Profile}
	}

	// Affinity
	if a := t.Affinity; a.PodAffinityPreset != "" || a.PodAntiAffinityPreset != "" || a.NodeAffinityPreset.Type != "" || a.NodeAffinityPreset.Key != "" || len(a.NodeAffinityPreset.Values) > 0 {
		cfg.Affinity.PodAffinityPreset = providerconfig.ConfigVarString{Value: a.PodAffinityPreset}
		cfg.Affinity.PodAntiAffinityPreset = providerconfig.ConfigVarString{Value: a.PodAntiAffinityPreset}
		cfg.Affinity.NodeAffinityPreset = kubevirt.NodeAffinityPreset{
			Type: providerconfig.ConfigVarString{Value: a.NodeAffinityPreset.Type},
			Key:  providerconfig.ConfigVarString{Value: a.NodeAffinityPreset.Key},
		}
		if len(a.NodeAffinityPreset.Values) > 0 {
			cfg.Affinity.NodeAffinityPreset.Values = make([]providerconfig.ConfigVarString, 0, len(a.NodeAffinityPreset.Values))
			for _, v := range a.NodeAffinityPreset.Values {
				cfg.Affinity.NodeAffinityPreset.Values = append(cfg.Affinity.NodeAffinityPreset.Values, providerconfig.ConfigVarString{Value: v})
			}
		}
	}

	// TopologySpreadConstraints
	if tscs := t.TopologySpreadConstraints; len(tscs) > 0 {
		cfg.TopologySpreadConstraints = make([]kubevirt.TopologySpreadConstraint, 0, len(tscs))
		for _, c := range tscs {
			cfg.TopologySpreadConstraints = append(cfg.TopologySpreadConstraints, kubevirt.TopologySpreadConstraint{
				MaxSkew:          providerconfig.ConfigVarString{Value: c.MaxSkew},
				TopologyKey:      providerconfig.ConfigVarString{Value: c.TopologyKey},
				WhenUnsatisfiable: providerconfig.ConfigVarString{Value: c.WhenUnsatisfiable},
			})
		}
	}

	return &cfg, nil
}
