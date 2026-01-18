/*
Copyright 2025 The Kubermatic Kubernetes Platform contributors.

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

package template

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"strconv"

	semverlib "github.com/Masterminds/semver/v3"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	kubermaticv1 "k8c.io/kubermatic/sdk/v2/apis/kubermatic/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrlruntimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// TemplateData is the root context injected into each application values.
type TemplateData struct {
	Cluster ClusterData
}

// ClusterData contains data related to the user cluster
// the application is rendered for.
type ClusterData struct {
	Name string
	// HumanReadableName is the user-specified cluster name.
	HumanReadableName string
	// OwnerEmail is the owner's e-mail address.
	OwnerEmail string
	// ClusterAddress stores access and address information of a cluster.
	Address kubermaticv1.ClusterAddress
	// Version is the exact current cluster version.
	Version string
	// MajorMinorVersion is a shortcut for common testing on "Major.Minor" on the
	// current cluster version.
	MajorMinorVersion string
	// AutoscalerVersion is the tag which should be used for the cluster autoscaler
	AutoscalerVersion string
	// Annotations holds arbitrary non-identifying metadata attached to the cluster.
	// Transferred from the Kubermatic cluster object.
	Annotations map[string]string
	// Labels are key-value pairs used to organize, categorize, and select clusters.
	// Transferred from the Kubermatic cluster object.
	Labels map[string]string
}

var (
	ErrBadTemplate     = errors.New("failed to render template:")
	ErrExistingAddon   = errors.New("addon is installed and enforced. Cannot continue with application installation.")
	AddonEnforcedLabel = "addons.kubermatic.io/ensure"
)

// HandleAddonCleanup deletes all addon resources for the specified one to avoid problems when migrating to applications.
func HandleAddonCleanup(ctx context.Context, applicationName string, seedClusterNamespace string, seedClient ctrlruntimeclient.Client, log *zap.SugaredLogger) error {
	existingAddon := &kubermaticv1.Addon{}

	if err := seedClient.Get(ctx, types.NamespacedName{Name: applicationName, Namespace: seedClusterNamespace}, existingAddon); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		// when we cannot found an addon anymore we expect no conflicting resources for the application
		return nil
	}

	addonEnforcedLabel, labelFound := existingAddon.GetLabels()[AddonEnforcedLabel]
	// we can suppress the error here because false is also correct for an empty string which is the case when no label is set
	addonEnforcedLabelParsed, _ := strconv.ParseBool(addonEnforcedLabel)

	// if the addon was found and the label to enforce the addon is set we cannot continue because we would have conflicting resources which are reconciled from addon and application for same workloads
	if labelFound && addonEnforcedLabelParsed {
		return ErrExistingAddon
	}

	if err := seedClient.Delete(ctx, existingAddon); err != nil {
		return err
	}

	log.Infof("addon %s removed. Continue with application installation.", applicationName)

	return nil
}

// GetTemplateData fetches the related cluster object by the given cluster namespace, parses pre defined values to a template data struct.
func GetTemplateData(ctx context.Context, seedClient ctrlruntimeclient.Client, clusterName string) (*TemplateData, error) {
	cluster := &kubermaticv1.Cluster{}
	if err := seedClient.Get(ctx, types.NamespacedName{Name: clusterName}, cluster); err != nil {
		return nil, err
	}
	var clusterVersion *semverlib.Version
	if s := cluster.Status.Versions.ControlPlane.Semver(); s != nil {
		clusterVersion = s
	} else {
		clusterVersion = cluster.Spec.Version.Semver()
	}
	if clusterVersion != nil {
		clusterAutoscalerVersion, err := GetAutoscalerImageTag(fmt.Sprintf("%d.%d", clusterVersion.Major(), clusterVersion.Minor()))
		if err != nil {
			return nil, fmt.Errorf("failed to parse autoscaler version for cluster %q", clusterName)
		}
		return &TemplateData{
			Cluster: ClusterData{
				Name:              cluster.Name,
				HumanReadableName: cluster.Spec.HumanReadableName,
				OwnerEmail:        cluster.Status.UserEmail,
				Address:           cluster.Status.Address,
				Version:           fmt.Sprintf("%d.%d.%d", clusterVersion.Major(), clusterVersion.Minor(), clusterVersion.Patch()),
				MajorMinorVersion: fmt.Sprintf("%d.%d", clusterVersion.Major(), clusterVersion.Minor()),
				AutoscalerVersion: clusterAutoscalerVersion,
				Annotations:       cluster.Annotations,
				Labels:            cluster.Labels,
			},
		}, nil
	}

	return nil, fmt.Errorf("failed to parse semver version for cluster %q", clusterName)
}

// RenderValueTemplate is rendering the given template data into the given map of values an error is returned when undefined values are used in the values map values.
func RenderValueTemplate(applicationValues map[string]interface{}, templateData *TemplateData) (map[string]interface{}, error) {
	yamlData, err := yaml.Marshal(applicationValues)
	if err != nil {
		return nil, err
	}

	parser := template.New("application-pre-defined-values")
	parsed, err := parser.Parse(string(yamlData))
	if err != nil {
		return nil, err
	}

	var buffer bytes.Buffer
	if err := parsed.Execute(&buffer, templateData); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrBadTemplate, err.Error())
	}

	parsedMap := make(map[string]interface{})
	err = yaml.Unmarshal(buffer.Bytes(), parsedMap)
	if err != nil {
		return nil, err
	}
	return parsedMap, nil
}

func GetAutoscalerImageTag(majorMinorVersion string) (string, error) {
	switch majorMinorVersion {
	case "1.32":
		return "v1.32.1", nil
	case "1.33":
		return "v1.33.0", nil
	case "1.34":
		return "v1.33.0", nil
	}
	return "", fmt.Errorf("could not find cluster autoscaler tag for cluster minor-major version %v", majorMinorVersion)
}
