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

package main

import (
	"fmt"
	"log"
	"os"
	"sort"

	ldap "github.com/go-ldap/ldap/v3"
	"gopkg.in/yaml.v3"
)

type Person struct {
	DN    string `yaml:"dn"`
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
}

type Group struct {
	DN      string   `yaml:"dn"`
	Name    string   `yaml:"name"`
	Members []Person `yaml:"members"`
}

type Organization struct {
	Groups []Group `yaml:"groups"`
}

func main() {
	config, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	l, err := ldap.DialURL(config.Address)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	var org *Organization

	if config.Mapping.Grouped != nil {
		org, err = fetchGroupedData(l, config.Mapping.Grouped)
	} else {
		org, err = fetchTaggedData(l, config.Mapping.Tagged)
	}

	if err != nil {
		log.Fatal(err)
	}

	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)

	if err := encoder.Encode(org); err != nil {
		log.Fatal(err)
	}
}

func fetchGroupedData(conn *ldap.Conn, config *GroupedConfig) (*Organization, error) {
	groupAttributes := []string{
		"dn",
		config.GroupNameAttribute,
		config.MemberAttribute,
	}

	// step 1, find all relevant groups
	searchRequest := ldap.NewSearchRequest(
		config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		config.Query,
		groupAttributes,
		nil,
	)

	log.Println("Fetching groups…")

	groupsResult, err := conn.SearchWithPaging(searchRequest, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to list groups: %w", err)
	}

	// go through all groups and collect the DNs for every member
	persons := map[string]Person{}
	for _, group := range groupsResult.Entries {
		for _, member := range group.GetAttributeValues(config.MemberAttribute) {
			persons[member] = Person{}
		}
	}

	// step2, fetch all the members individually
	personAttributes := []string{
		"dn",
		config.EmailAttribute,
		config.PersonNameAttribute,
	}

	for personDN := range persons {
		searchRequest := ldap.NewSearchRequest(
			personDN,
			ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
			"(objectClass=*)",
			personAttributes,
			nil,
		)

		log.Printf("Fetching person %q…", personDN)
		sr, err := conn.Search(searchRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch person %q: %w", personDN, err)
		}

		for _, person := range sr.Entries {
			persons[personDN] = Person{
				DN:    personDN,
				Name:  person.GetAttributeValue(config.PersonNameAttribute),
				Email: person.GetAttributeValue(config.EmailAttribute),
			}
		}
	}

	// step 3, create a nice result structure with org and members
	org := &Organization{
		Groups: []Group{},
	}

	for _, group := range groupsResult.Entries {
		members := []Person{}
		for _, member := range group.GetAttributeValues(config.MemberAttribute) {
			members = append(members, persons[member])
		}

		sortMembers(members)
		org.Groups = append(org.Groups, Group{
			DN:      group.DN,
			Name:    group.GetAttributeValue(config.GroupNameAttribute),
			Members: members,
		})
	}

	return org, nil
}

func fetchTaggedData(conn *ldap.Conn, config *TaggedConfig) (*Organization, error) {
	personAttributes := []string{
		"dn",
		config.EmailAttribute,
		config.PersonNameAttribute,
	}

	searchRequest := ldap.NewSearchRequest(
		config.BaseDN,
		ldap.ScopeWholeSubtree, ldap.NeverDerefAliases, 0, 0, false,
		config.Query,
		personAttributes,
		nil,
	)

	log.Println("Fetching persons…")

	personsResult, err := conn.SearchWithPaging(searchRequest, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to list persons: %w", err)
	}

	// go through all persons and collect the DNs for every group
	groups := map[string]Group{}
	for _, person := range personsResult.Entries {
		for _, groupDN := range person.GetAttributeValues(config.GroupAttribute) {
			groups[groupDN] = Group{}
		}
	}

	// step2, fetch all the groups individually
	groupAttributes := []string{
		"dn",
		config.GroupNameAttribute,
	}

	for groupDN := range groups {
		searchRequest := ldap.NewSearchRequest(
			groupDN,
			ldap.ScopeBaseObject, ldap.NeverDerefAliases, 0, 0, false,
			"(objectClass=*)",
			groupAttributes,
			nil,
		)

		log.Printf("Fetching group %q…", groupDN)
		sr, err := conn.Search(searchRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch group %q: %w", groupDN, err)
		}

		for _, group := range sr.Entries {
			groups[groupDN] = Group{
				DN:      group.DN,
				Name:    group.GetAttributeValue(config.GroupNameAttribute),
				Members: []Person{},
			}
		}
	}

	// step 3, create a nice result structure with org and members
	for _, person := range personsResult.Entries {
		for _, groupDN := range person.GetAttributeValues(config.GroupAttribute) {
			group := groups[groupDN]
			group.Members = append(group.Members, Person{
				DN:    person.DN,
				Name:  person.GetAttributeValue(config.PersonNameAttribute),
				Email: person.GetAttributeValue(config.EmailAttribute),
			})

			groups[groupDN] = group
		}
	}

	org := &Organization{
		Groups: []Group{},
	}

	for i := range groups {
		sortMembers(groups[i].Members)
		org.Groups = append(org.Groups, groups[i])
	}

	return org, nil
}

func sortMembers(persons []Person) {
	sort.Slice(persons, func(i, j int) bool {
		return persons[i].Email < persons[j].Email
	})
}

// authenticate demonstrate a BIND operation using behera password auth.
func authenticate() error {
	// controls := []ldap.Control{}
	// controls = append(controls, ldap.NewControlBeheraPasswordPolicy())
	// bindRequest := ldap.NewSimpleBindRequest("cn=admin,dc=planetexpress,dc=com", "GoodNewsEveryone", controls)

	// r, err := l.SimpleBind(bindRequest)
	// ppolicyControl := ldap.FindControl(r.Controls, ldap.ControlTypeBeheraPasswordPolicy)

	// var ppolicy *ldap.ControlBeheraPasswordPolicy
	// if ppolicyControl != nil {
	// 	ppolicy = ppolicyControl.(*ldap.ControlBeheraPasswordPolicy)
	// } else {
	// 	log.Printf("ppolicyControl response not available.\n")
	// }
	// if err != nil {
	// 	errStr := "ERROR: Cannot bind: " + err.Error()
	// 	if ppolicy != nil && ppolicy.Error >= 0 {
	// 		errStr += ":" + ppolicy.ErrorString
	// 	}
	// 	log.Print(errStr)
	// } else {
	// 	logStr := "Login Ok"
	// 	if ppolicy != nil {
	// 		if ppolicy.Expire >= 0 {
	// 			logStr += fmt.Sprintf(". Password expires in %d seconds\n", ppolicy.Expire)
	// 		} else if ppolicy.Grace >= 0 {
	// 			logStr += fmt.Sprintf(". Password expired, %d grace logins remain\n", ppolicy.Grace)
	// 		}
	// 	}
	// 	log.Print(logStr)
	// }

	return nil
}
