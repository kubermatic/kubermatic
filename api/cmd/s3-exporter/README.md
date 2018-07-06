# S3 exporter

A simple exporter for S3-compatible buckets.

Avaiable metrics:

```
# HELP s3_empty_object_count The amount of object with a size of zero
# TYPE s3_empty_object_count gauge
s3_empty_object_count{prefix="q8z2hgk94f"} 1
# HELP s3_object_count The amount of objects
# TYPE s3_object_count gauge
s3_object_count{prefix="q8z2hgk94f"} 12
# HELP s3_object_last_modified_object_time_seconds The amount of objects
# TYPE s3_object_last_modified_object_time_seconds gauge
s3_object_last_modified_object_time_seconds{prefix="q8z2hgk94f"} 1.530821994927e+18
# HELP s3_query_success Whether querying the S3 was successful
# TYPE s3_query_success gauge
s3_query_success 0
```


Usage:

```
s3-exporter:
  -access-key-id string
    	S3 Access key
  -alsologtostderr
    	log to standard error as well as files
  -bucket string
    	The bucket to monitor (default "kubermatic-etcd-backups")
  -endpoint string
    	https://my-s3.com:9000
  -listen-port int
    	The port to listen on (default 9340)
  -log_backtrace_at value
    	when logging hits line file:N, emit a stack trace
  -log_dir string
    	If non-empty, write log files in this directory
  -logtostderr
    	log to standard error instead of files
  -secret-access-key string
    	S3 Secret Access Key
  -stderrthreshold value
    	logs at or above this threshold go to stderr
  -v value
    	log level for V logs
  -vmodule value
    	comma-separated list of pattern=N settings for file-filtered logging
```


Releasing:

```
# Go to the project dir
cd $GOPATH/src/github.com/kubermatic/kubermatic/api

# Increment the tag variable in the publish script
vim hack/publish-s3-exporter.sh

# Publish the new version
./hack/publish-s3-exporter.sh

# Optional: Set the new version in the chart
vim ../config/kubermatic/values.yaml
```
