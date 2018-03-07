package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/go-kit/kit/endpoint"
	nodesetv1alpha1 "github.com/kube-node/nodeset/pkg/nodeset/v1alpha1"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
)

const (
	// NodeDeletionWaitInterval defines how long to wait between the checks if the node has already gone
	NodeDeletionWaitInterval = 500 * time.Millisecond
)

// Copy from k8s.io/kubernetes/pkg/kubelet/apis
const (
	LabelHostname          = "kubernetes.io/hostname"
	LabelZoneFailureDomain = "failure-domain.beta.kubernetes.io/zone"
	LabelZoneRegion        = "failure-domain.beta.kubernetes.io/region"

	LabelOS   = "beta.kubernetes.io/os"
	LabelArch = "beta.kubernetes.io/arch"
)

func nodesEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(ClusterReq)
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)

		c, err := clusterProvider.Cluster(user, req.ClusterName)
		if err != nil {
			return nil, err
		}

		client, err := c.GetClient()
		if err != nil {
			return nil, err
		}

		nodes, err := client.CoreV1().Nodes().List(metav1.ListOptions{})
		if err != nil {
			return nil, err
		}
		return nodes.Items, err
	}
}

func deleteNodeEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NodeReq)
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)

		c, err := clusterProvider.Cluster(user, req.ClusterName)
		if err != nil {
			return nil, err
		}

		deleteNodeLocking := func(clientset *kubernetes.Clientset, name string) error {
			err := clientset.CoreV1().Nodes().Delete(name, &metav1.DeleteOptions{})
			if err != nil {
				return err
			}

			for {
				_, err := clientset.CoreV1().Nodes().Get(name, metav1.GetOptions{})
				if err != nil {
					if apierrors.IsNotFound(err) {
						return nil
					}

					return fmt.Errorf("failed to get nodes: %v", err)
				}
				time.Sleep(NodeDeletionWaitInterval)
			}
		}

		client, err := c.GetClient()
		if err != nil {
			return nil, fmt.Errorf("failed to get client: %v", err)
		}
		return nil, deleteNodeLocking(client, req.NodeName)
	}
}

func createNodesEndpoint(cps map[string]provider.CloudProvider, dp provider.SSHKeyProvider, versions map[string]*apiv1.MasterVersion) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(CreateNodesReq)
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)

		c, err := clusterProvider.Cluster(user, req.ClusterName)
		if err != nil {
			return nil, err
		}

		keys, err := dp.ClusterSSHKeys(user, c.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to retrieve ssh keys: %v", err)
		}
		version, found := versions[c.Spec.MasterVersion]
		if !found {
			return nil, fmt.Errorf("unknown cluster version %s", c.Spec.MasterVersion)
		}

		_, cp, err := provider.ClusterCloudProvider(cps, c)
		if err != nil {
			return nil, err
		}
		if cp == nil {
			return nil, errors.NewBadRequest("cannot create nodes without cloud sshKeyProvider")
		}

		err = cp.ValidateNodeSpec(c.Spec.Cloud, &req.Body.Spec)
		if err != nil {
			return nil, err
		}

		nclient, err := c.GetNodesetClient()
		if err != nil {
			return nil, err
		}

		nc, err := nclient.NodesetV1alpha1().NodeClasses().Get(cp.NodeClassName(&req.Body.Spec), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			nc, err = cp.CreateNodeClass(c, &req.Body.Spec, keys, version)
			if err != nil {
				return nil, err
			}
		}

		client, err := c.GetClient()
		if err != nil {
			return nil, err
		}

		for i := 1; i <= req.Body.Instances; i++ {
			n := &corev1.Node{}
			n.Name = fmt.Sprintf("kubermatic-%s-%s", c.Name, rand.String(5))
			n.Labels = map[string]string{
				LabelArch:     "amd64",
				LabelOS:       "linux",
				LabelHostname: n.Name,
			}

			n.Annotations = map[string]string{
				nodesetv1alpha1.NodeClassNameAnnotationKey: nc.Name,
			}

			n, err = client.CoreV1().Nodes().Create(n)
			if err != nil {
				return nil, err
			}
		}
		return nil, nil
	}
}
