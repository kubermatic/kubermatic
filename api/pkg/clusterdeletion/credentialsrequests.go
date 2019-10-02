package clusterdeletion

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (d *Deletion) cleanupCredentialsRequest(ctx context.Context, log *zap.SugaredLogger, cluster *kubermaticv1.Cluster) (deletedSomething bool, err error) {
	log = log.Named("CredentialsRequestCleanup")

	userClusterClient, err := d.userClusterClientGetter()
	if err != nil {
		return false, err
	}

	credentialRequests := &unstructured.UnstructuredList{}
	credentialRequests.SetAPIVersion("cloudcredential.openshift.io/v1")
	credentialRequests.SetKind("CredentialsRequest")

	if err := userClusterClient.List(ctx, &ctrlruntimeclient.ListOptions{}, credentialRequests); err != nil {
		if _, matches := err.(*meta.NoKindMatchError); matches {
			log.Debug("Got a NoKindMatchError when listing CredentialsRequests, skipping their cleanup")
			return false, nil
		}
		return false, fmt.Errorf("failed to list CredentialsRequests: %v", err)
	}

	if len(credentialRequests.Items) == 0 {
		log.Debug("No CredentialsRequests found, nothing to clean up")
		return false, nil
	}

	log.Debug("Found CredentialsRequests", "num-credentials-requests", len(credentialRequests.Items))

	for _, credentialRequest := range credentialRequests.Items {
		if err := userClusterClient.Delete(ctx, &credentialRequest); err != nil {
			return false, fmt.Errorf("failed to delete CloudCredentia")
		}
	}

	log.Debug("Successfully issued DELETE for all CredentialsRequests")
	return true, nil
}
