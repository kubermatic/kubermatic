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

package docker

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/distribution/distribution/v3/reference"
	"github.com/sirupsen/logrus"
)

// execCommand is an internal helper function to execute commands and log them.
func execCommand(log logrus.FieldLogger, dryRun bool, cmd *exec.Cmd) error {
	log = log.WithField("command", strings.Join(cmd.Args, " "))
	if dryRun {
		log.Debug("Would execute command but this is a dry-run")
		return nil
	}

	log.Info("Executing command…")
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Errorf("Command failed: %s", string(out))
		return err
	}

	log.Debug("Executed command")
	return err
}

// DownloadImages pulls all given images using the Docker CLI
// Invokes DownloadImage for actual pulling.
func DownloadImages(ctx context.Context, log logrus.FieldLogger, dockerBinary string, dryRun bool, images []string) error {
	for _, image := range images {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := DownloadImage(ctx, log, dockerBinary, dryRun, image); err != nil {
			return fmt.Errorf("failed to download %s: %w", image, err)
		}
	}

	return nil
}

// DownloadImage invokes the Docker CLI and pulls an image.
func DownloadImage(ctx context.Context, log logrus.FieldLogger, dockerBinary string, dryRun bool, image string) error {
	log = log.WithField("image", image)
	log.Info("Downloading image…")

	cmd := exec.CommandContext(ctx, dockerBinary, "pull", image)
	if err := execCommand(log, dryRun, cmd); err != nil {
		return fmt.Errorf("failed to pull image %s: %w", image, err)
	}

	return nil
}

// RetagImages invokes the Docker CLI and tags the given images so they belongs to the given registry.
// Invokes RetagImage for actual tagging.
func RetagImages(ctx context.Context, log logrus.FieldLogger, dockerBinary string, dryRun bool, images []string, registry string) ([]string, error) {
	var retaggedImages []string
	for _, image := range images {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		retaggedImage, err := RetagImage(ctx, log, dockerBinary, dryRun, image, registry)
		if err != nil {
			return nil, fmt.Errorf("failed to re-tag %q: %w", image, err)
		}

		retaggedImages = append(retaggedImages, retaggedImage)
	}

	return retaggedImages, nil
}

// RetagImage invokes the Docker CLI and tags the given image so it belongs to the given registry.
func RetagImage(ctx context.Context, log logrus.FieldLogger, dockerBinary string, dryRun bool, sourceImage, registry string) (string, error) {
	log = log.WithField("source-image", sourceImage)
	imageRef, err := reference.ParseNamed(sourceImage)
	if err != nil {
		return "", fmt.Errorf("failed to parse image: %w", err)
	}
	taggedImageRef, ok := imageRef.(reference.NamedTagged)
	if !ok {
		return "", errors.New("image has no tag")
	}

	targetImage := fmt.Sprintf("%s/%s:%s", registry, reference.Path(imageRef), taggedImageRef.Tag())
	log = log.WithField("target-image", targetImage)

	if dryRun {
		log.Info("Image found")
	} else {
		log.Info("Tagging image…")
	}

	cmd := exec.CommandContext(ctx, dockerBinary, "tag", sourceImage, targetImage)
	if err := execCommand(log, dryRun, cmd); err != nil {
		return "", fmt.Errorf("failed to tag image %s to %s: %w", sourceImage, targetImage, err)
	}

	return targetImage, nil
}

// PushImages pushes all given images using the Docker CLI
// Invokes PushImage for actual pushing.
func PushImages(ctx context.Context, log logrus.FieldLogger, dockerBinary string, dryRun bool, images []string) error {
	for _, image := range images {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := PushImage(ctx, log, dockerBinary, dryRun, image); err != nil {
			return fmt.Errorf("failed to push image %s: %w", image, err)
		}
	}

	return nil
}

// PushImage invokes the Docker CLI and pushes the given image.
func PushImage(ctx context.Context, log logrus.FieldLogger, dockerBinary string, dryRun bool, image string) error {
	log = log.WithField("image", image)

	log.Info("Pushing image…")
	cmd := exec.CommandContext(ctx, dockerBinary, "push", image)
	if err := execCommand(log, dryRun, cmd); err != nil {
		return err
	}

	return nil
}

// Copy copies the content from a directory out of the
// container onto the host system.
func Copy(ctx context.Context, log logrus.FieldLogger, dockerBinary string, image string, dst string, src string) error {
	var err error

	log = log.WithField("image", image)

	log.WithFields(logrus.Fields{
		"source":      src,
		"destination": dst,
	}).Info("Extracting image…")

	dst, err = filepath.Abs(dst)
	if err != nil {
		return fmt.Errorf("failed to determine absolute path: %w", err)
	}

	mountPoint := "/kubermaticextractor"
	args := []string{
		"run",
		"--rm",
		"-v", fmt.Sprintf("%s:%s", dst, mountPoint),
		"-w", src,
		image,
		"cp", "-ar", ".", mountPoint,
	}

	cmd := exec.CommandContext(ctx, dockerBinary, args...)
	if err := execCommand(log, false, cmd); err != nil {
		return err
	}

	return nil
}
