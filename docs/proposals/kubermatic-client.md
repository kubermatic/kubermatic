# Kubermatic client

**Author**: Igor Komlew

**Status**: Proposal

*short description of the topic*
For the `kubermatic` product we need to have a command line client similar to `kubectl`

## Motivation and Background

In the moment the only possibility to create/edit/delete a user cluster is the kubermatic web-UI. The Web-UI is also the only way to get a `kubeconfig` after the user cluster is created.

The main purpose of `kubermatic-cli` is to have the same functionalities in the command line client as in the Web-UI. One motivation to have a command line client is to make an integration with other products (like CI/CD pipeline) possible.

## Implementation
For the client we can use [cobra](https://github.com/spf13/cobra). It provides an easy way to create a command line client. For a client library/SDK we can use swagger codegen. For the auth part we can either wait for the implementation of the service accounts or try to implement something like oauth client credentials flow.

## Tasks
* Validate if `dex` can be used to implement oauth client credentials flow. This way we can implement something like `kubermatic-cli login username password` and store an auth token in the way `docker` does it.
* Validate swagger code generation for the client library
* Define a list of operations for the first version of the client (like login and get list of existing  clusters)
