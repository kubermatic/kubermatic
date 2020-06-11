/*
Package kubeletdnat contains the kubeletdnat controller which:

	* Is needed for all controlplane components running in the seed that need to reach nodes
	* Is not needed if reaching the pods is sufficient
	* Must be used in conjunction with the openvpn client
	* Creates NAT rules for both the public and private node IP that tunnels access to them via the VPN
	* Its counterpart runs within the openvpn client pod in the usercluster, is part of the openvpn addon and written in bash

*/
package kubeletdnat
