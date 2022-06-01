//go:build ldaptest

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

package ldap

import (
	"reflect"
	"testing"

	"k8c.io/kubermatic/v2/cmd/ldap-group-syncer/pkg/types"

	"k8s.io/utils/diff"
)

func TestFetchGroupedData(t *testing.T) {
	cfg, err := types.LoadConfig("../../testdata/pawnee-grouped/config.yaml")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	client, err := NewClient("ldap://localhost:10389")
	if err != nil {
		t.Fatalf("Failed to create LDAP client: %v", err)
	}

	org, err := client.FetchGroupedData(cfg.Mapping.Grouped)
	if err != nil {
		t.Fatalf("Failed to fetch data: %v", err)
	}

	expected := &types.Organization{
		Groups: []types.Group{
			{
				DN:   "cn=City Government,ou=people,dc=pawnee,dc=gov",
				Name: "City Government",
				Members: []types.Person{
					{DN: "cn=Ben Wyatt,ou=people,dc=pawnee,dc=gov", Name: "Ben Wyatt", Email: "bwyatt@pawnee.gov"},
					{DN: "cn=Chris Traeger,ou=people,dc=pawnee,dc=gov", Name: "Chris Traeger", Email: "ctrager@pawnee.gov"},
					{DN: "cn=Leslie Knope,ou=people,dc=pawnee,dc=gov", Name: "Leslie Knope", Email: "lknope@pawnee.gov"},
				},
			},
			{
				DN:   "cn=Lot 48 Project,ou=people,dc=pawnee,dc=gov",
				Name: "Lot 48 Project",
				Members: []types.Person{
					{DN: "cn=Andy Dwyer,ou=people,dc=pawnee,dc=gov", Name: "Andy Dwyer", Email: "adwyer@pawnee.gov"},
					{DN: "cn=Ann Perkins,ou=people,dc=pawnee,dc=gov", Name: "Ann Perkins", Email: "aperkins@pawnee.gov"},
					{DN: "cn=Leslie Knope,ou=people,dc=pawnee,dc=gov", Name: "Leslie Knope", Email: "lknope@pawnee.gov"},
					{DN: "cn=Mark Brendanawicz,ou=people,dc=pawnee,dc=gov", Name: "Mark Brendanawicz", Email: "mbrendanawicz@pawnee.gov"},
				},
			},
			{
				DN:   "cn=Media,ou=people,dc=pawnee,dc=gov",
				Name: "Media",
				Members: []types.Person{
					{DN: "cn=Joan Callamezzo,ou=people,dc=pawnee,dc=gov", Name: "Joan Callamezzo", Email: "jcallamezzo@pawnee.gov"},
					{DN: "cn=Perd Hapley,ou=people,dc=pawnee,dc=gov", Name: "Perd Hapley", Email: "phapley@pawnee.gov"},
				},
			},
			{
				DN:   "cn=Parks & Recreation,ou=people,dc=pawnee,dc=gov",
				Name: "Parks & Recreation",
				Members: []types.Person{
					{DN: "cn=April Ludgate,ou=people,dc=pawnee,dc=gov", Name: "April Ludgate", Email: "aludgate@pawnee.gov"},
					{DN: "cn=Donna Meagle,ou=people,dc=pawnee,dc=gov", Name: "Donna Meagle", Email: "dmeagle@pawnee.gov"},
					{DN: "cn=Garry Gergich,ou=people,dc=pawnee,dc=gov", Name: "Garry Gergich", Email: "ggergich@pawnee.gov"},
					{DN: "cn=Leslie Knope,ou=people,dc=pawnee,dc=gov", Name: "Leslie Knope", Email: "lknope@pawnee.gov"},
					{DN: "cn=Mark Brendanawicz,ou=people,dc=pawnee,dc=gov", Name: "Mark Brendanawicz", Email: "mbrendanawicz@pawnee.gov"},
					{DN: "cn=Ron Swanson,ou=people,dc=pawnee,dc=gov", Name: "Ron Swanson", Email: "rswanson@pawnee.gov"},
					{DN: "cn=Tom Haverford,ou=people,dc=pawnee,dc=gov", Name: "Tom Haverford", Email: "thaverford@pawnee.gov"},
				},
			},
			{
				DN:   "cn=The Dynamic Duo,ou=people,dc=pawnee,dc=gov",
				Name: "The Dynamic Duo",
				Members: []types.Person{
					{DN: "cn=Ben Wyatt,ou=people,dc=pawnee,dc=gov", Name: "Ben Wyatt", Email: "bwyatt@pawnee.gov"},
					{DN: "cn=Chris Traeger,ou=people,dc=pawnee,dc=gov", Name: "Chris Traeger", Email: "ctrager@pawnee.gov"},
				},
			},
		},
	}

	if !reflect.DeepEqual(expected, org) {
		t.Fatalf("diff: %s", diff.ObjectGoPrintSideBySide(expected, org))
	}
}
