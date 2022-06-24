//go:build ee

/*
Copyright 2022 The Kubermatic Kubernetes Platform contributors.

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

package mutation

import (
	"context"

	"github.com/go-logr/logr"

	eeresourcequotamutation "k8c.io/kubermatic/v2/pkg/ee/mutation/resourcequota"

	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func handle(ctx context.Context, req webhook.AdmissionRequest, decoder *admission.Decoder,
	logger logr.Logger, client ctrlruntimeclient.Client) webhook.AdmissionResponse {
	return eeresourcequotamutation.Handle(ctx, req, decoder, logger, client)
}
