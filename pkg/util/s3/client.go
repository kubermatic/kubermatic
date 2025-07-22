/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package s3

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var customTransport = &http.Transport{
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
	DisableCompression:    true,
}

func NewClient(endpoint, accessKeyID, secretKey string, caBundle *x509.CertPool) (*minio.Client, error) {
	secure := true

	if strings.HasPrefix(endpoint, "https://") {
		endpoint = strings.Replace(endpoint, "https://", "", 1)
	} else if strings.HasPrefix(endpoint, "http://") {
		endpoint = strings.Replace(endpoint, "http://", "", 1)
		secure = false
	}

	options := &minio.Options{
		Creds:     credentials.NewStaticV4(accessKeyID, secretKey, ""),
		Secure:    secure,
		Transport: http.DefaultTransport,
	}

	if secure {
		systemCerts, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("unable to get system ca certificate pool: %w", err)
		}
		if !caBundle.Equal(systemCerts) {
			customTransport.TLSClientConfig = &tls.Config{RootCAs: caBundle}
		}
		options.Transport = customTransport
	}

	return minio.New(endpoint, options)
}
