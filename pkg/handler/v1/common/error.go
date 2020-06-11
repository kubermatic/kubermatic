package common

import (
	"github.com/kubermatic/kubermatic/api/pkg/util/errors"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
)

// kubernetesErrorToHTTPError constructs HTTPError only if the given err is of type *StatusError.
// Otherwise unmodified err will be returned to the caller.
func KubernetesErrorToHTTPError(err error) error {
	if kubernetesError, ok := err.(*kerrors.StatusError); ok {
		httpCode := kubernetesError.Status().Code
		httpMessage := kubernetesError.Status().Message
		return errors.New(int(httpCode), httpMessage)
	}
	return err
}
