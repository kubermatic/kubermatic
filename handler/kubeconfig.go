package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ghodss/yaml"
	"github.com/go-kit/kit/endpoint"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"
	kerrors "k8s.io/kubernetes/pkg/api/errors"
	capi "k8s.io/kubernetes/pkg/client/unversioned/clientcmd/api"
)

func kubeconfigEndpoint(
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

		id := fmt.Sprintf("%s-%s", req.dc, c.Metadata.Name)
		cfg := capi.Config{
			Kind:           "Config",
			APIVersion:     "v1",
			CurrentContext: id,
			Clusters: map[string]*capi.Cluster{
				id: &capi.Cluster{
					Server: c.Address.URL,
				},
			},
			Contexts: map[string]*capi.Context{
				id: &capi.Context{
					Cluster:  id,
					AuthInfo: id,
				},
			},
			AuthInfos: map[string]*capi.AuthInfo{
				id: &capi.AuthInfo{
					Token: c.Address.Token,
				},
			},
		}

		return &cfg, nil
	}
}

type kubeconfigReq struct {
	clusterReq
}

func decodeKubeconfigReq(r *http.Request) (interface{}, error) {
	var req kubeconfigReq

	cr, err := decodeClusterReq(r)
	if err != nil {
		return nil, err
	}
	req.clusterReq = cr.(clusterReq)

	return req, nil
}

func encodeKubeconfig(w http.ResponseWriter, response interface{}) (err error) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Content-disposition", "attachment; filename=kubeconfig")

	cfg := response.(*capi.Config)

	jcfg, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	ycfg, err := yaml.JSONToYAML(jcfg)
	_, err = w.Write(ycfg)
	return err
}
