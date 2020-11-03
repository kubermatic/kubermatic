# Kubermatic Operator

**Author**: Christoph (@xrstf)
**Status**: Draft proposal

This is a proposal to create a Kubermatic Operator to deploying and managing the various parts
that make up a Kubermatic setup.

## Motivation

* Managing complexer migrations via Helm is complicated and brittle.
* Helm templating is limited and can quickly lead to unreadable templates.
* Not all required setup steps can be performed with Helm, like labelling the cert-manager’s
  namespace or managing (i.e. updating) CRDs.
* Configuration is extremely fine-grained (feature creep over time) and hard to manage.
* It’s not obvious which Helm charts need to be installed (Minio vs. external storage?
  Is the nodeport-proxy needed?) and in which order.
* Testing Helm upgrades is tedious and complicated.
* The existing Kubermatic installer is barely maintained by a single person in seed-team
  and quickly grows ouf-of-sync with actual Kubermatic.

## Proposal

Let us create a Kubermatic Operator that takes care of setting up the various components
required for running Kubermatic in production. It should be configured via one or multiple
CRDs and over time replace Helm entirely.

In this first iteration we will focus on installing *Kubermatic*. The logging and monitoring
stacks are out-of-scope for now, as we need to decide if and how to migrate the existing
configuration flexibility to a 3rd party operator or something we write ourselves.

An example for a minimal starter configuration could look like this:

```yaml
apiVersion: operator.kubermatic.io/v1alpha1
kind: KubermaticConfiguration
metadata:
  name: kubermatic
spec:
  # External domain for the Kubermatic installation; additional
  # services as well as user clusters will be hosted as subdomains.
  domain: my.kubermatic.io

  # The secrets are used to pull images from private Docker repositories;
  # this is effectively a copy of the "auth" section in your ~/.docker/config.json.
  # You must configure credentials for quay.io and docker.io at the
  # very least.
  imagePullSecret: |
    {
      "auths": {
        "https://index.docker.io/v1/": {
          "auth": "[base64-encoded credentials here]",
          "email": ""
        },
        "quay.io": {
          "auth": "[base64-encoded credentials here]",
          "email": ""
        }
      }
    }

  # Dex integration
  auth:
    # these are required
    issuerClientSecret: "..."
    issuerCookieKey: "..."
    caBundle: "..."
    serviceAccountKey: "..."

    # these can be set, but would otherwise be inferred as shown
    tokenIssuer: "https://<domain>/dex"
    clientID: kubermatic
    issuerRedirectURL: "https://<domain>/api/v1/kubeconfig"
    issuerClientID: "<clientID>Issuer"

  # Feature gates are structs instead of simple booleans to make
  # extending individual features with additional fields easier.
  featureGates:
    OIDCKubeCfgEndpoint:
      enabled: true
    OpenIDAuthPlugin:
      enabled: true
    VerticalPodAutoscaler:
      enabled: true

  ui:
    config: |
      {
        "share_kubeconfig": true
      }
```

The above roughly represents the settings we currently override ourselves when deploying
our development environment, so these are the minimum viable set of flags we need to
carry over. After this, it's expected that we will over time extend the configuration to
provide the non-essentials like resource constraints, tolerations, etc.

This builds upon the new Datacenter and NodeLocation CRDs and assumes that these CRs are
created by the cluster admin.

## Goals

* Whoever changes stuff in Kubermatic should also be the one to update the installation
  procedure. Teams shall be responsible for managing their part of the deployment process.
* Have a working Kubermatic Operator for configuring the Kubermatic Stack and then think
  about Monitoring/Logging stacks.
* Get rid of all Helm charts (long term).
* Retain most of the flexibility of our charts, not necessarily using the same configuration
  structure (from the Helm values.yaml).
* Be able to opt-out of the operator managing certain pieces of the stacks (like the "pause"
  flag for clusters).
* We want to put the easy-configuration-experience into a (future) CLI that asks simple
  questions and generates one or more CRD manifests.

## Non-Goals

* Running arbitrary combinations of components (master branch of X and fearure-xyz branch
  of Y).
* Setting up DNS after the LoadBalancers have been created is not something for the operator,
  this remains something for the cluster admin to do.
* There is no need for a Big Bang release where we replace all charts with the operator
  at one point in time. Instead we will have a migration phase where both the operator and
  the charts are used.
* The operator is solely responsible for managing Kubermatic installations. It’s not a
  toolset for consulting jobs to setup pretty monitoring stacks.
