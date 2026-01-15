/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"
	"k8c.io/kubermatic/v2/pkg/install/images"
	"k8c.io/kubermatic/v2/pkg/version"
	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"
)

// Constants for default values and base URLs.
const (
	DefaultCNIPluginsVersion = "v1.5.1"
	CNIPluginsBaseURL        = "https://github.com/containernetworking/plugins/releases/download"
	CRIToolsBaseURL          = "https://github.com/kubernetes-sigs/cri-tools/releases/download"
	KubeBaseURLFormat        = "https://dl.k8s.io"
	KubeBinaryPath           = "release/%s/bin/linux/%s"
	SHA256Exentsion          = ".sha256"
	// Default output directory for binaries.
	DefaultOutputDir = "/usr/share/nginx/html/"
)

var httpClient = &http.Client{
	Timeout: 2 * time.Minute,
}

// MirrorBinariesOptions holds options for the mirror-binaries command.
type MirrorBinariesOptions struct {
	Config        string
	Versions      kubermaticversion.Versions
	VersionFilter string
	Architectures string
	// Destination directory for binaries.
	OutputDir string
}

// MirrorBinariesCommand creates the cobra command for mirror-binaries.
func MirrorBinariesCommand(logger *logrus.Logger, versions kubermaticversion.Versions) *cobra.Command {
	opt := MirrorBinariesOptions{
		OutputDir: DefaultOutputDir,
	}
	cmd := &cobra.Command{
		Use:   "mirror-binaries",
		Short: "Mirror binaries used by KKP",
		Long:  "Downloads all binaries used by KKP and copies them into a local path.",
		PreRun: func(cmd *cobra.Command, args []string) {
			if opt.Config == "" {
				opt.Config = os.Getenv("CONFIG_YAML")
			}
			if len(args) >= 1 {
				opt.Config = args[0]
			}

			opt.Versions = versions
		},
		RunE:         MirrorBinariesFunc(logger, &opt),
		SilenceUsage: true,
	}
	cmd.PersistentFlags().StringVar(&opt.Config, "config", "", "Path to the KubermaticConfiguration YAML file")
	cmd.PersistentFlags().StringVar(&opt.VersionFilter, "version-filter", "", "Version constraint (not used; all versions from the configuration are processed)")
	cmd.PersistentFlags().StringVar(&opt.OutputDir, "output-dir", opt.OutputDir, "Destination directory for binaries")
	cmd.PersistentFlags().StringVar(&opt.Architectures, "architectures", "amd64,arm64", "Comma-separated list of architectures to mirror binaries for (e.g., amd64,arm64)")
	return cmd
}

func getKubermaticConfigurationFromYaml(options *MirrorBinariesOptions) (*kubermaticv1.KubermaticConfiguration, error) {
	config, _, err := loadKubermaticConfiguration(options.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to load KubermaticConfiguration: %w", err)
	}
	if config == nil {
		return nil, errors.New("please specify your KubermaticConfiguration via --config")
	}
	kubermaticConfig, err := defaulting.DefaultConfiguration(config, zap.NewNop().Sugar())
	if err != nil {
		return nil, fmt.Errorf("failed to default KubermaticConfiguration: %w", err)
	}
	return kubermaticConfig, nil
}

func downloadFromURL(ctx context.Context, url, fileDownloadPath string) error {
	// Create a request with the provided context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %w", url, err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download file from url %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http error while downloading file from url %s: %s", url, resp.Status)
	}
	file, err := os.Create(fileDownloadPath)
	if err != nil {
		return fmt.Errorf("failed to create file at path %s: %w", fileDownloadPath, err)
	}
	defer file.Close()
	_, err = io.Copy(file, resp.Body)
	return err
}

// validateArchitectures checks if the given architectures are valid.
func validateArchitectures(archs string) ([]string, error) {
	validArchs := map[string]bool{"amd64": true, "arm64": true}
	archList := strings.Split(archs, ",")

	for _, arch := range archList {
		arch = strings.TrimSpace(arch)
		if !validArchs[arch] {
			return nil, fmt.Errorf("invalid architecture: %s (allowed: amd64, arm64)", arch)
		}
	}

	return archList, nil
}

func getChecksumFromURL(ctx context.Context, url string) (string, error) {
	// Create a request with the provided context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for %s: %w", url, err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve checksum from: %w", err)
	}
	defer resp.Body.Close()
	reader := bufio.NewReader(resp.Body)
	checksumLine, _, err := reader.ReadLine()
	if err != nil {
		return "", fmt.Errorf("failed to read checksum line: %w", err)
	}
	checksum := strings.Split(string(checksumLine), " ")[0]
	return checksum, nil
}

func getChecksumOfFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("unable to open file %s: %w", path, err)
	}
	defer file.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate checksum of file: %w", err)
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func verifyChecksum(ctx context.Context, checksumURL string, binaryFilePath string) error {
	expectedChecksum, err := getChecksumFromURL(ctx, checksumURL)
	if err != nil {
		return fmt.Errorf("error getting checksum from url %s: %w", checksumURL, err)
	}
	actualChecksum, err := getChecksumOfFile(binaryFilePath)
	if err != nil {
		return fmt.Errorf("error getting checksum of file %s: %w", binaryFilePath, err)
	}
	if expectedChecksum != actualChecksum {
		return fmt.Errorf("checksum verification failed for %s: expected %s, got %s", binaryFilePath, expectedChecksum, actualChecksum)
	}
	return nil
}

