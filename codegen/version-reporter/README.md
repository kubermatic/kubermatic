# Version Reporter

This is a small, hackish tool that extracts version numbers from the KKP codebase. The idea is to be able to get a quick overview over the used/deployed pieces of software in a running KKP setup (i.e. not Go dependencies).

## Mode of Operation

Version numbers are used all over the place in KKP. Helm charts have versions, in `pkg/resources/` there are many version constants (for things like etcd's version, CoreDNS' version, etc.) and there are even more.

The version reporter can be configured with a single YAML configuration file. In this file all software products that KKP uses are listed (like CoreDNS, etc, Prometheus, AWS CCM, ...). For each product, a list of occurrences is defined. Each occurrence is one single place in the codebase where a version is notated. An occurrence can be either

* a Go constant (also supports private constants)
* a function call (only to pre-defined helper functions, see below)
* a Helm chart (either its `appVersion` or an arbitrary value from the `values.yaml`).

When the version-reporter is run, it will scan all the occurrences and print a nice, human readable report.

Note that each occurrence can not just produce a single version number, but multiple. This is used for things like the CCM versions, which depend on the user-cluster version. You would configure a single occurrence, usually a Go function call, and the version-reporter will call the function once for each supported Kubernetes _minor_ release (i.e. if `v1.27.2` and `v1.27.5` are supported, the function is called once with `v1.27.0`).

## Usage

Simply run `hack/versions-gen.sh`. Use `-json` for JSON output.

## Maintenance

When an occurrence cannot be resolved anymore, version-reporter will exit with a non-zero code, triggering a presubmit to fail. This is a sign to the developer that they somehow refactored the code in a way where the reporter is not sure what happened. If this was you, then it's now your task to simply update the `hack/versions.yaml` accordingly.

When new software products are added to KKP, there is no mechanism that forces us to remember to add it to the `versions.yaml`.
