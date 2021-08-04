# KKP 3.0 Requirements

**Author**: Christoph Mewes (@xrstf)

**Status**: Drafting

## Motivation and Background

This document describes the basic assumptions and requirements we
set for "KKP 3" (not an official name). See https://github.com/kubermatic/kubermatic/issues/7454
for more information.

## Requirements

### Allow users to work declaratively

Currently, while KKP uses Custom Resources (CRs) to store its data,
most* interactions must happen through the REST API (e.g. creating a cluster).
The CRs were never documented or considered stable, though of course
that didn't stop people from manually editing them. Which is understandable
because many things _must_ be done by editing them directly (for example
to pause a cluster).

To become even more Kubernetes-native and allow a GitOps-based workflow,
we want to allow users to always (and possibly only) interact with our
CRs directly.

TL;DR: Allow `kubectl create -f mycluster.yaml`.

### Have a stable, versioned API

Currently our CRDs are unspec'ed and not very user friendly. When we
turn them into our primary API, we need to make sure they are properly
spec'ed.

This also includes immediate feedback to the user if they try to create
a malformed resource. Such resources must be rejected immediately and not
simply cause errors later during reconciling.

TL;DR: Generate spec/schema for types automatically, do not use unspec'ed CRDs
anymore.

### Do not use `*.k8s.io` anymore

Only Kubernetes-approved APIs may use this domain, so we must move away to
our own domain.

TL;DR: Turn `kubermatic.k8s.io` into `kubermatic.io` for example.

### Keep Master-Seed-Usercluster architecture

We want to keep the architecture unchanged, but the naming is up for discussion.
