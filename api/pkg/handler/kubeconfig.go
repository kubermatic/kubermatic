package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
	"k8s.io/client-go/tools/clientcmd"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func kubeconfigEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(LegacyGetClusterReq)
		user := ctx.Value(apiUserContextKey).(apiv1.User)
		clusterProvider := ctx.Value(clusterProviderContextKey).(provider.ClusterProvider)

		c, err := clusterProvider.Cluster(user, req.ClusterName)
		if err != nil {
			if kerrors.IsNotFound(err) {
				return nil, errors.NewNotFound("cluster", req.ClusterName)
			}
			return nil, err
		}
		return clusterProvider.GetAdminKubeconfig(c)
	}
}

func newGetClusterKubeconfig(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(NewGetClusterReq)
		clusterProvider := ctx.Value(newClusterProviderContextKey).(provider.NewClusterProvider)
		userInfo := ctx.Value(userInfoContextKey).(*provider.UserInfo)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		return clusterProvider.GetAdminKubeconfigForCustomerCluster(cluster)
	}
}

func encodeKubeconfig(c context.Context, w http.ResponseWriter, response interface{}) (err error) {
	cfg := response.(*clientcmdapi.Config)

	filename := "kubeconfig"

	if len(cfg.Contexts) > 0 {
		filename = fmt.Sprintf("%s-%s", filename, cfg.Contexts[cfg.CurrentContext].Cluster)
	}

	w.Header().Set("Content-Type", "application/yaml")
	w.Header().Set("Content-disposition", fmt.Sprintf("attachment; filename=%s", filename))

	b, err := clientcmd.Write(*cfg)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func newDecodeGetClusterKubeconfig(c context.Context, r *http.Request) (interface{}, error) {
	req, err := newDecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	return req, nil
}