func getCriToolsRelease(version semverlib.Version) string {
	release := fmt.Sprintf("%d.%d", version.Major(), version.Minor())
	criToolsReleases := map[string]string{
		"1.32": "v1.32.0",
		"1.30": "v1.30.1",
	}
	if criToolRelease, ok := criToolsReleases[release]; ok {
		return criToolRelease
	}

	return "v1.32.0"
}

// downloadCRITools downloads the CRI tools tarball and its checksum for the given Kubernetes version.
func downloadCRITools(ctx context.Context, logger *logrus.Logger, version semverlib.Version, binPath, hostArch string) error {
	criToolsRelease := getCriToolsRelease(version)
	criToolsDir := filepath.Join(binPath, "kubernetes-sigs", "cri-tools", "releases", "download", criToolsRelease)

	// Ensure the directory exists and is empty
	if err := ensureCleanDir(criToolsDir); err != nil {
		return fmt.Errorf("failed to prepare CRI tools directory: %w", err)
	}

	logger.Debugf("⏳ Downloading CRI tools %s...", criToolsRelease)

	criToolsFileName := fmt.Sprintf("crictl-%s-linux-%s.tar.gz", criToolsRelease, hostArch)
	criToolsURL := fmt.Sprintf("%s/%s/%s", CRIToolsBaseURL, criToolsRelease, criToolsFileName)
	criToolsFilePath := filepath.Join(criToolsDir, criToolsFileName)

	// Download tarball
	if err := downloadFromURL(ctx, criToolsURL, criToolsFilePath); err != nil {
		return fmt.Errorf("failed to download CRI tools tarball (%s): %w", criToolsRelease, err)
	}

	// Download and save checksum file
	checksumFileName := criToolsFileName + SHA256Exentsion
	checksumURL := fmt.Sprintf("%s/%s/%s", CRIToolsBaseURL, criToolsRelease, checksumFileName)
	checksumFilePath := filepath.Join(criToolsDir, checksumFileName)

	// Doownload and verify checksum
	if err := downloadAndVerifyChecksum(ctx, checksumURL, checksumFilePath, criToolsFilePath); err != nil {
		return err
	}

	logger.Debugf("✔ Successfully downloaded CRI tools %s.", criToolsRelease)
	return nil
}

// downloadKubeBinaries downloads the kube binaries (kubelet, kubeadm, kubectl) for a given Kubernetes version.
func downloadKubeBinaries(ctx context.Context, logger *logrus.Logger, version *version.Version, binPath, hostArch string) error {
	kubeVersion := fmt.Sprintf("v%s", version.Version.String())
	versionPath := fmt.Sprintf(KubeBinaryPath, kubeVersion, hostArch)
	kubeDir := filepath.Join(binPath, versionPath)
	kubeBaseURL := fmt.Sprintf("%s/%s", KubeBaseURLFormat, versionPath)

	// Ensure kubeDir exists and is empty
	if err := ensureCleanDir(kubeDir); err != nil {
		return fmt.Errorf("failed to prepare kube directory: %w", err)
	}

	logger.Debugf("⏳ Downloading Kubernetes binaries for version %s...", kubeVersion)

	binaries := []string{"kubelet", "kubeadm", "kubectl"}
	for _, binary := range binaries {
		if err := downloadAndVerifyBinary(ctx, binary, kubeBaseURL, kubeDir); err != nil {
			return err
		}
	}

	logger.Debugf("✔ Successfully downloaded Kubernetes binaries for version %s.", kubeVersion)
	return nil
}

// downloadAndVerifyBinary downloads a binary, its checksum, verifies it, and makes it executable.
func downloadAndVerifyBinary(ctx context.Context, binary, baseURL, targetDir string) error {
	binaryURL := fmt.Sprintf("%s/%s", baseURL, binary)
	binaryPath := filepath.Join(targetDir, binary)

	if err := downloadFromURL(ctx, binaryURL, binaryPath); err != nil {
		return fmt.Errorf("failed to download %s: %w", binary, err)
	}

	checksumURL := binaryURL + SHA256Exentsion
	checksumPath := binaryPath + SHA256Exentsion

	// Doownload and verify checksum
	if err := downloadAndVerifyChecksum(ctx, checksumURL, checksumPath, binaryPath); err != nil {
		return err
	}

	return nil
}

