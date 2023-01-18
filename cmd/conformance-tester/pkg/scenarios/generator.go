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

	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/resources"
	"k8c.io/kubermatic/v2/pkg/semver"

	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Generator struct {
	cloudProviders    sets.Set[string]
	operatingSystems  sets.Set[string]
	versions          sets.Set[string]
	containerRuntimes sets.Set[string]
	enableOSM         bool
	enableDualstack   bool
}

func NewGenerator() *Generator {
	return &Generator{
		cloudProviders:    sets.New[string](),
		operatingSystems:  sets.New[string](),
		versions:          sets.New[string](),
		containerRuntimes: sets.New[string](),
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

func (g *Generator) WithContainerRuntimes(runtimes ...string) *Generator {
	for _, runtime := range runtimes {
		g.containerRuntimes.Insert(runtime)
	}
	return g
}

func (g *Generator) WithOSM(enable bool) *Generator {
	g.enableOSM = enable
	return g
}

func (g *Generator) WithDualstack(enable bool) *Generator {
	g.enableDualstack = enable
	return g
}

func (g *Generator) Scenarios(ctx context.Context, opts *types.Options, log *zap.SugaredLogger) ([]Scenario, error) {
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
				for _, cri := range sets.List(g.containerRuntimes) {
					scenario, err := providerScenario(opts, kubermaticv1.ProviderType(providerName), providerconfig.OperatingSystem(operatingSystem), *s, cri, datacenter)
					if err != nil {
						return nil, err
					}

					scenario.SetDualstackEnabled(g.enableDualstack)

					if isValidNewScenario(opts, log, scenario) {
						scenarios = append(scenarios, scenario)
					}
				}
			}
		}
	}

	return shuffle(scenarios), nil
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
	case kubermaticv1.PacketCloudProvider:
		datacenterName = secrets.Packet.KKPDatacenter
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
	containerRuntime string,
	datacenter *kubermaticv1.Datacenter,
) (Scenario, error) {
	base := baseScenario{
		cloudProvider:    provider,
		operatingSystem:  os,
		version:          version,
		containerRuntime: containerRuntime,
		datacenter:       datacenter,
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
		return &kubevirtScenario{baseScenario: base}, nil
	case kubermaticv1.NutanixCloudProvider:
		return &nutanixScenario{baseScenario: base}, nil
	case kubermaticv1.OpenstackCloudProvider:
		return &openStackScenario{baseScenario: base}, nil
	case kubermaticv1.PacketCloudProvider:
		return &packetScenario{baseScenario: base}, nil
	case kubermaticv1.VMwareCloudDirectorCloudProvider:
		return &vmwareCloudDirectorScenario{baseScenario: base}, nil
	case kubermaticv1.VSphereCloudProvider:
		scenario := &vSphereScenario{baseScenario: base}
		scenario.customFolder = opts.ScenarioOptions.Has("custom-folder")
		scenario.datastoreCluster = opts.ScenarioOptions.Has("datastore-cluster")

		return scenario, nil
	default:
		return nil, fmt.Errorf("cloud provider %q is not supported yet in conformance-tester", provider)
	}
}

func isValidNewScenario(opts *types.Options, log *zap.SugaredLogger, scenario Scenario) bool {
	// check if the CRI is enabled by the user
	if !opts.ContainerRuntimes.Has(scenario.ContainerRuntime()) {
		scenario.Log(log).Debugw("Skipping because this CRI is not enabled.")
		return false
	}

	// check if the OS is enabled by the user
	if !opts.Distributions.Has(string(scenario.OperatingSystem())) {
		scenario.Log(log).Debugw("Skipping because this OS is not enabled.")
		return false
	}

	// apply static filters
	clusterVersion := scenario.Version()
	dockerSupported := clusterVersion.LessThan(semver.NewSemverOrDie("1.24"))
	if !dockerSupported && scenario.ContainerRuntime() == resources.ContainerRuntimeDocker {
		scenario.Log(log).Infow("Skipping because CRI is not supported in this Kubernetes version.")
		return false
	}

	return scenario.IsValid(opts, log)
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
