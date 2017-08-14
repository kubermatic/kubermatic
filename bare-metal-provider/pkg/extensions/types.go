package extensions

import (
	"fmt"

	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/meta"
	"k8s.io/client-go/pkg/api/unversioned"
	apiv1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apimachinery/announced"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

const (
	// GroupName is the group for all our extension
	GroupName string = "bare-metal.kubermatic.io"
	// Version is the version of our extensions
	Version string = "v1"

	// NodeResourceName defines the resource name on kubernetes for bare-metal-nodes
	NodeResourceName = "bmnodes"
	// ClusterResourceName defines the the resource name on kubernetes for clusters
	ClusterResourceName = "bmclusters"
)

var (
	// SchemeGroupVersion is the combination of group name and version for the kubernetes client
	SchemeGroupVersion = unversioned.GroupVersion{Group: GroupName, Version: Version}
	// SchemeBuilder provides scheme information about our extensions
	SchemeBuilder = runtime.NewSchemeBuilder(addTypes)
)

func addTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		SchemeGroupVersion,
		&api.ListOptions{},
	)
	m := map[string]runtime.Object{
		"BmNode":        &Node{},
		"BmNodeList":    &NodeList{},
		"BmCluster":     &Cluster{},
		"BmClusterList": &ClusterList{},
	}
	for k, v := range m {
		scheme.AddKnownTypeWithName(
			unversioned.GroupVersionKind{
				Group:   SchemeGroupVersion.Group,
				Version: SchemeGroupVersion.Version,
				Kind:    k,
			},
			v,
		)
	}

	return nil
}

func init() {
	if err := announced.NewGroupMetaFactory(
		&announced.GroupMetaFactoryArgs{
			GroupName:                  GroupName,
			VersionPreferenceOrder:     []string{SchemeGroupVersion.Version},
			AddInternalObjectsToScheme: SchemeBuilder.AddToScheme,
		},
		announced.VersionToSchemeFunc{
			SchemeGroupVersion.Version: SchemeBuilder.AddToScheme,
		},
	).Announce().RegisterAndEnable(); err != nil {
		panic(err)
	}
}

// Node specifies a bare metal node
type Node struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             apiv1.ObjectMeta `json:"metadata"`
	ID                   string           `json:"id"`
	CPUs                 []*CPU           `json:"cpus"`
	Memory               uint64           `json:"memory"`
	Space                uint64           `json:"space"`
	PublicIP             string           `json:"public_ip"`
	LastHeartbeat        int64            `json:"last_heartbeat"`
}

// NodeList specifies a list of nodes
type NodeList struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             unversioned.ListMeta `json:"metadata"`

	Items []Node `json:"items"`
}

// NodeStore is a cache store with filtering by label
type NodeStore struct {
	Cache cache.Store
}

// GetListByLabel returns only nodes from the cache store which have the given label
func (ns *NodeStore) GetListByLabel(key, value string, limit int) ([]*Node, error) {
	list := ns.Cache.List()
	nodes := []*Node{}
	for _, n := range list {
		if len(nodes) == limit {
			return nodes, nil
		}
		node, ok := n.(*Node)
		if !ok {
			return nodes, fmt.Errorf("entry in node cache is not of type Node instead got %T", n)
		}
		if node.Metadata.Labels[key] == value {
			nodes = append(nodes, node)
		}
	}
	return nodes, nil
}

// List returns all nodes
func (ns *NodeStore) List() ([]*Node, error) {
	list := ns.Cache.List()
	nodes := []*Node{}
	for _, n := range list {
		node, ok := n.(*Node)
		if !ok {
			return nodes, fmt.Errorf("entry in node cache is not of type Node instead got %T", n)
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

// GetByKey returns the node by the given key
func (ns *NodeStore) GetByKey(key string) (*Node, bool, error) {
	n, exists, err := ns.Cache.GetByKey(key)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get node %s from cache: %v", key, err)
	}
	if !exists {
		return nil, false, nil
	}
	node, ok := n.(*Node)
	if !ok {
		return nil, false, fmt.Errorf("key=%q from cache is not of type *Node instead got: %T", key, n)
	}
	return node, true, nil
}

// CPU Specifies a cpu of a node
type CPU struct {
	Cores     uint64  `json:"cores"`
	Frequency float64 `json:"frequency"`
}

// Cluster Specifies details of a kubernetes cluster to which the node should connect to
type Cluster struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             apiv1.ObjectMeta `json:"metadata"`
	Name                 string           `json:"name"`
	ApiserverURL         string           `json:"apiserver_url"`
	Kubeconfig           string           `json:"kubeconfig"`
	ApiserverSSHPubKey   string           `json:"apiserver_ssh_pub_key"`
}

// ClusterStore is a cache store with filtering by label
type ClusterStore struct {
	Cache cache.Store
}

// GetListByLabel returns only clusters from the cache store which have the given label
func (cs *ClusterStore) GetListByLabel(key, value string, limit int) ([]*Cluster, error) {
	list := cs.Cache.List()
	clusters := []*Cluster{}
	for _, c := range list {
		if len(clusters) == limit {
			return clusters, nil
		}
		cluster, ok := c.(*Cluster)
		if !ok {
			return clusters, fmt.Errorf("entry in cluster cache is not of type Cluster instead got %T", c)
		}
		if cluster.Metadata.Labels[key] == value {
			clusters = append(clusters, cluster)
		}
	}
	return clusters, nil
}

// GetByKey returns the node by the given key
func (cs *ClusterStore) GetByKey(key string) (*Cluster, bool, error) {
	c, exists, err := cs.Cache.GetByKey(key)
	if err != nil {
		return nil, false, fmt.Errorf("failed to get cluster %s from cache: %v", key, err)
	}
	if !exists {
		return nil, false, nil
	}
	cluster, ok := c.(*Cluster)
	if !ok {
		return nil, false, fmt.Errorf("key=%q from cache is not of type *Cluster instead got: %T", key, c)
	}
	return cluster, true, nil
}

// ClusterList specifies a list of clusters
type ClusterList struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             unversioned.ListMeta `json:"metadata"`

	Items []Cluster `json:"items"`
}

//GetObjectKind returns the object typemeta information
func (n *Node) GetObjectKind() unversioned.ObjectKind {
	return &n.TypeMeta
}

//GetObjectMeta returns the object metadata
func (n *Node) GetObjectMeta() meta.Object {
	return &n.Metadata
}

//GetObjectKind returns the object typemeta information
func (nl *NodeList) GetObjectKind() unversioned.ObjectKind {
	return &nl.TypeMeta
}

//GetListMeta returns the list object metadata
func (nl *NodeList) GetListMeta() unversioned.List {
	return &nl.Metadata
}

//GetObjectKind returns the object typemeta information
func (n *Cluster) GetObjectKind() unversioned.ObjectKind {
	return &n.TypeMeta
}

//GetObjectMeta returns the object metadata
func (n *Cluster) GetObjectMeta() meta.Object {
	return &n.Metadata
}

//GetObjectKind returns the object typemeta information
func (nl *ClusterList) GetObjectKind() unversioned.ObjectKind {
	return &nl.TypeMeta
}

//GetListMeta returns the list object metadata
func (nl *ClusterList) GetListMeta() unversioned.List {
	return &nl.Metadata
}
