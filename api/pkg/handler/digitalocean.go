package handler

import (
	"context"
	"fmt"
	"regexp"

	"github.com/digitalocean/godo"
	"github.com/go-kit/kit/endpoint"
	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"golang.org/x/oauth2"
)

func digitaloceanSizeEndpoint() endpoint.Endpoint {
	reStandard := regexp.MustCompile("(^s|S)")
	reOptimized := regexp.MustCompile("(^c|C)")

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
			return nil, fmt.Errorf("failed to list sizes: %v", err)
		}

		sizeList := apiv1.DigitaloceanSizeList{}
		// currently there are 3 types of sizes: 1) starting with s, 2) starting with c and 3) the old ones
		// type 3 isn't listed in the pricing anymore and only will be available for legacy issues until July 1st, 2018
		// therefor we might not want to log all cases that aren't starting with s or c
		for k := range sizes {
			if reStandard.MatchString(sizes[k].Slug) {
				sizeList.Standard = append(sizeList.Standard, apiv1.DigitaloceanSize{
					Slug:         sizes[k].Slug,
					Available:    sizes[k].Available,
					Transfer:     sizes[k].Transfer,
					PriceMonthly: sizes[k].PriceMonthly,
					PriceHourly:  sizes[k].PriceHourly,
					Memory:       sizes[k].Memory,
					VCPUs:        sizes[k].Vcpus,
					Disk:         sizes[k].Disk,
					Regions:      sizes[k].Regions,
				})
				continue
			} else if reOptimized.MatchString(sizes[k].Slug) {
				sizeList.Optimized = append(sizeList.Optimized, apiv1.DigitaloceanSize{
					Slug:         sizes[k].Slug,
					Available:    sizes[k].Available,
					Transfer:     sizes[k].Transfer,
					PriceMonthly: sizes[k].PriceMonthly,
					PriceHourly:  sizes[k].PriceHourly,
					Memory:       sizes[k].Memory,
					VCPUs:        sizes[k].Vcpus,
					Disk:         sizes[k].Disk,
					Regions:      sizes[k].Regions,
				})
				continue
			}
		}

		return sizeList, nil
	}
}
