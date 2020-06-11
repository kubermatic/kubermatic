/*
Package ipam contains a controller responsible for assigning IP addresses from a configured
pool to machines that have an annotation keyed `machine-controller.kubermatic.io/initializers`
which contains the value ipam. After that is done, the `ipam` value is removed.

This is used for environments where no DHCP is available. The aforementioned annotation will keep
the machine-controller from reconciling the machine.
*/
package ipam
