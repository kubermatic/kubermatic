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
	// revisionsToKeep is the amount of revisions to keep
	revisionsToKeep int
	// bucket is the name of the bucket to use
	bucket string
	// prefix allows to optionally set a prefix for all uploaded files
	prefix string
	// client is a pointer to an initialized client
	client *minio.Client
	// file defines the file to upload to S3
	file string
}

// New returns a new instance of the StoreUploader
func New(endpoint string, secure bool, accessKeyID, secretAccessKey string, revisionsToKeep int, bucket, prefix, file string) (*StoreUploader, error) {
	client, err := minio.New(endpoint, accessKeyID, secretAccessKey, secure)
	if err != nil {
		return nil, err
	}
	client.SetAppInfo("kubermatic-store-uploader", "v0.1")
	return &StoreUploader{
		revisionsToKeep: revisionsToKeep,
		bucket:          bucket,
		prefix:          prefix,
		client:          client,
		file:            file,
	}, nil
}

// Store uploads the given file to S3
func (u *StoreUploader) Store() error {
	if _, err := os.Stat(u.file); os.IsNotExist(err) {
		return fmt.Errorf("%s not found", u.file)
	}

	objectName := fmt.Sprintf("%s-%s-%s", u.getObjectPrefix(), time.Now().Format("2006-01-02T15:04:05"), path.Base(u.file))
	_, err := u.client.FPutObject(u.bucket, objectName, u.file, minio.PutObjectOptions{})
	return err
}

func (u *StoreUploader) getObjectPrefix() string {
	return fmt.Sprintf("%s-%s", u.prefix, prefixSeparator)
}

// DeleteOldBackups deletes revisions of the given file which are older than max-revisions
func (u *StoreUploader) DeleteOldBackups() error {
	doneCh := make(chan struct{})
	defer close(doneCh)

	var existingObjects []minio.ObjectInfo
	for object := range u.client.ListObjects(u.bucket, u.getObjectPrefix(), true, doneCh) {
		existingObjects = append(existingObjects, object)
	}

	for _, object := range u.getObjectsToDelete(existingObjects) {
		if err := u.client.RemoveObject(u.bucket, object.Key); err != nil {
			return err
		}
	}

	return nil
}

// DeleteAll deletes all revisions of the given file
func (u *StoreUploader) DeleteAll() error {
	doneCh := make(chan struct{})
	defer close(doneCh)

	var existingObjects []minio.ObjectInfo
	for object := range u.client.ListObjects(u.bucket, u.getObjectPrefix(), true, doneCh) {
		existingObjects = append(existingObjects, object)
	}

	for _, object := range existingObjects {
		if err := u.client.RemoveObject(u.bucket, object.Key); err != nil {
			return err
		}
	}

	return nil
}

func (u *StoreUploader) getObjectsToDelete(objects []minio.ObjectInfo) []minio.ObjectInfo {
	if len(objects) <= u.revisionsToKeep {
		return nil
	}

	sort.Slice(objects,
		func(i, j int) bool {
			return objects[i].LastModified.After(objects[j].LastModified)
		})

	numRevisionsToDelete := len(objects) - u.revisionsToKeep

	var objectsToDelete []minio.ObjectInfo
	for idx, object := range objects {
		if idx >= numRevisionsToDelete {
			return objectsToDelete
		}
		objectsToDelete = append(objectsToDelete, object)
	}

	return objectsToDelete
}
