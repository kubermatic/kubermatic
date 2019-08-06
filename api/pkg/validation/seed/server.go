package seed

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func New(
	ctx context.Context,
	listenAddress, workerName string,
	seedsGetter provider.SeedsGetter,
	seedKubeconfigGetter provider.SeedKubeconfigGetter) (*Server, error) {

	labelSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return nil, err
	}
	listOpts := &ctrlruntimeclient.ListOptions{Selector: labelSelector}

	return &Server{
		listenAddress: listenAddress,
		validator:     newValidator(ctx, seedsGetter, seedKubeconfigGetter, listOpts),
	}
}

type Server struct {
	listenAddress string
	validator     *seedValidator
}

func (s *Server) handleSeedValidationRequests(resp http.ResponseWriter, req *http.Request) {
	body := bytes.NewBuffer([]byte{})
	if _, err := body.ReadFrom(req.Body); err != nil {
		http.Error(resp, "failed to read request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	admissionRequest := &admissionv1beta1.AdmissionRequest{}
	if err := json.Unmarshal(body.Bytes(), admissionRequest); err != nil {
		http.Error(resp, "failed to unmarshal request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	seed := &kubermaticv1.Seed{}
	if err := json.Unmarshal(admissionRequest.Object.Raw, seed); err != nil {
		http.Error(resp, "failed to unmarshal admissionRequest.Body into a Seed: "+err.Error, http.StatusBadRequest)
	}

	validationErr := s.validator.Validate(seed, admissionRequest.Operation == admissionv1beta1.Delete)
}
