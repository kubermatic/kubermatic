package test

import (
	"errors"
	"net/http"
	"time"

	kubermaticapiv1 "github.com/kubermatic/kubermatic/api/pkg/crd/kubermatic/v1"
	"github.com/kubermatic/kubermatic/api/pkg/provider"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NoExistingFakeProject      = "NoExistingFakeProject"
	NoExistingFakeProjectID    = "NoExistingFakeProject-ID"
	ForbiddenFakeProject       = "ForbiddenFakeProject"
	ForbiddenFakeProjectID     = "ForbiddenFakeProject-ID"
	ExistingFakeProject        = "ExistingFakeProject"
	ExistingFakeProjectID      = "ExistingFakeProject-ID"
	ImpersonatedClientErrorMsg = "forbidden"
)

type FakePrivilegedProjectProvider struct {
}

type FakeProjectProvider struct {
}

func NewFakeProjectProvider() *FakeProjectProvider {
	return &FakeProjectProvider{}
}

func NewFakePrivilegedProjectProvider() *FakePrivilegedProjectProvider {
	return &FakePrivilegedProjectProvider{}
}

func (f *FakeProjectProvider) New(user *kubermaticapiv1.User, name string, labels map[string]string) (*kubermaticapiv1.Project, error) {
	return nil, errors.New("not implemented")
}

// Delete deletes the given project as the given user
//
// Note:
// Before deletion project's status.phase is set to ProjectTerminating
func (f *FakeProjectProvider) Delete(userInfo *provider.UserInfo, projectInternalName string) error {
	return errors.New("not implemented")
}

// Get returns the project with the given name
func (f *FakeProjectProvider) Get(userInfo *provider.UserInfo, projectInternalName string, options *provider.ProjectGetOptions) (*kubermaticapiv1.Project, error) {
	if NoExistingFakeProjectID == projectInternalName || ForbiddenFakeProjectID == projectInternalName {
		return nil, createError(http.StatusForbidden, ImpersonatedClientErrorMsg)
	}

	return GenProject(ExistingFakeProject, kubermaticapiv1.ProjectActive, DefaultCreationTimestamp().Add(2*time.Minute)), nil
}

// Update update an existing project and returns it
func (f *FakeProjectProvider) Update(userInfo *provider.UserInfo, newProject *kubermaticapiv1.Project) (*kubermaticapiv1.Project, error) {
	return nil, errors.New("not implemented")
}

// List gets a list of projects, by default it returns all resources.
// If you want to filter the result please set ProjectListOptions
//
// Note that the list is taken from the cache
func (f *FakeProjectProvider) List(options *provider.ProjectListOptions) ([]*kubermaticapiv1.Project, error) {
	return nil, errors.New("not implemented")
}

// GetUnsecured returns the project with the given name
// This function is unsafe in a sense that it uses privileged account to get project with the given name
func (f *FakePrivilegedProjectProvider) GetUnsecured(projectInternalName string, options *provider.ProjectGetOptions) (*kubermaticapiv1.Project, error) {
	if NoExistingFakeProjectID == projectInternalName {
		return nil, createError(http.StatusNotFound, "")
	}
	if ForbiddenFakeProjectID == projectInternalName {
		return nil, createError(http.StatusForbidden, "")
	}

	return nil, nil
}

// DeleteUnsecured deletes any given project
// This function is unsafe in a sense that it uses privileged account to delete project with the given name
func (f *FakePrivilegedProjectProvider) DeleteUnsecured(projectInternalName string) error {
	return nil
}

// UpdateUnsecured update an existing project and returns it
// This function is unsafe in a sense that it uses privileged account to update project
func (f *FakePrivilegedProjectProvider) UpdateUnsecured(project *kubermaticapiv1.Project) (*kubermaticapiv1.Project, error) {
	return project, nil
}

func createError(status int32, message string) error {
	return &kerrors.StatusError{ErrStatus: metav1.Status{
		Status:  metav1.StatusFailure,
		Code:    status,
		Reason:  metav1.StatusReasonBadRequest,
		Message: message,
	}}
}
