/*
Copyright 2023 The Kubermatic Kubernetes Platform contributors.

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

package etcd

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	client "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/etcdutl/v3/snapshot"
	"go.uber.org/zap"
)

type SnapshotOptions struct {
	File        string
	Compression string
}

var ValidCompressions = []string{"gzip"}

func CreateSnapshot(ctx context.Context, log *zap.SugaredLogger, etcdConfig client.Config, opt *SnapshotOptions) error {
	snapv3 := snapshot.NewV3(log.Desugar())

	if opt.Compression == "" {
		if _, err := snapv3.Save(ctx, etcdConfig, opt.File); err != nil {
			return err
		}
	}

	tmpFile := opt.File + ".tmp"
	defer os.Remove(tmpFile)

	if _, err := snapv3.Save(ctx, etcdConfig, tmpFile); err != nil {
		return err
	}

	compressedFile, err := os.Create(opt.File)
	if err != nil {
		return err
	}
	defer compressedFile.Close()

	rawFile, err := os.Open(tmpFile)
	if err != nil {
		return err
	}
	defer rawFile.Close()

	var compressor io.WriteCloser

	switch opt.Compression {
	case "gzip":
		compressor, err = gzip.NewWriterLevel(compressedFile, gzip.BestCompression)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unknown compression algorithm %q", opt.Compression)
	}

	defer compressor.Close()

	if _, err = io.Copy(compressor, rawFile); err != nil {
		return err
	}

	return nil
}

func DecompressSnapshot(filename string) (string, error) {
	ext := filepath.Ext(filename)

	switch ext {
	case ".db":
		return filename, nil

	case ".gz", ".gzip":
		rawFilename := strings.TrimSuffix(filename, ext)

		rawFile, err := os.Create(rawFilename)
		if err != nil {
			return "", err
		}
		defer rawFile.Close()

		compressedFile, err := os.Open(filename)
		if err != nil {
			return "", err
		}
		defer compressedFile.Close()

		var decompressor io.ReadCloser

		switch ext {
		case ".gz", ".gzip":
			decompressor, err = gzip.NewReader(compressedFile)
			if err != nil {
				return "", err
			}
		default:
			panic("Inner switch statement is out-of-sync with outer switch statement.")
		}

		defer decompressor.Close()

		if _, err = io.Copy(rawFile, decompressor); err != nil {
			return "", err
		}

		return rawFilename, nil

	default:
		return "", fmt.Errorf("unsupported backup file extension %q", ext)
	}
}
