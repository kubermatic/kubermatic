/*
The usersshkeyssynchronizer controller is responsible for synchronizing usersshkeys into
a secret in the cluster namespace. From there, the usercluster controller synchronizes them
into the usercluster and then a DaemonSet that runs on all nodes synchronizes them onto the
.ssh/authorized_keys file.
*/
package usersshkeyssynchronizer
