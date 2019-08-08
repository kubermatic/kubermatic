package seed

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"net/http"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type WebhookOpts struct {
	listenAddress string
	certFile      string
	keyFile       string
}

func (opts *WebhookOpts) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&opts.listenAddress, "seed-admisisonwebhook-listen-address", ":8100", "The listen address for the seed amission webhook")
	fs.StringVar(&opts.certFile, "seed-admissionwebhook-cert-file", "", "The location of the certificate file")
	fs.StringVar(&opts.keyFile, "seed-admissionwebhook-key-file", "", "The location of the certificate key file")
}

// Server returns a Server that validates AdmissionRequests for Seed CRs
func (opts *WebhookOpts) Server(
	ctx context.Context,
	log *zap.SugaredLogger,
	workerName string,
	seedsGetter provider.SeedsGetter,
	seedKubeconfigGetter provider.SeedKubeconfigGetter) (*Server, error) {

	labelSelector, err := workerlabel.LabelSelector(workerName)
	if err != nil {
		return nil, err
	}
	listOpts := &ctrlruntimeclient.ListOptions{LabelSelector: labelSelector}

	server := &Server{
		Server: &http.Server{
			Addr: opts.listenAddress,
		},
		log:           log.Named("seed-webhook-server"),
		listenAddress: opts.listenAddress,
		certFile:      opts.certFile,
		keyFile:       opts.keyFile,
		validator:     newValidator(ctx, seedsGetter, seedKubeconfigGetter, listOpts),
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/seed-validation", server.handleSeedValidationRequests)
	server.Handler = mux

	return server, nil
}

type Server struct {
	*http.Server
	log           *zap.SugaredLogger
	listenAddress string
	certFile      string
	keyFile       string
	validator     *seedValidator
}

// This implements sigs.k8s.io/controller-runtime/pkg/manager.Runnable
func (s *Server) Start(_ <-chan struct{}) error {
	return s.ListenAndServeTLS(s.certFile, s.keyFile)
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
		http.Error(resp, "failed to unmarshal admissionRequest.Body into a Seed: "+err.Error(), http.StatusBadRequest)
	}
	log := s.log.With("seed", seed.Name)

	var result *metav1.Status
	validationErr := s.validator.Validate(seed, admissionRequest.Operation == admissionv1beta1.Delete)
	if validationErr != nil {
		log.Errorw("seed failed validation", "validationError", validationErr.Error())
		result = &metav1.Status{Message: validationErr.Error()}
	}

	admissionResponse := &admissionv1beta1.AdmissionResponse{
		UID:     admissionRequest.UID,
		Allowed: validationErr == nil,
		Result:  result,
	}
	serializedAdmissionResponse, err := json.Marshal(admissionResponse)
	if err != nil {
		log.Errorw("failed to serialize admission response", zap.Error(err))
		http.Error(resp, "failed to serialize response", http.StatusInternalServerError)
		return
	}

	resp.WriteHeader(http.StatusOK)
	if _, err := resp.Write(serializedAdmissionResponse); err != nil {
		log.Errorw("failed to write response body", zap.Error(err))
	}
}
