package crd

import (
	"reflect"
	"time"

	"github.com/golang/glog"
	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	etcdoperatorv1beta2 "github.com/kubermatic/kubermatic/api/pkg/crd/etcdoperator/v1beta2"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

//EnsureCustomResourceDefinitions creates all needed CustomResourceDefinitions and waits until they are populated
func EnsureCustomResourceDefinitions(clientset apiextensionsclient.Interface) error {
	type resource struct {
		plural  string
		kind    string
		group   string
		version string
		scope   apiextensionsv1beta1.ResourceScope
	}

	resourceNames := []resource{
		{
			plural:  kubermaticv1.SSHKeyPlural,
			kind:    reflect.TypeOf(kubermaticv1.UserSSHKey{}).Name(),
			group:   kubermaticv1.GroupName,
			version: kubermaticv1.SchemeGroupVersion.Version,
			scope:   apiextensionsv1beta1.ClusterScoped,
		},
		{
			plural:  kubermaticv1.ClusterPlural,
			kind:    reflect.TypeOf(kubermaticv1.Cluster{}).Name(),
			group:   kubermaticv1.GroupName,
			version: kubermaticv1.SchemeGroupVersion.Version,
			scope:   apiextensionsv1beta1.ClusterScoped,
		},
		{
			plural:  kubermaticv1.UserPlural,
			kind:    reflect.TypeOf(kubermaticv1.User{}).Name(),
			group:   kubermaticv1.GroupName,
			version: kubermaticv1.SchemeGroupVersion.Version,
			scope:   apiextensionsv1beta1.ClusterScoped,
		},
		// TODO(p0lyn0mial) : add project custom resource
		{
			plural:  etcdoperatorv1beta2.EtcdClusterPlural,
			kind:    reflect.TypeOf(etcdoperatorv1beta2.EtcdCluster{}).Name(),
			group:   etcdoperatorv1beta2.GroupName,
			version: etcdoperatorv1beta2.SchemeGroupVersion.Version,
			scope:   apiextensionsv1beta1.NamespaceScoped,
		},
	}

	for _, res := range resourceNames {
		if err := createCustomResourceDefinition(res.plural, res.kind, res.group, res.version, res.scope, clientset); err != nil {
			return err
		}
	}

	return nil
}

func createCustomResourceDefinition(plural, kind, group, version string, scope apiextensionsv1beta1.ResourceScope, clientset apiextensionsclient.Interface) error {
	name := plural + "." + group
	crd := &apiextensionsv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: apiextensionsv1beta1.CustomResourceDefinitionSpec{
			Group:   group,
			Version: version,
			Scope:   scope,
			Names: apiextensionsv1beta1.CustomResourceDefinitionNames{
				Plural: plural,
				Kind:   kind,
			},
		},
	}
	_, err := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	if err != nil {
		if kerrors.IsAlreadyExists(err) {
			return nil
		}
		return err
	}

	// wait for CRD being established
	err = wait.Poll(500*time.Millisecond, 60*time.Second, func() (bool, error) {
		crd, err = clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		for _, cond := range crd.Status.Conditions {
			switch cond.Type {
			case apiextensionsv1beta1.Established:
				if cond.Status == apiextensionsv1beta1.ConditionTrue {
					return true, err
				}
			case apiextensionsv1beta1.NamesAccepted:
				if cond.Status == apiextensionsv1beta1.ConditionFalse {
					glog.Errorf("Name conflict: %v\n", cond.Reason)
				}
			}
		}
		return false, err
	})
	if err != nil {
		deleteErr := clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(name, nil)
		if deleteErr != nil {
			return errors.NewAggregate([]error{err, deleteErr})
		}
		return err
	}
	return nil
}
