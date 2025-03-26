//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2025 Kubermatic GmbH

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

package synccontroller

import (
	"context"
	"testing"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kuberneteshelper "k8c.io/kubermatic/v2/pkg/kubernetes"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"
	"k8c.io/kubermatic/v2/pkg/test/fake"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const testNamespace = "kubermatic"

func TestSync(t *testing.T) {
	cbsl := &kubermaticv1.ClusterBackupStorageLocation{}
	cbsl.Name = "test-location"
	cbsl.Namespace = testNamespace
	cbsl.UID = types.UID("1234-5678")
	cbsl.Spec = velerov1.BackupStorageLocationSpec{
		Provider: "test",
	}

	// When master and seed are the same cluster, the controller detects this by
	// comparing the UID from the CBSL it loaded from both clients. If they match,
	// no sync is attempted. To check if this UID comparison works, we simulate
	// "the identical" CBSL, but with a change in it (something that could not
	// happen in a real world scenario) and see if that change survives the
	// reconciliation.
	// If this was a real world scenario, the seedClients map below would simply
	// use the masterClient for the "local" seed, instead of creating a new
	// client.
	cbslClone := cbsl.DeepCopy()
	cbslClone.Spec = velerov1.BackupStorageLocationSpec{
		Provider: "do-not-change-me",
	}

	masterClient := fake.
		NewClientBuilder().
		WithObjects(cbsl).
		Build()

	seedClients := kuberneteshelper.SeedClientMap{
		"local": fake.
			NewClientBuilder().
			WithObjects(cbslClone).
			Build(),
		"remote": fake.
			NewClientBuilder().
			Build(),
	}

	rec := reconciler{
		masterClient: masterClient,
		seedClients:  seedClients,
		recorder:     &record.FakeRecorder{},
		log:          kubermaticlog.Logger,
	}

	ctx := context.Background()

	_, err := rec.Reconcile(ctx, reconcile.Request{
		NamespacedName: ctrlruntimeclient.ObjectKeyFromObject(cbsl),
	})
	if err != nil {
		t.Fatalf("Failed to reconcile CBSL: %v", err)
	}

	// check that the cloned CBSL remained unchanged
	currentClone := &kubermaticv1.ClusterBackupStorageLocation{}
	if err := seedClients["local"].Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cbslClone), currentClone); err != nil {
		t.Fatalf("Failed to fetch CBSL clone on the seed: %v", err)
	}

	if currentClone.Spec.Provider != cbslClone.Spec.Provider {
		t.Fatalf("CBSL clone should not have been modified, but provider was changed from %q to %q.", cbslClone.Spec.Provider, currentClone.Spec.Provider)
	}

	// check that a copy of the CBSL has been created on the remote seed
	remoteCBSL := &kubermaticv1.ClusterBackupStorageLocation{}
	if err := seedClients["remote"].Get(ctx, ctrlruntimeclient.ObjectKeyFromObject(cbsl), remoteCBSL); err != nil {
		t.Fatalf("Failed to fetch CBSL on remote seed: %v", err)
	}

	if remoteCBSL.Spec.Provider != cbsl.Spec.Provider {
		t.Fatalf("CBSL was created on remote seed, but provider was not set correctly, should be %q, but is %q.", cbsl.Spec.Provider, remoteCBSL.Spec.Provider)
	}
}
