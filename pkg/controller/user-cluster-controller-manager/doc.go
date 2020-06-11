/*
Package userclustercontrollermanager contains all controllers running in the usercluster controller
manager binary.

Controllers running in here:

	* Must not access master resources like userprojectbindings or usersshkeys
	* May access seed resources if they are namespace-scoped and in the cluster namespace
	* Must need to access resources that are inside the usercluster

*/
package userclustercontrollermanager
