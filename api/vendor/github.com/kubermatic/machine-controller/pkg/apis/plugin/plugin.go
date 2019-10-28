/*
Copyright 2019 The Machine Controller Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

//
// Environment and serialisation types for UserData plugins.
//

package plugin

import (
	"net"

	clusterv1alpha1 "github.com/kubermatic/machine-controller/pkg/apis/cluster/v1alpha1"

	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
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
	Kubeconfig            *clientcmdapi.Config
	CloudProviderName     string
	CloudConfig           string
	DNSIPs                []net.IP
	ExternalCloudProvider bool
	HTTPProxy             string
	NoProxy               string
	InsecureRegistries    []string
	RegistryMirrors       []string
	PauseImage            string
	HyperkubeImage        string
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
