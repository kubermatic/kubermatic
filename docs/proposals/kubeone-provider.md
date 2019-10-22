# Enabling KubeOne cluster management in Kubermatic

**Authors:** Tobias Hintze, Indradhanush Gupta, Alexander Sowitzki, Manuel Stößel

**Status:** Draft proposal; prototype in progress.

**Abstract:* To be able to support edge compute functionality with Kubermatic there is a need to support the management of user-clusters provisioned by KubeOne within Kubermatic.

## Motivation and Background

The current approaches to deploy and manage k8s nodes with Kubermatic won’t work well with the edge computing use-case:
- Unstable network connectivity to on-premise client infrastructure could break the user-clusters because the connection between master components deployed in the seed cluster to the on-premise nodes will be unstable
- Clients willing to pay extra for on-premise management of k8s have interest in running the whole k8s infrastructure onsite for data protection/security reasons
- Innovo needs the client infrastructure setup to be independently timeable of cluster creation in Kubermatic

With the enhancements described in this proposal we want to enable:
- Independent hardware deployment on the customer site
- Provision the customer hardware with k8s via a central Kubermatic installation
- Enable possibility to update of on-premise customer cluster from a central Kubermatic installation
- Kubermatic installation can run completely separate from customers infrastructure
- Kubermatic and customer clusters can tolerate network connectivity issues between customer site and Kubermatic

## Implementation

### Kubermatic:

#### Add “KubeOne” provider:

Add a new provider to Kubermatic that will not deploy a k8s control plane in the seed cluster. The cluster CRD will be used with some sort of specification that it is a KubeOne cluster (e.g. cluster-type) and will have a status field for the health of the gRPC connection. The cluster namespace (unique per user-cluster) will be created. Kubermatic generates and stores a random string in a K8s-Secret for the gRPC server to use as the token. Kubermatic will also create a KubeOne SSH key-pair and store it in a K8s-Secret. The KubeOne configuration will be created in a ConfigMap in the cluster namespace. In addition to that the gRPC server for that cluster needs to be deployed in the cluster namespace and wait for the agent registration.

#### Start KubeOne install:
When the agent successfully establishes connection with the gRPC server (running in the user-cluster namespace) a KubeOne install will be triggered (by watching gRPC agent connection health status in cluster CRD) that uses the created KubeOne ConfigMap, the KubeOne SSH key-pair, and provisions the master node via SSH through the gRPC connection (TCP tunnel). This could be accomplished by e.g. starting a Job with a simple KubeOne install.

### UI:

#### Add “KubeOne” provider for cluster creation:

To enable the UI there needs to be a new provider added. For the POC phase there will be no changes to the region selection although for this use-case there is no need for a region. We’ll use a dummy region for now that does nothing.

#### Display gRPC agent config details:

To be able to configure the gRPC agent the credentials/config for the agent needs to be displayed after cluster creation. Details on how these infos look like still need to be defined.

### gRPC server:

- Needs to be reachable from the outside to accept connection from agent. This will be achieved by a Service of type LoadBalancer created for the gRPC server.
- Needs to expose a health endpoint that shows the status of the gRPC connection with the agent.
- Takes generated TLS certificates for itself and the agent from a k8s secret
- Takes generated SSH key pair from a k8s secret
- Accepts agent connection with token and then sends the certificates for the agent to the agent via the gRPC connection
- Copies TLS certificates to the agent
- Accepts new gRPC connection from the agent with TLS certificates
- Copies SSH public key to the agent
- Creates a TCP tunnel to the agent
