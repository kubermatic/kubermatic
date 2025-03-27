/*
Copyright 2024 The Kubermatic Kubernetes Platform contributors.

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

package kubernetesdashboard

import (
	"k8c.io/kubermatic/sdk/v2/semver"
)

func APIVersion(_ semver.Semver) (string, error) {
	return "1.10.1", nil
}

func AuthVersion(_ semver.Semver) (string, error) {
	return "1.2.2", nil
}

func WebVersion(_ semver.Semver) (string, error) {
	return "1.6.0", nil
}

func MetricsScraperVersion(_ semver.Semver) (string, error) {
	return "1.2.1", nil
}

func KongVersion(_ semver.Semver) (string, error) {
	return "3.6", nil
}
