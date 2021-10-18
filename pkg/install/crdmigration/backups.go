/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package crdmigration

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"

	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateBackups(ctx context.Context, logger logrus.FieldLogger, opt *Options) error {
	now := time.Now()
	identifier := now.Format("kkpcrdmigration-2006-01-02T150405")

	logger.WithField("identifier", identifier).Info("Creating backups…")

	// backup master cluster
	filename := identifier + "-master.tar.gz"
	if err := createClusterBackup(ctx, logger.WithField("master", true), now, opt.MasterClient, filename, allKubermaticKinds); err != nil {
		return fmt.Errorf("backing up the master cluster failed: %w", err)
	}

	// backup seed clusters
	for seedName, seedClient := range opt.SeedClients {
		filename := fmt.Sprintf("%s-seed-%s.tar.gz", identifier, seedName)
		if err := createClusterBackup(ctx, logger.WithField("seed", seedName), now, seedClient, filename, allKubermaticKinds); err != nil {
			return fmt.Errorf("backing up the seed cluster failed: %w", err)
		}
	}

	return nil
}

func createClusterBackup(ctx context.Context, logger logrus.FieldLogger, ts time.Time, client ctrlruntimeclient.Client, filename string, kinds []string) error {
	logger.Info("Creating backup…")

	// Create output file
	backupFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer backupFile.Close()

	// wrap the file in gzip compression
	gzipWriter := gzip.NewWriter(backupFile)
	defer gzipWriter.Close()

	// create a tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	for _, kind := range kinds {
		logger.Debugf("Backing up %s resources…", kind)

		objectList := &metav1unstructured.UnstructuredList{}
		objectList.SetAPIVersion(kubermaticv1.SchemeGroupVersion.String())
		objectList.SetKind(kind)

		if err := client.List(ctx, objectList); err != nil {
			return fmt.Errorf("failed to list %s objects: %w", kind, err)
		}

		for _, object := range objectList.Items {
			objectLogger := logger.WithField(strings.ToLower(object.GetKind()), object.GetName())
			objectLogger.Debug("Dumping…")

			// create a filename like "cluster/3948rfhsf.yaml"
			filename := getBackupResourceFilename(object, kind)

			// encode resource as YAML
			encoded, err := encodeResourceAsYAML(object)
			if err != nil {
				return fmt.Errorf("failed to encode %s as YAML: %w", kind, err)
			}

			// write file header
			err = tarWriter.WriteHeader(&tar.Header{
				Name:     filename,
				ModTime:  ts,
				Mode:     int64(0644),
				Typeflag: tar.TypeReg,
				Size:     int64(len(encoded)),
				Uid:      os.Getuid(),
				Gid:      os.Getgid(),
			})
			if err != nil {
				return fmt.Errorf("failed to write tar file header: %w", err)
			}

			// write file contents
			if _, err := tarWriter.Write(encoded); err != nil {
				return fmt.Errorf("failed to append file to backup: %w", err)
			}
		}
	}

	return nil
}

func getBackupResourceFilename(obj metav1unstructured.Unstructured, kind string) string {
	filename := obj.GetName()
	if obj.GetNamespace() != "" {
		filename = fmt.Sprintf("%s-%s", obj.GetNamespace(), filename)
	}

	return fmt.Sprintf("%s/%s.yaml", strings.ToLower(kind), filename)
}

func encodeResourceAsYAML(obj metav1unstructured.Unstructured) ([]byte, error) {
	var buf bytes.Buffer

	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	// do not output grepping noise
	obj.SetManagedFields(nil)

	err := encoder.Encode(obj.Object)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
