# master-controller-manager
The controller is designed to monitor cluster resources and it's deployed only on master nodes.

## Controllers

### rbac-generator
A controller that is wrapped in an application. The main purpose of this controller is to generate a proper set of RBAC
Roles/Bindings for projects and their resources. See also https://github.com/kubermatic/kubermatic/blob/main/docs/proposals/user-management.md
