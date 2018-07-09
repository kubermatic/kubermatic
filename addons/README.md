**Add-ons**

This directory contains all possible default addons.
All addons will be build into a docker container which the kubermatic-addon-controller uses to install addons.
The docker image should be freely accessible to let customers extend & modify this image for their own purpose.

### Releasing

* Increment the tag in the `release.sh` script
* Execute `release.sh`
* Increment the tag in `../config/kubermatic/values.yaml`

### Using in the kubermatic-addon-controller
The addons docker image will be used as a init-container to copy all addon-manifests to a shared volume.
