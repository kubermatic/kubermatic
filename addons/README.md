**Add-ons**

This directory contains all possible default addons.
All addons will be build into a docker container which the kubermatic-addon-controller uses to install addons.
The docker image should be freely accessible to let customers extend & modify this image for their own purpose.

### Releasing

```bash
export TAG=v0.0.1
docker build -t quay.io/kubermatic/addons:${TAG} .
docker push quay.io/kubermatic/addons:${TAG}
```

### Using in the kubermatic-addon-controller
The addons docker image will be used as a init-container to copy all addon-manifests to a shared volume.
