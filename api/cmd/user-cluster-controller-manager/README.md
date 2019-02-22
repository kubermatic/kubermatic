# user-cluster-controller-manager

The controller is designed to monitor user cluster resources. It watches the shared state of the cluster through the 
apiserver and makes changes attempting to move the current state towards the desired state.

## Controllers

### RBAC user cluster controller
Enables automation of RBAC configuration. It's responsible for creating Cluster Roles for `machinedeployments` and
`machines` resources and ClusterRoleBindings for groups: `owners`, `editors` and `viewers`. This essentially gives the
`Groups` certain permissions to the `Resources`. In the future, it's going to support other types of resources.