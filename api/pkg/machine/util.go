package machine

import (
	"fmt"
	"reflect"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
)

var userNameMap = map[string]string{
	"Digitalocean:Ubuntu":         "root",
	"Digitalocean:ContainerLinux": "core",
	"Digitalocean:CentOS":         "root",
	"Hetzner:Ubuntu":              "root",
	"Hetzner:CentOS":              "root",
	"Azure:Ubuntu":                "ubuntu",
	"Azure:ContainerLinux":        "core",
	"Azure:CentOS":                "centos",
	"Azure:RHEL":                  "rhel",
	"VSphere:Ubuntu":              "ubuntu",
	"VSphere:ContainerLinux":      "core",
	"VSphere:CentOS":              "centos",
	"VSphere:RHEL":                "cloud-user",
	"AWS:Ubuntu":                  "ubuntu",
	"AWS:ContainerLinux":          "core",
	"AWS:CentOS":                  "centos",
	"AWS:SLES":                    "ec2-user",
	"AWS:RHEL":                    "ec2-user",
	"Openstack:RHEL":              "cloud-user",
	"Openstack:Ubuntu":            "ubuntu",
	"Openstack:ContainerLinux":    "core",
	"Openstack:CentOS":            "centos",
	"Packet:Ubuntu":               "root",
	"Packet:ContainerLinux":       "core",
	"Packet:CentOS":               "root",
	"GCP:Ubuntu":                  "ubuntu",
	"GCP:RHEL":                    "cloud-user",
	"GCP:ContainerLinux":          "core",
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
