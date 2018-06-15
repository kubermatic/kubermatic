package storeuploader

import (
	"testing"
	"time"

	"github.com/minio/minio-go"
)

func TestGetObjectsToDelete(t *testing.T) {
	tests := []struct {
		existingObjects []minio.ObjectInfo
		expected        []string
	}{
		{
			existingObjects: []minio.ObjectInfo{
				{
					Key:          "foo",
					LastModified: time.Unix(1, 0),
				},
			},
		},
		{
			existingObjects: []minio.ObjectInfo{
				{
					Key:          "foo",
					LastModified: time.Unix(1, 0),
				},
				{
					Key:          "bar",
					LastModified: time.Unix(10, 0),
				},
			},
			expected: []string{"bar"},
		},
	}

	uploader := StoreUploader{}
	for _, test := range tests {
		objectsToDelete := uploader.getObjectsToDelete(test.existingObjects, 1)
		if len(objectsToDelete) != len(test.expected) {
			t.Errorf("Expected objectsToDelete to be of size %v but was %v", len(test.expected), len(objectsToDelete))
		}
		for _, expectedObject := range test.expected {
			if !isExpectedObjectInObjectList(objectsToDelete, expectedObject) {
				t.Errorf("Expected to find object %s in objectsToDelete list!", expectedObject)
			}
		}
	}
}

func isExpectedObjectInObjectList(objects []minio.ObjectInfo, expected string) bool {
	for _, object := range objects {
		if object.Key == expected {
			return true
		}
	}

	return false
}
