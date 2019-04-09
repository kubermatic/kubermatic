//
// Environment and serialisation types for UserData plugins.
//

package plugin

import (
	"net"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"
)

const (
	// EnvUserDataRequest names the environment variable containing
	// the user data request.
	EnvUserDataRequest = "MACHINE_CONTROLLER_USER_DATA_REQUEST"

	// EnvPluginDir names the environment variable containing
	// a user defined location of the plugins.
	EnvPluginDir = "MACHINE_CONTROLLER_USERDATA_PLUGIN_DIR"
)

// UserDataRequest requests user data with the given arguments.
type UserDataRequest struct {
	MachineSpec           clusterv1alpha1.MachineSpec
	KubeConfig            *clientcmdapi.Config
	CloudProviderName     string
	CloudConfig           string
	DNSIPs                []net.IP
	ExternalCloudProvider bool
}

// UserDataResponse contains the responded user data.
type UserDataResponse struct {
	UserData string
	Err      string
}

// ErrorResponse contains a single responded error.
type ErrorResponse struct {
	Err string
}
