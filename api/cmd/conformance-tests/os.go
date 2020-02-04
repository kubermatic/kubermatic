package main

import (
	apimodels "github.com/kubermatic/kubermatic/api/pkg/test/e2e/api/utils/apiclient/models"
)

func getOSNameFromSpec(spec apimodels.OperatingSystemSpec) string {
	if spec.Centos != nil {
		return "centos"
	}
	if spec.ContainerLinux != nil {
		return "coreos"
	}
	if spec.Ubuntu != nil {
		return "ubuntu"
	}
	if spec.SLES != nil {
		return "sles"
	}

	return ""
}
