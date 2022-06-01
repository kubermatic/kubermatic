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
	"flag"
	"log"
	"os"

	"gopkg.in/yaml.v3"
	"k8c.io/kubermatic/v2/cmd/ldap-group-syncer/pkg/ldap"
	"k8c.io/kubermatic/v2/cmd/ldap-group-syncer/pkg/types"
)

func main() {
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("No configuration file given.")
	}

	config, err := types.LoadConfig(args[0])
	if err != nil {
		log.Fatal(err)
	}

	l, err := ldap.NewClient(config.Address)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	var org *types.Organization

	if config.Mapping.Grouped != nil {
		org, err = l.FetchGroupedData(config.Mapping.Grouped)
	} else {
		org, err = l.FetchTaggedData(config.Mapping.Tagged)
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
