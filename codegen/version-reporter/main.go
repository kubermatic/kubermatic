/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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
	"strings"

	"k8c.io/kubermatic/v2/codegen/version-reporter/pkg/config"
	"k8c.io/kubermatic/v2/codegen/version-reporter/pkg/output"
	"k8c.io/kubermatic/v2/codegen/version-reporter/pkg/reader"
)

type options struct {
	configFile string
	json       bool
}

func main() {
	opt := options{}
	flag.StringVar(&opt.configFile, "config", "", "path to the config file")
	flag.BoolVar(&opt.json, "json", false, "output JSON instead of plaintext")
	flag.Parse()

	cfg, err := config.Load(opt.configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// detect all versions
	success := reader.ResolveConfig(cfg)

	// cheat: remove common prefix from package names for easier readability
	for pdx, product := range cfg.Products {
		for odx, occ := range product.Occurrences {
			if occ.GoConstant != nil {
				newPackage := strings.TrimPrefix(occ.GoConstant.Package, "k8c.io/kubermatic/v2/")

				cfg.Products[pdx].Occurrences[odx].GoConstant.Package = newPackage
			}
		}
	}

	// make output look nicer :)
	cfg.Sort()

	if opt.json {
		err = output.FormatJSON(cfg, os.Stdout)
	} else {
		err = output.FormatTable(cfg, os.Stdout)
	}

	if err != nil {
		log.Fatalf("Failed to output results: %v", err)
	}

	if !success {
		os.Exit(1)
	}
}
