# Kubernetes NodePort-Proxy

Controller which exposes NodePorts via a LoadBalancer service through [Envoy](https://envoyproxy.io).

## Overview
The NodePort-Proxy watches services with the annotation `nodeport-proxy.k8s.io/expose="true"` and exposes all pods via a single `LoadBalancer` service.

## Release

```bash
TAG=v2.0.0 make docker
```
