package resources

const (
	// DistributionLabelKey is the label that gets applied.
	DistributionLabelKey = "kubermatic.io/distribution"

	// CentOSLabelValue is the value of the label for CentOS
	CentOSLabelValue = "centos"

	// UbuntuLabelValue is the value of the label for Ubuntu
	UbuntuLabelValue = "ubuntu"

	// ContainerLinuxLabelValue is the value of the label for Container Linux
	ContainerLinuxLabelValue = "container-linux"
)

// OSLabelMatchValues is a mapping between OS labels and the strings to match on in OSImage.
// Note that these are all lower case.
var OSLabelMatchValues = map[string]string{
	CentOSLabelValue:         "centos",
	UbuntuLabelValue:         "ubuntu",
	ContainerLinuxLabelValue: "container linux",
}
