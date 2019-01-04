package handler

import (
	"context"
	"fmt"
	"regexp"

	"github.com/digitalocean/godo"
	"github.com/go-kit/kit/endpoint"
	"golang.org/x/oauth2"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"
)

var reStandard = regexp.MustCompile("(^s|S)")
var reOptimized = regexp.MustCompile("(^c|C)")

func digitaloceanSizeNoCredentialsEndpoint(projectProvider provider.ProjectProvider) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DoSizesNoCredentialsReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)
		_, err := projectProvider.Get(userInfo, req.ProjectID, &provider.ProjectGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{})
		if err != nil {
			return nil, kubernetesErrorToHTTPError(err)
		}
		if cluster.Spec.Cloud.Digitalocean == nil {
			return nil, errors.NewNotFound("cloud spec for ", req.ClusterID)
		}

		doToken := cluster.Spec.Cloud.Digitalocean.Token
		return digitaloceanSize(ctx, doToken)
	}
}

func digitaloceanSizeEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DoSizesReq)
		return digitaloceanSize(ctx, req.DoToken)
	}
}

func digitaloceanSize(ctx context.Context, token string) (apiv1.DigitaloceanSizeList, error) {
	static := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := godo.NewClient(oauth2.NewClient(context.Background(), static))

	listOptions := &godo.ListOptions{
		Page:    1,
		PerPage: 1000,
	}

	sizes, _, err := client.Sizes.List(ctx, listOptions)
	if err != nil {
		return apiv1.DigitaloceanSizeList{}, fmt.Errorf("failed to list sizes: %v", err)
	}

	sizeList := apiv1.DigitaloceanSizeList{}
	// currently there are 3 types of sizes: 1) starting with s, 2) starting with c and 3) the old ones
	// type 3 isn't listed in the pricing anymore and only will be available for legacy issues until July 1st, 2018
	// therefor we might not want to log all cases that aren't starting with s or c
	for k := range sizes {
		s := apiv1.DigitaloceanSize{
			Slug:         sizes[k].Slug,
			Available:    sizes[k].Available,
			Transfer:     sizes[k].Transfer,
			PriceMonthly: sizes[k].PriceMonthly,
			PriceHourly:  sizes[k].PriceHourly,
			Memory:       sizes[k].Memory,
			VCPUs:        sizes[k].Vcpus,
			Disk:         sizes[k].Disk,
			Regions:      sizes[k].Regions,
		}
		switch {
		case reStandard.MatchString(sizes[k].Slug):
			sizeList.Standard = append(sizeList.Standard, s)
		case reOptimized.MatchString(sizes[k].Slug):
			sizeList.Optimized = append(sizeList.Optimized, s)
		}
	}

	return sizeList, nil
}
