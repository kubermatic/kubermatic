## Addons

This directory contains all possible default addons.

All addons will be built into a container image which the addon-controller (in the
seed-controller-manager) uses to install addons. The container image should be freely accessible to
let admins extend & modify it for their own purpose. The Dockerfile will ignore auxiliary files like
Makefiles or Kustomizations, as they serve no purpose to KKP at runtime.

### Release / Development Cycle

The tag of the image is automatically set to the HEAD commit (the commit referenced by
`KUBERMATICCOMMIT`), so no additional steps are necessary.

### Usage in the addon-controller

The `addons` container image will be used as an init-container to copy all addon manifests to a
shared volume.
