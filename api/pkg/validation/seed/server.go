package seed

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/workerlabel"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	mux.HandleFunc("/", server.handleSeedValidationRequests)
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
	admissionRequest, validationErr := s.handle(req)
	var uid types.UID
	if admissionRequest != nil {
		uid = admissionRequest.UID
	}
	response := &admissionv1beta1.AdmissionReview{
		Request: admissionRequest,
		Response: &admissionv1beta1.AdmissionResponse{
			UID:     uid,
			Allowed: validationErr == nil,
			Result: &metav1.Status{
				Message: fmt.Sprintf("%v", validationErr),
			},
		},
	}
	serializedAdmissionResponse, err := json.Marshal(response)
	if err != nil {
		s.log.Errorw("failed to serialize admission response", zap.Error(err))
		http.Error(resp, "failed to serialize response", http.StatusInternalServerError)
		return
	}
	resp.WriteHeader(http.StatusOK)
	if _, err := resp.Write(serializedAdmissionResponse); err != nil {
		s.log.Errorw("failed to write response body", zap.Error(err))
		return
	}
	s.log.Debug("Successfully validated seed")
}

func (s *Server) handle(req *http.Request) (*admissionv1beta1.AdmissionRequest, error) {
	body := bytes.NewBuffer([]byte{})
	if _, err := body.ReadFrom(req.Body); err != nil {
		return nil, fmt.Errorf("failed to read request body: %v", err)
	}

	admissionReview := &admissionv1beta1.AdmissionReview{}
	if err := json.Unmarshal(body.Bytes(), admissionReview); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request body: %v", err)
	}

	seed := &kubermaticv1.Seed{}
	if err := json.Unmarshal(admissionReview.Request.Object.Raw, seed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal object from request into a Seed: %v", err)
	}

	validationErr := s.validator.Validate(seed, admissionReview.Request.Operation == admissionv1beta1.Delete)
	if validationErr != nil {
		s.log.Errorw("seed failed validation", "seed", seed.Name, "validationError", validationErr.Error())
	}

	return admissionReview.Request, validationErr
}
