package storeuploader

import (
	"fmt"
	"io/ioutil"
	"sort"

	"github.com/minio/minio-go"
)

// Prefix separator separates the prefix
// from the file name. This is required to not have
// the storeuploader manage all files when Prefix
// is an empty string
const prefixSeparator = "storeuploader"

// StoreUploader is the configuration
// for the StoreUploader
type StoreUploader struct {
	// RevisionsToKeep is the amount of revisions to keep
	RevisionsToKeep int
	// Bucket is the name of the Bucket to use
	Bucket string
	// Prefix allows to optionally set a prefix for all uploaded files
	Prefix string
	// BackupSourceDir specifies the backup dir to read the files from
	BackupSourceDir string
	// Client is a pointer to an initialized Client
	Client *minio.Client
}

func New(endpoint, accessKeyID, secretAccessKey string, secure bool, revisionsToKeep int, bucket, prefix, backupSourceDir string) (*StoreUploader, error) {
	client, err := minio.New(endpoint, accessKeyID, secretAccessKey, secure)
	if err != nil {
		return nil, err
	}
	client.SetAppInfo("kubermatic-store-uploader", "v0.1")
	return &StoreUploader{
		RevisionsToKeep: revisionsToKeep,
		Bucket:          bucket,
		Prefix:          prefix,
		BackupSourceDir: backupSourceDir,
		Client:          client,
	}, nil
}

func (uploader *StoreUploader) Run() error {
	fileInfos, err := ioutil.ReadDir(uploader.BackupSourceDir)
	if err != nil {
		return fmt.Errorf("failed to read backup source dir: %v", err)
	}

	var fileNames []string
	for _, file := range fileInfos {
		if !file.IsDir() {
			fileNames = append(fileNames, file.Name())
		}
	}

	for _, fileName := range fileNames {
		objectName := fmt.Sprintf("%s-%s-%s", uploader.Prefix, prefixSeparator, fileName)
		filePath := fmt.Sprintf("%s/%s", uploader.BackupSourceDir, fileName)

		if _, err := uploader.Client.FPutObject(uploader.Bucket, objectName, filePath, minio.PutObjectOptions{}); err != nil {
			return err
		}
	}

	doneCh := make(chan struct{})
	defer close(doneCh)

	return uploader.deleteOldBackups()
}

func (uploader *StoreUploader) deleteOldBackups() error {
	doneCh := make(chan struct{})
	defer close(doneCh)

	var existingObjects []minio.ObjectInfo
	for object := range uploader.Client.ListObjects(uploader.Bucket, fmt.Sprintf("%s-%s", uploader.Prefix, prefixSeparator), true, doneCh) {
		existingObjects = append(existingObjects, object)
	}

	for _, object := range uploader.getObjectsToDelete(existingObjects) {
		if err := uploader.Client.RemoveObject(uploader.Bucket, object.Key); err != nil {
			return err
		}
	}

	return nil
}

func (uploader *StoreUploader) getObjectsToDelete(objects []minio.ObjectInfo) []minio.ObjectInfo {
	if len(objects) <= uploader.RevisionsToKeep {
		return nil
	}

	sort.Slice(objects,
		func(i, j int) bool {
			return objects[i].LastModified.After(objects[j].LastModified)
		})

	numRevisionsToDelete := len(objects) - uploader.RevisionsToKeep

	var objectsToDelete []minio.ObjectInfo
	for idx, object := range objects {
		if idx >= numRevisionsToDelete {
			return objectsToDelete
		}
		objectsToDelete = append(objectsToDelete, object)
	}

	return objectsToDelete
}
