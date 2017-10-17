package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ghodss/yaml"
	"github.com/go-kit/kit/endpoint"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/auth"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/clientcmd/api/v1"
)

func kubeconfigEndpoint(kp provider.ClusterProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		user := auth.GetUser(ctx)
		req := request.(kubeconfigReq)

		c, err := kp.Cluster(user, req.cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, errors.NewNotFound("cluster", req.cluster)
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
