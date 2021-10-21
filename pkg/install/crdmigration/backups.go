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
	operatorv1alpha1 "k8c.io/kubermatic/v2/pkg/crd/operator/v1alpha1"

	metav1unstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func CreateBackups(ctx context.Context, logger logrus.FieldLogger, opt *Options) error {
	now := time.Now()
	identifier := now.Format("kkpcrdmigration-2006-01-02T150405")

	logger.WithField("identifier", identifier).Info("Creating backups…")

	// Create output file
	filename := fmt.Sprintf("%s.tar.gz", identifier)
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

	// add KubermaticConfiguration
	if err := addKubermaticConfigurationToBackup(tarWriter, now, opt.KubermaticConfiguration); err != nil {
		return fmt.Errorf("backing up KubermaticConfiguration failed: %w", err)
	}

	// backup master cluster
	if err := createClusterBackup(ctx, logger.WithField("master", true), now, opt.MasterClient, tarWriter, "master", getMasterClusterKinds()); err != nil {
		return fmt.Errorf("backing up the master cluster failed: %w", err)
	}

	// backup seed clusters
	seedClusterKinds := getSeedClusterKinds()

	for seedName, seedClient := range opt.SeedClients {
		directory := fmt.Sprintf("seed-%s", seedName)
		if err := createClusterBackup(ctx, logger.WithField("seed", seedName), now, seedClient, tarWriter, directory, seedClusterKinds); err != nil {
			return fmt.Errorf("backing up the seed cluster failed: %w", err)
		}
	}

	return nil
}

func createClusterBackup(ctx context.Context, logger logrus.FieldLogger, ts time.Time, client ctrlruntimeclient.Client, tarWriter *tar.Writer, pathPrefix string, kinds []Kind) error {
	logger.Info("Creating backup…")

	for _, kind := range kinds {
		logger.Debugf("Backing up %s resources…", kind)

		objectList := &metav1unstructured.UnstructuredList{}
		objectList.SetAPIVersion(kubermaticv1.SchemeGroupVersion.String())
		objectList.SetKind(kind.Name)

		if err := client.List(ctx, objectList); err != nil {
			return fmt.Errorf("failed to list %s objects: %w", kind.Name, err)
		}

		for _, object := range objectList.Items {
			objectLogger := logger.WithField(strings.ToLower(object.GetKind()), object.GetName())
			objectLogger.Debug("Dumping…")

			// create a filename like "cluster/3948rfhsf.yaml"
			filename := getBackupResourceFilenameForObject(pathPrefix, object, kind.Name)

			// encode resource as YAML
			encoded, err := encodeResourceAsYAML(object)
			if err != nil {
				return fmt.Errorf("failed to encode %s as YAML: %w", kind.Name, err)
			}

			if err := addDataToBackup(tarWriter, ts, encoded, filename); err != nil {
				return err
			}
		}
	}

	return nil
}

func addKubermaticConfigurationToBackup(out *tar.Writer, t time.Time, config *operatorv1alpha1.KubermaticConfiguration) error {
	filename := getBackupResourceFilename("master", "KubermaticConfiguration", config.GetNamespace(), config.GetName())

	var buf bytes.Buffer

	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)

	// do not output grepping noise
	config.SetManagedFields(nil)

	err := encoder.Encode(config)
	if err != nil {
		return fmt.Errorf("failed to encode KubermaticConfiguration as YAML: %w", err)
	}

	return addDataToBackup(out, t, buf.Bytes(), filename)
}

func addDataToBackup(out *tar.Writer, t time.Time, data []byte, filename string) error {
	// write file header
	err := out.WriteHeader(&tar.Header{
		Name:     filename,
		ModTime:  t,
		Mode:     int64(0644),
		Typeflag: tar.TypeReg,
		Size:     int64(len(data)),
		Uid:      os.Getuid(),
		Gid:      os.Getgid(),
	})
	if err != nil {
		return fmt.Errorf("failed to write tar file header: %w", err)
	}

	// write file contents
	if _, err := out.Write(data); err != nil {
		return fmt.Errorf("failed to append file to backup: %w", err)
	}

	return nil
}

func getBackupResourceFilenameForObject(prefix string, obj metav1unstructured.Unstructured, kind string) string {
	return getBackupResourceFilename(prefix, kind, obj.GetNamespace(), obj.GetName())
}

func getBackupResourceFilename(prefix string, kind string, namespace string, name string) string {
	filename := name
	if namespace != "" {
		filename = fmt.Sprintf("%s-%s", namespace, filename)
	}

	return fmt.Sprintf("%s/%s/%s.yaml", prefix, strings.ToLower(kind), filename)
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
