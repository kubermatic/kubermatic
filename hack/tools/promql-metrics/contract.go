/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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
	"os"
	"sort"
	"strings"

	"sigs.k8s.io/yaml"
)

// contract lists every metric that rules and dashboards may reference,
// grouped by where CI can observe it.
type contract struct {
	// Seed metrics are scraped by the seed Prometheus in CI.
	Seed []string `json:"seed"`
	// UserCluster metrics come from user cluster MLA agents in CI.
	UserCluster []string `json:"userCluster"`
	// NotInCI metrics are referenced but never exported in CI.
	NotInCI []notInCIEntry `json:"notInCI"`
}

type notInCIEntry struct {
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

func loadContract(path string) (*contract, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var c contract
	if err := yaml.UnmarshalStrict(data, &c); err != nil {
		return nil, fmt.Errorf("parse contract %s: %w", path, err)
	}

	return &c, nil
}

// check returns one line per problem: duplicate or empty entries,
// referenced metrics missing from the contract and contract entries that
// nothing references any more.
func (c *contract) check(referenced map[string]map[string]struct{}) []string {
	var problems []string

	listed := map[string][]string{}
	add := func(list, name string) {
		listed[name] = append(listed[name], list)
	}

	for _, n := range c.Seed {
		add("seed", n)
	}
	for _, n := range c.UserCluster {
		add("userCluster", n)
	}
	for _, e := range c.NotInCI {
		add("notInCI", e.Name)
		if strings.TrimSpace(e.Reason) == "" {
			problems = append(problems, fmt.Sprintf("notInCI entry %q has no reason", e.Name))
		}
	}

	names := make([]string, 0, len(listed))
	for n := range listed {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, n := range names {
		if strings.TrimSpace(n) == "" {
			problems = append(problems, "contract contains an empty metric name")
			continue
		}
		if lists := listed[n]; len(lists) > 1 {
			problems = append(problems, fmt.Sprintf("duplicate: %s is listed more than once (%s)", n, strings.Join(lists, ", ")))
		}
		if _, ok := referenced[n]; !ok {
			problems = append(problems, fmt.Sprintf("stale: %s is in the contract but no rule or dashboard references it", n))
		}
	}

	refNames := make([]string, 0, len(referenced))
	for n := range referenced {
		refNames = append(refNames, n)
	}
	sort.Strings(refNames)

	for _, n := range refNames {
		if _, ok := listed[n]; ok {
			continue
		}

		files := make([]string, 0, len(referenced[n]))
		for f := range referenced[n] {
			files = append(files, f)
		}
		sort.Strings(files)
		problems = append(problems, fmt.Sprintf("missing: %s is referenced by %s but not in the contract", n, strings.Join(files, ", ")))
	}

	return problems
}
