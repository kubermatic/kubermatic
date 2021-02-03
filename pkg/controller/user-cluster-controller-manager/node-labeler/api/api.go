/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package api

const (
	// DistributionLabelKey is the label that gets applied.
	DistributionLabelKey = "x-kubernetes.io/distribution"

	// CentOSLabelValue is the value of the label for CentOS
	CentOSLabelValue = "centos"

	// UbuntuLabelValue is the value of the label for Ubuntu
	UbuntuLabelValue = "ubuntu"

	// SLESLabelValue is the value of the label for SLES
	SLESLabelValue = "sles"

	// RHELLabelValue is the value of the label for RHEL
	RHELLabelValue = "rhel"

	// FlatcarLabelValue is the value of the label for Flatcar Linux
	FlatcarLabelValue = "flatcar"
)

// OSLabelMatchValues is a mapping between OS labels and the strings to match on in OSImage.
// Note that these are all lower case.
var OSLabelMatchValues = map[string]string{
	CentOSLabelValue:  "centos",
	UbuntuLabelValue:  "ubuntu",
	SLESLabelValue:    "sles",
	RHELLabelValue:    "rhel",
	FlatcarLabelValue: "flatcar container linux",
}
