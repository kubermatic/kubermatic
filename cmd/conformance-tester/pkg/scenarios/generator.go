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
	"math/rand"
	"time"

	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/sdk/v2/semver"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/config"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	"k8c.io/kubermatic/v2/pkg/test"
	"k8c.io/machine-controller/pkg/cloudprovider"
	"k8c.io/machine-controller/sdk/providerconfig"
	providerconfigtypes "k8c.io/machine-controller/sdk/providerconfig"
	"k8c.io/machine-controller/sdk/providerconfig/configvar"

	clusterv1alpha1 "k8c.io/machine-controller/sdk/apis/cluster/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/utils/ptr"
)

// ProviderSpecBuilder defines an interface for scenarios that can build a
// provider-specific machine spec.
type ProviderSpecBuilder interface {
	// ProviderSpec returns the raw provider spec for a machine.
	ProviderSpec() (*runtime.RawExtension, error)
}

// Flavorable defines an interface for scenarios that can be configured with a flavor.
type Flavorable interface {
	SetFlavor(*config.Flavor) error
}

type Generator struct {
	cloudProviders   sets.Set[string]
	operatingSystems sets.Set[string]
	versions         sets.Set[string]
	enableDualstack  bool
	config           *config.Config
}

func NewGenerator() *Generator {
	return &Generator{
		cloudProviders:   sets.New[string](),
		operatingSystems: sets.New[string](),
		versions:         sets.New[string](),
	}
}

func (g *Generator) WithCloudProviders(providerNames ...string) *Generator {
	for _, provider := range providerNames {
		g.cloudProviders.Insert(provider)
	}
	return g
}

func (g *Generator) WithOperatingSystems(operatingSystems ...string) *Generator {
	for _, os := range operatingSystems {
		g.operatingSystems.Insert(os)
	}
	return g
}

func (g *Generator) WithVersions(versions ...*semver.Semver) *Generator {
	for _, version := range versions {
		g.versions.Insert(version.String())
	}
	return g
}

func (g *Generator) WithDualstack(enable bool) *Generator {
	g.enableDualstack = enable
	return g
}

func (g *Generator) WithConfig(config *config.Config) *Generator {
	g.config = config
	return g
}

func (g *Generator) Scenarios(ctx context.Context, opts *types.Options, log *zap.SugaredLogger) ([]Scenario, error) {
	if g.config != nil {
		return g.scenariosFromConfig(ctx, opts, log)
	}

	return g.scenariosFromFlags(ctx, opts, log)
}

func (g *Generator) scenariosFromConfig(ctx context.Context, opts *types.Options, log *zap.SugaredLogger) ([]Scenario, error) {
	scenarios := []Scenario{}

	// Prefer versions from the scenarios file if present; otherwise use CLI-provided versions
	versionList := opts.Versions
	if g.config != nil && len(g.config.Versions) > 0 {
		versionList = []*semver.Semver{}
		for _, v := range g.config.Versions {
			latest := test.LatestKubernetesVersionForRelease(v, nil)
			if latest == nil {
				return nil, fmt.Errorf("no version found for release %q", v)
			}
			versionList = append(versionList, latest)
		}
	}

	for _, s := range versionList {
		for _, scenarioConfig := range g.config.Scenarios {
			provider := kubermaticv1.ProviderType(scenarioConfig.Provider)
			os := providerconfig.OperatingSystem(scenarioConfig.OperatingSystem)

			datacenter, err := g.datacenter(ctx, opts.SeedClusterClient, opts.Secrets, provider)
			if err != nil {
				return nil, fmt.Errorf("failed to determine target datacenter for provider %q: %w", provider, err)
			}

			inner, err := providerScenario(opts, provider, os, *s, datacenter)
			if err != nil {
				return nil, err
			}

			// Validate the base scenario once (flavor-specific validation happens implicitly when building MDs)
			if err := g.validate(inner, log); err != nil {
				log.Infow("Skipping invalid scenario", "provider", provider, "os", os, "version", s.String(), "reason", err)
				continue
			}

			wrapped := &multiFlavorScenario{inner: inner, flavors: scenarioConfig.Flavors}
			scenarios = append(scenarios, wrapped)
		}
	}

	return shuffle(scenarios), nil
}

