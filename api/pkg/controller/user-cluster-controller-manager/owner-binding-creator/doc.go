/*
The ownerbindingcreator controller is responsible for init the API Cluster Role Bindings.
It creates Cluster Role Bindings for the all API Cluster Roles (containing label `component=userClusterRole`)
and adds subject with the user who created the cluster for the `admin` Cluster Role.
*/
package ownerbindingcreator
