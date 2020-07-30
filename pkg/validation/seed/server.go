/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package seed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"

	"go.uber.org/zap"

	kubermaticv1 "github.com/kubermatic/kubermatic/pkg/crd/kubermatic/v1"

	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type WebhookOpts struct {
	ListenAddress string
	CertFile      string
	KeyFile       string
}

func (opts *WebhookOpts) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&opts.ListenAddress, "seed-admissionwebhook-listen-address", ":8100", "The listen address for the seed amission webhook")
	fs.StringVar(&opts.CertFile, "seed-admissionwebhook-cert-file", "", "The location of the certificate file")
	fs.StringVar(&opts.KeyFile, "seed-admissionwebhook-key-file", "", "The location of the certificate key file")
}

// Server returns a Server that validates AdmissionRequests for Seed CRs.
func (opts *WebhookOpts) Server(
	ctx context.Context,
	log *zap.SugaredLogger,
	namespace string,
	validateFunc ValidateFunc) (*Server, error) {

	if opts.CertFile == "" || opts.KeyFile == "" {
		return nil, fmt.Errorf("seed-admissionwebhook-cert-file or seed-admissionwebhook-key-file cannot be empty")
	}

	server := &Server{
		Server: &http.Server{
			Addr: opts.ListenAddress,
		},
		log:           log.Named("seed-webhook-server"),
		listenAddress: opts.ListenAddress,
		certFile:      opts.CertFile,
		keyFile:       opts.KeyFile,
		validateFunc:  validateFunc,
		namespace:     namespace,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", server.handleSeedValidationRequests)
	server.Handler = mux

	return server, nil
}

type Server struct {
	*http.Server
	ctx           context.Context
	log           *zap.SugaredLogger
	listenAddress string
	certFile      string
	keyFile       string
	validateFunc  ValidateFunc
	namespace     string
}

// Server implements LeaderElectionRunnable to indicate that it does not require to run
// within an elected leader
var _ manager.LeaderElectionRunnable = &Server{}

func (s *Server) NeedLeaderElection() bool {
	return false
}

// Start implements sigs.k8s.io/controller-runtime/pkg/manager.Runnable
func (s *Server) Start(_ <-chan struct{}) error {
	return s.ListenAndServeTLS(s.certFile, s.keyFile)
}

func (s *Server) handleSeedValidationRequests(resp http.ResponseWriter, req *http.Request) {
	admissionRequest, validationErr := s.handle(req)
	if validationErr != nil {
		s.log.Warnw("Seed admission failed", zap.Error(validationErr))
	}

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
		s.log.Errorw("Failed to serialize admission response", zap.Error(err))
		http.Error(resp, "failed to serialize response", http.StatusInternalServerError)
		return
	}
	resp.WriteHeader(http.StatusOK)
	if _, err := resp.Write(serializedAdmissionResponse); err != nil {
		s.log.Errorw("Failed to write response body", zap.Error(err))
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

	if admissionReview.Request == nil {
		return nil, errors.New("received malformed admission review: no request defined")
	}

	s.log.Debugw(
		"Received admission request",
		"kind", admissionReview.Request.Kind,
		"name", admissionReview.Request.Name,
		"namespace", admissionReview.Request.Namespace,
		"operation", admissionReview.Request.Operation)

	// Under normal circumstances, the Kubermatic Operator will setup a Webhook
	// that has a namespace selector (and it will also label the kubermatic ns),
	// so that a seed webhook never receives requests for other namespaces.
	// However the old Helm chart could not do this and deployed a "global" webhook.
	// Until all seeds are migrated to the Operator, this check ensures that the
	// old-style webhook ignores foreign namespace requests entirely.
	if admissionReview.Request.Namespace != s.namespace {
		s.log.Warn("Request is for foreign namespace, ignoring")
		return admissionReview.Request, nil
	}

	seed := &kubermaticv1.Seed{}
	// On DELETE, the admissionReview.Request.Object is unset
	// Ref: https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#webhook-request-and-response
	if admissionReview.Request.Operation == admissionv1beta1.Delete {
		seed.Name = admissionReview.Request.Name
		seed.Namespace = admissionReview.Request.Namespace
	} else if err := json.Unmarshal(admissionReview.Request.Object.Raw, seed); err != nil {
		return nil, fmt.Errorf("failed to unmarshal object from request into a Seed: %v", err)
	}

	validationErr := s.validateFunc(s.ctx, seed, admissionReview.Request.Operation)
	if validationErr != nil {
		s.log.Errorw("Seed failed validation", "seed", seed.Name, "validationError", validationErr.Error())
	}

	return admissionReview.Request, validationErr
}
