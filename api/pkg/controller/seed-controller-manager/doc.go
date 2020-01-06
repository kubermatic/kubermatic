/*
Package seedcontrollermanager contains a package for each controller that runs within the seed
controller manager binary.

Controllers running in here:

  * Must not access master resources like userprojectbindings or usersshkeys
  * Must need to access seed resources like the cluster object or the controlplane deployments
  * Must not need to access resources within the usercluster like nodes or machines except for cleanup

*/
package seedcontrollermanager
