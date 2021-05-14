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

	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"

	coreV3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	typeV3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"

	kubermaticapiv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func (a *authorizationServer) parse() {
	flag.StringVar(&a.listenAddress, "address", ":50051", "the address to listen on")
	flag.StringVar(&a.authHeaderName, "auth-header-name", "X-Forwarded-Email", "alertmanager authorization proxy http Header that will contain the email")
	flag.StringVar(&a.orgIDHeaderName, "org-id-header-name", "X-Scope-OrgID", "the header that alertmanager uses for multi-tenancy support")
	flag.Parse()
}

type authorizationServer struct {
	listenAddress   string
	authHeaderName  string
	orgIDHeaderName string

	client ctrlruntimeclient.Client
	log    *zap.SugaredLogger
}

func (a *authorizationServer) Check(ctx context.Context, req *authv3.CheckRequest) (*authv3.CheckResponse, error) {
	userEmail, ok := req.Attributes.Request.Http.Headers[a.authHeaderName]
	if !ok {
		a.log.Debug("missing user id passed from OAuth proxy")
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

	// parse projectID from tbe original request path
	projectID := ""
	arr := strings.Split(req.Attributes.Request.Http.Path, "/")
	if len(arr) > 1 {
		projectID = arr[1]
	}
	if projectID == "" {
		a.log.Debug("ordID cannot be parsed")
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

	allMembers := &kubermaticapiv1.UserProjectBindingList{}
	if err := a.client.List(ctx, allMembers); err != nil {
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
		}, fmt.Errorf("listing userProjectBinding: %w", err)
	}

	for _, member := range allMembers.Items {
		if strings.EqualFold(member.Spec.UserEmail, userEmail) && member.Spec.ProjectID == projectID {
			a.log.Debugf("user %q authorized for project: %s\n", userEmail, projectID)
			return &authv3.CheckResponse{
				Status: &status.Status{
					Code: int32(code.Code_OK),
				},
				HttpResponse: &authv3.CheckResponse_OkResponse{
					OkResponse: &authv3.OkHttpResponse{
						Headers: []*coreV3.HeaderValueOption{
							{
								Header: &coreV3.HeaderValue{
									Key:   a.orgIDHeaderName,
									Value: projectID,
								},
							},
						},
					},
				},
			}, nil
		}
	}

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

func main() {
	log := createLogger()
	cfg, err := ctrlruntime.GetConfig()
	if err != nil {
		log.Fatalw("failed to get kubeconfig", zap.Error(err))
	}
	client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{})
	if err != nil {
		log.Panicw("failed to get client", zap.Error(err))
	}
	s := authorizationServer{}
	s.parse()
	s.client = client
	s.log = log

	lis, err := net.Listen("tcp", s.listenAddress)
	if err != nil {
		log.Fatalw("alertmanager authorization proxy failed to listen", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	authv3.RegisterAuthorizationServer(grpcServer, &s)

	if err = grpcServer.Serve(lis); err != nil {
		log.Fatalw("alertmanager authorization proxy failed to serve requests", zap.Error(err))
	}
}

func createLogger() *zap.SugaredLogger {
	logOpts := kubermaticlog.NewDefaultOptions()
	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	return rawLog.Sugar()
}
