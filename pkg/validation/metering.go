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

	"github.com/robfig/cron/v3"

	v1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
)

func GetCronExpressionParser() cron.Parser {
	return cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
}

func ValidateMeteringConfiguration(configuration *v1.MeteringConfiguration) error {
	if configuration != nil && len(configuration.ReportConfigurations) > 0 {
		parser := GetCronExpressionParser()
		for _, reportConfig := range configuration.ReportConfigurations {
			if _, err := parser.Parse(reportConfig.Schedule); err != nil {
				return fmt.Errorf("invalid cron expression format: %s", reportConfig.Schedule)
			}
		}
	}
	return nil
}
