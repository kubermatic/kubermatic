package addons

import (
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/tiller"
	"k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	"k8s.io/helm/pkg/engine"
	"k8s.io/helm/pkg/tiller/environment"
	"k8s.io/helm/pkg/kube"
	"k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/helm/pkg/storage"
	"k8s.io/helm/pkg/storage/driver"
	"github.com/golang/glog"
)

type Interface interface {
	ListReleases() error //TODO: Proper return
	Install(name string) error
	Delete(rlsName string) error
	UpdateRelease(rlsName string) error
	RollbackRelease(rlsName string) error
}

func NewAddonManager(config clientcmd.ClientConfig) (*AddonManager, error){
	t, err := getTiller(config)
	if err != nil {
		return nil, err
	}
	h := helm.NewClient()
	return &AddonManager{
		tiller: t,
		helm: h,
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
	helm   *helm.Client
}

func (a *AddonManager) Install(name string) error {
	glog.Infof("Would install now '%s'", name)
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
