/*
Package cloud contains a controller that is responsible for creating cluster-level resources
at the cloud provider, like networks, subnets or security groups. The concrete implementation
used differes based on the cloudprovider and may be a no-op.
*/
package cloud
