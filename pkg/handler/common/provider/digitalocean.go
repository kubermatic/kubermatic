/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package provider

import (
	"context"
	"fmt"

	"github.com/digitalocean/godo"
	"golang.org/x/oauth2"
)

func ListDigitaloceanSizes(ctx context.Context, token string) ([]godo.Size, error) {
	client, err := getDigitalOceanClient(ctx, token)
	if err != nil {
		return nil, err
	}

	listOptions := &godo.ListOptions{
		Page:    1,
		PerPage: 1000,
	}
	godoSizes, _, err := client.Sizes.List(ctx, listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list digital ocean sizes: %w", err)
	}
	return godoSizes, nil
}

func DescribeDigitaloceanSize(ctx context.Context, token, sizeName string) (godo.Size, error) {
	godoSize := godo.Size{}
	godoSizes, err := ListDigitaloceanSizes(ctx, token)
	if err != nil {
		return godoSize, err
	}

	for _, godosize := range godoSizes {
		if godosize.Slug == sizeName {
			return godosize, nil
		}
	}
	return godoSize, fmt.Errorf("digital ocean size:%s not found", sizeName)
}

func getDigitalOceanClient(ctx context.Context, token string) (*godo.Client, error) {
	if token == "" {
		return nil, fmt.Errorf("digital ocean token cannot be empty")
	}
	static := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := godo.NewClient(oauth2.NewClient(ctx, static))
	return client, nil
}
