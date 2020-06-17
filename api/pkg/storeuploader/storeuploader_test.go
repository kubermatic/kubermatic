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
