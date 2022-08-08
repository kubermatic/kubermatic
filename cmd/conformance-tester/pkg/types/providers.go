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

package types

import (
	providerconfig "github.com/kubermatic/machine-controller/pkg/providerconfig/types"

	"k8s.io/apimachinery/pkg/util/sets"
)

var AllProviders = sets.NewString(
	string(providerconfig.CloudProviderAWS),
	string(providerconfig.CloudProviderAlibaba),
	string(providerconfig.CloudProviderAnexia),
	string(providerconfig.CloudProviderAzure),
	string(providerconfig.CloudProviderDigitalocean),
	string(providerconfig.CloudProviderGoogle),
	string(providerconfig.CloudProviderHetzner),
	string(providerconfig.CloudProviderKubeVirt),
	string(providerconfig.CloudProviderNutanix),
	string(providerconfig.CloudProviderOpenstack),
	string(providerconfig.CloudProviderPacket),
	string(providerconfig.CloudProviderVMwareCloudDirector),
	string(providerconfig.CloudProviderVsphere),
)
