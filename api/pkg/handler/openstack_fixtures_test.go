package handler

// for more responses see: https://developer.openstack.org/api-ref/network/v2/
// and https://developer.openstack.org/api-ref/identity/v3/

// GetFlaivorsDetail: GET /flavors/detail
const GetFlaivorsDetail = `
{
    "flavors": [
        {
            "OS-FLV-DISABLED:disabled": false,
            "disk": 40,
            "OS-FLV-EXT-DATA:ephemeral": 0,
            "os-flavor-access:is_public": true,
            "id": "3",
            "links": [
                {
                    "href": "http://openstack.example.com/v2/6f70656e737461636b20342065766572/flavors/3",
                    "rel": "self"
                },
                {
                    "href": "http://openstack.example.com/6f70656e737461636b20342065766572/flavors/3",
                    "rel": "bookmark"
                }
            ],
            "name": "m1.medium",
            "ram": 4096,
            "swap": "",
            "vcpus": 2,
            "rxtx_factor": 1.0,
            "description": null,
            "extra_specs": {}
        },
        {
            "OS-FLV-DISABLED:disabled": false,
            "disk": 80,
            "OS-FLV-EXT-DATA:ephemeral": 0,
            "os-flavor-access:is_public": true,
            "id": "4",
            "links": [
                {
                    "href": "http://openstack.example.com/v2/6f70656e737461636b20342065766572/flavors/4",
                    "rel": "self"
                },
                {
                    "href": "http://openstack.example.com/6f70656e737461636b20342065766572/flavors/4",
                    "rel": "bookmark"
                }
            ],
            "name": "m1.large",
            "ram": 8192,
            "swap": "",
            "vcpus": 4,
            "rxtx_factor": 1.0,
            "description": null,
            "extra_specs": {}
        },
        {
            "OS-FLV-DISABLED:disabled": false,
            "disk": 1,
            "OS-FLV-EXT-DATA:ephemeral": 0,
            "os-flavor-access:is_public": true,
            "id": "6",
            "links": [
                {
                    "href": "http://openstack.example.com/v2/6f70656e737461636b20342065766572/flavors/6",
                    "rel": "self"
                },
                {
                    "href": "http://openstack.example.com/6f70656e737461636b20342065766572/flavors/6",
                    "rel": "bookmark"
                }
            ],
            "name": "m1.tiny.specs",
            "ram": 512,
            "swap": "",
            "vcpus": 1,
            "rxtx_factor": 1.0,
            "description": null,
            "extra_specs": {
                "hw:cpu_model": "SandyBridge",
                "hw:mem_page_size": "2048",
                "hw:cpu_policy": "dedicated"
            }
        }
    ]
}`

//GetUserProjects is response  pf GET /users/{user_id}/projects
const GetUserProjects = `
{
    "projects": [
        {
            "description": "description of project Foo",
            "domain_id": "161718",
            "enabled": true,
            "id": "456788",
            "links": {
                "self": "http://example.com/identity/v3/projects/456788"
            },
            "name": "a project name",
            "parent_id": "212223"
        },
        {
            "description": "description of project Bar",
            "domain_id": "161718",
            "enabled": true,
            "id": "456789",
            "links": {
                "self": "http://example.com/identity/v3/projects/456789"
            },
            "name": "another domain",
            "parent_id": "212223"
        }
    ],
    "links": {
        "self": "http://example.com/identity/v3/users/313233/projects",
        "previous": null,
        "next": null
    }
}
`

// const GetTokens = `
// `

