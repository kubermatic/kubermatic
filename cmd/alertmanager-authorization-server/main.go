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
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"strings"

	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntime "sigs.k8s.io/controller-runtime"

	coreV3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	typeV3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

type authorizationServer struct {
	listenAddress   string
	authHeaderName  string
	orgIDHeaderName string

	client ctrlruntimeclient.Client
	log    *zap.SugaredLogger
}

func (a *authorizationServer) Check(ctx context.Context, req *authv3.CheckRequest) (*authv3.CheckResponse, error) {
	a.log.Debug(">>> Authorization Check")

	b, err := json.MarshalIndent(req.Attributes.Request.Http.Headers, "", "  ")
	if err == nil {
		a.log.Debug("Inbound Headers: ")
		a.log.Debug(string(b))
	}
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

	// parse cluster ID from the original request path
	clusterID := ""
	arr := strings.Split(req.Attributes.Request.Http.Path, "/")
	if len(arr) > 1 {
		clusterID = arr[1]
	}
	if clusterID == "" {
		a.log.Debug("cluster ID cannot be parsed")
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

	authorized, err := a.authorize(ctx, userEmail, clusterID)
	if err != nil {
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
								Value: clusterID,
							},
						},
					},
				},
			},
		}, nil
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

func (a *authorizationServer) authorize(ctx context.Context, userEmail, clusterID string) (authorized bool, err error) {
	cluster := &kubermaticv1.Cluster{}
	if err := a.client.Get(ctx, types.NamespacedName{
		Name: clusterID,
	}, cluster); err != nil {
		return false, fmt.Errorf("getting cluster: %w", err)
	}

	projectID := cluster.Labels[kubermaticv1.ProjectIDLabelKey]
	if len(projectID) == 0 {
		return false, fmt.Errorf("cluster %s is missing '%s' label", cluster.Name, kubermaticv1.ProjectIDLabelKey)
	}

	allMembers := &kubermaticv1.UserProjectBindingList{}
	if err := a.client.List(ctx, allMembers); err != nil {
		return false, fmt.Errorf("listing userProjectBinding: %w", err)
	}

	for _, member := range allMembers.Items {
		if strings.EqualFold(member.Spec.UserEmail, userEmail) && member.Spec.ProjectID == projectID {
			a.log.Debugf("user %q authorized for project: %s", userEmail, projectID)
			return true, nil
		}
	}
	a.log.Debugf("user %q is NOT authorized for project: %s, cluster %s", userEmail, projectID, clusterID)
	return false, nil
}

func (a *authorizationServer) addFlags() {
	flag.StringVar(&a.listenAddress, "address", ":50051", "the address to listen on")
	flag.StringVar(&a.authHeaderName, "auth-header-name", "x-forwarded-email", "alertmanager authorization server http header that will contain the email")
	flag.StringVar(&a.orgIDHeaderName, "org-id-header-name", "X-Scope-OrgID", "the header that alertmanager uses for multi-tenancy support")
}

func main() {
	logOpts := kubermaticlog.NewDefaultOptions()
	logOpts.AddFlags(flag.CommandLine)
	s := authorizationServer{}
	s.addFlags()
	flag.Parse()

	rawLog := kubermaticlog.New(logOpts.Debug, logOpts.Format)
	log := rawLog.Sugar()
	cfg, err := ctrlruntime.GetConfig()
	if err != nil {
		log.Fatalw("failed to get kubeconfig", zap.Error(err))
	}
	client, err := ctrlruntimeclient.New(cfg, ctrlruntimeclient.Options{})
	if err != nil {
		log.Panicw("failed to get client", zap.Error(err))
	}
	s.client = client
	s.log = log

	lis, err := net.Listen("tcp", s.listenAddress)
	if err != nil {
		log.Fatalw("alertmanager authorization server failed to listen", zap.Error(err))
	}
	grpcServer := grpc.NewServer()
	authv3.RegisterAuthorizationServer(grpcServer, &s)
	if err = grpcServer.Serve(lis); err != nil {
		log.Fatalw("alertmanager authorization server failed to serve requests", zap.Error(err))
	}
}
