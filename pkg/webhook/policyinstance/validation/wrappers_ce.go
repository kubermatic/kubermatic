//go:build !ee

package validation

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func validateCreate(_ context.Context, _ runtime.Object, _ ctrlruntimeclient.Client) error {
	return nil
}

func validateUpdate(_ context.Context, _ runtime.Object, _ runtime.Object, _ ctrlruntimeclient.Client) error {
	return nil
}

func validateDelete(_ context.Context, _ runtime.Object, _ ctrlruntimeclient.Client) error {
	return nil
}
