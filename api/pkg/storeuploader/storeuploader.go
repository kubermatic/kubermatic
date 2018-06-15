package storeuploader

import (
	"fmt"
	"os"
	"path"
	"sort"
	"time"

	"github.com/minio/minio-go"
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
}

// New returns a new instance of the StoreUploader
func New(endpoint string, secure bool, accessKeyID, secretAccessKey string) (*StoreUploader, error) {
	client, err := minio.New(endpoint, accessKeyID, secretAccessKey, secure)
	if err != nil {
		return nil, err
	}
	client.SetAppInfo("kubermatic-store-uploader", "v0.1")
	return &StoreUploader{
		client: client,
	}, nil
}

// Store uploads the given file to S3
func (u *StoreUploader) Store(file, bucket, prefix string, createBucket bool) error {
	if _, err := os.Stat(file); os.IsNotExist(err) {
		return fmt.Errorf("%s not found", file)
	}

	if createBucket {
		exists, err := u.client.BucketExists(bucket)
		if err != nil {
			return err
		}
		if !exists {
			if err := u.client.MakeBucket(bucket, ""); err != nil {
				return err
			}
		}
	}

	objectName := fmt.Sprintf("%s-%s-%s-%s", prefix, prefixSeparator, time.Now().Format("2006-01-02T15:04:05"), path.Base(file))
	_, err := u.client.FPutObject(bucket, objectName, file, minio.PutObjectOptions{})
	return err
}

// DeleteOldBackups deletes revisions of the given file which are older than max-revisions
func (u *StoreUploader) DeleteOldBackups(file, bucket, prefix string, revisionsToKeep int) error {
	doneCh := make(chan struct{})
	defer close(doneCh)

	var existingObjects []minio.ObjectInfo
	for object := range u.client.ListObjects(bucket, fmt.Sprintf("%s-%s", prefix, prefixSeparator), true, doneCh) {
		existingObjects = append(existingObjects, object)
	}

	for _, object := range u.getObjectsToDelete(existingObjects, revisionsToKeep) {
		if err := u.client.RemoveObject(bucket, object.Key); err != nil {
			return err
		}
	}

	return nil
}

// DeleteAll deletes all revisions of the given file
func (u *StoreUploader) DeleteAll(file, bucket, prefix string) error {
	doneCh := make(chan struct{})
	defer close(doneCh)

	var existingObjects []minio.ObjectInfo
	for object := range u.client.ListObjects(bucket, fmt.Sprintf("%s-%s", prefix, prefixSeparator), true, doneCh) {
		existingObjects = append(existingObjects, object)
	}

	for _, object := range existingObjects {
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

	sort.Slice(objects,
		func(i, j int) bool {
			return objects[i].LastModified.After(objects[j].LastModified)
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
