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
	"k8c.io/kubermatic/v2/cmd/conformance-tester/pkg/types"
	"k8c.io/machine-controller/sdk/providerconfig"

	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type Generator struct {
	cloudProviders   sets.Set[string]
	operatingSystems sets.Set[string]
	versions         sets.Set[string]
	enableDualstack  bool
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
				scenario, err := providerScenario(opts, kubermaticv1.ProviderType(providerName), providerconfig.OperatingSystem(operatingSystem), *s, datacenter)
				if err != nil {
					return nil, err
				}

				scenarios = append(scenarios, scenario)
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
	base := BaseScenario{
		cloudProvider:    provider,
		operatingSystem:  os,
		clusterVersion:   version,
		datacenter:       datacenter,
		dualstackEnabled: opts.DualStackEnabled,
	}

	switch provider {
	case kubermaticv1.AlibabaCloudProvider:
		return &alibabaScenario{BaseScenario: base}, nil
	case kubermaticv1.AnexiaCloudProvider:
		return &anexiaScenario{BaseScenario: base}, nil
	case kubermaticv1.AWSCloudProvider:
		return &awsScenario{BaseScenario: base}, nil
	case kubermaticv1.AzureCloudProvider:
		return &azureScenario{BaseScenario: base}, nil
	case kubermaticv1.DigitaloceanCloudProvider:
		return &digitaloceanScenario{BaseScenario: base}, nil
	case kubermaticv1.GCPCloudProvider:
		return &googleScenario{BaseScenario: base}, nil
	case kubermaticv1.HetznerCloudProvider:
		return &hetznerScenario{BaseScenario: base}, nil
	case kubermaticv1.KubevirtCloudProvider:
		return &KubevirtScenario{BaseScenario: base}, nil
	case kubermaticv1.NutanixCloudProvider:
		return &nutanixScenario{BaseScenario: base}, nil
	case kubermaticv1.OpenstackCloudProvider:
		return &openStackScenario{BaseScenario: base}, nil
	case kubermaticv1.VMwareCloudDirectorCloudProvider:
		return &vmwareCloudDirectorScenario{BaseScenario: base}, nil
	case kubermaticv1.VSphereCloudProvider:
		scenario := &vSphereScenario{BaseScenario: base}
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

func shuffle(vals []Scenario) []Scenario {
	r := rand.New(rand.NewSource(time.Now().Unix()))
	ret := make([]Scenario, len(vals))
	n := len(vals)
	for i := range n {
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