func (g *Generator) scenariosFromFlags(ctx context.Context, opts *types.Options, log *zap.SugaredLogger) ([]Scenario, error) {
	scenarios := []Scenario{}

	for _, version := range sets.List(g.versions) {
		s, err := semver.NewSemver(version)
		if err != nil {
			return nil, fmt.Errorf("invalid version %q: %w", version, err)
		}

		for _, providerName := range sets.List(g.cloudProviders) {
			datacenter, err := g.datacenter(ctx, opts.SeedClusterClient, opts.Secrets, kubermaticv1.ProviderType(providerName))
			if err != nil {
				return nil, fmt.Errorf("failed to determine target datacenter for provider %q: %w", providerName, err)
			}

			for _, operatingSystem := range sets.List(g.operatingSystems) {
				scenario, err := providerScenario(opts, kubermaticv1.ProviderType(providerName), providerconfig.OperatingSystem(operatingSystem), *s, datacenter)
				if err != nil {
					return nil, err
				}

				if err := g.validate(scenario, log); err != nil {
					log.Infow("Skipping invalid scenario", "provider", providerName, "os", operatingSystem, "version", version, "reason", err)
					continue
				}

				scenarios = append(scenarios, scenario)
			}
		}
	}

	return shuffle(scenarios), nil
}

func (g *Generator) validate(scenario Scenario, log *zap.SugaredLogger) error {
	specBuilder, ok := scenario.(ProviderSpecBuilder)
	if !ok {
		log.Debugf("Scenario for provider %s does not implement ProviderSpecBuilder, skipping validation.", scenario.CloudProvider())
		return nil
	}

	if err := scenario.IsValid(); err != nil {
		return err
	}

	spec, err := specBuilder.ProviderSpec()
	if err != nil {
		return fmt.Errorf("failed to build provider spec: %w", err)
	}

	dummySpec := clusterv1alpha1.MachineSpec{
		ProviderSpec: clusterv1alpha1.ProviderSpec{
			Value: spec,
		},
	}

	cvr := configvar.NewResolver(context.Background(), nil)
	p, err := cloudprovider.ForProvider(providerconfigtypes.CloudProvider(scenario.CloudProvider()), cvr)
	if err != nil {
		return fmt.Errorf("failed to get provider %s: %w", scenario.CloudProvider(), err)
	}

	if err := p.Validate(context.Background(), zap.NewNop().Sugar(), dummySpec); err != nil {
		return fmt.Errorf("failed to validate kubevirt spec: %w", err)
	}

	return nil
}

func (g *Generator) datacenter(ctx context.Context, client ctrlruntimeclient.Client, secrets types.Secrets, provider kubermaticv1.ProviderType) (*kubermaticv1.Datacenter, error) {
	var datacenterName string

	switch provider {
	case kubermaticv1.AlibabaCloudProvider:
		datacenterName = secrets.Alibaba.KKPDatacenter
	case kubermaticv1.AnexiaCloudProvider:
		datacenterName = secrets.Anexia.KKPDatacenter
	case kubermaticv1.AWSCloudProvider:
		datacenterName = secrets.AWS.KKPDatacenter
	case kubermaticv1.AzureCloudProvider:
		datacenterName = secrets.Azure.KKPDatacenter
	case kubermaticv1.DigitaloceanCloudProvider:
		datacenterName = secrets.Digitalocean.KKPDatacenter
	case kubermaticv1.GCPCloudProvider:
		datacenterName = secrets.GCP.KKPDatacenter
	case kubermaticv1.HetznerCloudProvider:
		datacenterName = secrets.Hetzner.KKPDatacenter
	case kubermaticv1.KubevirtCloudProvider:
		datacenterName = secrets.Kubevirt.KKPDatacenter
	case kubermaticv1.NutanixCloudProvider:
		datacenterName = secrets.Nutanix.KKPDatacenter
	case kubermaticv1.OpenstackCloudProvider:
		datacenterName = secrets.OpenStack.KKPDatacenter
	case kubermaticv1.VMwareCloudDirectorCloudProvider:
		datacenterName = secrets.VMwareCloudDirector.KKPDatacenter
	case kubermaticv1.VSphereCloudProvider:
		datacenterName = secrets.VSphere.KKPDatacenter
	default:
		return nil, fmt.Errorf("cloud provider %q is not supported yet in conformance-tester", provider)
	}

	return getDatacenter(ctx, client, datacenterName)
}

