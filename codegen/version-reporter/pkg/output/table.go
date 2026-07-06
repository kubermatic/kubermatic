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

package output

import (
	"io"

	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"

	"k8c.io/kubermatic/v2/codegen/version-reporter/pkg/config"

	"k8s.io/apimachinery/pkg/util/sets"
)

func FormatTable(cfg *config.Config, dest io.Writer) error {
	columns := []string{"Product", "Type", "Location"}
	allVersions := getVersionSuperset(cfg)

	if len(allVersions) > 0 {
		columns = append(columns, allVersions...)
	} else {
		columns = append(columns, "Version")
	}

	versionColumns := func(versions map[string]string) []string {
		result := []string{}

		for _, versionKey := range allVersions {
			if versions == nil {
				result = append(result, "?")
				continue
			}

			version, ok := versions[versionKey]
			if ok {
				result = append(result, version)
			} else {
				result = append(result, "")
			}
		}

		return result
	}

	mergeCfg := tablewriter.NewConfigBuilder().
		// merge identical adjacent cells vertically in the Product/Type columns,
		// so repeated values collapse instead of printing on every row
		Row().Merging().WithMode(tw.MergeVertical).ByColumnIndex([]int{0, 1}).Build().Build().Build()

	table := tablewriter.NewWriter(dest)
	table.Options(
		tablewriter.WithHeader(columns),
		tablewriter.WithHeaderAutoWrap(tw.WrapNone),
		tablewriter.WithRowAutoWrap(tw.WrapNone),
		tablewriter.WithConfig(mergeCfg),
	)

	for _, product := range cfg.Products {
		for _, occ := range product.Occurrences {
			row := []string{product.Name, occ.TypeName(), occ.String()}
			if err := table.Append(append(row, versionColumns(occ.Versions)...)); err != nil {
				return err
			}
		}
	}

	return table.Render()
}

func getVersionSuperset(c *config.Config) []string {
	superset := sets.New[string]()

	for _, p := range c.Products {
		for _, o := range p.Occurrences {
			superset = superset.Union(sets.KeySet(o.Versions))
		}
	}

	if superset.Len() == 1 && superset.Has(config.Unversioned) {
		return nil
	}

	return sets.List(superset)
}
