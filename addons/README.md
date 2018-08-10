**Add-ons**

This directory contains all possible default addons.
All addons will be build into a docker container which the kubermatic-addon-controller uses to install addons.
The docker image should be freely accessible to let customers extend & modify this image for their own purpose.

### Release / Development Cycle

1) Change the tag in the `release.sh` script to the next version + `-dev` suffix
2) Do your development and testing. Feel free to `./release.sh` the image with the `-dev` as you need it.
3) When finishing your PR for approval, change the tag-suffix to `-rc` and `./release.sh` so that CI pipeline will refetch it.
4) Get your PR approval.
5) Change the tag to the final version without suffix.
6) Increment the tag in `../config/kubermatic/values.yaml`
7) `./release.sh`, update PR, get re-approval, merge.

If your `-rc` image needs yet another change, add indices as well. Of course you might need step 6) also during your development cycle.

### Using in the kubermatic-addon-controller
The addons docker image will be used as a init-container to copy all addon-manifests to a shared volume.
