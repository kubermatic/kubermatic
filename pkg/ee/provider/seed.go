//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2020 Kubermatic GmbH

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

package provider

import (
	"context"
	"fmt"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	"k8s.io/client-go/rest"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

type EESeedsGetter func() (map[string]*kubermaticv1.Seed, error)

func SeedsGetterFactory(ctx context.Context, client ctrlruntimeclient.Client, namespace string) (EESeedsGetter, error) {
	// We only have a options func for raw *metav1.ListOpts as the rbac controller currently required that
	listOpts := &ctrlruntimeclient.ListOptions{
		Namespace: namespace,
	}

	return func() (map[string]*kubermaticv1.Seed, error) {
		seeds := &kubermaticv1.SeedList{}
		if err := client.List(ctx, seeds, listOpts); err != nil {
			return nil, fmt.Errorf("failed to list the seeds: %w", err)
		}
		seedMap := map[string]*kubermaticv1.Seed{}
		for idx, seed := range seeds.Items {
			seedMap[seed.Name] = &seeds.Items[idx]
		}
		return seedMap, nil
	}, nil
}

type EESeedKubeconfigGetter = func(seed *kubermaticv1.Seed) (*rest.Config, error)

// Ensures that SeedKubeconfigGetter implements EESeedKubeconfigGetter.
var _ EESeedKubeconfigGetter = SeedKubeconfigGetter

// SeedKubeconfigGetter implements provider.SeedKubeconfigGetter.
func SeedKubeconfigGetter(seed *kubermaticv1.Seed) (*rest.Config, error) {
	cfg, err := ctrlruntimeconfig.GetConfigWithContext(seed.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to get restConfig for seed %q: %w", seed.Name, err)
	}
	return cfg, nil
}
