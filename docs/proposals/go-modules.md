# Using Go Modules

**Author**: Christoph Mewes (@xrstf)

**Status**: Draft proposal; prototype in progress.

## Goals

Kubermatic uses [dep](https://github.com/golang/dep) to manage its dependencies. Since its introduction
to the codebase it has become apparent that

* dep is brittle and confusing to use sometimes,
* dep is not developed anymore and of course,
* [Go modules](https://github.com/golang/go/wiki/Modules) are the way forward.

The goal of this proposal is to remove dep from Kubermatic and instead manage dependencies using
Go modules.

## Implementation

### Module Naming

Go modules must have a name and so does Kubermatic. The [Modules specs](https://github.com/golang/go/wiki/Modules#semantic-import-versioning)
say

> If the module is version v2 or higher, the major version of the module must be included as a `/vN` at
> the end of the module paths used in `go.mod` files and in the package import path.

Given that Kubermatic is currently in version 2.14, this requires us to include a version number in
the module path. This gives us `github.com/kubermatic/kubermatic/v2` as the module name (note the missing
`/api`).

### api/

The `api` directory is a leftover from when this repository was a monorepo containing Kubermatic, the
bare metal provider and an addon manager (see PR [#385](https://github.com/kubermatic/kubermatic/pull/385)).
Now that this is no longer the case, I propose that we get rid of it. This also nicely solves the question
of whether or not, and where, to put the `api` directory in the module path: just don't.

So we would do a `mv api/* .` in the repository.

### Custom Repository

As Kubernetes does, I propose we use our `kubermatic.com` domain to abstract away GitHub and have custom
module names: `kubermatic.com/kubermatic/v2` sounds good. We will also use this for other products like
KubeOne and KubeCarrier.

As [documented](https://golang.org/cmd/go/#hdr-Remote_import_paths), we need to host a HTML snippet on
`kubermatic.com` to direct `go get` to the GitHub repository. As the HTML snippet is pretty static (does
not change with versioning, only for new majors), I propose we just keep it in our website's repository.

This is how the HTML snippet for `k8s.io/klog/v2` looks like, as an example:

```html
<html>
  <head>
    <meta name="go-import"
          content="k8s.io/klog
                   git https://github.com/kubernetes/klog">
    <meta name="go-source"
          content="k8s.io/klog
                   https://github.com/kubernetes/klog
                   https://github.com/kubernetes/klog/tree/master{/dir}
                   https://github.com/kubernetes/klog/blob/master{/dir}/{file}#L{line}">
  </head>
</html>
```

### New Major?

The [Modules specs](https://github.com/golang/go/wiki/Modules#releasing-modules-v2-or-higher) say

> Note that if you are adopting modules for the first time for a pre-existing repository or set of packages
> that have already been tagged v2.0.0 or higher before adopting modules, then the recommended best
> practice is to increment the major version when first adopting modules.

Given that Kubermatic is a product and not a library that others consume, and bumping to version 3 just
because the dependency management changed (an invisible change to the end user) sends the wrong signal.
I propose that we do not increase the major and stay on v2 (unless the implementation of this proposal
coincides with a "real" version 3).

### Release Strategy

Our branching model is not affected by the change. We can still have `release/vX.Y` branches, as long as we
keep semver-compatible version tags.

### Vendoring

Currently we use vendoring to have the repository all contained. This speeds up compile times and makes
builds more resilient against some cloud provider's network infrastructure and peering to GitHub. However
moving forward the build time speed up should be achieved with proper modcache handling instead, making
the `vendor` directory obsolete. I propose that we remove it from the repository.
