package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/kubermatic/kubermatic/bare-metal-provider/pkg/extensions"
	"k8s.io/client-go/pkg/api/v1"
)

// CreateClusterEndpoint returns the endpoint to create a cluster
func CreateClusterEndpoint(c extensions.Clientset, clusterStore extensions.ClusterStore) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var cluster *extensions.Cluster
		if err := json.NewDecoder(r.Body).Decode(&cluster); err != nil {
			glog.Errorf("Unable to decode json: %v", err)
			http.Error(w, "Unable to decode json", http.StatusBadRequest)
			return
		}
		cluster.Metadata.Name = cluster.Name

		key := fmt.Sprintf("%s/%s", Namespace, cluster.Metadata.Name)
		_, exists, err := clusterStore.GetByKey(key)
		if err != nil {
			glog.Errorf("failed to get cluster %s from clusterStore: %v", key, err)
			http.Error(w, "failed to get cluster from clusterStore", http.StatusInternalServerError)
			return
		}
		if exists {
			glog.Errorf("Cluster %q does already exist", key)
			http.Error(w, "Cluster does already exist", http.StatusBadRequest)
			return
		}

		cluster, err = c.Clusters(Namespace).Create(cluster)
		if err != nil {
			glog.Errorf("failed to create cluster %s: %v", key, err)
			http.Error(w, "failed to create cluster", http.StatusInternalServerError)
			return
		}
		glog.Infof("created cluster %s", cluster.Metadata.Name)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(&cluster)
		if err != nil {
			glog.Errorf("failed to encode cluster %s: %v", key, err)
			http.Error(w, "failed encode cluster", http.StatusInternalServerError)
			return

		}
	}
}

// GetClusterEndpoint returns the endpoint to get a cluster
func GetClusterEndpoint(clusterStore extensions.ClusterStore) httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, ps httprouter.Params) {
		name := ps.ByName("name")
		if name == "" {
			http.Error(w, "No name given", http.StatusBadRequest)
			return
		}

		key := fmt.Sprintf("%s/%s", Namespace, name)
		cluster, exist, err := clusterStore.GetByKey(key)
		if err != nil {
			glog.Errorf("failed to get cluster %s from clusterStore: %v", key, err)
			http.Error(w, "failed to get cluster from clusterStore", http.StatusInternalServerError)
			return
		}
		if !exist {
			http.Error(w, "cluster not found", http.StatusNotFound)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(&cluster)
		if err != nil {
			glog.Errorf("failed to encode cluster %s: %v", key, err)
			http.Error(w, "failed encode cluster", http.StatusInternalServerError)
			return
		}
	}
}

// DeleteClusterEndpoint Returns the endpoint to delete the cluster
func DeleteClusterEndpoint(c extensions.Clientset, nodeStore extensions.NodeStore, clusterStore extensions.ClusterStore) httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, ps httprouter.Params) {
		name := ps.ByName("name")
		if name == "" {
			http.Error(w, "No name given", http.StatusBadRequest)
			return
		}

		key := fmt.Sprintf("%s/%s", Namespace, name)
		cluster, exist, err := clusterStore.GetByKey(key)
		if err != nil {
			glog.Errorf("failed to get cluster %s from clusterStore: %v", key, err)
			http.Error(w, "failed to get cluster from clusterStore", http.StatusInternalServerError)
			return
		}
		if !exist {
			http.Error(w, "cluster not found", http.StatusNotFound)
			return
		}

		clusterNodes, err := nodeStore.GetListByLabel(LabelKeyNodeClusterName, cluster.Metadata.Name, -1)
		if err != nil {
			glog.Errorf("failed to get nodes for cluster %q: %v", key, err)
			http.Error(w, "failed to get cluster nodes", http.StatusInternalServerError)
			return
		}
		for _, node := range clusterNodes {
			node.Metadata.Labels[LabelKeyNodeClusterName] = ""
			node.Metadata.Labels[LabelKeyNodeStatus] = NodeStatusFree
			node, err = c.Nodes(Namespace).Update(node)
			if err != nil {
				glog.Errorf("failed to unassign node %s: %v", node.Metadata.Name, err)
				http.Error(w, "failed to unassign node", http.StatusInternalServerError)
				return
			}
		}

		err = c.Clusters(Namespace).Delete(cluster.Metadata.Name, &v1.DeleteOptions{})
		if err != nil {
			glog.Errorf("failed to delete cluster %s: %v", key, err)
			http.Error(w, "failed to delete cluster", http.StatusInternalServerError)
			return
		}
		glog.Infof("deleted cluster %s", cluster.Metadata.Name)

		w.WriteHeader(http.StatusOK)
	}
}

