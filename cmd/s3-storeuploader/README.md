# S3 Store Uploader

```
Helper tool to backup files to S3 and maintain a given number of revisions

Usage:
  s3-storeuploader [command]

Available Commands:
  completion           generate the autocompletion script for the specified shell
  delete-all           Deletes all backups of the filename
  delete-old-revisions Deletes backups which are older than max-revisions
  help                 Help about any command
  store                Stores the given file on S3

Flags:
      --access-key-id string       S3 access key ID ($ACCESS_KEY_ID)
  -b, --bucket string              S3 bucket in which to store the snapshots (default "kubermatic-backups")
      --ca-bundle string           Filename of the CA bundle to use (if not given, default system certificates are used)
      --create-bucket              Create the bucket if it does not exist yet
  -e, --endpoint string            S3 endpoint
  -f, --file string                Path to the file to store in S3 (default "/backup/snapshot.db")
  -h, --help                       help for s3-storeuploader
      --log-debug                  Enables debug logging
      --log-format string          Log format, one of JSON, Console (default "JSON")
      --max-revisions int          Maximum number of revisions of the file to keep in S3. Older ones will be deleted (default 20)
  -p, --prefix string              Prefix to use for all objects stored in S3
      --secret-access-key string   S3 secret access key ($SECRET_ACCESS_KEY)
      --secure                     Enable TLS validation
  -v, --version                    version for s3-storeuploader

Use "s3-storeuploader [command] --help" for more information about a command.
```

# Building the docker image

```bash
CGO_ENABLED=0 go build -ldflags '-w -extldflags "-static"' -o s3-storeuploader k8c.io/kubermatic/v3/cmd/s3-storeuploader
docker build -t quay.io/kubermatic/s3-storer:v0.1.6 .
docker push quay.io/kubermatic/s3-storer:v0.1.6
```
