package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	"github.com/gorilla/mux"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/extensions"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

func newClusterEndpointV2(
	kps map[string]provider.KubernetesProvider,
	dcs map[string]provider.DatacenterMeta,
	masterClientset extensions.Clientset,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(newClusterReqV2)

		if req.Cloud == nil {
			return nil, NewBadRequest("no cloud spec given")
		}

		dc, found := dcs[req.Cloud.Region]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.Cloud.Region)
		}

		kp, found := kps[dc.Seed]
		if !found {
			return nil, NewBadRequest("unknown datacenter %q", dc.Seed)
		}

		if len(req.SSHKeys) < 1 {
			return nil, NewBadRequest("please provide at least one key")
		}

		switch req.Cloud.Name {
		// Move from implementation use fingerprint / name but not both
		case provider.AWSCloudProvider:
			req.Cloud.AWS = &api.AWSCloudSpec{
				AccessKeyID:     req.Cloud.User,
				SecretAccessKey: req.Cloud.Secret,
				// TODO: More keys!
				SSHKeyName: req.SSHKeys[0],
			}
		case provider.DigitaloceanCloudProvider:
			req.Cloud.Digitalocean = &api.DigitaloceanCloudSpec{
				Token:   req.Cloud.Secret,
				SSHKeys: req.SSHKeys,
			}
		case provider.FakeCloudProvider:
			req.Cloud.Fake = &api.FakeCloudSpec{
				Token: req.Cloud.Secret,
			}
		case provider.BringYourOwnCloudProvider:
			req.Cloud.BringYourOwn = &api.BringYourOwnCloudSpec{
				PrivateIntf: req.Cloud.BringYourOwn.PrivateIntf,
			}
		}

		req.Cloud.DatacenterName = req.Cloud.Region
		c, err := kp.NewClusterWithCloud(req.user, req.Spec, req.Cloud)
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				return nil, NewConflict("cluster", req.Cloud.Region, req.Spec.HumanReadableName)
			}
			return nil, err
		}
		c.Seed = dc.Seed

		// TODO(realfake): Duplicated code move to function
		sshClient := masterClientset.SSHKeyTPR(req.user.Name)
		keys, err := sshClient.List()
		if err != nil {
			return nil, err
		}

		for _, key := range keys.Items {
			for _, sshKeyName := range req.SSHKeys {
				if sshKeyName != key.Metadata.Name {
					continue
				}
				key.Clusters = append(key.Clusters, c.Metadata.Name)
				// TODO(realfake): This takes a long time look forward to async / batch implementation
				_, err := sshClient.Update(&key)
				if err != nil {
					return nil, err
				}
			}
		}

		return c, nil
	}
}

// Deprecated at V2 of create cluster endpoint
// @TODO Remove with https://github.com/kubermatic/api/issues/220
func newClusterEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(newClusterReq)

		if req.cluster.Spec.Cloud != nil {
			return nil, NewBadRequest("new clusters cannot have a cloud assigned")
		}

		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		c, err := kp.NewCluster(req.user, &req.cluster.Spec)
		if err != nil {
			if kerrors.IsAlreadyExists(err) {
				return nil, NewConflict("cluster", req.dc, req.cluster.Spec.HumanReadableName)
			}
			return nil, err
		}

		return c, nil
	}
}

func clusterEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(clusterReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		c, err := kp.Cluster(req.user, req.cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, NewInDcNotFound("cluster", req.dc, req.cluster)
			}
			return nil, err
		}

		return c, nil
	}
}

// Deprecated at V2 of create cluster endpoint
// @TODO Remove with https://github.com/kubermatic/api/issues/220
func setCloudEndpoint(
	dcs map[string]provider.DatacenterMeta,
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(setCloudReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		if req.provider != "" && req.provider != provider.BringYourOwnCloudProvider {
			if _, found := cps[req.provider]; !found {
				return nil, fmt.Errorf("invalid cloud provider %q", req.provider)
			}

			if _, found := dcs[req.cloud.DatacenterName]; !found {
				return nil, fmt.Errorf("invalid node datacenter %q", req.cloud.DatacenterName)
			}

			// TODO(sttts): add cloud credential smoke test
		}

		c, err := kp.SetCloud(req.user, req.cluster, &req.cloud)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, NewInDcNotFound("cluster", req.dc, req.cluster)
			}
			return nil, err
		}

		return c, nil
	}
}

func clustersEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(clustersReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		cs, err := kp.Clusters(req.user)
		if err != nil {
			return nil, err
		}

		return cs, nil
	}
}

func deleteClusterEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
	masterClientset extensions.Clientset,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(deleteClusterReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		//Delete all nodes in the cluster
		c, err := kp.Cluster(req.user, req.cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, NewInDcNotFound("cluster", req.dc, req.cluster)
			}
			return nil, err
		}

		_, cp, err := provider.ClusterCloudProvider(cps, c)
		if err != nil {
			return nil, err
		}

		var deleteErrors []error
		if cp != nil {
			// This code requires the valid credentials of the
			// CloudProvider (eg. AWS, DO, ...) however, when
			// there is an error the customer cluster laying on
			// our side does not get deleted properly.
			// We will save errors and later return them after
			// the cluster on our side got deleted.
			nodes, err := cp.Nodes(ctx, c)
			if err != nil {
				err = fmt.Errorf("error getting nodes: %v", err)
				deleteErrors = append(deleteErrors, err)
			}

			for _, node := range nodes {
				err := cp.DeleteNodes(ctx, c, []string{node.Metadata.UID})
				if err != nil {
					deleteErrors = append(deleteErrors, err)
				}
			}

			err = cp.CleanUp(c)
			if err != nil {
				deleteErrors = append(deleteErrors, err)
			}
		}

		err = kp.DeleteCluster(req.user, req.cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, NewInDcNotFound("cluster", req.dc, req.cluster)
			}
			return nil, err
		}

		// TODO(realfake): Duplicated code move to function
		sshClient := masterClientset.SSHKeyTPR(req.user.Name)
		keys, err := sshClient.List()
		if err != nil {
			return nil, err
		}

		for _, key := range keys.Items {
			for i, clusterName := range key.Clusters {
				if clusterName != req.cluster {
					continue
				}
				// TODO(realfake): This takes a long time look forward to async / batch implementation
				key.Clusters = append(key.Clusters[:i], key.Clusters[i+1:]...)
				_, err := sshClient.Update(&key)
				if err != nil {
					return nil, err
				}
			}
		}

		if len(deleteErrors) > 0 {
			err = fmt.Errorf("Please manually clean up any Storage, Nodes or Load Balancers associated with %q, got errors while cleaning up: %v", req.cluster, deleteErrors)
		}

		return nil, err
	}
}

func createAddonEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createAddonRequest)

		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		addon, err := kp.CreateAddon(req.user, req.cluster, req.addonName)

		if err != nil {
			return nil, err
		}

		return addon, nil
	}
}

// Deprecated at V2 of create cluster endpoint
// @TODO Remove with https://github.com/kubermatic/api/issues/220
type newClusterReq struct {
	dcReq
	cluster api.Cluster
}

// Deprecated at V2 of create cluster endpoint
// @TODO Remove with https://github.com/kubermatic/api/issues/220
func decodeNewClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req newClusterReq

	dr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.dcReq = dr.(dcReq)

	if err := json.NewDecoder(r.Body).Decode(&req.cluster); err != nil {
		return nil, err
	}

	return req, nil
}

type newClusterReqV2 struct {
	userReq
	Cloud   *api.CloudSpec   `json:"cloud"`
	Spec    *api.ClusterSpec `json:"spec"`
	SSHKeys []string         `json:"ssh_keys"`
}

func decodeNewClusterReqV2(c context.Context, r *http.Request) (interface{}, error) {
	var req newClusterReqV2

	ur, err := decodeUserReq(r)
	if err != nil {
		return nil, err
	}
	req.userReq = ur.(userReq)

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}

	return req, nil
}

type clustersReq struct {
	dcReq
}

func decodeClustersReq(c context.Context, r *http.Request) (interface{}, error) {
	var req clustersReq

	dr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.dcReq = dr.(dcReq)

	return req, nil
}

type clusterReq struct {
	dcReq
	cluster string
}

func decodeClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req clusterReq

	dr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.dcReq = dr.(dcReq)

	req.cluster = mux.Vars(r)["cluster"]

	return req, nil
}

type setCloudReq struct {
	clusterReq
	provider string
	cloud    api.CloudSpec
}

func decodeSetCloudReq(c context.Context, r *http.Request) (interface{}, error) {
	var req setCloudReq

	cr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.clusterReq = cr.(clusterReq)

	if err = json.NewDecoder(r.Body).Decode(&req.cloud); err != nil {
		return nil, err
	}

	req.provider, err = provider.ClusterCloudProviderName(&req.cloud)
	if err != nil {
		return nil, err
	}

	if req.provider != "" && req.provider != provider.BringYourOwnCloudProvider &&
		req.cloud.DatacenterName == "" {
		return nil, errors.New("dc cannot be empty when a cloud provider is set")
	}

	return req, nil
}

type deleteClusterReq struct {
	dcReq
	cluster string
}

func decodeDeleteClusterReq(c context.Context, r *http.Request) (interface{}, error) {
	var req deleteClusterReq

	dr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.dcReq = dr.(dcReq)

	req.cluster = mux.Vars(r)["cluster"]

	return req, nil
}

type createAddonRequest struct {
	dcReq
	addonName string
	cluster   string
}

func decodeCreateAddonRequest(c context.Context, r *http.Request) (interface{}, error) {
	var req createAddonRequest

	dr, err := decodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.dcReq = dr.(dcReq)
	req.cluster = mux.Vars(r)["cluster"]

	var addon struct {
		Name string `json:"name"`
	}

	if err = json.NewDecoder(r.Body).Decode(&addon); err != nil {
		return nil, err
	}
	req.addonName = addon.Name

	return req, nil
}