// downloadCNIPlugins downloads the CNI plugins tarball and its checksum, then verifies the integrity.
func downloadCNIPlugins(ctx context.Context, logger *logrus.Logger, binPath, hostArch string) error {
	// Get the CNI plugins version from the environment or use default
	cniPluginsVersion := os.Getenv("CNI_VERSION")
	if cniPluginsVersion == "" {
		cniPluginsVersion = DefaultCNIPluginsVersion
	}

	// Define the target directory
	cniPluginsDir := filepath.Join(binPath, "containernetworking", "plugins", "releases", "download", cniPluginsVersion)

	// Ensure the directory exists
	if err := ensureCleanDir(cniPluginsDir); err != nil {
		return fmt.Errorf("failed to prepare CNI plugins directory: %w", err)
	}

	logger.Debugf("⏳ Downloading CNI plugins version %s...", cniPluginsVersion)

	// Define file names and URLs
	cniPluginsFileName := fmt.Sprintf("cni-plugins-linux-%s-%s.tgz", hostArch, cniPluginsVersion)
	cniPluginsURL := fmt.Sprintf("%s/%s/%s", CNIPluginsBaseURL, cniPluginsVersion, cniPluginsFileName)
	cniPluginsFilePath := filepath.Join(cniPluginsDir, cniPluginsFileName)

	// Download CNI plugins tarball
	if err := downloadFromURL(ctx, cniPluginsURL, cniPluginsFilePath); err != nil {
		return fmt.Errorf("failed to download CNI plugins tarball (%s): %w", cniPluginsVersion, err)
	}

	// Define checksum file paths
	checksumFileName := cniPluginsFileName + SHA256Exentsion
	checksumURL := fmt.Sprintf("%s/%s/%s", CNIPluginsBaseURL, cniPluginsVersion, checksumFileName)
	checksumFilePath := filepath.Join(cniPluginsDir, checksumFileName)

	// Download and Verify checksum
	if err := downloadAndVerifyChecksum(ctx, checksumURL, checksumFilePath, cniPluginsFilePath); err != nil {
		return err
	}

	logger.Debugf("✔ Successfully downloaded CNI plugins version %s.", cniPluginsVersion)
	return nil
}

// MirrorBinariesFunc is the main function for the mirror-binaries command.
func MirrorBinariesFunc(logger *logrus.Logger, options *MirrorBinariesOptions) cobraFuncE {
	return handleErrors(logger, func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Validate flag after parsing
		archList, err := validateArchitectures(options.Architectures)
		if err != nil {
			return fmt.Errorf("invalid architectures: %w", err)
		}

		kubermaticConfig, err := getKubermaticConfigurationFromYaml(options)
		if err != nil {
			return fmt.Errorf("failed to get KubermaticConfiguration: %w", err)
		}

		// Extract all Kubernetes versions from the configuration.
		versions, err := images.GetVersions(logger, kubermaticConfig, options.VersionFilter)
		if err != nil {
			return fmt.Errorf("failed to load versions: %w", err)
		}

		logger.Debugf("Found %d Kubernetes version(s) in the configuration.", len(versions))

		binPath := options.OutputDir

		for _, arch := range archList {
			logger.Infof("🚀 Starting mirroring for architecture: %s", arch)
			logger.Debugf("⏳ Starting CNI plugins download for %s...", arch)
			if err := downloadCNIPlugins(ctx, logger, binPath, arch); err != nil {
				return fmt.Errorf("failed to download CNI plugins for %s: %w", arch, err)
			}
			logger.Infof("✅ CNI plugins download complete for %s.", arch)

			logger.Debugf("⏳ Starting CRI tools download for all available Kubernetes versions (%s)...", arch)
			for _, version := range versions {
				if err := downloadCRITools(ctx, logger, *version.Version, binPath, arch); err != nil {
					return fmt.Errorf("failed to download CRI tools for Kubernetes version %s (%s): %w", version.Version, arch, err)
				}
			}
			logger.Infof("✅ CRI tools download complete for all available Kubernetes versions (%s).", arch)

			logger.Debugf("⏳ Starting kube binaries download for all available Kubernetes versions (%s)...", arch)
			for _, version := range versions {
				if err := downloadKubeBinaries(ctx, logger, version, binPath, arch); err != nil {
					return fmt.Errorf("failed to download kube binaries for Kubernetes version %s (%s): %w", version.Version, arch, err)
				}
			}
			logger.Infof("✅ Kube binaries download complete for all available Kubernetes versions (%s).", arch)
		}
		logger.Info("✅ Finished loading images.")

		return nil
	})
}

// Ensure the directory exists without deleting it if it already exists.
func ensureCleanDir(dir string) error {
	// Check if the directory already exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// Directory does not exist, create it
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	} else if err != nil {
		// Some other error occurred while checking the directory
		return fmt.Errorf("failed to check directory %s: %w", dir, err)
	}

	return nil
}

// downloadAndVerifyChecksum downloads the checksum from the specified URL and verifies the file's integrity.
func downloadAndVerifyChecksum(ctx context.Context, checksumURL, checksumPath, filePath string) error {
	if err := downloadFromURL(ctx, checksumURL, checksumPath); err != nil {
		return fmt.Errorf("failed to download checksum: %w", err)
	}

	if err := verifyChecksum(ctx, checksumURL, filePath); err != nil {
		return fmt.Errorf("failed to verify checksum for %s: %w", filePath, err)
	}

	return nil
}