func providerScenario(
	opts *types.Options,
	provider kubermaticv1.ProviderType,
	os providerconfig.OperatingSystem,
	version semver.Semver,
	datacenter *kubermaticv1.Datacenter,
) (Scenario, error) {
	base := baseScenario{
		cloudProvider:    provider,
		operatingSystem:  os,
		clusterVersion:   version,
		datacenter:       datacenter,
		dualstackEnabled: opts.DualStackEnabled,
	}

	switch provider {
	case kubermaticv1.AlibabaCloudProvider:
		return &alibabaScenario{baseScenario: base}, nil
	case kubermaticv1.AnexiaCloudProvider:
		return &anexiaScenario{baseScenario: base}, nil
	case kubermaticv1.AWSCloudProvider:
		return &awsScenario{baseScenario: base}, nil
	case kubermaticv1.AzureCloudProvider:
		return &azureScenario{baseScenario: base}, nil
	case kubermaticv1.DigitaloceanCloudProvider:
		return &digitaloceanScenario{baseScenario: base}, nil
	case kubermaticv1.GCPCloudProvider:
		return &googleScenario{baseScenario: base}, nil
	case kubermaticv1.HetznerCloudProvider:
		return &hetznerScenario{baseScenario: base}, nil
	case kubermaticv1.KubevirtCloudProvider:
		return &kubevirtScenario{baseScenario: base, secrets: opts.Secrets}, nil
	case kubermaticv1.NutanixCloudProvider:
		return &nutanixScenario{baseScenario: base}, nil
	case kubermaticv1.OpenstackCloudProvider:
		return &openStackScenario{baseScenario: base}, nil
	case kubermaticv1.VMwareCloudDirectorCloudProvider:
		return &vmwareCloudDirectorScenario{baseScenario: base}, nil
	case kubermaticv1.VSphereCloudProvider:
		scenario := &vSphereScenario{baseScenario: base}
		scenario.customFolder = opts.ScenarioOptions.Has("custom-folder")
		scenario.basePath = opts.ScenarioOptions.Has("basepath")
		scenario.datastoreCluster = opts.ScenarioOptions.Has("datastore-cluster")

		if scenario.customFolder && scenario.basePath {
			return nil, fmt.Errorf("cannot run mutually exclusive %q scenarios 'custom-folder' and 'basepath' together", provider)
		}

		return scenario, nil
	default:
		return nil, fmt.Errorf("cloud provider %q is not supported yet in conformance-tester", provider)
	}
}

// multiFlavorScenario wraps a single provider/OS/version scenario and expands it
// into multiple MachineDeployments, one per flavor.
type multiFlavorScenario struct {
	inner   Scenario
	flavors []config.Flavor
}