// PostTokens is response of POST /v3/auth/tokens
const PostTokens = `
{
    "token": {
        "audit_ids": [
            "3T2dc1CGQxyJsHdDu1xkcw"
        ],
        "catalog": [
            {
                "endpoints": [
                    {
                        "id": "068d1b359ee84b438266cb736d81de97",
                        "interface": "public",
                        "region": "{{.Region}}",
                        "url": "{{.URL}}"
                    },
                    {
                        "id": "8bfc846841ab441ca38471be6d164ced",
                        "interface": "admin",
                        "region": "{{.Region}}",
                        "url": "{{.URL}}"
                    },
                    {
                        "id": "beb6d358c3654b4bada04d4663b640b9",
                        "interface": "internal",
                        "region": "{{.Region}}",
                        "url": "{{.URL}}"
                    }
                ],
                "type": "compute",
                "id": "a50726f278654128aba89757ae25910c",
                "name": "keystone"
            },
            {
                "endpoints": [
                    {
                        "id": "068d1b359ee84b438266cb736d81de97",
                        "interface": "public",
                        "region": "{{.Region}}",
                        "region_id": "RegionOne",
                        "url": "{{.URL}}"
                    },
                    {
                        "id": "8bfc846841ab441ca38471be6d164ced",
                        "interface": "admin",
                        "region": "{{.Region}}",
                        "region_id": "RegionOne",
                        "url": "{{.URL}}"
                    },
                    {
                        "id": "beb6d358c3654b4bada04d4663b640b9",
                        "interface": "internal",
                        "region": "{{.Region}}",
                        "region_id": "RegionOne",
                        "url": "{{.URL}}"
                    }
                ],
                "type": "network",
                "id": "050726f278654128aba89757ae25950c",
                "name": "keystone"
            },
            {
                "endpoints": [
                    {
                        "id": "068d1b359ee84b438266cb736d81de97",
                        "interface": "public",
                        "region": "{{.Region}}",
                        "region_id": "RegionOne",
                        "url": "{{.URL}}"
                    },
                    {
                        "id": "8bfc846841ab441ca38471be6d164ced",
                        "interface": "admin",
                        "region": "{{.Region}}",
                        "region_id": "RegionOne",
                        "url": "{{.URL}}"
                    },
                    {
                        "id": "beb6d358c3654b4bada04d4663b640b9",
                        "interface": "internal",
                        "region": "{{.Region}}",
                        "region_id": "RegionOne",
                        "url": "{{.URL}}"
                    }
                ],
                "type": "identity",
                "id": "050726f278654128aba89757ae25950c",
                "name": "keystone"
            }
        ],
        "domain": {
            "id": "default",
            "name": "{{.Domain}}"
        },
        "expires_at": "2015-11-07T02:58:43.578887Z",
        "issued_at": "2015-11-07T01:58:43.578929Z",
        "methods": [
            "password"
        ],
        "roles": [
            {
                "id": "51cc68287d524c759f47c811e6463340",
                "name": "{{.User}}"
            }
        ],
        "user": {
            "domain": {
                "id": "default",
                "name": "{{.Domain}}"
            },
            "id": "{{.TokenID}}",
            "name": "{{.User}}",
            "password_expires_at": "3016-11-06T15:32:17.000000"
        }
    }
}
`

// GetNetworks is response of GET /v2.0/networks
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

// GetSubnets is response of GET /v2.0/subnets
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

const ExpectedSubnets = `
[
    {"id": "08eae331-0402-425a-923c-34f7cfe39c1b", "name": "private-subnet"},
    {"id": "54d6f61d-db07-451c-9ab3-b9609b6b6f0b", "name": "my_subnet"}
]
`

const ExpectedNetworks = `
[
    {"id": "396f12f8-521e-4b91-8e21-2e003500433a", "name": "net3"},
    {"id": "71c1e68c-171a-4aa2-aca5-50ea153a3718", "name": "net2"}
]
`
const ExpectedTenants = `
[
    {"id":"456788", "name": "a project name"},
    {"id":"456789", "name": "another domain"}
]
`
const ExpectedSizes = `
[
    {
        "disk":40,
        "isPublic":true,
        "memory":4096,
        "region":"RegionOne",
        "slug":"m1.medium",
        "swap":0,
        "vcpus":2
    },
    {
        "disk":80,
        "isPublic":true,
        "memory":8192,
        "region":"RegionOne",
        "slug":"m1.large",
        "swap":0,
        "vcpus":4
    },
    {
        "disk":1,
        "isPublic":true,
        "memory":512,
        "region":"RegionOne",
        "slug":"m1.tiny.specs",
        "swap":0,
        "vcpus":1
    }
]
`
