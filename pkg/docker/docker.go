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

	"github.com/docker/distribution/reference"
	"go.uber.org/zap"
)

// execCommand is an internal helper function to execute commands and log them
func execCommand(log *zap.SugaredLogger, dryRun bool, cmd *exec.Cmd) error {
	log = log.With(zap.String("command", strings.Join(cmd.Args, " ")))
	if dryRun {
		log.Info("Would execute Docker command but this is a dry-run")
		return nil
	}

	log.Debug("Executing command...")
	out, err := cmd.CombinedOutput()
	log = log.With(zap.ByteString("output", out))
	if err != nil {
		log.Error("Command failed")
		return err
	}

	log.Debug("Executed command")
	return err
}

// DownloadImages pulls all given images using the Docker CLI
// Invokes DownloadImage for actual pulling
func DownloadImages(ctx context.Context, log *zap.SugaredLogger, dryRun bool, images []string) error {
	for _, image := range images {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := DownloadImage(ctx, log, dryRun, image); err != nil {
			return fmt.Errorf("failed to download %s: %v", image, err)
		}
	}

	return nil
}

// DownloadImage invokes the Docker CLI and pulls an image
func DownloadImage(ctx context.Context, log *zap.SugaredLogger, dryRun bool, image string) error {
	log = log.With(zap.String("image", image))
	log.Info("Downloading image...")

	cmd := exec.CommandContext(ctx, "docker", "pull", image)
	if err := execCommand(log, dryRun, cmd); err != nil {
		return fmt.Errorf("failed to pull image %s: %v", image, err)
	}

	return nil
}

// RetagImages invokes the Docker CLI and tags the given images so they belongs to the given registry.
// Invokes RetagImage for actual tagging
func RetagImages(ctx context.Context, log *zap.SugaredLogger, dryRun bool, images []string, registry string) ([]string, error) {
	var retaggedImages []string
	for _, image := range images {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		retaggedImage, err := RetagImage(ctx, log, dryRun, image, registry)
		if err != nil {
			return nil, fmt.Errorf("failed to re-tag %q: %v", image, err)
		}

		retaggedImages = append(retaggedImages, retaggedImage)
	}

	return retaggedImages, nil
}

// RetagImage invokes the Docker CLI and tags the given image so it belongs to the given registry.
func RetagImage(ctx context.Context, log *zap.SugaredLogger, dryRun bool, sourceImage, registry string) (string, error) {
	log = log.With(zap.String("source-image", sourceImage))
	imageRef, err := reference.ParseNamed(sourceImage)
	if err != nil {
		return "", fmt.Errorf("failed to parse image: %v", err)
	}
	taggedImageRef, ok := imageRef.(reference.NamedTagged)
	if !ok {
		return "", errors.New("image has no tag")
	}

	targetImage := fmt.Sprintf("%s/%s:%s", registry, reference.Path(imageRef), taggedImageRef.Tag())
	log = log.With(zap.String("target-image", targetImage))

	log.Info("Tagging image...")
	cmd := exec.CommandContext(ctx, "docker", "tag", sourceImage, targetImage)
	if err := execCommand(log, dryRun, cmd); err != nil {
		return "", fmt.Errorf("failed to tag image %s to %s: %v", sourceImage, targetImage, err)
	}

	return targetImage, nil
}

// PushImages pushes all given images using the Docker CLI
// Invokes PushImage for actual pushing
func PushImages(ctx context.Context, log *zap.SugaredLogger, dryRun bool, images []string) error {
	for _, image := range images {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := PushImage(ctx, log, dryRun, image); err != nil {
			return fmt.Errorf("failed to push image %s: %v", image, err)
		}
	}

	return nil
}

// PushImage invokes the Docker CLI and pushes the given image
func PushImage(ctx context.Context, log *zap.SugaredLogger, dryRun bool, image string) error {
	log = log.With(zap.String("image", image))

	log.Info("Pushing image...")
	cmd := exec.CommandContext(ctx, "docker", "push", image)
	if err := execCommand(log, dryRun, cmd); err != nil {
		return err
	}

	return nil
}

// ExtractDirectory copies the content from a directory out of the
// container onto the host system
func ExtractDirectory(ctx context.Context, log *zap.SugaredLogger, image string, dst string, src string) error {
	var err error

	log = log.With(zap.String("image", image))

	log.Info("Extracting image...", "src", src, "dst", dst)

	dst, err = filepath.Abs(dst)
	if err != nil {
		return fmt.Errorf("failed to determine absolute path: %v", err)
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

	cmd := exec.CommandContext(ctx, "docker", args...)
	if err := execCommand(log, false, cmd); err != nil {
		return err
	}

	return nil
}
