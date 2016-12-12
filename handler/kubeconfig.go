package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ghodss/yaml"
	"github.com/go-kit/kit/endpoint"
	"github.com/kubermatic/api"
	"github.com/kubermatic/api/provider"
	"golang.org/x/net/context"
	kerrors "k8s.io/client-go/pkg/api/errors"
	capi "k8s.io/client-go/tools/clientcmd/api"
)

func getKubeConfig(dc string, c *api.Cluster) capi.Config {
	id := fmt.Sprintf("%s-%s", dc, c.Metadata.Name)
	cfg := capi.Config{
		Kind:           "Config",
		APIVersion:     "v1",
		CurrentContext: id,
		Clusters: map[string]*capi.Cluster{
			id: {
				Server: c.Address.URL,
				CertificateAuthorityData: c.Status.RootCA.Cert,
			}},
		Contexts: map[string]*capi.Context{
			id: {
				Cluster:  id,
				AuthInfo: id,
			}},
		AuthInfos: map[string]*capi.AuthInfo{
			id: {
				Token: c.Address.Token,
			}},
	}
	cfg.CurrentContext = id

	return cfg
}

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

		cfg := getKubeConfig(req.dc, c)

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

	cfg := response.(*capi.Config)

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
