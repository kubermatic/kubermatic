package handler

import (
	"context"
	"regexp"

	"github.com/digitalocean/godo"
	"github.com/go-kit/kit/endpoint"
	"github.com/golang/glog"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"golang.org/x/oauth2"
)

func digitaloceanSizeEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(DoSizesReq)
		static := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: req.DoToken})
		client := godo.NewClient(oauth2.NewClient(context.Background(), static))

		listOptions := &godo.ListOptions{
			Page:    1,
			PerPage: 1000,
		}

		sizes, _, err := client.Sizes.List(ctx, listOptions)
		if err != nil {
			glog.Errorf("Sizes.List returned error: %v", err)
		}

		sizeList := apiv1.DigitaloceanSizeList{}
		sizesStandard := []apiv1.DigitaloceanSize{}
		sizesOptimized := []apiv1.DigitaloceanSize{}

		for k := range sizes {
			reStandard := regexp.MustCompile("(^s|S)")
			reOptimized := regexp.MustCompile("(^c|C)")

			if reStandard.MatchString(sizes[k].Slug) {
				sizesStandard = append(sizesStandard, apiv1.DigitaloceanSize{
					Slug:         sizes[k].Slug,
					Available:    sizes[k].Available,
					Transfer:     sizes[k].Transfer,
					PriceMonthly: sizes[k].PriceMonthly,
					PriceHourly:  sizes[k].PriceHourly,
					Memory:       sizes[k].Memory,
					Vcpus:        sizes[k].Vcpus,
					Disk:         sizes[k].Disk,
					Regions:      sizes[k].Regions,
				})
			} else if reOptimized.MatchString(sizes[k].Slug) {
				sizesOptimized = append(sizesOptimized, apiv1.DigitaloceanSize{
					Slug:         sizes[k].Slug,
					Available:    sizes[k].Available,
					Transfer:     sizes[k].Transfer,
					PriceMonthly: sizes[k].PriceMonthly,
					PriceHourly:  sizes[k].PriceHourly,
					Memory:       sizes[k].Memory,
					Vcpus:        sizes[k].Vcpus,
					Disk:         sizes[k].Disk,
					Regions:      sizes[k].Regions,
				})
			} else {
				continue
			}
		}

		sizeList.Standard = sizesStandard
		sizeList.Optimized = sizesOptimized

		return sizeList, nil
	}
}
