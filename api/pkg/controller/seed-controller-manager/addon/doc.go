/*
Package addon contains a controller that applies addons based on a Addon CRD. It needs
a folder per addon that contains all manifests, then adds a label to all objects and applies
the addon via `kubectl apply --purge -l $added-label`, which result in all objects that
do have the label but are not in the on-disk manifests being removed.
*/
package addon
