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

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/provider"
	"k8c.io/kubermatic/v2/pkg/resources"

	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/clientcmd"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var scheme = runtime.NewScheme()

// kubevirtInfranGetter is a function to retrieve the currently relevant
// kube client for the used kubevirt infra cluster.
type kubevirtInfranGetter = func(ctx context.Context, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client) (ctrlruntimeclient.Client, error)

func init() {
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

func getKubeVirtInfraKConfig(ctx context.Context, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client) (string, error) {
	if cluster.Spec.Cloud.Kubevirt.Kubeconfig != "" {
		return cluster.Spec.Cloud.Kubevirt.Kubeconfig, nil
	}

	if cluster.Spec.Cloud.Kubevirt.CredentialsReference == nil {
		return "", errors.New("no credentials provided")
	}

	secretKeySelectorFunc := provider.SecretKeySelectorValueFuncFactory(ctx, seedClient)
	kubeconfig, err := secretKeySelectorFunc(cluster.Spec.Cloud.Kubevirt.CredentialsReference, resources.KubeVirtKubeconfig)
	if err != nil {
		return "", err
	}

	return kubeconfig, nil
}

func SetupKubeVirtInfraClient(ctx context.Context, cluster *kubermaticv1.Cluster, seedClient ctrlruntimeclient.Client) (ctrlruntimeclient.Client, error) {
	kubeconfig, err := getKubeVirtInfraKConfig(ctx, cluster, seedClient)
	if err != nil {
		return nil, err
	}

	kubeVirtInfraClient, err := newClient(kubeconfig)
	if err != nil {
		return nil, err
	}

	return kubeVirtInfraClient, nil
}