// AssignNodesEndpoint Returns the endpoint to assign nodes to a cluster
func AssignNodesEndpoint(c extensions.Clientset, nodeStore extensions.NodeStore, clusterStore extensions.ClusterStore) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		clusterName := ps.ByName("name")
		if clusterName == "" {
			http.Error(w, "No name given", http.StatusBadRequest)
			return
		}

		clusterKey := fmt.Sprintf("%s/%s", Namespace, clusterName)
		cluster, exist, err := clusterStore.GetByKey(clusterKey)
		if err != nil {
			glog.Errorf("failed to get cluster %s from clusterStore: %v", clusterKey, err)
			http.Error(w, "failed to get cluster from clusterStore", http.StatusInternalServerError)
			return
		}
		if !exist {
			http.Error(w, "cluster not found", http.StatusNotFound)
			return
		}

		var reqData struct {
			Number int `json:"number"`
		}

		if err = json.NewDecoder(r.Body).Decode(&reqData); err != nil {
			glog.Errorf("Unable to decode json: %v", err)
			http.Error(w, "Unable to decode json", http.StatusBadRequest)
			return
		}

		//Get free nodes
		freeNodes, err := nodeStore.GetListByLabel(LabelKeyNodeStatus, NodeStatusFree, reqData.Number)
		if err != nil {
			glog.Errorf("failed to get free nodes: %v", err)
			http.Error(w, "failed to get free nodes", http.StatusInternalServerError)
			return
		}
		if len(freeNodes) != reqData.Number {
			glog.Errorf("Unable to assign %d nodes to cluster %s, not enough(%d) free nodes available", reqData.Number, clusterKey, len(freeNodes))
			http.Error(w, "Not enough free nodes", http.StatusRequestedRangeNotSatisfiable)
			return
		}

		//We first set the label, to reduce the risk of concurrent assignments
		for _, node := range freeNodes {
			node.Metadata.Labels[LabelKeyNodeStatus] = NodeStatusAssigned
			node.Metadata.Labels[LabelKeyNodeClusterName] = cluster.Metadata.Name
		}

		assignedNodes := []*extensions.Node{}
		for _, node := range freeNodes {
			assignedNode, err := c.Nodes(Namespace).Update(node)
			if err != nil {
				glog.Errorf("failed to assign node %s to cluster %s: %v", node.Metadata.Name, cluster.Metadata.Name, err)
				http.Error(w, "failed to assign node to cluster", http.StatusInternalServerError)
				return
			}
			glog.Infof("assigned node %q to cluster %q", node.Metadata.Name, cluster.Metadata.Name)
			assignedNodes = append(assignedNodes, assignedNode)
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(&assignedNodes)
		if err != nil {
			glog.Errorf("failed to encode nodes list  for cluster %s: %v", clusterKey, err)
			http.Error(w, "failed encode nodes list", http.StatusInternalServerError)
			return
		}
	}
}

// GetClusterNodesEndpoint returns the endpoint to return the cluster nodes
func GetClusterNodesEndpoint(nodeStore extensions.NodeStore, clusterStore extensions.ClusterStore) httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, ps httprouter.Params) {
		name := ps.ByName("name")
		if name == "" {
			http.Error(w, "No name given", http.StatusBadRequest)
			return
		}

		clusterKey := fmt.Sprintf("%s/%s", Namespace, name)
		cluster, exist, err := clusterStore.GetByKey(clusterKey)
		if err != nil {
			glog.Errorf("failed to get cluster %s from clusterStore: %v", clusterKey, err)
			http.Error(w, "failed to get cluster from clusterStore", http.StatusInternalServerError)
			return
		}
		if !exist {
			http.Error(w, "cluster not found", http.StatusNotFound)
			return
		}

		clusterNodes, err := nodeStore.GetListByLabel(LabelKeyNodeClusterName, cluster.Metadata.Name, -1)
		if err != nil {
			glog.Errorf("failed to get nodes for cluster %q: %v", cluster.Metadata.Name, err)
			http.Error(w, "failed to get cluster nodes", http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(&clusterNodes)
		if err != nil {
			glog.Errorf("failed to encode nodes list  for cluster %s: %v", clusterKey, err)
			http.Error(w, "failed encode cluster nodes", http.StatusInternalServerError)
			return
		}
	}
}

// DeleteClusterNodeEndpoint returns the endpoint to delete a cluster node
func DeleteClusterNodeEndpoint(c extensions.Clientset, nodeStore extensions.NodeStore, clusterStore extensions.ClusterStore) httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, ps httprouter.Params) {
		name := ps.ByName("name")
		if name == "" {
			http.Error(w, "No cluster name given", http.StatusBadRequest)
			return
		}

		clusterKey := fmt.Sprintf("%s/%s", Namespace, name)
		cluster, exist, err := clusterStore.GetByKey(clusterKey)
		if err != nil {
			glog.Errorf("failed to get cluster %s from clusterStore: %v", clusterKey, err)
			http.Error(w, "failed to get cluster from clusterStore", http.StatusInternalServerError)
			return
		}
		if !exist {
			http.Error(w, "cluster not found", http.StatusNotFound)
			return
		}

		nodeID := ps.ByName("id")
		if nodeID == "" {
			http.Error(w, "No node id given", http.StatusBadRequest)
			return
		}

		nodeKey := fmt.Sprintf("%s/%s", Namespace, nodeID)
		node, exist, err := nodeStore.GetByKey(nodeKey)
		if err != nil {
			glog.Errorf("failed to get node %q: %v", nodeID, err)
			http.Error(w, "failed to get node", http.StatusInternalServerError)
			return
		}
		if !exist {
			http.Error(w, "node not found", http.StatusNotFound)
			return
		}

		err = c.Nodes(Namespace).Delete(node.Metadata.Name, v1.NewDeleteOptions(60))
		if err != nil {
			glog.Errorf("failed to delete node %s: %v", node.Metadata.Name, err)
			http.Error(w, "failed to delete node", http.StatusInternalServerError)
			return
		}
		glog.Infof("deleted node %q from cluster %q", node.Metadata.Name, cluster.Metadata.Name)

		w.WriteHeader(http.StatusOK)
	}
}
