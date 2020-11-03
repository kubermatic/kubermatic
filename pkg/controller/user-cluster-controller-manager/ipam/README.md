# IP Address Management controller

Responsibilities:
* Used for customers that do not have DHCP available
* Can only be used for vsphere, as all other platforms have DHCP
* The IPAM controller gets configured with a set of subnets
* For all machines with an `machine-controller.kubermatic.io/initializers` annotation that contains the value `ipam`, it will allocate an IP address
