//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package kubevirtnetworkcontroller

import (
	"context"
	"encoding/base64"
	"errors"

	kubeovnv1 "github.com/kubeovn/kube-ovn/pkg/apis/kubeovn/v1"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(kubeovnv1.AddToScheme(scheme))
	utilruntime.Must(networkingv1.AddToScheme(scheme))
}

func newClient(kubeconfig string) (ctrlruntimeclient.Client, error) {
	var client ctrlruntimeclient.Client
	config, err := base64.StdEncoding.DecodeString(kubeconfig)
	if err != nil {
		// if the decoding failed, the kubeconfig is sent already decoded without the need of decoding it,
		// for example the value has been read from Vault during the ci tests, which is saved as json format.
		config = []byte(kubeconfig)
	}
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(config)
	if err != nil {
		return nil, err
	}

	client, err = ctrlruntimeclient.New(restConfig, ctrlruntimeclient.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}

	return client, nil
}

func (r *Reconciler) getKubeVirtInfraKConfig(ctx context.Context, cluster *kubermaticv1.Cluster) (string, error) {
	if cluster.Spec.Cloud.Kubevirt.Kubeconfig != "" {
		return cluster.Spec.Cloud.Kubevirt.Kubeconfig, nil
	}

	if cluster.Spec.Cloud.Kubevirt.CredentialsReference == nil {
		return "", errors.New("no credentials provided")
	}

	secretKeySelectorFunc := provider.SecretKeySelectorValueFuncFactory(ctx, r.Client)
	kubeconfig, err := secretKeySelectorFunc(cluster.Spec.Cloud.Kubevirt.CredentialsReference, resources.KubeVirtKubeconfig)
	if err != nil {
		return "", err
	}

	return kubeconfig, nil
}

func (r *Reconciler) SetupKubeVirtInfraClient(ctx context.Context, cluster *kubermaticv1.Cluster) (ctrlruntimeclient.Client, error) {
	kubeconfig, err := r.getKubeVirtInfraKConfig(ctx, cluster)
	if err != nil {
		return nil, err
	}

	kubeVirtInfraClient, err := newClient(kubeconfig)
	if err != nil {
		return nil, err
	}

	return kubeVirtInfraClient, nil
}
