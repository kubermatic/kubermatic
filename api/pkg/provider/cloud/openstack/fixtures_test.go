package openstack

import (
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
)

// GetNetworks is response of GET /v2.0/networks call
const GetNetworks = `
{
    "networks": [
        {
            "admin_state_up": true,
            "id": "396f12f8-521e-4b91-8e21-2e003500433a",
            "name": "net3",
            "provider:network_type": "vlan",
            "provider:physical_network": "physnet1",
            "provider:segmentation_id": 1002,
            "router:external": false,
            "shared": false,
            "status": "ACTIVE",
            "subnets": [],
            "tenant_id": "20bd52ff3e1b40039c312395b04683cf",
            "project_id": "20bd52ff3e1b40039c312395b04683cf"
        },
        {
            "admin_state_up": true,
            "id": "71c1e68c-171a-4aa2-aca5-50ea153a3718",
            "name": "net2",
            "provider:network_type": "vlan",
            "provider:physical_network": "physnet1",
            "provider:segmentation_id": 1001,
            "router:external": false,
            "shared": false,
            "status": "ACTIVE",
            "subnets": [],
            "tenant_id": "20bd52ff3e1b40039c312395b04683cf",
            "project_id": "20bd52ff3e1b40039c312395b04683cf"
        }
    ],
    "networks_links": [

    ]
}
`

// GetSubnets is response of GET /v2.0/subnets call
const GetSubnets = `
{
    "subnets": [
        {
            "name": "private-subnet",
            "enable_dhcp": true,
            "network_id": "db193ab3-96e3-4cb3-8fc5-05f4296d0324",
            "segment_id": null,
            "project_id": "26a7980765d0414dbc1fc1f88cdb7e6e",
            "tenant_id": "26a7980765d0414dbc1fc1f88cdb7e6e",
            "dns_nameservers": [],
            "allocation_pools": [
                {
                    "start": "10.0.0.2",
                    "end": "10.0.0.254"
                }
            ],
            "host_routes": [],
            "ip_version": 4,
            "gateway_ip": "10.0.0.1",
            "cidr": "10.0.0.0/24",
            "id": "08eae331-0402-425a-923c-34f7cfe39c1b",
            "created_at": "2016-10-10T14:35:34Z",
            "description": "",
            "ipv6_address_mode": null,
            "ipv6_ra_mode": null,
            "revision_number": 2,
            "service_types": [],
            "subnetpool_id": null,
            "tags": ["tag1,tag2"],
            "updated_at": "2016-10-10T14:35:34Z"
        },
        {
            "name": "my_subnet",
            "enable_dhcp": true,
            "network_id": "d32019d3-bc6e-4319-9c1d-6722fc136a22",
            "segment_id": null,
            "project_id": "4fd44f30292945e481c7b8a0c8908869",
            "tenant_id": "4fd44f30292945e481c7b8a0c8908869",
            "dns_nameservers": [],
            "allocation_pools": [
                {
                    "start": "192.0.0.2",
                    "end": "192.255.255.254"
                }
            ],
            "host_routes": [],
            "ip_version": 4,
            "gateway_ip": "192.0.0.1",
            "cidr": "192.0.0.0/8",
            "id": "54d6f61d-db07-451c-9ab3-b9609b6b6f0b",
            "created_at": "2016-10-10T14:35:47Z",
            "description": "",
            "ipv6_address_mode": null,
            "ipv6_ra_mode": null,
            "revision_number": 2,
            "service_types": [],
            "subnetpool_id": null,
            "tags": ["tag1,tag2"],
            "updated_at": "2016-10-10T14:35:47Z"
        }
    ]
}
`

var subnet1 = subnets.Subnet{
	ID:             "08eae331-0402-425a-923c-34f7cfe39c1b",
	NetworkID:      "db193ab3-96e3-4cb3-8fc5-05f4296d0324",
	Name:           "private-subnet",
	IPVersion:      4,
	CIDR:           "10.0.0.0/24",
	GatewayIP:      "10.0.0.1",
	DNSNameservers: []string{},
	AllocationPools: []subnets.AllocationPool{
		subnets.AllocationPool{
			Start: "10.0.0.2",
			End:   "10.0.0.254",
		},
	},
	HostRoutes:      []subnets.HostRoute{},
	EnableDHCP:      true,
	TenantID:        "26a7980765d0414dbc1fc1f88cdb7e6e",
	ProjectID:       "26a7980765d0414dbc1fc1f88cdb7e6e",
	IPv6AddressMode: "",
	IPv6RAMode:      "",
	SubnetPoolID:    "",
}

var subnet2 = subnets.Subnet{
	ID:             "54d6f61d-db07-451c-9ab3-b9609b6b6f0b",
	NetworkID:      "d32019d3-bc6e-4319-9c1d-6722fc136a22",
	Name:           "my_subnet",
	IPVersion:      4,
	CIDR:           "192.0.0.0/8",
	GatewayIP:      "192.0.0.1",
	DNSNameservers: []string{},
	AllocationPools: []subnets.AllocationPool{
		subnets.AllocationPool{
			Start: "192.0.0.2",
			End:   "192.255.255.254",
		},
	},
	HostRoutes:      []subnets.HostRoute{},
	EnableDHCP:      true,
	TenantID:        "4fd44f30292945e481c7b8a0c8908869",
	ProjectID:       "4fd44f30292945e481c7b8a0c8908869",
	IPv6AddressMode: "",
	IPv6RAMode:      "",
	SubnetPoolID:    "",
}

var expectedSubnets = []subnets.Subnet{subnet1, subnet2}

var network1 = networks.Network{
	ID:                    "396f12f8-521e-4b91-8e21-2e003500433a",
	Name:                  "net3",
	AdminStateUp:          true,
	Status:                "ACTIVE",
	Subnets:               []string{},
	TenantID:              "20bd52ff3e1b40039c312395b04683cf",
	ProjectID:             "20bd52ff3e1b40039c312395b04683cf",
	Shared:                false,
	AvailabilityZoneHints: []string(nil),
}

// NetworkExternalExt: external.NetworkExternalExt{External: false},
// }

var network2 = networks.Network{
	ID:                    "71c1e68c-171a-4aa2-aca5-50ea153a3718",
	Name:                  "net2",
	AdminStateUp:          true,
	Status:                "ACTIVE",
	Subnets:               []string{},
	TenantID:              "20bd52ff3e1b40039c312395b04683cf",
	ProjectID:             "20bd52ff3e1b40039c312395b04683cf",
	Shared:                false,
	AvailabilityZoneHints: []string(nil),
}

var expectedNetworks = []networks.Network{network1, network2}