func (m *multiFlavorScenario) CloudProvider() kubermaticv1.ProviderType {
	return m.inner.CloudProvider()
}
func (m *multiFlavorScenario) OperatingSystem() providerconfig.OperatingSystem {
	return m.inner.OperatingSystem()
}
func (m *multiFlavorScenario) ClusterVersion() semver.Semver               { return m.inner.ClusterVersion() }
func (m *multiFlavorScenario) Datacenter() *kubermaticv1.Datacenter        { return m.inner.Datacenter() }
func (m *multiFlavorScenario) Name() string                                { return m.inner.Name() }
func (m *multiFlavorScenario) Log(l *zap.SugaredLogger) *zap.SugaredLogger { return m.inner.Log(l) }
func (m *multiFlavorScenario) NamedLog(l *zap.SugaredLogger) *zap.SugaredLogger {
	return m.inner.NamedLog(l)
}
func (m *multiFlavorScenario) IsValid() error { return m.inner.IsValid() }
func (m *multiFlavorScenario) Cluster(secrets types.Secrets) *kubermaticv1.ClusterSpec {
	spec := m.inner.Cluster(secrets)

	// For KubeVirt, include all storage classes used by flavors so all MDs can be created in the same cluster.
	if m.inner.CloudProvider() == kubermaticv1.KubevirtCloudProvider {
		seen := map[string]struct{}{}
		storageClasses := make([]kubermaticv1.KubeVirtInfraStorageClass, 0, len(m.flavors))

		getNestedValue := func(data map[string]interface{}, path ...string) (interface{}, bool) {
			var current interface{} = data
			for _, key := range path {
				m, ok := current.(map[string]interface{})
				if !ok {
					return nil, false
				}
				current, ok = m[key]
				if !ok {
					return nil, false
				}
			}
			return current, true
		}

		for _, fl := range m.flavors {
			providerConfig, ok := fl.Value.(map[string]interface{})
			if !ok {
				continue
			}
			if v, ok := getNestedValue(providerConfig, "virtualMachine", "template", "primaryDisk", "storageClassName"); ok {
				if name, ok := v.(string); ok && name != "" {
					if _, exists := seen[name]; !exists {
						seen[name] = struct{}{}
						storageClasses = append(storageClasses, kubermaticv1.KubeVirtInfraStorageClass{Name: name})
					}
				}
			}
		}

		if len(storageClasses) > 0 {
			// mark the first as default if none is already set
			storageClasses[0].IsDefaultClass = ptr.To(true)
			if spec.Cloud.Kubevirt == nil {
				spec.Cloud.Kubevirt = &kubermaticv1.KubevirtCloudSpec{}
			}
			spec.Cloud.Kubevirt.StorageClasses = storageClasses
		}
	}

	return spec
}

func (m *multiFlavorScenario) MachineDeployments(ctx context.Context, num int, secrets types.Secrets, cluster *kubermaticv1.Cluster, sshPubKeys []string) ([]clusterv1alpha1.MachineDeployment, error) {
	mds := []clusterv1alpha1.MachineDeployment{}

	if flavorable, ok := m.inner.(Flavorable); ok {
		for _, fl := range m.flavors {
			if err := flavorable.SetFlavor(&fl); err != nil {
				// skip invalid flavor
				continue
			}

			child, err := m.inner.MachineDeployments(ctx, num, secrets, cluster, sshPubKeys)
			if err != nil {
				// skip failures per flavor
				continue
			}
			mds = append(mds, child...)
		}
	} else {
		// Provider does not support flavors; just create its default MD(s) once
		child, err := m.inner.MachineDeployments(ctx, num, secrets, cluster, sshPubKeys)
		if err != nil {
			return nil, err
		}
		mds = append(mds, child...)
	}

	if len(mds) == 0 {
		return nil, fmt.Errorf("no valid machine deployments for scenario")
	}
	return mds, nil
}

func shuffle(vals []Scenario) []Scenario {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	ret := make([]Scenario, len(vals))
	n := len(vals)
	for i := 0; i < n; i++ {
		randIndex := r.Intn(len(vals))
		ret[i] = vals[randIndex]
		vals = append(vals[:randIndex], vals[randIndex+1:]...)
	}
	return ret
}

func getDatacenter(ctx context.Context, client ctrlruntimeclient.Client, datacenter string) (*kubermaticv1.Datacenter, error) {
	seeds := &kubermaticv1.SeedList{}
	if err := client.List(ctx, seeds); err != nil {
		return nil, fmt.Errorf("failed to list seeds: %w", err)
	}

	for _, seed := range seeds.Items {
		for name, dc := range seed.Spec.Datacenters {
			if name == datacenter {
				return &dc, nil
			}
		}
	}

	return nil, fmt.Errorf("no Seed contains datacenter %q", datacenter)
}
