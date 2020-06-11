package azure

import (
	"fmt"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
)

// For some reason, creting route table requires a subnetwork's full ID
// and not just the name and location. We could fetch the subnet by name and get the ID,
// but that's slow, so we assemble it ourselves.
func assembleSubnetID(cloud kubermaticv1.CloudSpec) string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/virtualNetworks/%s/subnets/%s",
		cloud.Azure.SubscriptionID, cloud.Azure.ResourceGroup, cloud.Azure.VNetName, cloud.Azure.SubnetName)
}
