## VPN Sidecar

This is about resources for the sidecar running alongside master components in cluster-namespace to provide connectivity into the user-cluster (worker). Two things:

### OpenVPN container

This container runs an OpenVPN client to connect to the OpenVPN server running in the cluster-namespace. This provides connectivity to the service and pod network.

### KubeletDnatController container

This container runs the KubeletDnatController. This controller watches nodes in the user-cluster and creates iptable rules based on node addresses. The rules implement:

  * DNAT translation for locally originated packets (pod network namespace) from node-addresses to respective addresses in the node-access-network. On the OpenVPN-client side in the user-cluster there is another DNAT translating from these node-access-addresses back to the actual node-addresses.
  * MASQUERADING for packets leaving via the VPN tunnel

All this makes sure that nodes (kubelets) can be reached by its unmodified node-addresses via the VPN. This allows using non-public (or firewalled) IP-addresses for the workers.
