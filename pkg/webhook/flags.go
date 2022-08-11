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

package webhook

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

type Options struct {
	listenHost string
	listenPort int
	certDir    string
	certName   string
	keyName    string
}

func (opts *Options) AddFlags(fs *flag.FlagSet, prefix string) {
	fs.StringVar(&opts.listenHost, fmt.Sprintf("%s-listen-host", prefix), "", "The listen host for the admission/mutation webhooks.")
	fs.IntVar(&opts.listenPort, fmt.Sprintf("%s-listen-port", prefix), 9443, "The listen port for the admission/mutation webhooks.")
	fs.StringVar(&opts.certDir, fmt.Sprintf("%s-cert-dir", prefix), "", "The directory containing the webhook serving certificate files.")
	fs.StringVar(&opts.certName, fmt.Sprintf("%s-cert-name", prefix), "", "The certificate file name.")
	fs.StringVar(&opts.keyName, fmt.Sprintf("%s-key-name", prefix), "", "The key file name.")
}

func checkValidFile(directory, filename string) error {
	if filename == "" {
		return errors.New("no filename configured")
	}

	fullPath := filepath.Join(directory, filename)

	stat, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("failed to stat %q: %w", fullPath, err)
	}

	if stat.IsDir() {
		return fmt.Errorf("%q is not a file", fullPath)
	}

	return nil
}

func (opts *Options) Validate() error {
	if opts.certDir == "" {
		return errors.New("no -webhook-cert-dir configured")
	}

	stat, err := os.Stat(opts.certDir)
	if err != nil {
		return fmt.Errorf("%q is not a valid directory: %w", opts.certDir, err)
	}

	if !stat.IsDir() {
		return fmt.Errorf("%q is not a directory", opts.certDir)
	}

	if err := checkValidFile(opts.certDir, opts.certName); err != nil {
		return fmt.Errorf("invalid certificate file: %w", err)
	}

	if err := checkValidFile(opts.certDir, opts.keyName); err != nil {
		return fmt.Errorf("invalid private key file: %w", err)
	}

	return nil
}

func (opts *Options) Configure(s *webhook.Server) error {
	s.CertDir = opts.certDir
	s.CertName = opts.certName
	s.KeyName = opts.keyName
	s.Host = opts.listenHost
	s.Port = opts.listenPort

	return nil
}
