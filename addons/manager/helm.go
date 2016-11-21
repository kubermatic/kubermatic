package manager

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/kubermatic/api"
	"io"
	"k8s.io/helm/cmd/helm/downloader"
	"k8s.io/helm/cmd/helm/helmpath"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/engine"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/kube"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/services"
	"k8s.io/helm/pkg/repo"
	"k8s.io/helm/pkg/storage"
	"k8s.io/helm/pkg/storage/driver"
	"k8s.io/helm/pkg/tiller"
	"k8s.io/helm/pkg/tiller/environment"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	stableRepository    = "stable"
	stableRepositoryURL = "http://storage.googleapis.com/kubernetes-charts"

	localRepoIndexFilePath = "index.yaml"
)

// NewHelmAddonManager returns a addon manager instance for the given kubeconfig based on the helm package manager
func NewHelmAddonManager(config clientcmd.ClientConfig) (AddonManager, error) {
	err := ensureHome(helmpath.Home("/tmp"), os.Stdout)
	if err != nil {
		return nil, err
	}

	t, err := getTiller(config)
	if err != nil {
		return nil, err
	}
	return &HelmAddonManager{
		tiller: t,
	}, nil
}

// getTiller returns an instance of the tiller for the given kubeconfig
func getTiller(config clientcmd.ClientConfig) (*tiller.ReleaseServer, error) {
	e := engine.New()
	var ey environment.EngineYard = map[string]environment.Engine{environment.GoTplEngine: e}

	helmKubeClient := kube.Client{
		Factory:               util.NewFactory(config),
		IncludeThirdPartyAPIs: true,
	}

	env := &environment.Environment{
		EngineYard: ey,
		Releases:   storage.Init(driver.NewMemory()),
		KubeClient: &helmKubeClient,
	}

	c, err := env.KubeClient.APIClient()
	if err != nil {
		return nil, err
	}
	env.Releases = storage.Init(driver.NewConfigMaps(c.ConfigMaps(environment.TillerNamespace)))

	return tiller.NewReleaseServer(env), nil
}

// HelmAddonManager represents a addon manager based on kubernetes/helm
type HelmAddonManager struct {
	tiller *tiller.ReleaseServer
}

// Install installs a given addon to the cluster
func (a *HelmAddonManager) Install(addon *api.ClusterAddon) (*api.ClusterAddon, error) {
	c, err := getChart(fmt.Sprintf("stable/%s", addon.Name), "")
	if err != nil {
		return nil, err
	}

	req := services.InstallReleaseRequest{}
	req.Chart = c
	req.Namespace = addon.Metadata.Namespace
	req.Values = &chart.Config{Raw: ""}
	ctx := helm.NewContext()

	res, err := a.tiller.InstallRelease(ctx, &req)

	if err != nil {
		return nil, err
	}

	addon.Version = res.Release.Version
	addon.Deployed = time.Unix(res.Release.Info.GetFirstDeployed().Seconds, 0)
	addon.ReleaseName = res.Release.Name

	return addon, nil
}

// ListReleases lists all installed releases on the cluster
func (a *HelmAddonManager) ListReleases() error {
	return nil
}

// Delete will delete a installed addon from the luster
func (a *HelmAddonManager) Delete(addon *api.ClusterAddon) error {
	req := &services.UninstallReleaseRequest{}
	req.Name = addon.ReleaseName
	ctx := helm.NewContext()

	_, err := a.tiller.UninstallRelease(ctx, req)
	if err != nil {
		return err
	}

	return nil
}

// Update will update a installed addon in the cluster
func (a *HelmAddonManager) Update(addon *api.ClusterAddon) error {
	return nil
}

// Rollback will rollback a release to the previous release
func (a *HelmAddonManager) Rollback(addon *api.ClusterAddon) error {
	return nil
}

// getChart will download and return a chart for the given name
func getChart(name, version string) (*chart.Chart, error) {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)

	dl := downloader.ChartDownloader{
		HelmHome: helmpath.Home("/tmp"),
		Out:      os.Stdout,
	}

	filename, _, err := dl.DownloadTo(name, version, ".")
	if err != nil {
		return nil, err
	}

	absName, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}

	c, err := chartutil.Load(absName)
	if err != nil {
		return nil, err
	}

	return c, nil
}

// From vendor/k8s.io/helm/cmd/helm/init.go:145
func ensureHome(home helmpath.Home, out io.Writer) error {
	configDirectories := []string{home.String(), home.Repository(), home.Cache(), home.LocalRepository()}
	for _, p := range configDirectories {
		if fi, err := os.Stat(p); err != nil {
			glog.V(4).Infof("Creating directory for addon manager: '%s'", p)
			if err := os.MkdirAll(p, 0755); err != nil {
				return fmt.Errorf("Could not create %s: %s", p, err)
			}
		} else if !fi.IsDir() {
			return fmt.Errorf("%s must be a directory", p)
		}
	}

	repoFile := home.RepositoryFile()
	if fi, err := os.Stat(repoFile); err != nil {
		glog.V(4).Infof("Creating repository file: '%s'", repoFile)
		r := repo.NewRepoFile()
		r.Add(&repo.Entry{
			Name:  stableRepository,
			URL:   stableRepositoryURL,
			Cache: "stable-index.yaml",
		})
		if err := r.WriteFile(repoFile, 0644); err != nil {
			return err
		}
		cif := home.CacheIndex(stableRepository)
		if err := repo.DownloadIndexFile(stableRepository, stableRepositoryURL, cif); err != nil {
			glog.Errorf("Failed to download %s: %s", stableRepository, err)
		}
	} else if fi.IsDir() {
		return fmt.Errorf("%s must be a file, not a directory", repoFile)
	}
	if r, err := repo.LoadRepositoriesFile(repoFile); err == repo.ErrRepoOutOfDate {
		if err := r.WriteFile(repoFile, 0644); err != nil {
			return err
		}
	}

	localRepoIndexFile := home.LocalRepository(localRepoIndexFilePath)
	if fi, err := os.Stat(localRepoIndexFile); err != nil {
		glog.V(4).Infof("Creating repository index file'%s'", localRepoIndexFile)
		i := repo.NewIndexFile()
		if err := i.WriteFile(localRepoIndexFile, 0644); err != nil {
			return err
		}

		err = os.Symlink(localRepoIndexFile, home.CacheIndex("local"))
		if err != nil {
			return err
		}
	} else if fi.IsDir() {
		return fmt.Errorf("%s must be a file, not a directory", localRepoIndexFile)
	}

	glog.V(4).Infof("Folder structure for addon manager created under %s", home.String())
	return nil
}
