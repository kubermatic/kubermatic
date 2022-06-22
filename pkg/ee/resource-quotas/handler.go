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

package resourcequotas

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"

	apiv1 "k8c.io/kubermatic/v2/pkg/api/v1"
	kubermaticv1 "k8c.io/kubermatic/v2/pkg/apis/kubermatic/v1"
	"k8c.io/kubermatic/v2/pkg/handler/v1/common"
	"k8c.io/kubermatic/v2/pkg/provider"
	utilerrors "k8c.io/kubermatic/v2/pkg/util/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
)

// swagger:parameters getResourceQuota deleteResourceQuota
type getResourceQuota struct {
	// in: path
	// required: true
	Name string `json:"quota_name"`
}

// swagger:parameters listResourceQuotas
type listResourceQuotas struct {
	// in: query
	// required: false
	SubjectName string `json:"subject_name,omitempty"`

	// in: query
	// required: false
	SubjectKind string `json:"subject_kind,omitempty"`
}

// swagger:parameters createResourceQuota
type createResourceQuota struct {
	// in: body
	// required: true
	Body struct {
		Subject kubermaticv1.Subject         `json:"subject"`
		Quota   kubermaticv1.ResourceDetails `json:"quota"`
	}
}

// swagger:parameters updateResourceQuota
type updateResourceQuota struct {
	// in: path
	// required: true
	Name string `json:"quota_name"`

	// in: body
	Body struct {
		CPU     *resource.Quantity `json:"cpu,omitempty"`
		Memory  *resource.Quantity `json:"memory,omitempty"`
		Storage *resource.Quantity `json:"storage,omitempty"`
	}
}

func (m createResourceQuota) Validate() error {
	if m.Body.Subject.Name == "" {
		return utilerrors.NewBadRequest("subject's name cannot be empty")
	}

	if m.Body.Subject.Kind == "" {
		return utilerrors.NewBadRequest("subject's kind cannot be empty")
	}

	return nil
}

func DecodeResourceQuotaReq(r *http.Request) (interface{}, error) {
	var req getResourceQuota

	req.Name = mux.Vars(r)["quota_name"]

	if req.Name == "" {
		return nil, utilerrors.NewBadRequest("`quota_name` cannot be empty")
	}

	return req, nil
}

func DecodeListResourceQuotaReq(r *http.Request) (interface{}, error) {
	var req listResourceQuotas

	req.SubjectName = r.URL.Query().Get("subjectName")
	req.SubjectKind = r.URL.Query().Get("subjectKind")

	return req, nil
}

func DecodeCreateResourceQuotaReq(r *http.Request) (interface{}, error) {
	var req createResourceQuota

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, utilerrors.NewBadRequest(err.Error())
	}

	return req, nil
}

func DecodeUpdateResourceQuotaReq(r *http.Request) (interface{}, error) {
	var req updateResourceQuota

	req.Name = mux.Vars(r)["quota_name"]

	if req.Name == "" {
		return nil, utilerrors.NewBadRequest("`quota_name` cannot be empty")
	}

	if err := json.NewDecoder(r.Body).Decode(&req.Body); err != nil {
		return nil, utilerrors.NewBadRequest(err.Error())
	}

	return req, nil
}

func GetResourceQuota(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider) (*apiv1.ResourceQuota, error) {
	req, ok := request.(getResourceQuota)
	if !ok {
		return nil, utilerrors.NewBadRequest("invalid request")
	}

	resourceQuota, err := provider.GetUnsecured(ctx, req.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, utilerrors.NewNotFound("ResourceQuota", req.Name)
		}
		return nil, err
	}

	resp := &apiv1.ResourceQuota{
		Name:        resourceQuota.Name,
		SubjectKind: resourceQuota.Spec.Subject.Kind,
		SubjectName: resourceQuota.Spec.Subject.Name,
		Quota:       resourceQuota.Spec.Quota,
		Status:      resourceQuota.Status,
	}

	return resp, nil
}

