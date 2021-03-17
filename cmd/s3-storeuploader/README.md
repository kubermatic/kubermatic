# S3 storer

```bash
NAME:
   S3 storer

USAGE:
   s3-storeuploader [global options] command [command options] [arguments...]

VERSION:
   v1.0.0

DESCRIPTION:
   Helper tool to backup files to S3 and maintain a given number of revisions

COMMANDS:
     store                 Stores the given file on S3
     delete-old-revisions  Deletes backups which are older than max-revisions
     delete-all            deletes all backups of the filename
     help, h               Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h     show help
   --version, -v  print the version
```


# Building the docker image

```bash
CGO_ENABLED=0 go build -ldflags '-w -extldflags "-static"' -o s3-storeuploader k8c.io/kubermatic/v2/cmd/s3-storeuploader
docker build -t quay.io/kubermatic/s3-storer:v0.1.5 .
docker push quay.io/kubermatic/s3-storer:v0.1.5
```
