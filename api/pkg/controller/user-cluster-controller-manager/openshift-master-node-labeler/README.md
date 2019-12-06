# Openshift master node labeler

A simple controller that makes sure there are always three randomly selected nodes
with a `node-role.kubernetes.io/master` label on them.

This is required because the following components have label selectors for that label:

* `sdn-controller`, managed by the `openshift-network-operator`
