package main

import (
	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
)

func getOSNameFromSpec(spec kubermaticapiv1.OperatingSystemSpec) string {
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
