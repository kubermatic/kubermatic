/*
Package usersshkeysagent contains the usersshkeysagent controller, which is deployed as a DaemonSet
on all usercluster nodes and responsible for synchronizing the `$HOME/.ssh/authorized_keys` file
for all users we know about (root, core, ubuntu, centos) and that exist with the content of a
secret.

This secret in turn is synchronized based on a secret in the seed namespace via a controller running
in the usercluster controller manager and that seed namespace secret is synchronized based on the
usersshkeys custom resources in the master cluster via a controller running in the master controller
manager.
*/
package usersshkeysagent
