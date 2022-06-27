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

package main

import (
	"crypto/x509"
	"fmt"
	"net/url"
	"os"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/storeuploader"
)

type options struct {
	Endpoint string
	Secure   bool
	CABundle string

	AccessKeyID     string
	SecretAccessKey string

	Bucket       string
	CreateBucket bool
	Prefix       string
	File         string
	MaxRevisions int

	LogOptions log.Options
}

func main() {
	opt := options{
		Bucket:       "kubermatic-backups",
		File:         "/backup/snapshot.db",
		MaxRevisions: 20,
		LogOptions:   log.NewDefaultOptions(),
	}

	var (
		logger   *zap.SugaredLogger
		uploader *storeuploader.StoreUploader
	)

	rootCmd := &cobra.Command{
		Use:           "s3-storeuploader",
		Short:         "Helper tool to backup files to S3 and maintain a given number of revisions",
		Version:       "v0.2.0",
		SilenceErrors: true,
		PersistentPreRunE: func(_ *cobra.Command, _ []string) (err error) {
			if opt.AccessKeyID == "" {
				opt.AccessKeyID = os.Getenv("ACCESS_KEY_ID")
			}

			if opt.SecretAccessKey == "" {
				opt.SecretAccessKey = os.Getenv("SECRET_ACCESS_KEY")
			}

			logger = log.New(opt.LogOptions.Debug, opt.LogOptions.Format).Sugar()
			uploader, err = getUploaderFromCtx(logger, opt)
			return
		},
	}

	pFlags := rootCmd.PersistentFlags()
	pFlags.StringVarP(&opt.Endpoint, "endpoint", "e", opt.Endpoint, "S3 endpoint")
	pFlags.StringVar(&opt.AccessKeyID, "access-key-id", "", "S3 access key ID ($ACCESS_KEY_ID)")
	pFlags.StringVar(&opt.SecretAccessKey, "secret-access-key", "", "S3 secret access key ($SECRET_ACCESS_KEY)")
	pFlags.StringVarP(&opt.Bucket, "bucket", "b", opt.Bucket, "S3 bucket in which to store the snapshots")
	pFlags.StringVarP(&opt.Prefix, "prefix", "p", opt.Prefix, "Prefix to use for all objects stored in S3")
	pFlags.StringVarP(&opt.File, "file", "f", opt.File, "Path to the file to store in S3")
	pFlags.BoolVar(&opt.Secure, "secure", opt.Secure, "Enable TLS validation")
	pFlags.BoolVar(&opt.CreateBucket, "create-bucket", opt.CreateBucket, "Create the bucket if it does not exist yet")
	pFlags.IntVar(&opt.MaxRevisions, "max-revisions", opt.MaxRevisions, "Maximum number of revisions of the file to keep in S3. Older ones will be deleted")
	pFlags.StringVar(&opt.CABundle, "ca-bundle", opt.CABundle, "Filename of the CA bundle to use (if not given, default system certificates are used)")
	opt.LogOptions.AddPFlags(pFlags)

	rootCmd.AddCommand(
		&cobra.Command{
			Use:   "store",
			Short: "Stores the given file on S3",
			RunE: func(c *cobra.Command, args []string) error {
				return uploader.Store(c.Context(), opt.File, opt.Bucket, opt.Prefix, opt.CreateBucket)
			},
		},

		&cobra.Command{
			Use:   "delete-old-revisions",
			Short: "Deletes backups which are older than max-revisions",
			RunE: func(c *cobra.Command, args []string) error {
				return uploader.DeleteOldBackups(c.Context(), opt.Bucket, opt.Prefix, opt.MaxRevisions)
			},
		},

		&cobra.Command{
			Use:   "delete-all",
			Short: "Deletes all backups of the filename",
			RunE: func(c *cobra.Command, args []string) error {
				return uploader.DeleteAll(c.Context(), opt.Bucket, opt.Prefix)
			},
		},
	)

	if err := rootCmd.Execute(); err != nil {
		logger.Fatalw("Failed to run command", zap.Error(err))
	}
}

func getUploaderFromCtx(log *zap.SugaredLogger, opt options) (*storeuploader.StoreUploader, error) {
	var rootCAs *x509.CertPool

	if opt.CABundle != "" {
		bundle, err := certificates.NewCABundleFromFile(opt.CABundle)
		if err != nil {
			return nil, fmt.Errorf("cannot open CA bundle: %w", err)
		}

		rootCAs = bundle.CertPool()
	}

	// prepend the desired scheme to the endpoint so that the storeuploader
	// can detect HTTP/HTTPS
	endpoint, err := prependScheme(opt.Endpoint, opt.Secure)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint: %w", err)
	}

	uploader, err := storeuploader.New(
		endpoint,
		opt.AccessKeyID,
		opt.SecretAccessKey,
		log,
		rootCAs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create store uploader: %w", err)
	}

	return uploader, nil
}

func prependScheme(u string, secure bool) (string, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return "", err
	}

	if parsed.Scheme != "" && parsed.Host == "" {
		parsed, err = url.Parse("https://" + u) // assume endpoints with no protocol are HTTPS by default
		if err != nil {
			return "", err
		}
	}

	if secure {
		parsed.Scheme = "https"
	} else {
		parsed.Scheme = "http"
	}

	return parsed.String(), nil
}
