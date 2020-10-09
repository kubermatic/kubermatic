**Addon-Controller**

**Author**: Henrik Schmidt

**Status**: Prototype in progress

*short description of the topic e.g.*
The addon-controller is a controller written in Go to replace the existing addon-manager written in bash. It works based on a Addon CRD and offers template possibilities.


## Motivation and Background

The current approaches to deploy and manage addons within user clusters comes with several downsides:

*   One addon-manager pod per user cluster
*   Only limited to kube-system namespace
*   No templating
*   No way of deploying just a subset of all addons. All clusters always get the same manifests

The idea is to make the addon deployment more configurable.

With the new addon-controller, we achieve that:

*   Offers templating
*   A resource can be in any namespace
*   One addon-manager for all clusters
*   Thanks to `--prune`, cleanup of unused manifests

## Implementation

Addons will be installed based upon a Addon CRD which lives inside the seed-cluster. The CRD is namespaced for better organization.  For Kubermatic, Addon's belonging to a cluster resources will live inside of its `cluster-xxxxx` namespace.
Example:
```yaml
apiVersion: kubermatic.k8s.io/v1
kind: Addon
metadata:
  name: canal
  namespace: cluster-xptj6dqmvt
spec:
  # Reference to the cluster, the addon should be installed in
  cluster:
    kind: Cluster
    name: xptj6dqmvt
    uid: cda65e12-6423-11e8-a141-42010a9c00c6
  # Name of the addon to install. Needs to exist as a folder accessible by the addon-controller. Contains all manifests
  name: canal
  # Generic variables to add. Can be used in the manifest templates
  variables:
    foo: bar
```

### Workflow
1. Addon resource gets created
2. Controller executes templates & adds addon specific label to all manifests
3. Controller combines all manifests to single manifest
4. Controller executes `kubectl apply -f combined_manifest.yaml --prune -l addon=foo-bar`
5. Addon resource gets deleted
6. Step 2+3 + Controller executes `kubectl delete -f combined_manifest.yaml`

### Manifests & Templates
All addons need to have a own folder. All addon-folders need to exist in one root folder which can be specified on the controller as `--addons=`.
Every manifest needs to exist as YAML & can use templating using the go-template syntax.
For simplicity the Sprig template library will be available to all templates.

The controller will always make 3 things accessible within the templates:
- Cluster (via `.Cluster`)
- Addon (via `.Addon`)
- Variables (via `.Variables`)
