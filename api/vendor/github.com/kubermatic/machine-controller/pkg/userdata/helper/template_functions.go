/*
Copyright 2019 The Machine Controller Authors.

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

package helper

import (
	"regexp"
	"text/template"

	"github.com/Masterminds/sprig"
)

// TxtFuncMap returns an aggregated template function map. Currently (custom functions + sprig)
func TxtFuncMap() template.FuncMap {
	funcMap := sprig.TxtFuncMap()

	funcMap["downloadBinariesScript"] = DownloadBinariesScript
	funcMap["safeDownloadBinariesScript"] = SafeDownloadBinariesScript
	funcMap["kubeletSystemdUnit"] = KubeletSystemdUnit
	funcMap["kubeletConfiguration"] = kubeletConfiguration
	funcMap["kubeletFlags"] = KubeletFlags
	funcMap["cloudProviderFlags"] = CloudProviderFlags
	funcMap["kernelModulesScript"] = LoadKernelModulesScript
	funcMap["kernelSettings"] = KernelSettings
	funcMap["journalDConfig"] = JournalDConfig
	funcMap["kubeletHealthCheckSystemdUnit"] = KubeletHealthCheckSystemdUnit
	funcMap["containerRuntimeHealthCheckSystemdUnit"] = ContainerRuntimeHealthCheckSystemdUnit
	funcMap["dockerConfig"] = DockerConfig
	funcMap["proxyEnvironment"] = ProxyEnvironment

	return funcMap
}

// CleanupTemplateOutput postprocesses the output of the template processing. Those
// may exist due to the working of template functions like those of the sprig package
// or template condition.
func CleanupTemplateOutput(output string) (string, error) {
	// Valid YAML files are not allowed to have empty lines containing spaces or tabs.
	// So far only cleanup.
	woBlankLines := regexp.MustCompile(`(?m)^[ \t]+$`).ReplaceAllString(output, "")
	return woBlankLines, nil
}
