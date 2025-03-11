package main

import (
	"bufio"
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

	"github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/defaulting"

	kubermaticversion "k8c.io/kubermatic/v2/pkg/version/kubermatic"
)

// Minimal CLIOptions definition to satisfy dependencies.

type CLIOptions struct{}

func (o CLIOptions) CopyInto(target *CLIOptions) {
	// No-op implementation.
}

// ClusterVersion is a minimal local type representing a Kubernetes version.
type ClusterVersion struct {
	Version *semver.Version
}

// MirrorBinariesOptions holds options for the mirror-binaries command.
type MirrorBinariesOptions struct {
	CLIOptions

	Config        string
	Versions      kubermaticversion.Versions // Not used for extraction.
	VersionFilter string                     // Ignored in our extraction logic.
	// Destination directory for binaries.
	OutputDir string
}

func MirrorBinariesCommand(logger *logrus.Logger, versions kubermaticversion.Versions) *cobra.Command {
	opt := MirrorBinariesOptions{
		OutputDir: "/usr/share/nginx/html/binaries/",
	}
	cmd := &cobra.Command{
		Use:   "mirror-binaries",
		Short: "Mirror binaries used by KKP",
		Long:  "Downloads all binaries used by KKP and copies them into a local path.",
		PreRun: func(cmd *cobra.Command, args []string) {
			opt.CLIOptions.CopyInto(&opt.CLIOptions)
			if opt.Config == "" {
				opt.Config = os.Getenv("CONFIG_YAML")
			}
			if len(args) >= 1 {
				opt.Config = args[0]
			}
			// Versions parameter is still passed but our extraction function will rely on the configuration.
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

// getAllKubernetesVersions extracts all Kubernetes versions from config.Spec.Versions.Versions
// and returns them as a slice of *ClusterVersion.
func getAllKubernetesVersions(config *kubermaticv1.KubermaticConfiguration) ([]*ClusterVersion, error) {
	if config.Spec.Versions.Versions == nil || len(config.Spec.Versions.Versions) == 0 {
		return nil, errors.New("no Kubernetes versions defined in KubermaticConfiguration.spec.versions.versions")
	}
	var versions []*ClusterVersion
	for _, verVal := range config.Spec.Versions.Versions {
		vStr := fmt.Sprintf("%v", verVal)
		clean := strings.TrimPrefix(vStr, "v")
		ver, err := semver.NewVersion(clean)
		if err != nil {
			return nil, fmt.Errorf("failed to parse version %s: %w", vStr, err)
		}
		versions = append(versions, &ClusterVersion{Version: ver})
	}
	return versions, nil
}

func checkIfDirExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("%w", err)
	}
	return true, nil
}

func createDir(path string) error {
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

func checkIfDirEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, fmt.Errorf("%w", err)
	}
	return len(entries) == 0, nil
}

func makeFileExecutable(path string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	newPermissions := fileInfo.Mode() | 0111
	if err := os.Chmod(path, newPermissions); err != nil {
		return fmt.Errorf("failed to set execute permissions: %w", err)
	}
	return nil
}

