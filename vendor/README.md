# Kubermatic vendoring

This directory is maintained with the `gvt` tool with the exception of the Kubernetes dependencies.

## gvt

Install gvt by executing:

```
$ go get github.com/FiloSottile/gvt
```

## Kubernetes

To update the Kubernetes source tree, execute:

```
$ cd $GOPATH/src/github.com/kubermatic/api
$ git subtree pull --prefix vendor/k8s.io/kubernetes git@github.com:kubernetes/kubernetes [GIT_REF] --squash
```

Where `GIT_REF` refers to an upstream git reference, a git sha, tag or branch.

### Remarks
Initially the Kubernetes source tree was imported using:

`$ git subtree add --prefix vendor/k8s.io/kubernetes git@github.com:kubernetes/kubernetes v1.1.7 --squash`
