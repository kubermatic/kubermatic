package helper

import (
	"text/template"

	"github.com/Masterminds/sprig"
)

// TxtFuncMap returns an aggregated template function map. Currently (custom functions + sprig)
func TxtFuncMap() template.FuncMap {
	funcMap := sprig.TxtFuncMap()

	funcMap["downloadBinariesScript"] = DownloadBinariesScript
	funcMap["kubeletSystemdUnit"] = KubeletSystemdUnit
	funcMap["kubeletFlags"] = KubeletFlags
	funcMap["kernelModules"] = KernelModules
	funcMap["kernelSettings"] = KernelSettings
	funcMap["journalDConfig"] = JournalDConfig
	funcMap["dockerDaemonConfig"] = DockerDaemonConfig
	funcMap["kubeletHealthCheckSystemdUnit"] = KubeletHealthCheckSystemdUnit
	funcMap["containerRuntimeHealthCheckSystemdUnit"] = ContainerRuntimeHealthCheckSystemdUnit
	funcMap["dockerSystemdUnit"] = DockerSystemdUnit
	funcMap["containerdSystemdUnit"] = ContainerdSystemdUnit
	funcMap["startAllUnits"] = StartAllUnits

	return funcMap
}
