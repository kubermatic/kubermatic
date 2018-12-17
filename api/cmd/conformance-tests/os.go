package main

import (
	"github.com/kubermatic/kubermatic/api/pkg/api/v2"
)

func getOSNameFromSpec(spec v2.OperatingSystemSpec) string {
	if spec.CentOS != nil {
		return "centos"
	}
	if spec.ContainerLinux != nil {
		return "coreos"
	}
	if spec.Ubuntu != nil {
		return "ubuntu"
	}

	return ""
}
