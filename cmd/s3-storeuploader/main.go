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
	"os"

	"github.com/urfave/cli"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/resources/certificates"
	"k8c.io/kubermatic/v2/pkg/storeuploader"
)

var logger *zap.SugaredLogger

func main() {
	app := cli.NewApp()
	app.Name = "S3 storer"
	app.Usage = ""
	app.Version = "v0.1.6"
	app.Description = "Helper tool to backup files to S3 and maintain a given number of revisions"

	endpointFlag := cli.StringFlag{
		Name:  "endpoint, e",
		Value: "",
		Usage: "S3 endpoint",
	}
	accessKeyIDFlag := cli.StringFlag{
		Name:   "access-key-id",
		Value:  "",
		EnvVar: "ACCESS_KEY_ID",
		Usage:  "S3 AccessKeyID",
	}
	secretAccessKeyFlag := cli.StringFlag{
		Name:   "secret-access-key",
		Value:  "",
		EnvVar: "SECRET_ACCESS_KEY",
		Usage:  "S3 SecretAccessKey",
	}
	bucketFlag := cli.StringFlag{
		Name:  "bucket, b",
		Value: "kubermatic-backups",
		Usage: "S3 bucket in which to store the snapshots",
	}
	prefixFlag := cli.StringFlag{
		Name:  "prefix, p",
		Value: "",
		Usage: "Prefix to use for all objects stored in S3",
	}
	fileFlag := cli.StringFlag{
		Name:  "file, f",
		Value: "/backup/snapshot.db",
		Usage: "Path to the file to store in S3",
	}
	secureFlag := cli.BoolFlag{
		Name:  "secure",
		Usage: "Enable tls validation",
	}
	caBundleFlag := cli.StringFlag{
		Name:  "ca-bundle",
		Usage: "Filename of the CA bundle to use (if not given, default system certificates are used)",
	}
	createBucketFlag := cli.BoolFlag{
		Name:  "create-bucket",
		Usage: "creates the bucket if it does not exist yet",
	}
	maxRevisionsFlag := cli.IntFlag{
		Name:  "max-revisions",
		Value: 20,
		Usage: "Maximum number of revisions of the file to keep in S3. Older ones will be deleted",
	}

	logDebugFlag := cli.BoolFlag{
		Name:  "log-debug",
		Usage: "Enables more verbose logging",
	}

	defaultLogFormat := log.FormatJSON
	logFormatFlag := cli.GenericFlag{
		Name:  "log-format",
		Value: &defaultLogFormat,
		Usage: fmt.Sprintf("Use one of [%v] to change the log output", log.AvailableFormats),
	}

	app.Flags = []cli.Flag{
		logDebugFlag,
		logFormatFlag,
	}

	app.Commands = []cli.Command{
		{
			Name:   "store",
			Usage:  "Stores the given file on S3",
			Action: store,
			Flags: []cli.Flag{
				endpointFlag,
				secureFlag,
				caBundleFlag,
				accessKeyIDFlag,
				secretAccessKeyFlag,
				bucketFlag,
				prefixFlag,
				fileFlag,
				createBucketFlag,
			},
		},
		{
			Name:   "delete-old-revisions",
			Usage:  "Deletes backups which are older than max-revisions",
			Action: deleteOldRevisions,
			Flags: []cli.Flag{
				endpointFlag,
				secureFlag,
				caBundleFlag,
				accessKeyIDFlag,
				secretAccessKeyFlag,
				bucketFlag,
				prefixFlag,
				maxRevisionsFlag,
				fileFlag, // unused but kept for BC compatibility with old cleanup scripts
			},
		},
		{
			Name:   "delete-all",
			Usage:  "deletes all backups of the filename",
			Action: deleteAll,
			Flags: []cli.Flag{
				endpointFlag,
				secureFlag,
				caBundleFlag,
				accessKeyIDFlag,
				secretAccessKeyFlag,
				bucketFlag,
				prefixFlag,
			},
		},
	}

	// setup logging
	app.Before = func(c *cli.Context) error {
		format := c.GlobalGeneric("log-format").(*log.Format)
		rawLog := log.New(c.GlobalBool("log-debug"), *format)
		logger = rawLog.Sugar()

		return nil
	}

	defer func() {
		if logger != nil {
			if err := logger.Sync(); err != nil {
				fmt.Println(err)
			}
		}
	}()

	err := app.Run(os.Args)
	// Only log failures when the logger has been setup, otherwise
	// we know it's been a CLI parsing failure and the cli package
	// has already output the error and printed the usage hints.
	if err != nil && logger != nil {
		logger.Fatalw("Failed to run command", zap.Error(err))
	}
}

func getUploaderFromCtx(c *cli.Context) (*storeuploader.StoreUploader, error) {
	var rootCAs *x509.CertPool

	if caBundleFile := c.String("ca-bundle"); caBundleFile != "" {
		bundle, err := certificates.NewCABundleFromFile(caBundleFile)
		if err != nil {
			return nil, fmt.Errorf("cannot open CA bundle: %v", err)
		}

		rootCAs = bundle.CertPool()
	}

	uploader, err := storeuploader.New(
		c.String("endpoint"),
		c.Bool("secure"),
		c.String("access-key-id"),
		c.String("secret-access-key"),
		logger,
		rootCAs,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create store uploader: %v", err)
	}

	return uploader, nil
}

func store(c *cli.Context) error {
	uploader, err := getUploaderFromCtx(c)
	if err != nil {
		return err
	}

	return uploader.Store(
		c.String("file"),
		c.String("bucket"),
		c.String("prefix"),
		c.Bool("create-bucket"),
	)
}

func deleteOldRevisions(c *cli.Context) error {
	uploader, err := getUploaderFromCtx(c)
	if err != nil {
		return err
	}

	return uploader.DeleteOldBackups(
		c.String("bucket"),
		c.String("prefix"),
		c.Int("max-revisions"),
	)
}

func deleteAll(c *cli.Context) error {
	uploader, err := getUploaderFromCtx(c)
	if err != nil {
		return err
	}

	return uploader.DeleteAll(
		c.String("bucket"),
		c.String("prefix"),
	)
}
