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
	"time"

	"go.uber.org/zap"
	"google.golang.org/genproto/googleapis/rpc/code"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc"

	coreV3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	authv3 "github.com/envoyproxy/go-control-plane/envoy/service/auth/v3"
	typeV3 "github.com/envoyproxy/go-control-plane/envoy/type/v3"

	kubermaticv1 "k8c.io/kubermatic/v2/pkg/crd/kubermatic/v1"
	kubermaticlog "k8c.io/kubermatic/v2/pkg/log"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrlruntime "sigs.k8s.io/controller-runtime"
	ctrlcache "sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(kubermaticv1.AddToScheme(scheme))
}

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

	t := time.Now()
	a.log.Debugf("request: %s,time now: %s", req.Attributes.Request.Http.Path, t.String())
	authorized, err := a.authorize(ctx, userEmail, clusterID)
	a.log.Debugf("request: %s,time to do authorization: %s", req.Attributes.Request.Http.Path, time.Since(t))
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
	isAdmin, err := a.isAdminUser(ctx, userEmail)
	if err != nil {
		return false, fmt.Errorf("checking if user is admin: %w", err)
	}
	if isAdmin {
		return true, nil
	}
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

func (a *authorizationServer) isAdminUser(ctx context.Context, userEmail string) (bool, error) {
	users := &kubermaticv1.UserList{}
	if err := a.client.List(ctx, users); err != nil {
		return false, fmt.Errorf("listing user: %w", err)
	}

	for _, user := range users.Items {
		if strings.EqualFold(user.Spec.Email, userEmail) && user.Spec.IsAdmin {
			a.log.Debugf("user %q authorized as an admin", userEmail)
			return true, nil
		}
	}
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
	mapper, err := apiutil.NewDynamicRESTMapper(cfg)
	if err != nil {
		log.Fatalw("failed to create rest mapper", zap.Error(err))
	}
	cache, err := ctrlcache.New(cfg, ctrlcache.Options{
		Scheme: scheme,
		Mapper: mapper,
	})
	if err != nil {
		log.Fatalw("failed to create cache", zap.Error(err))
	}
	ctx := context.Background()
	go func() {
		if err := cache.Start(ctx); err != nil {
			log.Fatalw("failed to start cache", zap.Error(err))
		}
	}()
	if !cache.WaitForCacheSync(ctx) {
		log.Fatalw("cache is outdated")
	}
	cachedClient, err := ctrlruntimeclient.NewDelegatingClient(ctrlruntimeclient.NewDelegatingClientInput{
		CacheReader: cache,
		Client:      client,
	})
	if err != nil {
		log.Fatalw("failed to create cached client", zap.Error(err))
	}
	s.client = cachedClient
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
