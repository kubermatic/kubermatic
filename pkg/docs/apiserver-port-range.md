#Apiserver public port

The apiserver of a client cluster is only reachable by a NodePort within the node-port range of the seed-cluster.
The range although can be limited by the `--api-server-port-range=30000-32767` flag on the kubermatic-cluster-controller.

**Just make sure the range is still withing the `--service-node-port-range` of the seed-cluster**

##GKE/AWS Setup
As nodes on cloud providers are ephemeral k8sniff can be deployed which will utilize a LoadBalancer of the cloud provider to expose the apiservers NodePort via one permanent IP.

##Bare-Metal
On bare-metal systems k8sniff must not get deployed. 
