package storeuploader

import (
	"errors"
	"fmt"
	"os"
	"path"
	"sort"
	"time"

	"github.com/minio/minio-go"
	"go.uber.org/zap"
)

// prefix separator separates the prefix
// from the file name. This is required to not have
// the storeuploader manage all files when prefix
// is an empty string
const prefixSeparator = "storeuploader"

// StoreUploader is the configuration
// for the StoreUploader
type StoreUploader struct {
	// client is a pointer to an initialized client
	client *minio.Client
	logger *zap.SugaredLogger
}

// New returns a new instance of the StoreUploader
func New(endpoint string, secure bool, accessKeyID, secretAccessKey string, logger *zap.SugaredLogger) (*StoreUploader, error) {
	client, err := minio.New(endpoint, accessKeyID, secretAccessKey, secure)
	if err != nil {
		return nil, err
	}
	client.SetAppInfo("kubermatic-store-uploader", "v0.1")
	return &StoreUploader{
		client: client,
		logger: logger,
	}, nil
}

// Store uploads the given file to S3
func (u *StoreUploader) Store(file, bucket, prefix string, createBucket bool) error {
	if len(prefix) == 0 {
		return errors.New("prefix cannot be empty")
	}

	if _, err := os.Stat(file); os.IsNotExist(err) {
		return fmt.Errorf("%s not found", file)
	}

	if createBucket {
		u.logger.Debugw("Check if bucket exists", "bucket", bucket)
		exists, err := u.client.BucketExists(bucket)
		if err != nil {
			return err
		}
		if !exists {
			u.logger.Infow("Creating bucket", "bucket", bucket)
			if err := u.client.MakeBucket(bucket, ""); err != nil {
				return err
			}
		}
	}

	objectName := fmt.Sprintf("%s-%s-%s-%s", prefix, prefixSeparator, time.Now().Format("2006-01-02T15:04:05"), path.Base(file))
	u.logger.Infow("Uploading file", "bucket", bucket, "src", file, "dst", objectName)

	_, err := u.client.FPutObject(bucket, objectName, file, minio.PutObjectOptions{})
	return err
}

// DeleteOldBackups deletes revisions of all files of the given prefix which are older than max-revisions
func (u *StoreUploader) DeleteOldBackups(bucket, prefix string, revisionsToKeep int) error {
	if len(prefix) == 0 {
		return errors.New("prefix cannot be empty")
	}

	doneCh := make(chan struct{})
	defer close(doneCh)

	u.logger.Debugw("Listing existing objects", "bucket", bucket, "prefix", prefix)

	var existingObjects []minio.ObjectInfo
	for object := range u.client.ListObjects(bucket, fmt.Sprintf("%s-%s", prefix, prefixSeparator), true, doneCh) {
		if object.Err != nil {
			return object.Err
		}
		existingObjects = append(existingObjects, object)
	}

	u.logger.Debugw("Done listing bucket", "bucket", bucket, "prefix", prefix, "objects", len(existingObjects))

	for _, object := range u.getObjectsToDelete(existingObjects, revisionsToKeep) {
		u.logger.Infow("Removing object", "bucket", bucket, "object", object.Key)
		if err := u.client.RemoveObject(bucket, object.Key); err != nil {
			return err
		}
	}

	return nil
}

// DeleteAll deletes all revisions of all files matching the given prefix
func (u *StoreUploader) DeleteAll(bucket, prefix string) error {
	if len(prefix) == 0 {
		return errors.New("prefix cannot be empty")
	}

	doneCh := make(chan struct{})
	defer close(doneCh)

	u.logger.Debugw("Listing existing objects", "bucket", bucket, "prefix", prefix)

	var existingObjects []minio.ObjectInfo
	for object := range u.client.ListObjects(bucket, fmt.Sprintf("%s-%s", prefix, prefixSeparator), true, doneCh) {
		if object.Err != nil {
			return object.Err
		}
		existingObjects = append(existingObjects, object)
	}

	u.logger.Debugw("Done listing bucket", "bucket", bucket, "prefix", prefix, "objects", len(existingObjects))

	for _, object := range existingObjects {
		u.logger.Infow("Removing object", "bucket", bucket, "object", object.Key)
		if err := u.client.RemoveObject(bucket, object.Key); err != nil {
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
