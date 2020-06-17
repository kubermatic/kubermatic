// +build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2020 Loodse GmbH

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

package master

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	fakectrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestMigrateAllDatacenterEmailRestrictions(t *testing.T) {
	seedName := "example-seed"
	nsName := "example-namespace"
	dcName := "example-datacenter"
	dcName2 := "another-datacenter"
	domain := "ripe.net"
	unmigratedSeed := &kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      seedName,
			Namespace: nsName,
		},
		Spec: kubermaticv1.SeedSpec{
			Datacenters: map[string]kubermaticv1.Datacenter{
				dcName: {
					Spec: kubermaticv1.DatacenterSpec{
						RequiredEmailDomain: domain,
					},
				},
				dcName2: {
					Spec: kubermaticv1.DatacenterSpec{},
				},
			},
		},
	}
	expectedSeed := &kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      seedName,
			Namespace: nsName,
		},
		Spec: kubermaticv1.SeedSpec{
			Datacenters: map[string]kubermaticv1.Datacenter{
				dcName: {
					Spec: kubermaticv1.DatacenterSpec{
						RequiredEmailDomains: []string{domain},
					},
				},
				dcName2: {
					Spec: kubermaticv1.DatacenterSpec{},
				},
			},
		},
	}

	client := fakectrlruntimeclient.NewFakeClient(unmigratedSeed)
	err := migrateAllDatacenterEmailRestrictions(context.Background(), zaptest.NewLogger(t).Sugar(), client, nsName, Options{})
	assert.NoError(t, err)

	key := ctrlruntimeclient.ObjectKey{
		Name:      seedName,
		Namespace: nsName,
	}

	migratedSeed := &kubermaticv1.Seed{}
	err = client.Get(context.Background(), key, migratedSeed)
	assert.NoError(t, err)
	assert.Equal(t, expectedSeed.Spec, migratedSeed.Spec)
}

func TestMigrateAllDatacenterEmailRestrictionsInvalid(t *testing.T) {
	seedName := "example-seed"
	nsName := "example-namespace"
	dcName := "example-datacenter"
	domain := "ripe.net"
	unmigratedSeed := &kubermaticv1.Seed{
		ObjectMeta: metav1.ObjectMeta{
			Name:      seedName,
			Namespace: nsName,
		},
		Spec: kubermaticv1.SeedSpec{
			Datacenters: map[string]kubermaticv1.Datacenter{
				dcName: {
					Spec: kubermaticv1.DatacenterSpec{
						RequiredEmailDomain:  domain,
						RequiredEmailDomains: []string{domain},
					},
				},
			},
		},
	}

	client := fakectrlruntimeclient.NewFakeClient(unmigratedSeed)
	err := migrateAllDatacenterEmailRestrictions(context.Background(), zaptest.NewLogger(t).Sugar(), client, nsName, Options{})
	assert.Error(t, err, "datacenter %s->%s has both `requiredEmailDomain` and `requiredEmailDomains` set", seedName, dcName)
}
