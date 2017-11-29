#!/bin/bash

# We replaced flannel with canal & we need to delete everything flannel related before proceeding
/etc/kubermatic/migrations/replace-flannel-with-canal.sh
# We changed a annotation key for the node controller
/etc/kubermatic/migrations/fix-node-label.sh
# We switched from externalName for NodePort, so we need to delete the service so the apiserver recreates it
/etc/kubermatic/migrations/delete-kubernetes-service.sh
