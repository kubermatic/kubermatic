/*
Package mastercontrollermanager contains all controllers that run within the master-controller-manager
binary.

Controllers that run in here:

  * Must need to access resources in the master cluster like userprojectbindings or usersshkeys
  * May need to access resources in seed clusters like clusters or secrets
  * Must consider that that master cluster may or may not be a seed as well
  * Must not access anything that is inside the usercluster like nodes or machines

*/
package mastercontrollermanager
