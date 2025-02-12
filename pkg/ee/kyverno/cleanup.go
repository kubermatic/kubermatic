//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0")
                     Copyright © 2021 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package kyverno

import (
	"context"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	admissionresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/admission-controller"
	backgroundresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/background-controller"
	cleanupresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/cleanup-controller"
	reportsresources "k8c.io/kubermatic/v2/pkg/ee/kyverno/resources/seed-cluster/reports-controller"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// CleanUpResources deletes all resources created for Kyverno in the seed cluster.
func CleanUpResources(ctx context.Context, client ctrlruntimeclient.Client, cluster *kubermaticv1.Cluster) error {
	// Clean up admission controller resources
	if err := admissionresources.CleanUpResources(ctx, client, cluster); err != nil {
		return err
	}

	// Clean up background controller resources
	if err := backgroundresources.CleanUpResources(ctx, client, cluster); err != nil {
		return err
	}

	// Clean up reports controller resources
	if err := reportsresources.CleanUpResources(ctx, client, cluster); err != nil {
		return err
	}

	// Clean up cleanup controller resources
	if err := cleanupresources.CleanUpResources(ctx, client, cluster); err != nil {
		return err
	}

	return nil
}
