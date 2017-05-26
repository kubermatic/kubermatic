package api

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/handlers"
	"github.com/julienschmidt/httprouter"
	"github.com/kubermatic/api/pkg/bare-metal-provider/api/handler"
	"github.com/kubermatic/api/pkg/bare-metal-provider/extensions"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	"k8s.io/kubernetes/pkg/util/wait"
)

const (
	// MaxHeartbeatAge defines the maximum age of the last heartbeat
	MaxHeartbeatAge = 60
)

// Server is the server which handles the bare-metal-nodes
type Server struct {
	address  string
	client   extensions.Clientset
	router   *httprouter.Router
	authUser string
	authPass string

	nodeController    cache.Controller
	nodeStore         extensions.NodeStore
	clusterController cache.Controller
	clusterStore      extensions.ClusterStore
}

func (s *Server) initRoutes() {
	s.router.GET("/ping", func(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
		fmt.Fprint(w, "Pong")
	})
	s.router.POST("/nodes", handler.BasicAuth(s.authUser, s.authPass, handler.CreateNodeEndpoint(s.client, s.nodeStore)))
	s.router.GET("/nodes/:id", handler.BasicAuth(s.authUser, s.authPass, handler.UpdateHeartbeat(s.client, s.nodeStore, handler.GetNodeEndpoint(s.client, s.nodeStore))))
	s.router.GET("/nodes/:id/cluster", handler.BasicAuth(s.authUser, s.authPass, handler.UpdateHeartbeat(s.client, s.nodeStore, handler.GetNodeClusterEndpoint(s.nodeStore, s.clusterStore))))
	s.router.GET("/free-nodes", handler.BasicAuth(s.authUser, s.authPass, handler.GetFreeNodesEndpoint(s.nodeStore)))

	s.router.POST("/clusters", handler.BasicAuth(s.authUser, s.authPass, handler.CreateClusterEndpoint(s.client, s.clusterStore)))
	s.router.GET("/clusters/:name", handler.BasicAuth(s.authUser, s.authPass, handler.GetClusterEndpoint(s.clusterStore)))
	s.router.DELETE("/clusters/:name", handler.BasicAuth(s.authUser, s.authPass, handler.DeleteClusterEndpoint(s.client, s.nodeStore, s.clusterStore)))
	s.router.POST("/clusters/:name/nodes", handler.BasicAuth(s.authUser, s.authPass, handler.AssignNodesEndpoint(s.client, s.nodeStore, s.clusterStore)))
	s.router.GET("/clusters/:name/nodes", handler.BasicAuth(s.authUser, s.authPass, handler.GetClusterNodesEndpoint(s.nodeStore, s.clusterStore)))
	s.router.DELETE("/clusters/:name/nodes/:id", handler.BasicAuth(s.authUser, s.authPass, handler.DeleteClusterNodeEndpoint(s.client, s.nodeStore, s.clusterStore)))
}

// RemoveDeadNodes removes nodes which did not send a request to the provider within the last MaxHeartbeatAge seconds
func (s *Server) RemoveDeadNodes() {
	nodes, err := s.nodeStore.List()
	if err != nil {
		glog.Errorf("cloud not delete dead nodes. failed to retrieve nodes from store: %v", err)
		return
	}
	for _, node := range nodes {
		if node.Metadata.Labels[handler.LabelKeyNodeStatus] != handler.NodeStatusFree {
			continue
		}

		diff := time.Now().Unix() - node.LastHeartbeat
		if diff < MaxHeartbeatAge {
			continue
		}

		//Ok we have a dead node
		err := s.client.Nodes(handler.Namespace).Delete(node.Metadata.Name, &metav1.DeleteOptions{})
		if err != nil {
			glog.Errorf("failed to delete dead node %q: %v", node.Metadata.Name, err)
			continue
		}

		glog.Infof("deleted dead node %q", node.Metadata.Name)
	}
}

// Run Starts the internal http server
func (s *Server) Run() error {
	s.initRoutes()
	go wait.Until(s.RemoveDeadNodes, 10*time.Second, wait.NeverStop)
	glog.Infof("listening on %s", s.address)
	return http.ListenAndServe(s.address, handlers.LoggingHandler(os.Stderr, handlers.RecoveryHandler(handlers.PrintRecoveryStack(true))(s.router)))
}

// New returns an instance of the bare-metal-provider server
func New(address string, c extensions.Clientset, authUser string, authPass string) *Server {
	s := &Server{
		address:  address,
		client:   c,
		router:   httprouter.New(),
		authUser: authUser,
		authPass: authPass,
	}

	nodeStore, nodeController := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return s.client.Nodes(metav1.NamespaceAll).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return s.client.Nodes(metav1.NamespaceAll).Watch(options)
			},
		},
		&extensions.Node{},
		10*time.Minute,
		cache.ResourceEventHandlerFuncs{},
	)
	s.nodeController = nodeController
	s.nodeStore = extensions.NodeStore{Cache: nodeStore}

	clusterStore, clusterController := cache.NewInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return s.client.Clusters(metav1.NamespaceAll).List(options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return s.client.Clusters(metav1.NamespaceAll).Watch(options)
			},
		},
		&extensions.Cluster{},
		10*time.Minute,
		cache.ResourceEventHandlerFuncs{},
	)
	s.clusterController = clusterController
	s.clusterStore = extensions.ClusterStore{Cache: clusterStore}

	go s.nodeController.Run(wait.NeverStop)
	go s.clusterController.Run(wait.NeverStop)

	glog.Info("Waiting until controllers have synced...")
	for {
		if s.clusterController.HasSynced() && s.nodeController.HasSynced() {
			glog.Info("Controllers are synced!")
			return s
		}
		time.Sleep(100 * time.Millisecond)
	}
}
