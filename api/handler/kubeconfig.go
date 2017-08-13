package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ghodss/yaml"
	"github.com/go-kit/kit/endpoint"
	"github.com/kubermatic/kubermatic/api/provider"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/clientcmd/api/v1"
)

func kubeconfigEndpoint(
	kps map[string]provider.KubernetesProvider,
	cps map[string]provider.CloudProvider,
) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(kubeconfigReq)

		kp, found := kps[req.dc]
		if !found {
			return nil, NewBadRequest("unknown kubernetes datacenter %q", req.dc)
		}

		c, err := kp.Cluster(req.user, req.cluster)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, NewInDcNotFound("cluster", req.dc, req.cluster)
			}
			return nil, err
		}
		cfg := c.GetKubeconfig()
		return cfg, nil
	}
}

type kubeconfigReq struct {
	clusterReq
}

func decodeKubeconfigReq(c context.Context, r *http.Request) (interface{}, error) {
	var req kubeconfigReq

	cr, err := decodeClusterReq(c, r)
	if err != nil {
		return nil, err
	}
	req.clusterReq = cr.(clusterReq)

	return req, nil
}

func encodeKubeconfig(c context.Context, w http.ResponseWriter, response interface{}) (err error) {
	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Content-disposition", "attachment; filename=kubeconfig")

	cfg := response.(*v1.Config)

	jcfg, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	ycfg, err := yaml.JSONToYAML(jcfg)
	if err != nil {
		return err
	}
	_, err = w.Write(ycfg)
	return err
}
