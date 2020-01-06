/*
Package update contains a controller that auto applies updates to both the cluster version
and the machine version based on a configuration file.

TODO: Make this controller wait for successfully convergation after an update was applied. Currently,
it may apply an update and then instantly apply another one, which is not supported, only n+1 minor
version updates are supported.
*/
package update
