package cluster

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-kit/kit/endpoint"

	apiv1 "github.com/kubermatic/kubermatic/api/pkg/api/v1"
	"github.com/kubermatic/kubermatic/api/pkg/handler/middleware"
	"github.com/kubermatic/kubermatic/api/pkg/handler/v1/common"
	"github.com/kubermatic/kubermatic/api/pkg/provider"
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	UserClusterRoleComponentKey   = "component"
	UserClusterRoleComponentValue = "userClusterRole"
)

func CreateClusterRoleEndpoint() endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(createReq)
		clusterProvider := ctx.Value(middleware.ClusterProviderContextKey).(provider.ClusterProvider)
		userInfo := ctx.Value(middleware.UserInfoContextKey).(*provider.UserInfo)

		cluster, err := clusterProvider.Get(userInfo, req.ClusterID, &provider.ClusterGetOptions{CheckInitStatus: true})
		if err != nil {
			return nil, err
		}

		client, err := clusterProvider.GetClientForCustomerCluster(userInfo, cluster)
		if err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		userClusterAPIRole := req.Body

		if userClusterAPIRole.Namespace != "" {
			role, err := generateRBACRole(userClusterAPIRole.Name, userClusterAPIRole.Namespace, userClusterAPIRole.Rules)
			if err != nil {
				return nil, errors.NewBadRequest("invalid role: %v", err)
			}
			if err := client.Create(ctx, role); err != nil {
				return nil, common.KubernetesErrorToHTTPError(err)
			}

			return convertInternalRoleToExternal(role), nil
		}

		clusterRole, err := generateRBACClusterRole(userClusterAPIRole.Name, userClusterAPIRole.Rules)
		if err != nil {
			return nil, errors.NewBadRequest("invalid cluster role: %v", err)
		}

		if err := client.Create(ctx, clusterRole); err != nil {
			return nil, common.KubernetesErrorToHTTPError(err)
		}
		return convertInternalClusterRoleToExternal(clusterRole), nil
	}
}

// createReq defines HTTP request for createClusterRole endpoint
// swagger:parameters createClusterRole
type createReq struct {
	common.DCReq
	// in: path
	// required: true
	ClusterID string `json:"cluster_id"`
	// in: body
	Body apiv1.UserClusterRole
}

func DecodeCreateClusterRoleReq(c context.Context, r *http.Request) (interface{}, error) {
	var req createReq
	clusterID, err := common.DecodeClusterID(c, r)
	if err != nil {
		return nil, err
	}

	req.ClusterID = clusterID

	dcr, err := common.DecodeDcReq(c, r)
	if err != nil {
		return nil, err
	}
	req.DCReq = dcr.(common.DCReq)

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}

	return req, nil
}

// generateRBACClusterRole creates cluster role
func generateRBACClusterRole(name string, rules []rbacv1.PolicyRule) (*rbacv1.ClusterRole, error) {
	if rules == nil {
		return nil, fmt.Errorf("the policy rule can not be nil")
	}
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: map[string]string{UserClusterRoleComponentKey: UserClusterRoleComponentValue},
		},
		Rules: rules,
	}
	return clusterRole, nil
}

// generateRBACRole creates role
func generateRBACRole(name, namespace string, rules []rbacv1.PolicyRule) (*rbacv1.Role, error) {
	if rules == nil {
		return nil, fmt.Errorf("the policy rule can not be nil")
	}
	clusterRole := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Labels:    map[string]string{UserClusterRoleComponentKey: UserClusterRoleComponentValue},
			Namespace: namespace,
		},
		Rules: rules,
	}
	return clusterRole, nil
}

func convertInternalClusterRoleToExternal(clusterRole *rbacv1.ClusterRole) *apiv1.UserClusterRole {
	return &apiv1.UserClusterRole{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                clusterRole.Name,
			Name:              clusterRole.Name,
			DeletionTimestamp: nil,
			CreationTimestamp: apiv1.NewTime(clusterRole.CreationTimestamp.Time),
		},
		Rules: clusterRole.Rules,
	}
}

func convertInternalRoleToExternal(clusterRole *rbacv1.Role) *apiv1.UserClusterRole {
	return &apiv1.UserClusterRole{
		ObjectMeta: apiv1.ObjectMeta{
			ID:                clusterRole.Name,
			Name:              clusterRole.Name,
			DeletionTimestamp: nil,
			CreationTimestamp: apiv1.NewTime(clusterRole.CreationTimestamp.Time),
		},
		Namespace: clusterRole.Namespace,
		Rules:     clusterRole.Rules,
	}
}
