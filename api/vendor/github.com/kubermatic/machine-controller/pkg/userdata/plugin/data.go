//
// Core UserData plugin.
//

package plugin

import (
	"net"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	clusterv1alpha1 "sigs.k8s.io/cluster-api/pkg/apis/cluster/v1alpha1"

	"github.com/kubermatic/machine-controller/pkg/userdata/cloud"
)

const (
	// EnvRequest names the environment variable signalling the
	// plugin which request the caller wants to have answered.
	EnvRequest = "REQUEST"

	// EnvUserDataRequest names the environment variable containing
	// the user data request.
	EnvUserDataRequest = "USER_DATA_REQUEST"
)

// UserDataRequest requests user data with the given arguments.
type UserDataRequest struct {
	MachineSpec           clusterv1alpha1.MachineSpec
	KubeConfig            *clientcmdapi.Config
	CloudConfig           cloud.ConfigProvider
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
