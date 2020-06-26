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

package restmapper

import (
	"fmt"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// New returns a new Cache
func New() *Cache {
	return &Cache{cache: &sync.Map{}}
}

// RestMapperCache is used to dynamically create controllerruntimeClients whilst caching the RestMapper. It uses properties of the
// *cfg as cache key
type Cache struct {
	cache *sync.Map
}

// Client returns a brand new controllerruntime.Client, using a cache for the restMapping to avoid doing discovery during startup.
// It uses properties of the *cfg as cache Key
func (c *Cache) Client(cfg *rest.Config) (ctrlruntimeclient.Client, error) {
	key := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s/%s", cfg.Host, cfg.APIPath, cfg.Username, cfg.Password, cfg.BearerToken, cfg.BearerTokenFile, string(cfg.CertData), string(cfg.KeyData), string(cfg.CAData))

	var mapper meta.RESTMapper

	rawMapper, exists := c.cache.Load(key)
	if !exists {
		var err error
		mapper, err = apiutil.NewDynamicRESTMapper(cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create restMapper: %v", err)
		}
		c.cache.Store(key, mapper)
	} else {
		var ok bool
		mapper, ok = rawMapper.(meta.RESTMapper)
		if !ok {
			return nil, fmt.Errorf("didn't get a restMapper from the cache")
		}
	}

	return ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{Mapper: mapper})
}
