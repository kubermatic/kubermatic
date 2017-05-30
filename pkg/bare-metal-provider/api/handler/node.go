package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang/glog"
	"github.com/julienschmidt/httprouter"
	"github.com/kubermatic/api/pkg/bare-metal-provider/extensions"
)

const (
	// Namespace is the default namespace we save our ThirdPartyResources in
	Namespace = "kubermatic"

	// LabelKeyNodeStatus is the name of the label for the node status
	LabelKeyNodeStatus = "status"
	// LabelKeyNodeClusterName is the name of the label for the node cluster
	LabelKeyNodeClusterName = "cluster-name"

	// NodeStatusFree determines that the node is free
	NodeStatusFree = "free"
	// NodeStatusAssigned determines that the node is assigned to a cluster
	NodeStatusAssigned = "assigned"
)

// CreateNodeEndpoint returns the endpoint to create/register a node
func CreateNodeEndpoint(c extensions.Clientset, nodeStore extensions.NodeStore) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		var node *extensions.Node
		if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
			glog.Errorf("Unable to decode json: %v", err)
			http.Error(w, "Unable to decode json", http.StatusBadRequest)
			return
		}
		node.Metadata.Name = node.ID
		node.LastHeartbeat = time.Now().Unix()

		node.Metadata.Labels = map[string]string{
			LabelKeyNodeClusterName: "",
			LabelKeyNodeStatus:      NodeStatusFree,
		}

		key := fmt.Sprintf("%s/%s", Namespace, node.Metadata.Name)
		_, exists, err := nodeStore.GetByKey(key)
		if err != nil {
			glog.Errorf("failed to get node %s from nodeStore: %v", key, err)
			http.Error(w, "failed to get node from nodeStore", http.StatusInternalServerError)
			return
		}
		if exists {
			glog.Errorf("Node %q does already exist", key)
			http.Error(w, "Node does already exist", http.StatusBadRequest)
			return
		}

		node, err = c.Nodes(Namespace).Create(node)
		if err != nil {
			glog.Errorf("failed to create node %s: %v", key, err)
			http.Error(w, "failed to create node", http.StatusInternalServerError)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		err = json.NewEncoder(w).Encode(&node)
		if err != nil {
			glog.Errorf("failed to encode node %s: %v", node.Metadata.Name, err)
			http.Error(w, "failed to encode node ", http.StatusInternalServerError)
			return
		}
		glog.Infof("created node %q", node.Metadata.Name)
	}
}

// GetNodeEndpoint returns the endpoint to retrieve a node
func GetNodeEndpoint(c extensions.Clientset, nodeStore extensions.NodeStore) httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, ps httprouter.Params) {
		id := ps.ByName("id")
		if id == "" {
			http.Error(w, "No id given", http.StatusBadRequest)
			return
		}

		key := fmt.Sprintf("%s/%s", Namespace, id)
		node, exist, err := nodeStore.GetByKey(key)
		if err != nil {
			glog.Errorf("failed to get node %s from nodeStore: %v", key, err)
			http.Error(w, "failed to get node from nodeStore", http.StatusInternalServerError)
			return
		}
		if !exist {
			http.Error(w, "node not found", http.StatusNotFound)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(node)
		if err != nil {
			glog.Errorf("failed to encode node %s: %v", node.Metadata.Name, err)
			http.Error(w, "failed to encode node ", http.StatusInternalServerError)
			return
		}
	}
}

// GetNodeClusterEndpoint returns the endpoint to retrieve a node's cluster
func GetNodeClusterEndpoint(nodeStore extensions.NodeStore, clusterStore extensions.ClusterStore) httprouter.Handle {
	return func(w http.ResponseWriter, _ *http.Request, ps httprouter.Params) {
		id := ps.ByName("id")
		if id == "" {
			http.Error(w, "No id given", http.StatusBadRequest)
			return
		}

		nodeKey := fmt.Sprintf("%s/%s", Namespace, id)
		node, exists, err := nodeStore.GetByKey(nodeKey)
		if err != nil {
			glog.Errorf("failed to get node %s from nodeStore: %v", nodeKey, err)
			http.Error(w, "failed to get node from nodeStore", http.StatusInternalServerError)
			return
		}
		if !exists {
			glog.Errorf("Node %q does not exist", nodeKey)
			http.Error(w, "Node does not exist", http.StatusNotFound)
			return
		}

		clusterName, _ := node.Metadata.Labels[LabelKeyNodeClusterName]
		if clusterName == "" {
			http.Error(w, "Node is not assigned", http.StatusNotFound)
			return
		}

		clusterKey := fmt.Sprintf("%s/%s", Namespace, clusterName)
		cluster, exists, err := clusterStore.GetByKey(clusterKey)
		if err != nil {
			glog.Errorf("failed to get cluster %s from store: %v", clusterKey, err)
			http.Error(w, "failed to get cluster from store", http.StatusInternalServerError)
			return
		}
		if !exists {
			http.Error(w, "Node is not assigned", http.StatusNotFound)
			return
		}

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		err = json.NewEncoder(w).Encode(&cluster)
		if err != nil {
			glog.Errorf("failed to encode cluster %s: %v", cluster.Metadata.Name, err)
			http.Error(w, "failed to encode cluster ", http.StatusInternalServerError)
			return
		}
	}
}

// GetFreeNodesEndpoint returns the endpoint to retrieve all free/unassigned nodes
func GetFreeNodesEndpoint(nodeStore extensions.NodeStore) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		freeNodes, err := nodeStore.GetListByLabel(LabelKeyNodeStatus, NodeStatusFree, -1)
		if err != nil {
			glog.Errorf("failed to get free nodes: %v", err)
			http.Error(w, "failed to get free nodes", http.StatusInternalServerError)
			return
		}
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		err = json.NewEncoder(w).Encode(freeNodes)
		if err != nil {
			glog.Errorf("failed to encode node list: %v", err)
			http.Error(w, "failed to encode node list", http.StatusInternalServerError)
			return
		}
	}
}

// UpdateHeartbeat will update the last heartbeat of a node
func UpdateHeartbeat(c extensions.Clientset, nodeStore extensions.NodeStore, next httprouter.Handle) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		next(w, r, p)
		go func() {
			id := p.ByName("id")
			if id == "" {
				return
			}

			nodeKey := fmt.Sprintf("%s/%s", Namespace, id)
			node, exists, err := nodeStore.GetByKey(nodeKey)
			if err != nil {
				glog.Errorf("failed to get node %s from nodeStore: %v", nodeKey, err)
				return
			}
			if !exists {
				return
			}

			node.LastHeartbeat = time.Now().Unix()
			_, err = c.Nodes(Namespace).Update(node)
			if err != nil {
				glog.Errorf("failed update node %s: %v", node.Metadata.Name, err)
			}
		}()
	}
}
