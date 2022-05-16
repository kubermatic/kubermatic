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

package provider_test

// for more responses see: https://developer.openstack.org/api-ref/network/v2/
// and https://developer.openstack.org/api-ref/identity/v3/

// PostTokens is response of POST /v3/auth/tokens.
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

// GetSubnetPools is response of GET /v2.0/subnetpools.
const GetSubnetPools = `
{
    "subnetpools": [
        {
            "min_prefixlen": "64",
            "address_scope_id": null,
            "default_prefixlen": "64",
            "id": "03f761e6-eee0-43fc-a921-8acf64c14988",
            "max_prefixlen": "64",
            "name": "my-subnet-pool-ipv6",
            "default_quota": null,
            "is_default": false,
            "project_id": "9fadcee8aa7c40cdb2114fff7d569c08",
            "tenant_id": "9fadcee8aa7c40cdb2114fff7d569c08",
            "prefixes": [
                "2001:db8:0:2::/64",
                "2001:db8::/63"
            ],
            "ip_version": 6,
            "shared": false,
            "description": "",
            "created_at": "2016-03-08T20:19:41",
            "updated_at": "2016-03-08T20:19:41",
            "revision_number": 2,
            "tags": ["tag1,tag2"]
        },
        {
            "min_prefixlen": "24",
            "address_scope_id": null,
            "default_prefixlen": "25",
            "id": "f49a1319-423a-4ee6-ba54-1d95a4f6cc68",
            "max_prefixlen": "30",
            "name": "my-subnet-pool-ipv4",
            "default_quota": null,
            "is_default": false,
            "project_id": "9fadcee8aa7c40cdb2114fff7d569c08",
            "tenant_id": "9fadcee8aa7c40cdb2114fff7d569c08",
            "prefixes": [
                "10.10.0.0/21",
                "192.168.0.0/16"
            ],
            "ip_version": 4,
            "shared": false,
            "description": "",
            "created_at": "2016-03-08T20:19:41",
            "updated_at": "2016-03-08T20:19:41",
            "revision_number": 2,
            "tags": ["tag1,tag2"]
        }
    ]
}
`
