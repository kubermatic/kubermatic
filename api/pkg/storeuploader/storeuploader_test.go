package storeuploader

import (
	"testing"
	"time"

	"github.com/go-test/deep"
	"github.com/minio/minio-go"
)

func TestGetObjectsToDelete(t *testing.T) {
	tests := []struct {
		name             string
		existingObjects  []minio.ObjectInfo
		expectedToDelete []minio.ObjectInfo
		revisions        int
	}{
		{
			name:      "nothing gets deleted as revisions==existing-backups",
			revisions: 1,
			existingObjects: []minio.ObjectInfo{
				{
					Key:          "foo",
					LastModified: time.Unix(1, 0),
				},
			},
			expectedToDelete: nil,
		},
		{
			name:      "oldest should be deleted as revisions < existing-backups",
			revisions: 1,
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
			expectedToDelete: []minio.ObjectInfo{
				{
					Key:          "foo",
					LastModified: time.Unix(1, 0),
				},
			},
		},
	}

	uploader := StoreUploader{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Log("existing objects:")
			for _, object := range test.existingObjects {
				t.Logf("existing object: %s - %s", object.LastModified.Format("2006-01-02T15:04:05"), object.Key)
			}

			gotToDelete := uploader.getObjectsToDelete(test.existingObjects, test.revisions)
			t.Log("objects to delete:")
			for _, object := range gotToDelete {
				t.Logf("existing object: %s - %s", object.LastModified.Format("2006-01-02T15:04:05"), object.Key)
			}

			if diff := deep.Equal(gotToDelete, test.expectedToDelete); diff != nil {
				t.Errorf("Expected: \n\n%v \n\nGot: \n\n%v", test.expectedToDelete, gotToDelete)
			}
		})
	}
}
