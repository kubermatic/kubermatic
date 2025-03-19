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
	"runtime"
	"strings"
	"time"

	semverlib "github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
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

	// Default output directory for binaries.
	DefaultOutputDir = "/usr/share/nginx/html/"
)

// MirrorBinariesOptions holds options for the mirror-binaries command.
type MirrorBinariesOptions struct {
	Config        string
	Versions      kubermaticversion.Versions // Not used for extraction.
	VersionFilter string                     // Ignored in our extraction logic.
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

func downloadFromUrl(ctx context.Context, url, fileDownloadPath string) error {
	// Create an HTTP client with a timeout.
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Create a request with the provided context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request for %s: %w", url, err)
	}

	resp, err := client.Do(req)
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

func getHostArchitecture() (string, error) {
	arch := os.Getenv("HOST_ARCH")
	if arch == "" {
		switch runtime.GOARCH {
		case "amd64":
			arch = "amd64"
		case "arm64":
			arch = "arm64"
		default:
			return "", fmt.Errorf("unsupported CPU architecture: %s", runtime.GOARCH)
		}
	}
	return arch, nil
}

func getChecksumFromURL(ctx context.Context, url string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	// Create a request with the provided context
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request for %s: %w", url, err)
	}

	resp, err := client.Do(req)
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

func verifyChecksum(ctx context.Context, checksumUrl string, binaryFilePath string) error {
	expectedChecksum, err := getChecksumFromURL(ctx, checksumUrl)
	if err != nil {
		return fmt.Errorf("error getting checksum from url %s: %w", checksumUrl, err)
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

func getCriToolsRelease(version semverlib.Version) (string, error) {
	release := fmt.Sprintf("%d.%d", version.Major(), version.Minor())
	var criToolsReleases = map[string]string{
		"1.32": "v1.32.0",
		"1.31": "v1.31.1",
		"1.30": "v1.30.1",
		"1.29": "v1.29.0",
		"1.28": "v1.28.0",
		"1.27": "v1.27.1",
		"1.26": "v1.26.1",
		"1.25": "v1.25.0",
		"1.24": "v1.24.2",
	}
	if criToolRelease, ok := criToolsReleases[release]; ok {
		return criToolRelease, nil
	}

	return "v1.32.0", nil
}

// downloadCriTools downloads the CRI tools tarball and its checksum for the given Kubernetes version.
func downloadCriTools(ctx context.Context, logger *logrus.Logger, version semverlib.Version, binPath, hostArch string) error {
	criToolsRelease, err := getCriToolsRelease(version)
	if err != nil {
		return fmt.Errorf("failed to get CRI tools release for Kubernetes version %s: %w", version, err)
	}

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
	if err := downloadFromUrl(ctx, criToolsURL, criToolsFilePath); err != nil {
		return fmt.Errorf("failed to download CRI tools tarball (%s): %w", criToolsRelease, err)
	}

	// Download and save checksum file
	checksumFileName := criToolsFileName + ".sha256"
	checksumURL := fmt.Sprintf("%s/%s/%s", CRIToolsBaseURL, criToolsRelease, checksumFileName)
	checksumFilePath := filepath.Join(criToolsDir, checksumFileName)

	if err := downloadFromUrl(ctx, checksumURL, checksumFilePath); err != nil {
		return fmt.Errorf("failed to download CRI tools checksum file (%s): %w", criToolsRelease, err)
	}

	// Verify checksum
	if err := verifyChecksum(ctx, checksumURL, criToolsFilePath); err != nil {
		return fmt.Errorf("failed to verify CRI tools tarball checksum (%s): %w", criToolsRelease, err)
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

	if err := downloadFromUrl(ctx, binaryURL, binaryPath); err != nil {
		return fmt.Errorf("failed to download %s: %w", binary, err)
	}

	checksumURL := fmt.Sprintf("%s.sha256", binaryURL)
	checksumPath := binaryPath + ".sha256"

	if err := downloadFromUrl(ctx, checksumURL, checksumPath); err != nil {
		return fmt.Errorf("failed to download checksum for %s: %w", binary, err)
	}

	if err := verifyChecksum(ctx, checksumURL, binaryPath); err != nil {
		return fmt.Errorf("failed to verify checksum for %s: %w", binary, err)
	}

	return nil
}

// downloadCniPlugins downloads the CNI plugins tarball and its checksum, then verifies the integrity.
func downloadCniPlugins(ctx context.Context, logger *logrus.Logger, binPath, hostArch string) error {
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
	if err := downloadFromUrl(ctx, cniPluginsURL, cniPluginsFilePath); err != nil {
		return fmt.Errorf("failed to download CNI plugins tarball (%s): %w", cniPluginsVersion, err)
	}

	// Define checksum file paths
	checksumFileName := cniPluginsFileName + ".sha256"
	checksumURL := fmt.Sprintf("%s/%s/%s", CNIPluginsBaseURL, cniPluginsVersion, checksumFileName)
	checksumFilePath := filepath.Join(cniPluginsDir, checksumFileName)

	// Download and save checksum file
	if err := downloadFromUrl(ctx, checksumURL, checksumFilePath); err != nil {
		return fmt.Errorf("failed to download CNI plugins checksum file (%s): %w", cniPluginsVersion, err)
	}

	// Verify checksum using the downloaded checksum file
	if err := verifyChecksum(ctx, checksumURL, cniPluginsFilePath); err != nil {
		return fmt.Errorf("failed to verify CNI plugins tarball checksum (%s): %w", cniPluginsVersion, err)
	}

	logger.Debugf("✔ Successfully downloaded CNI plugins version %s.", cniPluginsVersion)
	return nil
}

// MirrorBinariesFunc is the main function for the mirror-binaries command.
func MirrorBinariesFunc(logger *logrus.Logger, options *MirrorBinariesOptions) cobraFuncE {
	return handleErrors(logger, func(cmd *cobra.Command, args []string) error {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

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

		hostArch, err := getHostArchitecture()
		if err != nil {
			return fmt.Errorf("failed to get host architecture: %w", err)
		}

		binPath := options.OutputDir
		logger.Info("🚀 Starting mirroring the binaries")

		logger.Debugf("🚀 Starting CNI plugins download...")
		if err := downloadCniPlugins(ctx, logger, binPath, hostArch); err != nil {
			return fmt.Errorf("failed to download CNI plugins: %w", err)
		}
		logger.Infof("✅ CNI plugins download complete.")

		logger.Debugf("🚀 Starting CRI tools download for all available Kubernetes versions...")
		for _, version := range versions {
			if err := downloadCriTools(ctx, logger, *version.Version, binPath, hostArch); err != nil {
				return fmt.Errorf("failed to download CRI tools for Kubernetes version %s: %w", version.Version, err)
			}
		}
		logger.Infof("✅ CRI tools download complete for all available Kubernetes versions.")

		logger.Debugf("🚀 Starting kube binaries download for all available Kubernetes versions...")
		for _, version := range versions {
			if err := downloadKubeBinaries(ctx, logger, version, binPath, hostArch); err != nil {
				return fmt.Errorf("failed to download kube binaries for Kubernetes version %s: %w", version.Version, err)
			}
		}
		logger.Infof("✅ Kube binaries download complete for all available Kubernetes versions.")
		logger.Info("✅ Finished loading images.")

		return nil
	})
}

// Ensure clean directory.
func ensureCleanDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("failed to clean directory %s: %w", dir, err)
	}
	return os.MkdirAll(dir, 0755)
}
