/*
Package nodelabeler contains a controller that ensures Nodes have various labels present at all times:

	* A `x-kubernetes.io/distribution` label with a value of `centos`, `ubuntu`, `container-linux`, `rhel` or `sles`
	* A set of labels configured on the controller via a flag that are inherited from the cluster object
*/
package nodelabeler