func GetResourceQuotaForProject(ctx context.Context, request interface{}, projectProvider provider.ProjectProvider,
	privilegedProjectProvider provider.PrivilegedProjectProvider, userInfoGetter provider.UserInfoGetter,
	quotaProvider provider.ResourceQuotaProvider) (*apiv1.ResourceQuota, error) {
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
		return nil, err
	}

	projectResourceQuota, err := quotaProvider.Get(ctx, userInfo, kubermaticProject.Name, kubermaticProject.Kind)
	if err != nil {
		return nil, common.KubernetesErrorToHTTPError(err)
	}

	return &apiv1.ResourceQuota{
		Name:        projectResourceQuota.Name,
		SubjectKind: projectResourceQuota.Spec.Subject.Kind,
		SubjectName: projectResourceQuota.Spec.Subject.Name,
		Quota:       projectResourceQuota.Spec.Quota,
		Status:      projectResourceQuota.Status,
	}, nil
}

func ListResourceQuotas(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider) ([]apiv1.ResourceQuota, error) {
	req, ok := request.(listResourceQuotas)
	if !ok {
		return nil, utilerrors.NewBadRequest("invalid request")
	}

	labelSet := make(map[string]string)
	if req.SubjectKind != "" {
		labelSet[kubermaticv1.ResourceQuotaSubjectKindLabelKey] = req.SubjectKind
	}
	if req.SubjectName != "" {
		labelSet[kubermaticv1.ResourceQuotaSubjectNameLabelKey] = req.SubjectName
	}

	resourceQuotaList, err := provider.ListUnsecured(ctx, labelSet)
	if err != nil {
		return nil, err
	}

	resp := make([]apiv1.ResourceQuota, len(resourceQuotaList.Items))
	for idx, rq := range resourceQuotaList.Items {
		resp[idx] = apiv1.ResourceQuota{
			Name:        rq.Name,
			SubjectKind: rq.Spec.Subject.Kind,
			SubjectName: rq.Spec.Subject.Name,
			Quota:       rq.Spec.Quota,
			Status:      rq.Status,
		}
	}

	return resp, nil
}

func CreateResourceQuota(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider) error {
	req, ok := request.(createResourceQuota)
	if !ok {
		return utilerrors.NewBadRequest("invalid request")
	}

	if err := req.Validate(); err != nil {
		return utilerrors.NewBadRequest(err.Error())
	}

	if err := provider.CreateUnsecured(ctx, req.Body.Subject, req.Body.Quota); err != nil {
		if apierrors.IsAlreadyExists(err) {
			name := buildNameFromSubject(req.Body.Subject)
			return utilerrors.NewAlreadyExists("ResourceQuota", name)
		}
		return err
	}
	return nil
}

func UpdateResourceQuota(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider) error {
	req, ok := request.(updateResourceQuota)
	if !ok {
		return utilerrors.NewBadRequest("invalid request")
	}

	newQuota := kubermaticv1.ResourceDetails{
		CPU:     req.Body.CPU,
		Memory:  req.Body.Memory,
		Storage: req.Body.Storage,
	}

	if err := provider.UpdateUnsecured(ctx, req.Name, newQuota); err != nil {
		if apierrors.IsNotFound(err) {
			return utilerrors.NewNotFound("ResourceQuota", req.Name)
		}
		return err
	}
	return nil
}

func DeleteResourceQuota(ctx context.Context, request interface{}, provider provider.ResourceQuotaProvider) error {
	req, ok := request.(getResourceQuota)
	if !ok {
		return utilerrors.NewBadRequest("invalid request")
	}

	if err := provider.DeleteUnsecured(ctx, req.Name); err != nil {
		if apierrors.IsNotFound(err) {
			return utilerrors.NewNotFound("ResourceQuota", req.Name)
		}
		return err
	}
	return nil
}
