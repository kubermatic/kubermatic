/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package azure

import (
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
)

// For some reason, creting route table requires a subnetwork's full ID
// and not just the name and location. We could fetch the subnet by name and get the ID,
// but that's slow, so we assemble it ourselves.
func assembleSubnetID(cloud kubermaticv1.CloudSpec) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s",
		cloud.Azure.SubscriptionID, cloud.Azure.ResourceGroup, cloud.Azure.VNetName, cloud.Azure.SubnetName)
}
