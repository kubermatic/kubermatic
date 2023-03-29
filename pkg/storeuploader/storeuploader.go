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

package storeuploader

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"sort"
	"time"

	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"

	"k8c.io/kubermatic/v3/pkg/util/s3"
)

// prefix separator separates the prefix
// from the file name. This is required to not have
// the storeuploader manage all files when prefix
// is an empty string.
const prefixSeparator = "storeuploader"

// StoreUploader is the configuration
// for the StoreUploader.
type StoreUploader struct {
	// client is a pointer to an initialized client
	client *minio.Client
	logger *zap.SugaredLogger
}

// New returns a new instance of the StoreUploader.
func New(endpoint string, accessKeyID, secretAccessKey string, logger *zap.SugaredLogger, rootCAs *x509.CertPool) (*StoreUploader, error) {
	client, err := s3.NewClient(endpoint, accessKeyID, secretAccessKey, rootCAs)
	if err != nil {
		return nil, err
	}
	client.SetAppInfo("kubermatic-store-uploader", "v0.2")

	return &StoreUploader{
		client: client,
		logger: logger,
	}, nil
}

// Store uploads the given file to S3.
func (u *StoreUploader) Store(ctx context.Context, file, bucket, prefix string, createBucket bool) error {
	if len(prefix) == 0 {
		return errors.New("prefix cannot be empty")
	}

	if _, err := os.Stat(file); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s not found", file)
	}

	logger := u.logger.With("bucket", bucket)

	if createBucket {
		logger.Debug("Check if bucket exists")
		exists, err := u.client.BucketExists(ctx, bucket)
		if err != nil {
			return err
		}
		if !exists {
			logger.Info("Creating bucket")
			if err := u.client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
				return err
			}
		}
	}

	objectName := fmt.Sprintf("%s-%s-%s-%s", prefix, prefixSeparator, time.Now().Format("2006-01-02T150405"), path.Base(file))
	logger.Infow("Uploading file", "src", file, "dst", objectName)

	_, err := u.client.FPutObject(ctx, bucket, objectName, file, minio.PutObjectOptions{})
	return err
}

// DeleteOldBackups deletes revisions of all files of the given prefix which are older than max-revisions.
func (u *StoreUploader) DeleteOldBackups(ctx context.Context, bucket, prefix string, revisionsToKeep int) error {
	if len(prefix) == 0 {
		return errors.New("prefix cannot be empty")
	}

	doneCh := make(chan struct{})
	defer close(doneCh)

	logger := u.logger.With("bucket", bucket, "prefix", prefix, "keep", revisionsToKeep)

	logger.Debugw("Listing existing objects")

	listOpts := minio.ListObjectsOptions{
		Recursive: true,
		Prefix:    fmt.Sprintf("%s-%s", prefix, prefixSeparator),
	}

	var existingObjects []minio.ObjectInfo
	for object := range u.client.ListObjects(ctx, bucket, listOpts) {
		if object.Err != nil {
			return object.Err
		}
		existingObjects = append(existingObjects, object)
	}

	logger.Debugw("Done listing bucket", "objects", len(existingObjects))

	for _, object := range u.getObjectsToDelete(existingObjects, revisionsToKeep) {
		logger.Infow("Removing object", "object", object.Key)
		if err := u.client.RemoveObject(ctx, bucket, object.Key, minio.RemoveObjectOptions{}); err != nil {
			return err
		}
	}

	return nil
}

// DeleteAll deletes all revisions of all files matching the given prefix.
func (u *StoreUploader) DeleteAll(ctx context.Context, bucket, prefix string) error {
	if len(prefix) == 0 {
		return errors.New("prefix cannot be empty")
	}

	doneCh := make(chan struct{})
	defer close(doneCh)

	logger := u.logger.With("bucket", bucket, "prefix", prefix)

	logger.Debugw("Listing existing objects")

	listOpts := minio.ListObjectsOptions{
		Recursive: true,
		Prefix:    fmt.Sprintf("%s-%s", prefix, prefixSeparator),
	}

	var existingObjects []minio.ObjectInfo
	for object := range u.client.ListObjects(ctx, bucket, listOpts) {
		if object.Err != nil {
			return object.Err
		}
		existingObjects = append(existingObjects, object)
	}

	logger.Debugw("Done listing bucket", "objects", len(existingObjects))

	for _, object := range existingObjects {
		logger.Infow("Removing object", "object", object.Key)
		if err := u.client.RemoveObject(ctx, bucket, object.Key, minio.RemoveObjectOptions{}); err != nil {
			return err
		}
	}

	return nil
}

func (u *StoreUploader) getObjectsToDelete(objects []minio.ObjectInfo, revisionsToKeep int) []minio.ObjectInfo {
	if len(objects) <= revisionsToKeep {
		return nil
	}

	sort.Slice(objects, func(i, j int) bool {
		return objects[i].LastModified.Before(objects[j].LastModified)
	})

	numRevisionsToDelete := len(objects) - revisionsToKeep

	var objectsToDelete []minio.ObjectInfo
	for idx, object := range objects {
		if idx >= numRevisionsToDelete {
			return objectsToDelete
		}
		objectsToDelete = append(objectsToDelete, object)
	}

	return objectsToDelete
}
