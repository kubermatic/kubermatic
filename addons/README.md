**Add-ons**

This directory contains all possible default addons.
All addons will be build into a docker container which the kubermatic-addon-controller uses to install addons.
The docker image should be freely accessible to let admins extend & modify this image for their own purpose.

### Release / Development Cycle

The tag of the addons image is automatically set to the HEAD commit (the commit referenced by `KUBERMATICCOMMIT`)
so no additional steps are necessary.

### Using in the kubermatic-addon-controller
The addons docker image will be used as a init-container to copy all addon-manifests to a shared volume.
