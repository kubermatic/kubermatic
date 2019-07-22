package machine

import (
	"fmt"
	"reflect"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
)

var userNameMap = map[string]string{
	"Digitalocean:Ubuntu":         "",
	"Digitalocean:ContainerLinux": "",
	"Digitalocean:CentOS":         "",
	"Hetzner:Ubuntu":              "",
	"Hetzner:ContainerLinux":      "",
	"Hetzner:CentOS":              "",
	"Azure:Ubuntu":                "",
	"Azure:ContainerLinux":        "",
	"Azure:CentOS":                "",
	"VSphere:Ubuntu":              "",
	"VSphere:ContainerLinux":      "",
	"VSphere:CentOS":              "",
	"AWS:Ubuntu":                  "",
	"AWS:ContainerLinux":          "",
	"AWS:CentOS":                  "",
	"Openstack:Ubuntu":            "",
	"Openstack:ContainerLinux":    "",
	"Openstack:CentOS":            "",
	"Packet:Ubuntu":               "",
	"Packet:ContainerLinux":       "",
	"Packet:CentOS":               "",
	"GCP:Ubuntu":                  "",
	"GCP:ContainerLinux":          "",
	"GCP:CentOS":                  "",
}

// GetSSHUserName returns SSH login name for the provider and distribution
func GetSSHUserName(distribution *apiv1.OperatingSystemSpec, cloudProvider *apiv1.NodeCloudSpec) (string, error) {

	distributionName, err := getDistributionName(distribution)
	if err != nil {
		return "", err
	}

	providerName, err := getProviderName(cloudProvider)
	if err != nil {
		return "", err
	}

	loginName, ok := userNameMap[fmt.Sprintf("%s:%s", providerName, distributionName)]

	if ok {
		return loginName, nil
	}

	return "unknown", nil
}

func getDistributionName(distribution *apiv1.OperatingSystemSpec) (string, error) {
	val := reflect.ValueOf(distribution).Elem()

	for i := 0; i < val.NumField(); i++ {
		if !val.Field(i).IsNil() {
			return val.Type().Field(i).Name, nil
		}
	}

	return "", fmt.Errorf("no operating system set")
}

func getProviderName(cloudProvider *apiv1.NodeCloudSpec) (string, error) {
	val := reflect.ValueOf(cloudProvider).Elem()

	for i := 0; i < val.NumField(); i++ {
		if !val.Field(i).IsNil() {
			return val.Type().Field(i).Name, nil
		}
	}

	return "", fmt.Errorf("no cloud provider set")
}
