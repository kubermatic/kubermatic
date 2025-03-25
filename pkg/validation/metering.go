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

package validation

import (
	"fmt"
	"strings"

	"github.com/robfig/cron/v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/utils/strings/slices"
)

var reportTypes = []string{"cluster", "namespace"}

func GetCronExpressionParser() cron.Parser {
	return cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
}

func ValidateMeteringConfiguration(configuration *kubermaticv1.MeteringConfiguration) error {
	if configuration != nil {
		if configuration.RetentionDays < 0 {
			return fmt.Errorf("invalid retentionDays (%d): must be positive", configuration.RetentionDays)
		}

		if size := configuration.StorageSize; size != "" {
			if _, err := resource.ParseQuantity(size); err != nil {
				return fmt.Errorf("invalid storageSize (%q): %w", size, err)
			}
		}

		if len(configuration.ReportConfigurations) > 0 {
			parser := GetCronExpressionParser()
			for reportName, reportConfig := range configuration.ReportConfigurations {
				if errs := validation.IsDNS1035Label(reportName); len(errs) != 0 {
					return fmt.Errorf("metering report configuration name must be valid rfc1035 label: %s", strings.Join(errs, ","))
				}

				if _, err := parser.Parse(reportConfig.Schedule); err != nil {
					return fmt.Errorf("invalid cron expression format: %s", reportConfig.Schedule)
				}

				for _, t := range reportConfig.Types {
					if !slices.Contains(reportTypes, t) {
						return fmt.Errorf("invalid report type: %s", t)
					}
				}
			}
		}
	}

	return nil
}
