**kkp delete Add-on**

Place yaml files here that should be cleaned up manually by the addon controller.
Usually this is not necessary as `Kubeclt apply --prune` will take care of most of the changes.

However, changes like namespace migration will not work, as the resources in the old namespace won't be cleaned up.