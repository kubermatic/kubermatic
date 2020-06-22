/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

/*
Package kubeletdnat contains the kubeletdnat controller which:

	* Is needed for all controlplane components running in the seed that need to reach nodes
	* Is not needed if reaching the pods is sufficient
	* Must be used in conjunction with the openvpn client
	* Creates NAT rules for both the public and private node IP that tunnels access to them via the VPN
	* Its counterpart runs within the openvpn client pod in the usercluster, is part of the openvpn addon and written in bash

*/
package kubeletdnat
