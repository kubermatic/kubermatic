/*
Package clusterrolelabeler contains a controller that is responsible for ensuring that the
viewer, editor and admin clusterroles have a `component: userClusterRole` label associated.

This label is used by the API to determine which clusterroles to show.
*/
package clusterrolelabeler
