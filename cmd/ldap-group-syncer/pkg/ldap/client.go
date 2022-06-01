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
	"fmt"
	"log"

	ldap3 "github.com/go-ldap/ldap/v3"

	"k8c.io/kubermatic/v2/cmd/ldap-group-syncer/pkg/types"
)

type Client struct {
	*ldap3.Conn
}

// NewClient returns a new LDAP client.
func NewClient(addr string) (*Client, error) {
	conn, err := ldap3.DialURL(addr)
	if err != nil {
		return nil, err
	}

	return &Client{
		Conn: conn,
	}, nil
}

// authenticate demonstrate a BIND operation using behera password auth.
// func authenticate() error {
// 	controls := []ldap.Control{}
// 	controls = append(controls, ldap.NewControlBeheraPasswordPolicy())
// 	bindRequest := ldap.NewSimpleBindRequest("cn=admin,dc=planetexpress,dc=com", "GoodNewsEveryone", controls)

// 	r, err := l.SimpleBind(bindRequest)
// 	ppolicyControl := ldap.FindControl(r.Controls, ldap.ControlTypeBeheraPasswordPolicy)

// 	var ppolicy *ldap.ControlBeheraPasswordPolicy
// 	if ppolicyControl != nil {
// 		ppolicy = ppolicyControl.(*ldap.ControlBeheraPasswordPolicy)
// 	} else {
// 		log.Printf("ppolicyControl response not available.\n")
// 	}
// 	if err != nil {
// 		errStr := "ERROR: Cannot bind: " + err.Error()
// 		if ppolicy != nil && ppolicy.Error >= 0 {
// 			errStr += ":" + ppolicy.ErrorString
// 		}
// 		log.Print(errStr)
// 	} else {
// 		logStr := "Login Ok"
// 		if ppolicy != nil {
// 			if ppolicy.Expire >= 0 {
// 				logStr += fmt.Sprintf(". Password expires in %d seconds\n", ppolicy.Expire)
// 			} else if ppolicy.Grace >= 0 {
// 				logStr += fmt.Sprintf(". Password expired, %d grace logins remain\n", ppolicy.Grace)
// 			}
// 		}
// 		log.Print(logStr)
// 	}

// 	return nil
// }

func (c *Client) FetchGroupedData(config *types.GroupedConfig) (*types.Organization, error) {
	groupAttributes := []string{
		"dn",
		config.GroupNameAttribute,
		config.MemberAttribute,
	}

	// step 1, find all relevant groups
	searchRequest := ldap3.NewSearchRequest(
		config.BaseDN,
		ldap3.ScopeWholeSubtree, ldap3.NeverDerefAliases, 0, 0, false,
		config.Query,
		groupAttributes,
		nil,
	)

	log.Println("Fetching groups…")

	groupsResult, err := c.SearchWithPaging(searchRequest, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	// go through all groups and collect the DNs for every member
	persons := map[string]types.Person{}
	for _, group := range groupsResult.Entries {
		for _, member := range group.GetAttributeValues(config.MemberAttribute) {
			persons[member] = types.Person{}
		}
	}

	// step2, fetch all the members individually
	personAttributes := []string{
		"dn",
		config.EmailAttribute,
		config.PersonNameAttribute,
	}

	for personDN := range persons {
		searchRequest := ldap3.NewSearchRequest(
			personDN,
			ldap3.ScopeBaseObject, ldap3.NeverDerefAliases, 0, 0, false,
			"(objectClass=*)",
			personAttributes,
			nil,
		)

		log.Printf("Fetching person %q…", personDN)
		sr, err := c.Search(searchRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch person %q: %w", personDN, err)
		}

		for _, person := range sr.Entries {
			persons[personDN] = types.Person{
				DN:    personDN,
				Name:  person.GetAttributeValue(config.PersonNameAttribute),
				Email: person.GetAttributeValue(config.EmailAttribute),
			}
		}
	}

	// step 3, create a nice result structure with org and members
	org := &types.Organization{
		Groups: []types.Group{},
	}

	for _, group := range groupsResult.Entries {
		members := []types.Person{}
		for _, member := range group.GetAttributeValues(config.MemberAttribute) {
			members = append(members, persons[member])
		}

		types.SortPersons(members)
		org.Groups = append(org.Groups, types.Group{
			DN:      group.DN,
			Name:    group.GetAttributeValue(config.GroupNameAttribute),
			Members: members,
		})
	}

	types.SortGroups(org.Groups)

	return org, nil
}

func (c *Client) FetchTaggedData(config *types.TaggedConfig) (*types.Organization, error) {
	personAttributes := []string{
		"dn",
		config.EmailAttribute,
		config.PersonNameAttribute,
	}

	searchRequest := ldap3.NewSearchRequest(
		config.BaseDN,
		ldap3.ScopeWholeSubtree, ldap3.NeverDerefAliases, 0, 0, false,
		config.Query,
		personAttributes,
		nil,
	)

	log.Println("Fetching persons…")

	personsResult, err := c.SearchWithPaging(searchRequest, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to list persons: %w", err)
	}

	// go through all persons and collect the DNs for every group
	groups := map[string]types.Group{}
	for _, person := range personsResult.Entries {
		for _, groupDN := range person.GetAttributeValues(config.GroupAttribute) {
			groups[groupDN] = types.Group{}
		}
	}

	// step2, fetch all the groups individually
	groupAttributes := []string{
		"dn",
		config.GroupNameAttribute,
	}

	for groupDN := range groups {
		searchRequest := ldap3.NewSearchRequest(
			groupDN,
			ldap3.ScopeBaseObject, ldap3.NeverDerefAliases, 0, 0, false,
			"(objectClass=*)",
			groupAttributes,
			nil,
		)

		log.Printf("Fetching group %q…", groupDN)
		sr, err := c.Search(searchRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch group %q: %w", groupDN, err)
		}

		for _, group := range sr.Entries {
			groups[groupDN] = types.Group{
				DN:      group.DN,
				Name:    group.GetAttributeValue(config.GroupNameAttribute),
				Members: []types.Person{},
			}
		}
	}

	// step 3, create a nice result structure with org and members
	for _, person := range personsResult.Entries {
		for _, groupDN := range person.GetAttributeValues(config.GroupAttribute) {
			group := groups[groupDN]
			group.Members = append(group.Members, types.Person{
				DN:    person.DN,
				Name:  person.GetAttributeValue(config.PersonNameAttribute),
				Email: person.GetAttributeValue(config.EmailAttribute),
			})

			groups[groupDN] = group
		}
	}

	org := &types.Organization{
		Groups: []types.Group{},
	}

	for i := range groups {
		types.SortPersons(groups[i].Members)
		org.Groups = append(org.Groups, groups[i])
	}

	types.SortGroups(org.Groups)

	return org, nil
}
