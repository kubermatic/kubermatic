# Creating a gRPC agent component for enabling KubeOne cluster management in Kubermatic

**Authors:** Tobias Hintze, Indradhanush Gupta, Alexander Sowitzki, Manuel Stößel

**Status:** Draft proposal; prototype in progress.

**Abstract:** The gRPC agent will create a gRPC connection to the gRPC server component running in Kubermatic with a token and URL provided via a config file or the Kubermatic UI. Over that connection the agent will tunnel TCP to allow ssh connections to the node.

## Motivation and Background

To be able to provision k8s on on-premise customer infrastructure we need a simple and minimal way to pre-provision the customers infrastructure that enables Kubermatic to then provision the customer machines.

## Implementation

- Simple to configure (token + URL only?)
    - for PoC scope: token and URL are passed as flags to the agent
- Running as a service just on the machines OS
    - systemd unit
- Connects with the gRPC server running on the seed cluster via URL with the token
- Receives client TLS key pair from the gRPC server
- Establishes gRPC TLS connection to the gRPC server
- Accepts public SSH key from the gRPC server
- Inject public key into authorized_keys on the machines OS and tell Kubermatic which user id should be used for connecting to the customer machine
- Retries when connection is lost
- Allows SSH access to the node via the gRPC connection
