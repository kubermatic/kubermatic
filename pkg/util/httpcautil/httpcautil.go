/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package httpcautil

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

const defaultClientTimeout = 15 * time.Second

var (
	// CABundle is set globally once by the main() function
	// and is used to overwrite the default set of CA certificates
	// loaded from the host system/pod
	CABundle *x509.CertPool
)

// SetCABundleFile reads a PEM-encoded file and replaces the current
// global CABundle with a new one. The file must contain at least one
// valid certificate.
func SetCABundleFile(filename string) error {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	CABundle = x509.NewCertPool()
	if !CABundle.AppendCertsFromPEM(content) {
		return errors.New("file does not contain valid PEM-encoded certificates")
	}

	return nil
}

type HTTPClientConfig struct {
	// Global timeout used by the client
	Timeout time.Duration
}

// New return a custom HTTP client that uses the global CA bundle
func (c HTTPClientConfig) New() http.Client {
	timeout := c.Timeout
	// Enforce a global timeout
	if timeout <= 0 {
		timeout = defaultClientTimeout
	}

	return http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: CABundle,
			},
		},
		Timeout: timeout,
	}
}
