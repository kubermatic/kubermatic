package main

import (
	"log"
	"os"

	"github.com/golang/glog"
	"github.com/kubermatic/kubermatic/api/pkg/storeuploader"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "S3 storer"
	app.Usage = ""
	app.Version = "v1.0.0"
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
	createBucketFlag := cli.BoolFlag{
		Name:  "create-bucket",
		Usage: "creates the bucket if it does not exist yet",
	}
	maxRevisionsFlag := cli.IntFlag{
		Name:  "max-revisions",
		Value: 20,
		Usage: "Maximum number of revisions of the file to keep in S3. Older ones will be deleted",
	}

	app.Commands = []cli.Command{
		{
			Name:   "store",
			Usage:  "Stores the given file on S3",
			Action: store,
			Flags: []cli.Flag{
				endpointFlag,
				secureFlag,
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
				accessKeyIDFlag,
				secretAccessKeyFlag,
				bucketFlag,
				prefixFlag,
				maxRevisionsFlag,
				fileFlag,
			},
		},
		{
			Name:   "delete-all",
			Usage:  "deletes all backups of the filename",
			Action: deleteAll,
			Flags: []cli.Flag{
				endpointFlag,
				secureFlag,
				accessKeyIDFlag,
				secretAccessKeyFlag,
				bucketFlag,
				prefixFlag,
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func getUploaderFromCtx(c *cli.Context) *storeuploader.StoreUploader {
	uploader, err := storeuploader.New(
		c.String("endpoint"),
		c.Bool("secure"),
		c.String("access-key-id"),
		c.String("secret-access-key"),
	)
	if err != nil {
		glog.Fatal(err)
	}
	return uploader
}

func store(c *cli.Context) error {
	uploader := getUploaderFromCtx(c)
	return uploader.Store(
		c.String("file"),
		c.String("bucket"),
		c.String("prefix"),
		c.Bool("create-bucket"),
	)
}
func deleteOldRevisions(c *cli.Context) error {
	uploader := getUploaderFromCtx(c)
	return uploader.DeleteOldBackups(
		c.String("file"),
		c.String("bucket"),
		c.String("prefix"),
		c.Int("max-revisions"),
	)
}
func deleteAll(c *cli.Context) error {
	uploader := getUploaderFromCtx(c)
	return uploader.DeleteAll(
		c.String("file"),
		c.String("bucket"),
		c.String("prefix"),
	)
}
