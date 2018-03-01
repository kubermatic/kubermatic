package handler

import (
	"context"
	"encoding/json"
	"fmt"
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
		req := request.(ClusterReq)

		c, err := kp.Cluster(user, req.Cluster)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, errors.NewNotFound("cluster", req.Cluster)
			}
			return nil, err
		}
		cfg := c.GetKubeconfig()
		return cfg, nil
	}
}

func encodeKubeconfig(c context.Context, w http.ResponseWriter, response interface{}) (err error) {
	cfg := response.(*v1.Config)

	filename := "kubeconfig"

	if len(cfg.Clusters) > 0 {
		filename = fmt.Sprintf("%s-%s", filename, cfg.Clusters[0].Name)
	}

	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Content-disposition", fmt.Sprintf("attachment; filename=%s", filename))

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
