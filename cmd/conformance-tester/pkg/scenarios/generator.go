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

	kubermaticv1 "k8c.io/api/v2/pkg/apis/kubermatic/v1"
	"k8c.io/api/v2/pkg/semver"
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"

	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Generator struct {
	cloudProviders    sets.Set[kubermaticv1.CloudProvider]
	operatingSystems  sets.Set[kubermaticv1.OperatingSystem]
	versions          sets.Set[string]
	containerRuntimes sets.Set[string]
	enableOSM         bool
	enableDualstack   bool
}

func NewGenerator() *Generator {
	return &Generator{
		cloudProviders:    sets.New[kubermaticv1.CloudProvider](),
		operatingSystems:  sets.New[kubermaticv1.OperatingSystem](),
		versions:          sets.New[string](),
		containerRuntimes: sets.New[string](),
	}
}

func (g *Generator) WithCloudProviders(providerNames ...kubermaticv1.CloudProvider) *Generator {
	for _, provider := range providerNames {
		g.cloudProviders.Insert(provider)
	}
	return g
}

func (g *Generator) WithOperatingSystems(operatingSystems ...kubermaticv1.OperatingSystem) *Generator {
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
			datacenter, err := g.datacenter(ctx, opts.SeedClusterClient, opts.Secrets, providerName)
			if err != nil {
				return nil, fmt.Errorf("failed to determine target datacenter for provider %q: %w", providerName, err)
			}

			for _, operatingSystem := range sets.List(g.operatingSystems) {
				for _, cri := range sets.List(g.containerRuntimes) {
					scenario, err := providerScenario(opts, providerName, operatingSystem, *s, cri, datacenter)
					if err != nil {
						return nil, err
					}

					scenarios = append(scenarios, scenario)
				}
			}
		}
	}

	return shuffle(scenarios), nil
}

func (g *Generator) datacenter(ctx context.Context, client ctrlruntimeclient.Client, secrets types.Secrets, provider kubermaticv1.CloudProvider) (*kubermaticv1.Datacenter, error) {
	var datacenterName string

	switch provider {
	case kubermaticv1.CloudProviderAlibaba:
		datacenterName = secrets.Alibaba.KKPDatacenter
	case kubermaticv1.CloudProviderAnexia:
		datacenterName = secrets.Anexia.KKPDatacenter
	case kubermaticv1.CloudProviderAWS:
		datacenterName = secrets.AWS.KKPDatacenter
	case kubermaticv1.CloudProviderAzure:
		datacenterName = secrets.Azure.KKPDatacenter
	case kubermaticv1.CloudProviderDigitalocean:
		datacenterName = secrets.Digitalocean.KKPDatacenter
	case kubermaticv1.CloudProviderGCP:
		datacenterName = secrets.GCP.KKPDatacenter
	case kubermaticv1.CloudProviderHetzner:
		datacenterName = secrets.Hetzner.KKPDatacenter
	case kubermaticv1.CloudProviderKubeVirt:
		datacenterName = secrets.Kubevirt.KKPDatacenter
	case kubermaticv1.CloudProviderNutanix:
		datacenterName = secrets.Nutanix.KKPDatacenter
	case kubermaticv1.CloudProviderOpenStack:
		datacenterName = secrets.OpenStack.KKPDatacenter
	case kubermaticv1.CloudProviderPacket:
		datacenterName = secrets.Packet.KKPDatacenter
	case kubermaticv1.CloudProviderVMwareCloudDirector:
		datacenterName = secrets.VMwareCloudDirector.KKPDatacenter
	case kubermaticv1.CloudProviderVSphere:
		datacenterName = secrets.VSphere.KKPDatacenter
	default:
		return nil, fmt.Errorf("cloud provider %q is not supported yet in conformance-tester", provider)
	}

	return getDatacenter(ctx, client, datacenterName)
}

func providerScenario(
	opts *types.Options,
	provider kubermaticv1.CloudProvider,
	os kubermaticv1.OperatingSystem,
	version semver.Semver,
	containerRuntime string,
	datacenter *kubermaticv1.Datacenter,
) (Scenario, error) {
	base := baseScenario{
		cloudProvider:    provider,
		operatingSystem:  os,
		clusterVersion:   version,
		containerRuntime: containerRuntime,
		datacenter:       datacenter,
		dualstackEnabled: opts.DualStackEnabled,
	}

	switch provider {
	case kubermaticv1.CloudProviderAlibaba:
		return &alibabaScenario{baseScenario: base}, nil
	case kubermaticv1.CloudProviderAnexia:
		return &anexiaScenario{baseScenario: base}, nil
	case kubermaticv1.CloudProviderAWS:
		return &awsScenario{baseScenario: base}, nil
	case kubermaticv1.CloudProviderAzure:
		return &azureScenario{baseScenario: base}, nil
	case kubermaticv1.CloudProviderDigitalocean:
		return &digitaloceanScenario{baseScenario: base}, nil
	case kubermaticv1.CloudProviderGCP:
		return &googleScenario{baseScenario: base}, nil
	case kubermaticv1.CloudProviderHetzner:
		return &hetznerScenario{baseScenario: base}, nil
	case kubermaticv1.CloudProviderKubeVirt:
		return &kubeVirtScenario{baseScenario: base}, nil
	case kubermaticv1.CloudProviderNutanix:
		return &nutanixScenario{baseScenario: base}, nil
	case kubermaticv1.CloudProviderOpenStack:
		return &openStackScenario{baseScenario: base}, nil
	case kubermaticv1.CloudProviderPacket:
		return &packetScenario{baseScenario: base}, nil
	case kubermaticv1.CloudProviderVMwareCloudDirector:
		return &vmwareCloudDirectorScenario{baseScenario: base}, nil
	case kubermaticv1.CloudProviderVSphere:
		scenario := &vSphereScenario{baseScenario: base}
		scenario.customFolder = opts.ScenarioOptions.Has("custom-folder")
		scenario.datastoreCluster = opts.ScenarioOptions.Has("datastore-cluster")

		return scenario, nil
	default:
		return nil, fmt.Errorf("cloud provider %q is not supported yet in conformance-tester", provider)
	}
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
