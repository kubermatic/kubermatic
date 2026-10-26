/*
Copyright 2026 The Kubermatic Kubernetes Platform contributors.

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

package metering

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// Keys of the metering-s3 Secret, see pkg/ee/metering/const.go.
const (
	s3EndpointKey  = "endpoint"
	s3BucketKey    = "bucket"
	s3AccessKeyKey = "accessKey"
	s3SecretKeyKey = "secretKey"
)

// The seed MinIO from the KKP seed stack (charts/minio). EnsureS3 uses it when
// the seed has no metering-s3 Secret yet.
const (
	seedMinioNamespace = "minio"
	seedMinioName      = "minio"
	seedMinioPort      = 9000
)

// s3Bucket is an S3 client for the metering bucket. Callers must call stop.
type s3Bucket struct {
	client *minio.Client
	bucket string
	stop   func()
}

// meteringBucket connects to the bucket configured in the metering-s3
// Secret. An endpoint that is an in-cluster Service is reached through a
// port-forward.
func meteringBucket(ctx context.Context, in Input) (*s3Bucket, error) {
	secret := &corev1.Secret{}
	if err := in.SeedClient.Get(ctx, types.NamespacedName{Namespace: in.SeedNamespace, Name: s3SecretName}, secret); err != nil {
		return nil, fmt.Errorf("failed to get Secret %s: %w", s3SecretName, err)
	}

	endpoint := string(secret.Data[s3EndpointKey])
	bucket := string(secret.Data[s3BucketKey])
	if endpoint == "" || bucket == "" {
		return nil, fmt.Errorf("Secret %s has no endpoint or bucket", s3SecretName)
	}

	creds := credentials.NewStaticV4(string(secret.Data[s3AccessKeyKey]), string(secret.Data[s3SecretKeyKey]), "")

	// the metering tool treats an endpoint without scheme as https
	if !strings.Contains(endpoint, "://") {
		endpoint = "https://" + endpoint
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to parse S3 endpoint %q: %w", endpoint, err)
	}
	secure := u.Scheme == "https"

	stop := func() {}
	address := u.Host

	if name, namespace, ok := serviceAddress(u.Hostname()); ok {
		if secure {
			return nil, fmt.Errorf("S3 endpoint %s is an in-cluster Service with TLS, which the checks cannot reach through a port-forward", endpoint)
		}

		port := 80
		if u.Port() != "" {
			if port, err = strconv.Atoi(u.Port()); err != nil {
				return nil, fmt.Errorf("invalid port in S3 endpoint %q: %w", endpoint, err)
			}
		}

		localPort, stopForward, err := forwardServicePort(ctx, in, namespace, name, int32(port))
		if err != nil {
			return nil, fmt.Errorf("failed to port-forward to the S3 endpoint: %w", err)
		}

		stop = stopForward
		address = net.JoinHostPort("127.0.0.1", strconv.Itoa(int(localPort)))
	}

	client, err := minio.New(address, &minio.Options{Creds: creds, Secure: secure})
	if err != nil {
		stop()
		return nil, fmt.Errorf("failed to create S3 client: %w", err)
	}

	return &s3Bucket{client: client, bucket: bucket, stop: stop}, nil
}

// EnsureS3 makes sure the seed has a metering-s3 Secret and that its bucket
// exists. Without a Secret, it creates one for the seed MinIO of the KKP seed
// stack and bucket. created tells whether the Secret was created, so the caller
// can delete it again.
func EnsureS3(ctx context.Context, in Input, bucket string) (created bool, err error) {
	secret := &corev1.Secret{}
	err = in.SeedClient.Get(ctx, types.NamespacedName{Namespace: in.SeedNamespace, Name: s3SecretName}, secret)

	switch {
	case err == nil:
		in.Log.Infow("Using the existing metering S3 Secret", "bucket", string(secret.Data[s3BucketKey]), "endpoint", string(secret.Data[s3EndpointKey]))

	case apierrors.IsNotFound(err):
		minioSecret := &corev1.Secret{}
		if err := in.SeedClient.Get(ctx, types.NamespacedName{Namespace: seedMinioNamespace, Name: seedMinioName}, minioSecret); err != nil {
			return false, fmt.Errorf("no Secret %s and no seed MinIO credentials to create one: %w", s3SecretName, err)
		}

		secret = &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Namespace: in.SeedNamespace, Name: s3SecretName},
			Data: map[string][]byte{
				s3EndpointKey:  fmt.Appendf(nil, "http://%s.%s.svc.cluster.local:%d", seedMinioName, seedMinioNamespace, seedMinioPort),
				s3BucketKey:    []byte(bucket),
				s3AccessKeyKey: minioSecret.Data["accessKey"],
				s3SecretKeyKey: minioSecret.Data["secretKey"],
			},
		}
		if err := in.SeedClient.Create(ctx, secret); err != nil {
			return false, fmt.Errorf("failed to create Secret %s: %w", s3SecretName, err)
		}

		created = true
		in.Log.Infow("Created the metering S3 Secret for the seed MinIO", "bucket", bucket)

	default:
		return false, fmt.Errorf("failed to get Secret %s: %w", s3SecretName, err)
	}

	s3, err := meteringBucket(ctx, in)
	if err != nil {
		return created, err
	}
	defer s3.stop()

	exists, err := s3.client.BucketExists(ctx, s3.bucket)
	if err != nil {
		return created, fmt.Errorf("failed to check bucket %s: %w", s3.bucket, err)
	}

	if !exists {
		if err := s3.client.MakeBucket(ctx, s3.bucket, minio.MakeBucketOptions{}); err != nil {
			return created, fmt.Errorf("failed to create bucket %s: %w", s3.bucket, err)
		}
	}

	return created, nil
}

// DeleteS3Secret deletes the metering-s3 Secret that EnsureS3 created.
func DeleteS3Secret(ctx context.Context, in Input) error {
	secret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Namespace: in.SeedNamespace, Name: s3SecretName}}
	if err := in.SeedClient.Delete(ctx, secret); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("failed to delete Secret %s: %w", s3SecretName, err)
	}

	return nil
}

// CleanupObjects removes every object under the report prefixes of the suite.
func CleanupObjects(ctx context.Context, in Input) error {
	s3, err := meteringBucket(ctx, in)
	if err != nil {
		return err
	}
	defer s3.stop()

	var errs []error
	for object := range s3.client.ListObjects(ctx, s3.bucket, minio.ListObjectsOptions{Prefix: in.ReportPrefix, Recursive: true}) {
		if object.Err != nil {
			errs = append(errs, object.Err)
			continue
		}

		if err := s3.client.RemoveObject(ctx, s3.bucket, object.Key, minio.RemoveObjectOptions{}); err != nil {
			errs = append(errs, fmt.Errorf("failed to remove %s: %w", object.Key, err))
		}
	}

	return errors.Join(errs...)
}

// s3Object is a downloaded report.
type s3Object struct {
	key          string
	contentType  string
	lastModified time.Time
	body         []byte
}

func (b *s3Bucket) get(ctx context.Context, key string) (*s3Object, error) {
	info, err := b.client.StatObject(ctx, b.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("object %s: %w", key, err)
	}

	object, err := b.client.GetObject(ctx, b.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object %s: %w", key, err)
	}
	defer object.Close()

	body, err := io.ReadAll(object)
	if err != nil {
		return nil, fmt.Errorf("failed to read object %s: %w", key, err)
	}

	return &s3Object{key: key, contentType: info.ContentType, lastModified: info.LastModified, body: body}, nil
}
