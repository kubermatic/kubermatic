package addons

import (
	"errors"
	"fmt"
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
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	stableRepository    = "stable"
	localRepository     = "local"
	stableRepositoryURL = "http://storage.googleapis.com/kubernetes-charts"
	// This is the IPv4 loopback, not localhost, because we have to force IPv4
	// for Dockerized Helm: https://github.com/kubernetes/helm/issues/1410
	localRepositoryURL = "http://127.0.0.1:8879/charts"
)

const (
	localRepoIndexFilePath = "index.yaml"
)

var helmHome string = "/opt/helm"

type Interface interface {
	ListReleases() error //TODO: Proper return
	Install(name string) error
	Delete(rlsName string) error
	UpdateRelease(rlsName string) error
	RollbackRelease(rlsName string) error
}

func NewAddonManager(config clientcmd.ClientConfig) (*AddonManager, error) {
	err := ensureHome(helmpath.Home(homePath()), os.Stdout)
	if err != nil {
		return nil, err
	}

	t, err := getTiller(config)
	if err != nil {
		return nil, err
	}
	return &AddonManager{
		tiller: t,
	}, nil
}

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

type AddonManager struct {
	tiller *tiller.ReleaseServer
}

func (a *AddonManager) Install(name string) error {
	cp, err := locateChartPath(fmt.Sprintf("stable/%s", name), "", false, "/home/henrik/.gnupg/pubring.gpg")
	if err != nil {
		return err
	}
	log.Println("Downloaded chart to " + cp)

	// load the chart to install
	c, err := chartutil.Load(cp)
	if err != nil {
		return err
	}
	req := services.InstallReleaseRequest{}
	req.Chart = c
	req.Namespace = "default"
	req.Values = &chart.Config{Raw: ""}
	ctx := helm.NewContext()

	_, err = a.tiller.InstallRelease(ctx, &req)

	if err != nil {
		fmt.Println(err)
	}

	return nil
}

func (a *AddonManager) ListReleases() error {
	return nil
}

func (a *AddonManager) Delete(rlsName string) error {
	return nil
}
func (a *AddonManager) UpdateRelease(rlsName string) error {
	return nil
}
func (a *AddonManager) RollbackRelease(rlsName string) error {
	return nil
}

// locateChartPath looks for a chart directory in known places, and returns either the full path or an error.
//
// This does not ensure that the chart is well-formed; only that the requested filename exists.
//
// Order of resolution:
// - current working directory
// - if path is absolute or begins with '.', error out here
// - chart repos in $HELM_HOME
// - URL
//
// If 'verify' is true, this will attempt to also verify the chart.
func locateChartPath(name, version string, verify bool, keyring string) (string, error) {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	fmt.Println("locateChartPath.name: " + name)
	fmt.Println("locateChartPath.version" + version)
	fmt.Println("locateChartPath.verify" + fmt.Sprintf("%s", verify))
	fmt.Println("locateChartPath.keyring" + keyring)

	if fi, err := os.Stat(name); err == nil {
		abs, err := filepath.Abs(name)
		if err != nil {
			return abs, err
		}
		if verify {
			if fi.IsDir() {
				return "", errors.New("cannot verify a directory")
			}
			if _, err := downloader.VerifyChart(abs, keyring); err != nil {
				return "", err
			}
		}
		return abs, nil
	}
	if filepath.IsAbs(name) || strings.HasPrefix(name, ".") {
		return name, fmt.Errorf("path %q not found", name)
	}

	crepo := filepath.Join(helmpath.Home(homePath()).Repository(), name)
	fmt.Println("crepo: " + crepo)
	if _, err := os.Stat(crepo); err == nil {
		return filepath.Abs(crepo)
	}

	fmt.Println("helmpath.Home(homePath()): " + helmpath.Home(homePath()))
	fmt.Println("keyring: " + keyring)

	dl := downloader.ChartDownloader{
		HelmHome: helmpath.Home(homePath()),
		Out:      os.Stdout,
		Keyring:  keyring,
	}
	if verify {
		dl.Verify = downloader.VerifyAlways
	}

	filename, _, err := dl.DownloadTo(name, version, ".")
	fmt.Println("filename: " + filename)
	if err != nil {
		fmt.Println(err)
	}
	if err == nil {
		lname, err := filepath.Abs(filename)
		if err != nil {
			return filename, err
		}
		fmt.Printf("Fetched %s to %s\n", name, filename)
		return lname, nil
	}

	return filename, fmt.Errorf("file %q not found", name)
}

func homePath() string {
	return helmHome
}

// ensureHome checks to see if $HELM_HOME exists
//
// If $HELM_HOME does not exist, this function will create it.
func ensureHome(home helmpath.Home, out io.Writer) error {
	configDirectories := []string{home.String(), home.Repository(), home.Cache(), home.LocalRepository()}
	for _, p := range configDirectories {
		if fi, err := os.Stat(p); err != nil {
			fmt.Fprintf(out, "Creating %s \n", p)
			if err := os.MkdirAll(p, 0755); err != nil {
				return fmt.Errorf("Could not create %s: %s", p, err)
			}
		} else if !fi.IsDir() {
			return fmt.Errorf("%s must be a directory", p)
		}
	}

	repoFile := home.RepositoryFile()
	if fi, err := os.Stat(repoFile); err != nil {
		fmt.Fprintf(out, "Creating %s \n", repoFile)
		r := repo.NewRepoFile()
		r.Add(&repo.Entry{
			Name:  stableRepository,
			URL:   stableRepositoryURL,
			Cache: "stable-index.yaml",
		}, &repo.Entry{
			Name:  localRepository,
			URL:   localRepositoryURL,
			Cache: "local-index.yaml",
		})
		if err := r.WriteFile(repoFile, 0644); err != nil {
			return err
		}
		cif := home.CacheIndex(stableRepository)
		if err := repo.DownloadIndexFile(stableRepository, stableRepositoryURL, cif); err != nil {
			fmt.Fprintf(out, "WARNING: Failed to download %s: %s (run 'helm repo update')\n", stableRepository, err)
		}
	} else if fi.IsDir() {
		return fmt.Errorf("%s must be a file, not a directory", repoFile)
	}
	if r, err := repo.LoadRepositoriesFile(repoFile); err == repo.ErrRepoOutOfDate {
		fmt.Fprintln(out, "Updating repository file format...")
		if err := r.WriteFile(repoFile, 0644); err != nil {
			return err
		}
	}

	localRepoIndexFile := home.LocalRepository(localRepoIndexFilePath)
	if fi, err := os.Stat(localRepoIndexFile); err != nil {
		fmt.Fprintf(out, "Creating %s \n", localRepoIndexFile)
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

	fmt.Fprintf(out, "$HELM_HOME has been configured at %s.\n", helmHome)
	return nil
}
