/*
Package openshiftmasternodelabeler contains a controller that makes sure
there is always one randomly selected nodes with a `node-role.kubernetes.io/master` label on it.

This is required because the `sdn-controller` managed by the `openshift-network-operator` has a label
selector for that label on it.
*/
package openshiftmasternodelabeler
