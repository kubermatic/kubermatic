// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"k8c.io/kubermatic/v2/pkg/crd/client/clientset/versioned/scheme"
	v1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	rest "k8s.io/client-go/rest"
)

type KubermaticV1Interface interface {
	RESTClient() rest.Interface
	AddonsGetter
	AddonConfigsGetter
	AlertmanagersGetter
	AllowedRegistriesGetter
	ClustersGetter
	ClusterTemplatesGetter
	ClusterTemplateInstancesGetter
	ConstraintsGetter
	ConstraintTemplatesGetter
	EtcdBackupConfigsGetter
	EtcdRestoresGetter
	ExternalClustersGetter
	KubermaticSettingsGetter
	MLAAdminSettingsGetter
	ProjectsGetter
	RuleGroupsGetter
	UsersGetter
	UserProjectBindingsGetter
	UserSSHKeysGetter
}

// KubermaticV1Client is used to interact with features provided by the kubermatic.k8s.io group.
type KubermaticV1Client struct {
	restClient rest.Interface
}

func (c *KubermaticV1Client) Addons(namespace string) AddonInterface {
	return newAddons(c, namespace)
}

func (c *KubermaticV1Client) AddonConfigs() AddonConfigInterface {
	return newAddonConfigs(c)
}

func (c *KubermaticV1Client) Alertmanagers(namespace string) AlertmanagerInterface {
	return newAlertmanagers(c, namespace)
}

func (c *KubermaticV1Client) AllowedRegistries() AllowedRegistryInterface {
	return newAllowedRegistries(c)
}

func (c *KubermaticV1Client) Clusters() ClusterInterface {
	return newClusters(c)
}

func (c *KubermaticV1Client) ClusterTemplates() ClusterTemplateInterface {
	return newClusterTemplates(c)
}

func (c *KubermaticV1Client) ClusterTemplateInstances() ClusterTemplateInstanceInterface {
	return newClusterTemplateInstances(c)
}

func (c *KubermaticV1Client) Constraints(namespace string) ConstraintInterface {
	return newConstraints(c, namespace)
}

func (c *KubermaticV1Client) ConstraintTemplates() ConstraintTemplateInterface {
	return newConstraintTemplates(c)
}

func (c *KubermaticV1Client) EtcdBackupConfigs(namespace string) EtcdBackupConfigInterface {
	return newEtcdBackupConfigs(c, namespace)
}

func (c *KubermaticV1Client) EtcdRestores(namespace string) EtcdRestoreInterface {
	return newEtcdRestores(c, namespace)
}

func (c *KubermaticV1Client) ExternalClusters() ExternalClusterInterface {
	return newExternalClusters(c)
}

func (c *KubermaticV1Client) KubermaticSettings() KubermaticSettingInterface {
	return newKubermaticSettings(c)
}

func (c *KubermaticV1Client) MLAAdminSettings(namespace string) MLAAdminSettingInterface {
	return newMLAAdminSettings(c, namespace)
}

func (c *KubermaticV1Client) Projects() ProjectInterface {
	return newProjects(c)
}

func (c *KubermaticV1Client) RuleGroups(namespace string) RuleGroupInterface {
	return newRuleGroups(c, namespace)
}

func (c *KubermaticV1Client) Users() UserInterface {
	return newUsers(c)
}

func (c *KubermaticV1Client) UserProjectBindings() UserProjectBindingInterface {
	return newUserProjectBindings(c)
}

func (c *KubermaticV1Client) UserSSHKeys() UserSSHKeyInterface {
	return newUserSSHKeys(c)
}

// NewForConfig creates a new KubermaticV1Client for the given config.
func NewForConfig(c *rest.Config) (*KubermaticV1Client, error) {
	config := *c
	if err := setConfigDefaults(&config); err != nil {
		return nil, err
	}
	client, err := rest.RESTClientFor(&config)
	if err != nil {
		return nil, err
	}
	return &KubermaticV1Client{client}, nil
}

// NewForConfigOrDie creates a new KubermaticV1Client for the given config and
// panics if there is an error in the config.
func NewForConfigOrDie(c *rest.Config) *KubermaticV1Client {
	client, err := NewForConfig(c)
	if err != nil {
		panic(err)
	}
	return client
}

// New creates a new KubermaticV1Client for the given RESTClient.
func New(c rest.Interface) *KubermaticV1Client {
	return &KubermaticV1Client{c}
}

func setConfigDefaults(config *rest.Config) error {
	gv := v1.SchemeGroupVersion
	config.GroupVersion = &gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()

	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	return nil
}

// RESTClient returns a RESTClient that is used to communicate
// with API server by this client implementation.
func (c *KubermaticV1Client) RESTClient() rest.Interface {
	if c == nil {
		return nil
	}
	return c.restClient
}
