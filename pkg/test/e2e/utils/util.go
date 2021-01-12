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

package utils

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"time"

	"github.com/Masterminds/semver/v3"
	httptransport "github.com/go-openapi/runtime/client"

	"k8c.io/kubermatic/v2/pkg/controller/operator/common"
	"k8c.io/kubermatic/v2/pkg/test/e2e/utils/apiclient/client"

	"k8s.io/apimachinery/pkg/util/wait"
)

func NewKubermaticClient(endpointURL string) (*client.KubermaticAPI, error) {
	parsed, err := url.Parse(endpointURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %v", err)
	}

	if parsed.Host == "" || parsed.Scheme == "" {
		return nil, errors.New("Kubermatic endpoint must be scheme://host[:port]")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, errors.New("invalid scheme, must be HTTP or HTTPS")
	}

	return client.New(httptransport.New(parsed.Host, parsed.Path, []string{parsed.Scheme}), nil), nil
}

func APIEndpoint() (string, error) {
	endpoint := os.Getenv("KUBERMATIC_API_ENDPOINT")
	if len(endpoint) == 0 {
		return "", errors.New("no $KUBERMATIC_API_ENDPOINT (scheme://host:port) environment variable set")
	}

	return endpoint, nil
}

func KubernetesVersion() string {
	version := common.DefaultKubernetesVersioning.Default

	if v := os.Getenv("VERSION_TO_TEST"); v != "" {
		version = semver.MustParse(v)
	}

	return "v" + version.String()
}

// WaitFor is a convenience wrapper that makes simple, "brute force"
// waiting loops easier to write.
func WaitFor(interval time.Duration, timeout time.Duration, callback func() bool) bool {
	err := wait.PollImmediate(interval, timeout, func() (bool, error) {
		return callback(), nil
	})

	return err == nil
}
