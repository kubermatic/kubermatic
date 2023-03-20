# Cluster Mesh

**Author**: Rastislav Szabo (@rastislavs)

**Status**: Draft proposal

**Related Issue:** https://github.com/kubermatic/kubermatic/issues/11521

## Goals
Provide a way to interconnect services in multiple k8s clusters in an easy way using the Cilium Cluster Mesh feature.

The setup of the cluster mesh for KKP users should be as simple as selecting which KKP user clusters should form the mesh, without the need for any other knowledge on the underlying solution.

## Non-Goals
We do not focus on the networking datapath, or stitching the cross-cluster connectivity ourselves, instead we solely rely on the existing Cilium Cluster Mesh feature with all its benefits and limitations.

## Motivation and Background

The main motivation for interconnecting multiple clusters are the following use-cases:

- High availability of clusters (failover of a service to a different cluster, possibly in a different datacenter),
- Easy workload migration between clusters,
- Hybrid cloud scenarios, like providing special services by on-premises clusters to cloud clusters, or vice-versa.

This should be in the ideal case possible without the need for too complex setup on the cluster level and service level (e.g. with automatic service discovery), and in a secure manner (e.g. network policies should work across multiple clusters, traffic between the clusters can be encrypted).

All of this can be achieved by relying on the feature of the Cilium CNI called Cluster Mesh.

## Implementation

We will add a new CRD on the Seed level (`ClusterMesh`), that would define a list of clusters that are part of the mesh.

In the initial implementation it will be possible to mesh only clusters that belong to the same KKP seed cluster, to avoid flow of data from the user clusters / KKP seed clusters towards the KKP main cluster. This can be enhanced without major changes in the future, e.g. if Cilium Cluster Mesh deployment will be simplified.

The `ClusterMesh` CRD will be defined as follows:

```go
// ClusterMesh is the object representing a mesh of multiple KKP user clusters.
type ClusterMesh struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    // Spec describes the desired cluster mesh state.
    Spec ClusterMeshSpec `json:"spec,omitempty"`

    // Status contains reconciliation information for the cluster mesh.
    Status ClusterMeshStatus `json:"status,omitempty"`
}

// ClusterMeshSpec specifies the mesh of KKP user clusters.
type ClusterMeshSpec struct {
    // Clusters is a list of clusters connected in the ClusterMesh, indexed by the KKP cluster name.s
    Clusters map[string]ClusterMeshCluster `json:"clusters,omitempty"`
}

// ClusterMeshCluster contains mesh configuration for a cluster.
type ClusterMeshCluster struct {
    // +kubebuilder:validation:Enum=NodePort;LoadBalancer
    // +kubebuilder:default=NodePort
    // APIServerServiceType defines the k8s service type used to expose the clustermesh apiserver the other clusters.
    APIServerServiceType v1.ServiceType `json:"apiServerServiceType,omitempty"`
}

// ClusterMeshStatus stores status information about a cluster mesh.
type ClusterMeshStatus struct {
    // Clusters contains a map of cluster mesh status information per each cluster, indexed by the KKP cluster name.
    // +optional
    Clusters map[string]ClusterMeshClusterStatus `json:"address,omitempty"`

    // TODO: mesh status conditions etc.
}

// ClusterMeshClusterStatus contains mesh status information for a cluster.
    type ClusterMeshClusterStatus struct {
    // +kubebuilder:validation:Minimum:=1
    // +kubebuilder:validation:Maximum:=255
    // ID is an internal identifier of the cluster in the ClusterMesh as it was assigned by the KKP controller.
    ID int `json:"ID,omitempty"`

    // APIServerIPs contains a list of IPs on which the clustermesh apiserver of this cluster is exposed to the other clusters.
    // It can be a single IP if APIServerServiceType == LoadBalancer, or multiple IPs if APIServerServiceType == NodePort.
    APIServerIPs []string `json:"apiServerIPs,omitempty"`

    // APIServerPort defines the port on which the clustermesh apiserver of this cluster is exposed to the other clusters.
    APIServerPort int `json:"apiServerPort,omitempty"`
}
```

A new cluster-mesh controller on the seed level  (EE only) will make sure that:

- For each `ClusterMesh`, cluster mesh CA certificates are issued and stored in a secret in the kubermatic namespace,
- Each cluster in the `ClusterMesh` list is assigned with an unique cluster ID (1 byte) and stored in the `ClusterMesh` CR,
- For each cluster in the `ClusterMesh`, the necessary certificates are issued and saved in the cluster’s namespace,
- Each cluster’s clustermesh-apiserver external IP (LB IP or node IPs + port) will be stored in the `ClusterMesh` CR.

The CNI controller running on the seed level, which is responsible for the CNI deployment, will use the information from the `ClusterMesh` CR and secrets mentioned above to populate proper Cilium Cluster Mesh configuration in the CNI values.

For cloud providers that KKP manages security groups / firewalls, KKP will also ensure that the necessary ports - UDP 8472 (VXLAN) & TCP 4240 (HTTP health checks) - are open for each worker node in a cluster in the cluster mesh.

A validation webhook for the `ClusterMesh` will be added as well, which will ensure that the following conditions are always met:

- Each cluster can exist only in 1 cluster mesh,
- Clusters in the cluster mesh have non-overlapping pod CIDRs (while it is still necessary),
- Only clusters with CIlium CNI 1.13+ can be part of the cluster mesh,
- Only clusters from the same Seed cluster can be part of the cluster mesh (this will be for the time being a limitation of this KKP implementation).

Within KKP UI (Dashboard), it will be possible to add / remove clusters to / from cluster mesh easily. The exact design is still TBD, for example it could be:

- A new “Cluster Meshes” section under “Resources”, that would allow creating new cluster meshes and adding / removing existing clusters into / from it.
- A new “Add to Cluster Mesh” option for each cluster that could be added into an existing cluster mesh / “Remove from Cluster Mesh” for removal. 

## Alternatives Considered

Configuring the CIlium Cluster Mesh feature manually is also an option and [documented in KKP docs](https://docs.kubermatic.com/kubermatic/v2.22/tutorials-howtos/networking/cilium-cluster-mesh/), but that setup is very complex, especially for more than few clusters.

Apart from relying on Cilium for creating the Cluster Mesh, we could attempt to interconnect the clusters in some other way ourselves (e.g. by using WireGuard in some smart way). However, that would be unnecessarily complex, especially if it needs to be done in a proper fault-tolerant way.

We also considered some other solutions with similar functionality from the cloud-native ecosystem (such as [submariner.io](submariner.io), [skupper.io](skupper.io), [liqo.io](liqo.io)) but all of them have their own set of limitations and setup & usage complexity.

To conclude the alternatives, multi-cluster service meshes could be also a potential solution, but they would bring even more complexity into the solution.