func downloadFromUrl(url, fileDownloadPath string) error {
	resp, err := http.Get(url)
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

func getChecksumFromURL(url string) (string, error) {
	resp, err := http.Get(url)
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

func verifyChecksum(checksumUrl string, binaryFilePath string) error {
	expectedChecksum, err := getChecksumFromURL(checksumUrl)
	if err != nil {
		return fmt.Errorf("error getting checksum from url %s: %w", checksumUrl, err)
	}
	actualChecksum, err := getChecksumOfFile(binaryFilePath)
	if err != nil {
		return fmt.Errorf("error getting checksum of file %s: %w", binaryFilePath, err)
	}
	if expectedChecksum != actualChecksum {
		return fmt.Errorf("checksum verification failed for %s", binaryFilePath)
	}
	return nil
}

func verifyChecksumFile(checksumFilePath, binaryFilePath string) error {
	data, err := os.ReadFile(checksumFilePath)
	if err != nil {
		return fmt.Errorf("failed to read checksum file %s: %w", checksumFilePath, err)
	}
	expectedChecksum := strings.Split(string(data), " ")[0]
	actualChecksum, err := getChecksumOfFile(binaryFilePath)
	if err != nil {
		return fmt.Errorf("error calculating checksum for file %s: %w", binaryFilePath, err)
	}
	if expectedChecksum != actualChecksum {
		return fmt.Errorf("checksum verification failed for %s", binaryFilePath)
	}
	return nil
}

// getCriToolsRelease returns the CRI tools release tag based on the provided Kubernetes version.
func getCriToolsRelease(version string) (string, error) {
	newVersion, err := semver.NewVersion(version)
	if err != nil {
		return "", fmt.Errorf("invalid semantic version: %w", err)
	}
	release := fmt.Sprintf("%d.%d", newVersion.Major(), newVersion.Minor())
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
	// Default fallback.
	return "v1.32.0", nil
}

// downloadCniPlugins downloads the CNI plugins tarball and verifies its checksum.

func downloadCniPlugins(logger *logrus.Logger, binPath, hostArch string) error {
	cniPluginsVersion := os.Getenv("CNI_VERSION")
	if cniPluginsVersion == "" {
		cniPluginsVersion = "v1.5.1"
	}
	cniPluginsDir := filepath.Join(binPath, "containernetworking", "plugins", "releases", "download", cniPluginsVersion)
	if err := createDir(cniPluginsDir); err != nil {
		return fmt.Errorf("failed to create CNI plugins directory: %w", err)
	}
	logger.Infof("⏳ Downloading CNI plugins %s...", cniPluginsVersion)
	cniPluginsBaseUrl := "https://github.com/containernetworking/plugins/releases/download"
	cniPluginsFileName := fmt.Sprintf("cni-plugins-linux-%s-%s.tgz", hostArch, cniPluginsVersion)
	cniPluginsUrl := fmt.Sprintf("%s/%s/%s", cniPluginsBaseUrl, cniPluginsVersion, cniPluginsFileName)
	cniPluginsFilePath := filepath.Join(cniPluginsDir, cniPluginsFileName)
	if err := downloadFromUrl(cniPluginsUrl, cniPluginsFilePath); err != nil {
		return fmt.Errorf("failed to download CNI plugins tarball (%s): %w", cniPluginsVersion, err)
	}
	cniPluginsChecksumFileName := fmt.Sprintf("%s.sha256", cniPluginsFileName)
	cniPluginsChecksumUrl := fmt.Sprintf("%s/%s/%s", cniPluginsBaseUrl, cniPluginsVersion, cniPluginsChecksumFileName)
	if err := verifyChecksum(cniPluginsChecksumUrl, cniPluginsFilePath); err != nil {
		return fmt.Errorf("failed to verify CNI plugins tarball checksum (%s): %w", cniPluginsVersion, err)
	}
	logger.Infof("✔ Downloaded CNI plugins tarball %s.", cniPluginsVersion)
	return nil
}

// downloadCriTools downloads the CRI tools tarball for the given Kubernetes version.

func downloadCriTools(logger *logrus.Logger, version, binPath, hostArch string) error {
	criToolsRelease, err := getCriToolsRelease(version)
	if err != nil {
		return fmt.Errorf("failed to get CRI tools release for Kubernetes version %s: %w", version, err)
	}
	criToolsDir := filepath.Join(binPath, "kubernetes-sigs", "cri-tools", "releases", "download", criToolsRelease)
	exists, err := checkIfDirExists(criToolsDir)
	if err != nil {
		return err
	}
	if !exists {
		if err := createDir(criToolsDir); err != nil {
			return fmt.Errorf("failed to create CRI tools directory: %w", err)
		}
	} else {
		empty, err := checkIfDirEmpty(criToolsDir)
		if err != nil {
			return err
		}
		if !empty {
			// Already downloaded.
			return nil
		}
	}
	logger.Infof("⏳ Downloading CRI tools %s...", criToolsRelease)
	criToolsBaseUrl := "https://github.com/kubernetes-sigs/cri-tools/releases/download"
	criToolsFileName := fmt.Sprintf("crictl-%s-linux-%s.tar.gz", criToolsRelease, hostArch)
	criToolsUrl := fmt.Sprintf("%s/%s/%s", criToolsBaseUrl, criToolsRelease, criToolsFileName)
	criToolsFilePath := filepath.Join(criToolsDir, criToolsFileName)
	if err := downloadFromUrl(criToolsUrl, criToolsFilePath); err != nil {
		return fmt.Errorf("failed to download CRI tools tarball (%s): %w", criToolsRelease, err)
	}
	criToolsChecksumFileName := fmt.Sprintf("%s.sha256", criToolsFileName)
	criToolsChecksumUrl := fmt.Sprintf("%s/%s/%s", criToolsBaseUrl, criToolsRelease, criToolsChecksumFileName)
	if err := verifyChecksum(criToolsChecksumUrl, criToolsFilePath); err != nil {
		return fmt.Errorf("failed to verify CRI tools tarball checksum (%s): %w", criToolsRelease, err)
	}
	logger.Infof("✔ Downloaded CRI tools tarball %s.", criToolsRelease)
	return nil
}

// downloadKubeBinaries downloads the kube binaries (kubelet, kubeadm, kubectl) for a given Kubernetes version.
// The binaries are downloaded into the folder: kubernetes-$KUBE_VERSION

func downloadKubeBinaries(logger *logrus.Logger, version, binPath, hostArch string) error {
	kubeVersion := fmt.Sprintf("v%s", version)

	kubeDir := filepath.Join(binPath, fmt.Sprintf("kubernetes-%s", kubeVersion))
	exists, err := checkIfDirExists(kubeDir)
	if err != nil {
		return err
	}
	if !exists {
		if err := createDir(kubeDir); err != nil {
			return fmt.Errorf("failed to create kube directory: %w", err)
		}
	} else {
		empty, err := checkIfDirEmpty(kubeDir)
		if err != nil {
			return err
		}
		if !empty {
			// Already downloaded.
			return nil
		}
	}
	logger.Infof("⏳ Downloading kube binaries %s...", kubeVersion)
	kubeBaseUrl := fmt.Sprintf("https://dl.k8s.io/release/%s/bin/linux/%s", kubeVersion, hostArch)
	binaries := []string{"kubelet", "kubeadm", "kubectl"}
	for _, binary := range binaries {
		binaryURL := fmt.Sprintf("%s/%s", kubeBaseUrl, binary)
		binaryPath := filepath.Join(kubeDir, binary)
		if err := downloadFromUrl(binaryURL, binaryPath); err != nil {
			return fmt.Errorf("failed to download %s: %w", binary, err)
		}
		checksumURL := fmt.Sprintf("%s.sha256", binaryURL)
		checksumFilePath := binaryPath + ".sha256"
		if err := downloadFromUrl(checksumURL, checksumFilePath); err != nil {
			return fmt.Errorf("failed to download checksum for %s: %w", binary, err)
		}
		if err := verifyChecksum(checksumURL, binaryPath); err != nil {
			return fmt.Errorf("failed to verify %s checksum: %w", binary, err)
		}
		if err := makeFileExecutable(binaryPath); err != nil {
			return fmt.Errorf("failed to make %s executable: %w", binary, err)
		}
	}
	logger.Infof("✔ Downloaded kube binaries %s.", kubeVersion)
	return nil
}

// MirrorBinariesFunc is the main function for the mirror-binaries command.
func MirrorBinariesFunc(logger *logrus.Logger, options *MirrorBinariesOptions) cobraFuncE {
	return handleErrors(logger, func(cmd *cobra.Command, args []string) error {
		kubermaticConfig, err := getKubermaticConfigurationFromYaml(options)
		if err != nil {
			return fmt.Errorf("failed to get KubermaticConfiguration: %w", err)
		}

		// Extract all Kubernetes versions from the configuration.
		clusterVersions, err := getAllKubernetesVersions(kubermaticConfig)
		if err != nil {
			return fmt.Errorf("failed to extract Kubernetes versions: %w", err)
		}
		logger.Infof("Found %d Kubernetes version(s) in the configuration.", len(clusterVersions))

		hostArch, err := getHostArchitecture()
		if err != nil {
			return fmt.Errorf("failed to get host architecture: %w", err)
		}

		binPath := options.OutputDir

		logger.Info("🚀 Starting CNI plugins download...")
		if err := downloadCniPlugins(logger, binPath, hostArch); err != nil {
			return fmt.Errorf("failed to download CNI plugins: %w", err)
		}
		logger.Info("✅ CNI plugins download complete.")

		logger.Info("🚀 Starting CRI tools download for all available Kubernetes versions...")
		for _, clusterVersion := range clusterVersions {
			version := clusterVersion.Version.String()
			if err := downloadCriTools(logger, version, binPath, hostArch); err != nil {
				return fmt.Errorf("failed to download CRI tools for Kubernetes version %s: %w", version, err)
			}
		}
		logger.Info("✅ CRI tools download complete for all available Kubernetes versions.")

		logger.Info("🚀 Starting kube binaries download for all available Kubernetes versions...")
		for _, clusterVersion := range clusterVersions {
			version := clusterVersion.Version.String()
			if err := downloadKubeBinaries(logger, version, binPath, hostArch); err != nil {
				return fmt.Errorf("failed to download kube binaries for Kubernetes version %s: %w", version, err)
			}
		}
		logger.Info("✅ Kube binaries download complete for all available Kubernetes versions.")

		return nil
	})
}
