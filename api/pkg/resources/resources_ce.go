// +build !ee

package resources

const (
	KubermaticEdition = "Community Edition"

	// DefaultKubermaticImage defines the default Docker repository containing the Kubermatic API image.
	DefaultKubermaticImage = "quay.io/kubermatic/kubermatic"

	// DefaultDNATControllerImage defines the default Docker repository containing the DNAT controller image.
	DefaultDNATControllerImage = "quay.io/kubermatic/kubeletdnat-controller"

	// DefaultDashboardAddonImage defines the default Docker repository containing the dashboard image.
	DefaultDashboardImage = "quay.io/kubermatic/dashboard"

	// DefaultKubernetesAddonImage defines the default Docker repository containing the Kubernetes addons.
	DefaultKubernetesAddonImage = "quay.io/kubermatic/addons"

	// DefaultOpenshiftAddonImage defines the default Docker repository containing the Openshift addons.
	DefaultOpenshiftAddonImage = "quay.io/kubermatic/openshift-addons"
)
