//go:build ee

/*
                  Kubermatic Enterprise Read-Only License
                         Version 1.0 ("KERO-1.0”)
                     Copyright © 2022 Kubermatic GmbH

   1.	You may only view, read and display for studying purposes the source
      code of the software licensed under this license, and, to the extent
      explicitly provided under this license, the binary code.
   2.	Any use of the software which exceeds the foregoing right, including,
      without limitation, its execution, compilation, copying, modification
      and distribution, is expressly prohibited.
   3.	THE SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY OF ANY KIND,
      EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
      MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
      IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
      CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
      TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
      SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

   END OF TERMS AND CONDITIONS
*/

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	apiv2 "k8c.io/kubermatic/v2/pkg/api/v2"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	ViewersRole = "viewers"
	EditorsRole = "editors"
	OwnersRole  = "owners"
)

func ListGroupProjectBindings(ctx context.Context, request interface{},
	userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	bindingProvider provider.GroupProjectBindingProvider,
) ([]apiv2.GroupProjectBinding, error) {
	req, ok := request.(common.GetProjectRq)
	if !ok {
		return nil, utilerrors.NewBadRequest("invalid request")
	}

	if len(req.ProjectID) == 0 {
		return nil, utilerrors.NewBadRequest("the id of the project cannot be empty")
	}

	kubermaticProject, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	userInfo, err := userInfoGetter(ctx, kubermaticProject.Name)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	bindingList, err := bindingProvider.List(ctx, userInfo, kubermaticProject.Name)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	var bindingAPIObjList []apiv2.GroupProjectBinding
	for _, binding := range bindingList {
		bindingAPIObjList = append(bindingAPIObjList, apiv2.GroupProjectBinding{
			Name:      binding.Name,
			Group:     binding.Spec.Group,
			ProjectID: binding.Spec.ProjectID,
			Role:      binding.Spec.Role,
		})
	}

	return bindingAPIObjList, nil
}

// swagger:parameters getGroupProjectBinding
type groupProjectBindingReq struct {
	common.ProjectReq

	// in: path
	// required: true
	BindingName string `json:"binding_name"`
}

func DecodeGroupProjectBindingReq(r *http.Request) (interface{}, error) {
	var req groupProjectBindingReq

	req.ProjectID = mux.Vars(r)["project_id"]
	if req.ProjectID == "" {
		return nil, utilerrors.NewBadRequest("`project_id` cannot be empty")
	}

	req.BindingName = mux.Vars(r)["binding_name"]
	if req.BindingName == "" {
		return nil, utilerrors.NewBadRequest("`binding_name` cannot be empty")
	}

	return req, nil
}

func GetGroupProjectBinding(ctx context.Context, request interface{},
	userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	bindingProvider provider.GroupProjectBindingProvider,
) (*apiv2.GroupProjectBinding, error) {
	req, ok := request.(groupProjectBindingReq)
	if !ok {
		return nil, utilerrors.NewBadRequest("invalid request")
	}
	kubermaticProject, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	userInfo, err := userInfoGetter(ctx, kubermaticProject.Name)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	binding, err := bindingProvider.Get(ctx, userInfo, req.BindingName)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return &apiv2.GroupProjectBinding{
		Name:      binding.Name,
		Group:     binding.Spec.Group,
		ProjectID: binding.Spec.ProjectID,
		Role:      binding.Spec.Role,
	}, nil
}

// swagger:parameters createGroupProjectBinding
type createGroupProjectBindingReq struct {
	common.ProjectReq

	// in: body
	// required: true
	Body createGroupProjectBindingBody
}

func DecodeCreateGroupProjectBindingReq(r *http.Request) (interface{}, error) {
	var req createGroupProjectBindingReq

	req.ProjectID = mux.Vars(r)["project_id"]
	if req.ProjectID == "" {
		return nil, utilerrors.NewBadRequest("`project_id` cannot be empty")
	}

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, err
	}
	return req, nil
}

type createGroupProjectBindingBody struct {
	Role  string `json:"role"`
	Group string `json:"group"`
}

func (r createGroupProjectBindingReq) Validate() error {
	if r.Body.Group == "" {
		return utilerrors.NewBadRequest("`group` cannot be empty")
	}

	allowedRoles := sets.NewString(ViewersRole, EditorsRole, OwnersRole)
	if !allowedRoles.Has(r.Body.Role) {
		return utilerrors.NewBadRequest("allowed roles are: %v", strings.Join(allowedRoles.List(), ", "))
	}

	return nil
}

func CreateGroupProjectBinding(ctx context.Context, request interface{},
	userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	bindingProvider provider.GroupProjectBindingProvider,
) error {
	req, ok := request.(createGroupProjectBindingReq)
	if !ok {
		return utilerrors.NewBadRequest("invalid request")
	}

	err := req.Validate()
	if err != nil {
		return utilerrors.NewBadRequest(err.Error())
	}

	kubermaticProject, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
	if err != nil {
		return common.KubernetesErrorToHTTPError(err)
	}

	userInfo, err := userInfoGetter(ctx, kubermaticProject.Name)
	if err != nil {
		return err
	}

	if err := bindingProvider.Create(ctx, userInfo, &kubermaticv1.GroupProjectBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", kubermaticProject.Name, req.Body.Group),
		},
		Spec: kubermaticv1.GroupProjectBindingSpec{
			ProjectID: req.ProjectID,
			Group:     req.Body.Group,
			Role:      req.Body.Role,
		},
	}); err != nil {
		return common.KubernetesErrorToHTTPError(err)
	}
	return nil
}

func DeleteGroupProjectBinding(ctx context.Context, request interface{},
	userInfoGetter provider.UserInfoGetter,
	projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider,
	bindingProvider provider.GroupProjectBindingProvider,
) error {
	req, ok := request.(groupProjectBindingReq)
	if !ok {
		return utilerrors.NewBadRequest("invalid request")
	}
	kubermaticProject, err := common.GetProject(ctx, userInfoGetter, projectProvider, privilegedProjectProvider, req.ProjectID, nil)
	if err != nil {
		return common.KubernetesErrorToHTTPError(err)
	}

	userInfo, err := userInfoGetter(ctx, kubermaticProject.Name)
	if err != nil {
		return common.KubernetesErrorToHTTPError(err)
	}

	if err := bindingProvider.Delete(ctx, userInfo, req.BindingName); err != nil {
		return common.KubernetesErrorToHTTPError(err)
	}
	return nil
}
