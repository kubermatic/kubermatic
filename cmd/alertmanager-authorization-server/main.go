/*
Copyright 2021 The Kubermatic Kubernetes Platform contributors.

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

package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"strings"

	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"

	coreV3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	typeV3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	ctrlruntimecluster "sigs.k8s.io/controller-runtime/pkg/cluster"
	ctrlruntimelog "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(kubermaticv1.AddToScheme(scheme))
}

type authorizationServer struct {
	listenAddress   string
	authHeaderName  string
	orgIDHeaderName string

	client ctrlruntimeclient.Reader
	log    *zap.SugaredLogger
}

func (s *authorizationServer) Check(ctx context.Context, req *authv3.CheckRequest) (*authv3.CheckResponse, error) {
	log := s.log.With("reqid", req.Attributes.Request.Http.Id)

	userEmail, ok := req.Attributes.Request.Http.Headers[s.authHeaderName]
	if !ok {
		log.Warnw("No user ID passed from OAuth proxy via HTTP header.", "header", s.authHeaderName)

		return &authv3.CheckResponse{
			Status: &status.Status{
				Code: int32(code.Code_UNAUTHENTICATED),
			},
			HttpResponse: &authv3.CheckResponse_DeniedResponse{
				DeniedResponse: &authv3.DeniedHttpResponse{
					Status: &typeV3.HttpStatus{
						Code: typeV3.StatusCode_Unauthorized,
					},
				},
			},
		}, nil
	}

	log = log.With("user", userEmail)

	// parse cluster ID from the original request path
	clusterID := ""
	arr := strings.Split(req.Attributes.Request.Http.Path, "/")
	if len(arr) > 1 {
		clusterID = arr[1]
	}
	if clusterID == "" {
		log.Warnw("Malformed path, cannot determine cluster ID.", "path", req.Attributes.Request.Http.Path)
		return &authv3.CheckResponse{
			Status: &status.Status{
				Code: int32(code.Code_NOT_FOUND),
			},
			HttpResponse: &authv3.CheckResponse_DeniedResponse{
				DeniedResponse: &authv3.DeniedHttpResponse{
					Status: &typeV3.HttpStatus{
						Code: typeV3.StatusCode_NotFound,
					},
				},
			},
		}, nil
	}

	log = log.With("cluster", clusterID)

	authorized, reason, err := s.authorize(ctx, userEmail, clusterID)
	if err != nil {
		log.Warnw("Authorization failed", zap.Error(err))

		return &authv3.CheckResponse{
			Status: &status.Status{
				Code: int32(code.Code_INTERNAL),
			},
			HttpResponse: &authv3.CheckResponse_DeniedResponse{
				DeniedResponse: &authv3.DeniedHttpResponse{
					Status: &typeV3.HttpStatus{
						Code: typeV3.StatusCode_InternalServerError,
					},
				},
			},
		}, err
	}

	if authorized {
		log.Debugw("Request authorized", "reason", reason)

		return &authv3.CheckResponse{
			Status: &status.Status{
				Code: int32(code.Code_OK),
			},
			HttpResponse: &authv3.CheckResponse_OkResponse{
				OkResponse: &authv3.OkHttpResponse{
					Headers: []*coreV3.HeaderValueOption{
						{
							Header: &coreV3.HeaderValue{
								Key:   s.orgIDHeaderName,
								Value: clusterID,
							},
						},
					},
				},
			},
		}, nil
	}

	log.Debugw("Request rejected", "reason", reason)

	return &authv3.CheckResponse{
		Status: &status.Status{
			Code: int32(code.Code_UNAUTHENTICATED),
		},
		HttpResponse: &authv3.CheckResponse_DeniedResponse{
			DeniedResponse: &authv3.DeniedHttpResponse{
				Status: &typeV3.HttpStatus{
					Code: typeV3.StatusCode_Unauthorized,
				},
			},
		},
	}, nil
}

func (s *authorizationServer) authorize(ctx context.Context, userEmail, clusterID string) (authorized bool, reason string, err error) {
	isAdmin, err := s.isAdminUser(ctx, userEmail)
	if err != nil {
		return false, "", fmt.Errorf("checking if user is admin: %w", err)
	}
	if isAdmin {
		return true, "user is admin", nil
	}

	cluster := &kubermaticv1.Cluster{}
	if err := s.client.Get(ctx, types.NamespacedName{Name: clusterID}, cluster); err != nil {
		return false, "", fmt.Errorf("getting cluster: %w", err)
	}

	projectID := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
	if len(projectID) == 0 {
		return false, "", fmt.Errorf("cluster %s is missing '%s' label", cluster.Name, kubermaticv1.ProjectIDLabelKey)
	}

	allMembers := &kubermaticv1.UserProjectBindingList{}
	if err := s.client.List(ctx, allMembers); err != nil {
		return false, "", fmt.Errorf("listing userProjectBinding: %w", err)
	}

	for _, member := range allMembers.Items {
		if strings.EqualFold(member.Spec.UserEmail, userEmail) && member.Spec.ProjectID == projectID {
			return true, "user bound to cluster project", nil
		}
	}

	// authorize through group project bindings
	allGroupBindings := &kubermaticv1.GroupProjectBindingList{}
	if err := s.client.List(ctx, allGroupBindings, ctrlruntimeclient.MatchingLabels{kubermaticv1.ProjectIDLabelKey: projectID}); err != nil {
		return false, "", fmt.Errorf("listing groupProjectBinding: %w", err)
	}

	if len(allGroupBindings.Items) > 0 {
		groupSet := sets.New[string]()
		for _, gpb := range allGroupBindings.Items {
			groupSet.Insert(gpb.Spec.Group)
		}

		allUsers := &kubermaticv1.UserList{}
		if err := s.client.List(ctx, allUsers); err != nil {
			return false, "", fmt.Errorf("listing users: %w", err)
		}

		for _, user := range allUsers.Items {
			if strings.EqualFold(user.Spec.Email, userEmail) && groupSet.HasAny(user.Spec.Groups...) {
				return true, "user group(s) bound to cluster project", nil
			}
		}
	}

	return false, "not bound to project", nil
}

func (s *authorizationServer) isAdminUser(ctx context.Context, userEmail string) (bool, error) {
	users := &kubermaticv1.UserList{}
	if err := s.client.List(ctx, users); err != nil {
		return false, fmt.Errorf("listing user: %w", err)
	}

	for _, user := range users.Items {
		if user.Spec.IsAdmin && strings.EqualFold(user.Spec.Email, userEmail) {
			return true, nil
		}
	}

	return false, nil
}

func (s *authorizationServer) addFlags() {
	flag.StringVar(&s.listenAddress, "address", ":50051", "the address to listen on")
	flag.StringVar(&s.authHeaderName, "auth-header-name", "X-Forwarded-Email", "alertmanager authorization server HTTP header that will contain the email")
	flag.StringVar(&s.orgIDHeaderName, "org-id-header-name", "X-Scope-OrgID", "the header that alertmanager uses for multi-tenancy support")
}

func (s *authorizationServer) complete() error {
	// envoy's Header map only supports lowercased header names
	s.authHeaderName = strings.ToLower(s.authHeaderName)

	return nil
}

func main() {
	ctx := signals.SetupSignalHandler()

	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)
	server := authorizationServer{}
	server.addFlags()
	flag.Parse()

	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()

	// set the logger used by sigs.k8s.io/controller-runtime
	ctrlruntimelog.SetLogger(zapr.NewLogger(rawLog.WithOptions(zap.AddCallerSkip(1))))

	if err := server.complete(); err != nil {
		log.Fatalw("Invalid command line", zap.Error(err))
	}

	cfg, err := ctrlruntime.GetConfig()
	if err != nil {
		log.Fatalw("Failed to get kubeconfig", zap.Error(err))
	}

	cluster, err := ctrlruntimecluster.New(cfg)
	if err != nil {
		log.Fatalw("Failed to create cluster object", zap.Error(err))
	}

	go func() {
		if err := cluster.GetCache().Start(ctx); err != nil {
			log.Fatalw("Failed to start cache", zap.Error(err))
		}
	}()
	if !cluster.GetCache().WaitForCacheSync(ctx) {
		log.Fatal("Failed to wait for cache sync")
	}

	server.client = cluster.GetClient()
	server.log = log

	log.With(
		"address", server.listenAddress,
		"authHeader", server.authHeaderName,
		"orgIDHeader", server.orgIDHeaderName,
	).Info("Listening…")
	var lc net.ListenConfig
	lis, err := lc.Listen(ctx, "tcp", server.listenAddress)
	if err != nil {
		log.Fatalw("Alertmanager authorization server failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	authv3.RegisterAuthorizationServer(grpcServer, &server)
	if err = grpcServer.Serve(lis); err != nil {
		log.Fatalw("Alertmanager authorization server failed to serve requests", zap.Error(err))
	}
}
