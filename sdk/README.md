# KKP SDK

This directory contains the `k8c.io/kubermatic/sdk/v2` Go module. If you're looking
at integrating KKP into your application, this is where you should start.

## Contents

Contained herein are

* all Go types for the KKP Kubernetes CRDs (`apis/`)
* auxiliary types for the KKP dashboard (`api/`)

## Usage

Simply `go get` the SDK to use it in your application:

```shell
go get k8c.io/kubermatic/sdk/v2
```

If necessary, you can also import the main KKP module, but this comes with heavy
dependencies that might be too costly to maintain for you:

```shell
go get k8c.io/kubermatic/v2
go get k8c.io/kubermatic/sdk/v2
```

In this case it's recommended to always keep both dependencies on the exact same
version.

## Development

There are two main design criteria for the SDK:

1. The SDK should contain a minimal set of dependencies, in a perfect world it
   would be only Kube dependencies. The idea behind the SDK is to make importing
   KKP cheap and easy and to not force dependencies onto consumers.

   Not every Kube application out there is using controller-runtime, for example.
   And it can get famously complicated to find a working set of Kube dependencies
   especially if you also include 3rd party code that still uses the old
   `v12` client-go modules and has custom rewrite directives in its `go.mod`.

1. The SDK should not contain any _functions_, only _types_. Functions always
   contain application logic and if that logic is suddenly hardcoded into KKP
   clients, KKP can never change it again. For example if all clients had
   hardcoded the `"cluster-" + cluster.Name` rule to create cluster namespaces,
   that would be an unchangeable, eternal constant.

   This is especially important since usually an application compiled with the
   KKP SDK will talk to _different_ KKP versions running in some clusters.
   Nobody wants to re-compile their client for every single KKP version out
   there.

   There are some functions that are safe and harmless to include here, but
   for example all of the old helper functions in KKP < 2.28's API (like
   ClusterReconcileWrapper) would be way to specific to include in the SDK.

### Integrating 3rd Party Code into KKP CRDs

It would be tempting to just include another app's `Spec` type directly in our
CRDs, for example in the `ClusterPolicy` objects it would have been seemingly
convenient to just directly include the `KyvernoSpec`.

However this is actually posing a problem, because we cannot guarantee that
Kyverno treats its spec exactly like KKP does.

In KKP we never bump the API version, it has been and will be for the foreseeable
future `v1`. This is possible because as a policy, we do not break out CRDs'
backwards compatibility. We take great care in ensuring new versions don't break
older versions.

However for 3rd party components, we cannot guarantee this. Who is to say that
Kyverno (just as an example) would not include a breaking change in their CRDs
at some point? Or maybe we want to upgrade from Kyverno v1 to v2, which could
naturally include API breaks. In both cases the overall KKP CRD guarantee of
not breaking compatibility is broken. Users would upgrade KKP, install the new
CRDs and then, to stay with the example, try to edit a policy in the dashboard,
only for it to not save again because the KyvernoSpec has changed.

To handle this cleanly, such integrations should make use of `RawExtension` fields
instead. This allows our controllers to unmarshal and handle version changes
dynamically. As a side effect, this also prevents costly dependencies because now,
the Kyverno dependency needs to be with the controllers, not the SDK anymore.

There are also different approaches. For example with Helm, we just need one
special ENUM, and instead of depending on Helm, we simply copied that type into
our CRDs.
