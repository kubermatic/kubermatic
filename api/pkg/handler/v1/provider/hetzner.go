package provider

import (
	"context"
	"fmt"
	"net/http"
	"regexp"

	"github.com/go-kit/kit/endpoint"
	"github.com/hetznercloud/hcloud-go/hcloud"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/cluster"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/provider/cloud/hetzner"
	kubernetesprovider "github.com/kubermatic/kubermatic/api/pkg/provider/kubernetes"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

var reStandardSize = regexp.MustCompile("(^cx)")
var reDedicatedSize = regexp.MustCompile("(^ccx)")

func HetznerSizeWithClusterCredentialsEndpoint(projectProvider provider.ProjectProvider, privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(HetznerSizesNoCredentialsReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)

		cluster, err := cluster.GetCluster(ctx, projectProvider, privilegedProjectProvider, userInfoGetter, req.ProjectID, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		if cluster.Spec.Cloud.Hetzner == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
		}

		assertedClusterProvider, ok := clusterProvider.(*kubernetesprovider.ClusterProvider)
		if !ok {
			return nil, errors.New(http.StatusInternalServerError, "failed to assert clusterProvider")
		}

		secretKeySelector := provider.SecretKeySelectorValueFuncFactory(ctx, assertedClusterProvider.GetSeedClusterAdminRuntimeClient())
		hetznerToken, err := hetzner.GetCredentialsForCluster(cluster.Spec.Cloud, secretKeySelector)
		if err != nil {
			return nil, err
		}

		return hetznerSize(ctx, hetznerToken)
	}
}

func HetznerSizeEndpoint(presetsProvider provider.PresetProvider, userInfoGetter provider.UserInfoGetter) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(HetznerSizesReq)
		token := req.HetznerToken

		userInfo, err := userInfoGetter(ctx, "")
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		if len(req.Credential) > 0 {
			preset, err := presetsProvider.GetPreset(userInfo, req.Credential)
			if err != nil {
				return nil, errors.New(http.StatusInternalServerError, fmt.Sprintf("can not get preset %s for user %s", req.Credential, userInfo.Email))
			}
			if credentials := preset.Spec.Hetzner; credentials != nil {
				token = credentials.Token
			}
		}
		return hetznerSize(ctx, token)
	}
}

func hetznerSize(ctx context.Context, token string) (apiv1.HetznerSizeList, error) {
	client := hcloud.NewClient(hcloud.WithToken(token))

	listOptions := hcloud.ServerTypeListOpts{
		ListOpts: hcloud.ListOpts{
			Page:    1,
			PerPage: 1000,
		},
	}

	sizes, _, err := client.ServerType.List(ctx, listOptions)
	if err != nil {
		return apiv1.HetznerSizeList{}, fmt.Errorf("failed to list sizes: %v", err)
	}

	sizeList := apiv1.HetznerSizeList{}

	for _, size := range sizes {
		s := apiv1.HetznerSize{
			ID:          size.ID,
			Name:        size.Name,
			Description: size.Description,
			Cores:       size.Cores,
			Memory:      size.Memory,
			Disk:        size.Disk,
		}
		switch {
		case reStandardSize.MatchString(size.Name):
			sizeList.Standard = append(sizeList.Standard, s)
		case reDedicatedSize.MatchString(size.Name):
			sizeList.Dedicated = append(sizeList.Dedicated, s)
		}
	}

	return sizeList, nil
}

// HetznerSizesNoCredentialsReq represent a request for hetzner sizes EP
// swagger:parameters listHetznerSizesNoCredentials
type HetznerSizesNoCredentialsReq struct {
	common.GetClusterReq
}

func DecodeHetznerSizesNoCredentialsReq(c context.Context, r *http.Request) (interface{}, error) {
	var req HetznerSizesNoCredentialsReq
	cr, err := common.DecodeGetClusterReq(c, r)
	if err != nil {
		return nil, err
	}

	req.GetClusterReq = cr.(common.GetClusterReq)
	return req, nil
}

// HetznerSizesReq represent a request for hetzner sizes
type HetznerSizesReq struct {
	HetznerToken string
	Credential   string
}

func DecodeHetznerSizesReq(c context.Context, r *http.Request) (interface{}, error) {
	var req HetznerSizesReq

	req.HetznerToken = r.Header.Get("HetznerToken")
	req.Credential = r.Header.Get("Credential")
	return req, nil
}
