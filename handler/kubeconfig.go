package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ghodss/yaml"
	"github.com/go-kit/kit/endpoint"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"
	kerrors "k8s.io/client-go/1.5/pkg/api/errors"
	"k8s.io/client-go/1.5/tools/clientcmd/api/v1"
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
			if kerrors.IsNotFound(err) {
				return nil, NewInDcNotFound("cluster", req.dc, req.cluster)
			}
			return nil, err
		}

		id := fmt.Sprintf("%s-%s", req.dc, c.Metadata.Name)
		cfg := v1.Config{
			Kind:           "Config",
			APIVersion:     "v1",
			CurrentContext: id,
			Clusters: []v1.NamedCluster{v1.NamedCluster{
				Name: id,
				Cluster: v1.Cluster{
					Server: c.Address.URL,
				},
			}},
			Contexts: []v1.NamedContext{v1.NamedContext{
				Name: id,
				Context: v1.Context{
					Cluster:  id,
					AuthInfo: id,
				},
			}},
			AuthInfos: []v1.NamedAuthInfo{v1.NamedAuthInfo{
				Name: id,
				AuthInfo: v1.AuthInfo{
					Token: c.Address.Token,
				},
			}},
		}

		return &cfg, nil
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
