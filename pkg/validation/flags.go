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

package validation

import (
	"flag"
	"fmt"
	"net"
	"path/filepath"
	"strconv"

	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type WebhookOpts struct {
	listenHost string
	listenPort int
	certDir    string
	certName   string
	keyName    string
	// Deprecated fields to be removed
	ListenAddress string
	CertFile      string
	KeyFile       string
}

func (opts *WebhookOpts) AddFlags(fs *flag.FlagSet, includeDeprecatedFlags bool) {
	if includeDeprecatedFlags {
		fs.StringVar(&opts.ListenAddress, "seed-admissionwebhook-listen-address", ":8100", "The listen address for the seed amission webhook (Deprecated)")
		fs.StringVar(&opts.CertFile, "seed-admissionwebhook-cert-file", "", "The location of the certificate file (Deprecated)")
		fs.StringVar(&opts.KeyFile, "seed-admissionwebhook-key-file", "", "The location of the certificate key file (Deprecated)")
	}
	fs.StringVar(&opts.listenHost, "admissionwebhook-listen-host", "", "The listen host for the seed amission webhook")
	fs.IntVar(&opts.listenPort, "admissionwebhook-listen-port", 8100, "The listen port for the seed amission webhook")
	fs.StringVar(&opts.certDir, "admissionwebhook-cert-dir", "", "The directory containing certificate files")
	fs.StringVar(&opts.certName, "admissionwebhook-cert-name", "", "The certificate file name")
	fs.StringVar(&opts.keyName, "admissionwebhook-key-name", "", "The key file name")
}

// Configured() must be called after the Validate() function has normalized the
// deprecated flags.
func (opts *WebhookOpts) Configured() bool {
	return opts.certName != "" && opts.keyName != ""
}

func (opts *WebhookOpts) Validate() error {
	// translate deprecated flag into new structure
	if opts.ListenAddress != "" {
		host, port, err := net.SplitHostPort(opts.ListenAddress)
		if err != nil {
			return fmt.Errorf("failed to parse admission webhook listen address: %v", err)
		}

		opts.listenHost = host
		opts.listenPort, _ = strconv.Atoi(port)
		opts.ListenAddress = ""
	}

	// controller-runtime server do not support cert file and key file being in
	// different directories; this is not fully backward compatible
	if opts.CertFile != "" && opts.KeyFile != "" && filepath.Dir(opts.CertFile) != filepath.Dir(opts.KeyFile) {
		return fmt.Errorf("certificate file %q and key file %q must be located in the same directory", opts.CertFile, opts.certDir)
	}

	if opts.CertFile != "" {
		opts.certDir = filepath.Dir(opts.CertFile)
		opts.certName = filepath.Base(opts.CertFile)
		opts.CertFile = ""
	}

	if opts.KeyFile != "" {
		opts.certDir = filepath.Dir(opts.KeyFile)
		opts.keyName = filepath.Base(opts.KeyFile)
		opts.KeyFile = ""
	}

	return nil
}

func (opts *WebhookOpts) Configure(s *webhook.Server) error {
	s.CertDir = opts.certDir
	s.CertName = opts.certName
	s.KeyName = opts.keyName
	s.Host = opts.listenHost
	s.Port = opts.listenPort

	return nil
}
