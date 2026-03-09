/*
Copyright 2020 The Kubermatic Kubernetes Platform contributors.

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

package v1

import (
	"encoding/json"
	"errors"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"
)

const (
	// ApplicationInstallationResourceName represents "Resource" defined in Kubernetes.
	ApplicationInstallationResourceName = "applicationinstallations"

	// ApplicationInstallationKindName represents "Kind" defined in Kubernetes.
	ApplicationInstallationKindName = "ApplicationInstallations"

	// ApplicationInstallationsFQDNName represents "FQDN" defined in Kubernetes.
	ApplicationInstallationsFQDNName = ApplicationInstallationResourceName + "." + GroupName
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=appinstall

// ApplicationInstallation describes a single installation of an Application.
type ApplicationInstallation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationInstallationSpec   `json:"spec,omitempty"`
	Status ApplicationInstallationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ApplicationInstallationList is a list of ApplicationInstallations.
type ApplicationInstallationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []ApplicationInstallation `json:"items"`
}

type ApplicationInstallationSpec struct {
	// Namespace describe the desired state of the namespace where application will be created.
	Namespace *AppNamespaceSpec `json:"namespace,omitempty"`

	// ApplicationRef is a reference to identify which Application should be deployed
	ApplicationRef ApplicationRef `json:"applicationRef"`

	// Values specify values overrides that are passed to helm templating. Comments are not preserved.
	// +kubebuilder:pruning:PreserveUnknownFields
	//
	// Deprecated: Use ValuesBlock instead. This field was deprecated in KKP 2.25 and will be removed in KKP 2.27+.
	Values runtime.RawExtension `json:"values,omitempty"`
	// As kubebuilder does not support interface{} as a type, deferring json decoding, seems to be our best option (see https://github.com/kubernetes-sigs/controller-tools/issues/294#issuecomment-518379253)

	// ValuesBlock specifies values overrides that are passed to helm templating. Comments are preserved.
	ValuesBlock string `json:"valuesBlock,omitempty"`

	// ReconciliationInterval is the interval at which to force the reconciliation of the application. By default, Applications are only reconciled
	// on changes on spec, annotations, or the parent application definition. Meaning that if the user manually deletes the workload
	// deployed by the application, nothing will happen until the application CR change.
	//
	// Setting a value greater than zero force reconciliation even if no changes occurred on application CR.
	// Setting a value equal to 0 disables the force reconciliation of the application (default behavior).
	// Setting this too low can cause a heavy load and may disrupt your application workload depending on the template method.
	ReconciliationInterval metav1.Duration `json:"reconciliationInterval,omitempty"`

	// DeployOptions holds the settings specific to the templating method used to deploy the application.
	DeployOptions *DeployOptions `json:"deployOptions,omitempty"`
}

// DeployOptions holds the settings specific to the templating method used to deploy the application.
type DeployOptions struct {
	Helm *HelmDeployOptions `json:"helm,omitempty"`
}

// HelmDeployOptions holds the deployment settings when templating method is Helm.
type HelmDeployOptions struct {
	// Wait corresponds to the --wait flag on Helm cli.
	// if set, will wait until all Pods, PVCs, Services, and minimum number of Pods of a Deployment, StatefulSet, or ReplicaSet are in a ready state before marking the release as successful. It will wait for as long as timeout
	Wait bool `json:"wait,omitempty"`

	// Timeout corresponds to the --timeout flag on Helm cli.
	// time to wait for any individual Kubernetes operation.
	Timeout metav1.Duration `json:"timeout,omitempty"`

	// Atomic corresponds to the --atomic flag on Helm cli.
	// if set, the installation process deletes the installation on failure; the upgrade process rolls back changes made in case of failed upgrade.
	Atomic bool `json:"atomic,omitempty"`

	// EnableDNS  corresponds to the --enable-dns flag on Helm cli.
	// enable DNS lookups when rendering templates.
	// if you enable this flag, you have to verify that helm template function 'getHostByName' is not being used in a chart to disclose any information you do not want to be passed to DNS servers.(c.f. CVE-2023-25165)
	EnableDNS bool `json:"enableDNS,omitempty"`
}

// AppNamespaceSpec describe the desired state of the namespace where application will be created.
type AppNamespaceSpec struct {
	// Name is the namespace to deploy the Application into.
	// Should be a valid lowercase RFC1123 domain name
	// +kubebuilder:validation:Pattern:=`^(|[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*)`
	// +kubebuilder:validation:MaxLength:=63
	// +kubebuilder:validation:Type=string
	Name string `json:"name"`

	// +kubebuilder:default:=true

	// Create defines whether the namespace should be created if it does not exist. Defaults to true
	Create bool `json:"create"`

	// Labels of the namespace
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations of the namespace
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ApplicationRef describes a KKP-wide, unique reference to an Application.
type ApplicationRef struct {
	// Name of the Application.
	// Should be a valid lowercase RFC1123 domain name
	// +kubebuilder:validation:Pattern:=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`
	// +kubebuilder:validation:MaxLength:=63
	// +kubebuilder:validation:Type=string
	Name string `json:"name"`

	// +kubebuilder:validation:Pattern:=v?([0-9]+)(\.[0-9]+)?(\.[0-9]+)?(-([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?(\+([0-9A-Za-z\-]+(\.[0-9A-Za-z\-]+)*))?
	// +kubebuilder:validation:Type=string

	// Version of the Application. Must be a valid SemVer version
	Version string `json:"version"`
	// (pattern taken from masterminds/semver we use https://github.com/Masterminds/semver/blob/master/version.go#L42)

	// NOTE: We are not using Masterminds/semver here, as it keeps data in unexported fields witch causes issues for
	// DeepEqual used in our reconciliation packages. At the same time, we are not using pkg/semver because
	// of the reasons stated in https://github.com/kubermatic/kubermatic/pull/10891.
}

// ApplicationInstallationStatus denotes status information about an ApplicationInstallation.
type ApplicationInstallationStatus struct {
	// Conditions contains conditions an installation is in, its primary use case is status signaling between controllers or between controllers and the API
	Conditions map[ApplicationInstallationConditionType]ApplicationInstallationCondition `json:"conditions,omitempty"`

	// ApplicationVersion contains information installing / removing application
	ApplicationVersion *ApplicationVersion `json:"applicationVersion,omitempty"`

	// Method used to install the application
	Method TemplateMethod `json:"method"`

	// HelmRelease holds the information about the helm release installed by this application. This field is only filled if template method is 'helm'.
	HelmRelease *HelmRelease `json:"helmRelease,omitempty"`

	// Failures counts the number of failed installation or updagrade. it is reset on successful reconciliation.
	Failures int `json:"failures,omitempty"`
}

type HelmRelease struct {
	// Name is the name of the release.
	Name string `json:"name,omitempty"`

	// Version is an int which represents the revision of the release.
	Version int `json:"version,omitempty"`

	// Info provides information about a release.
	Info *HelmReleaseInfo `json:"info,omitempty"`
}

// HelmReleaseStatus is the status of a Helm release. This type mirrors
// helm/pkg/release/v1.Status, but was copied here to avoid a very costly dependency.
// Since this field is only used in the status of an App, no user should ever
// have to set it manually.
//
// +kubebuilder:validation:Enum=unknown;deployed;uninstalled;superseded;failed;uninstalling;pending-install;pending-upgrade;pending-rollback
type HelmReleaseStatus string

// Describe the status of a release
// NOTE: Make sure to update cmd/helm/status.go when adding or modifying any of these statuses.
const (
	// HelmReleaseStatusUnknown indicates that a release is in an uncertain state.
	HelmReleaseStatusUnknown HelmReleaseStatus = "unknown"
	// HelmReleaseStatusDeployed indicates that the release has been pushed to Kubernetes.
	HelmReleaseStatusDeployed HelmReleaseStatus = "deployed"
	// HelmReleaseStatusUninstalled indicates that a release has been uninstalled from Kubernetes.
	HelmReleaseStatusUninstalled HelmReleaseStatus = "uninstalled"
	// HelmReleaseStatusSuperseded indicates that this release object is outdated and a newer one exists.
	HelmReleaseStatusSuperseded HelmReleaseStatus = "superseded"
	// HelmReleaseStatusFailed indicates that the release was not successfully deployed.
	HelmReleaseStatusFailed HelmReleaseStatus = "failed"
	// HelmReleaseStatusUninstalling indicates that an uninstall operation is underway.
	HelmReleaseStatusUninstalling HelmReleaseStatus = "uninstalling"
	// HelmReleaseStatusPendingInstall indicates that an install operation is underway.
	HelmReleaseStatusPendingInstall HelmReleaseStatus = "pending-install"
	// HelmReleaseStatusPendingUpgrade indicates that an upgrade operation is underway.
	HelmReleaseStatusPendingUpgrade HelmReleaseStatus = "pending-upgrade"
	// HelmReleaseStatusPendingRollback indicates that a rollback operation is underway.
	HelmReleaseStatusPendingRollback HelmReleaseStatus = "pending-rollback"
)

// HelmReleaseInfo describes release information.
// tech note: we can not use release.Info from Helm because the underlying type used for time has no json tag.
type HelmReleaseInfo struct {
	// FirstDeployed is when the release was first deployed.
	FirstDeployed metav1.Time `json:"firstDeployed,omitempty"`

	// LastDeployed is when the release was last deployed.
	LastDeployed metav1.Time `json:"lastDeployed,omitempty"`

	// Deleted tracks when this object was deleted.
	Deleted metav1.Time `json:"deleted,omitempty"`

	// Description is human-friendly "log entry" about this release.
	Description string `json:"description,omitempty"`

	// Status is the current state of the release.
	Status HelmReleaseStatus `json:"status,omitempty"`

	// Notes is  the rendered templates/NOTES.txt if available.
	Notes string `json:"notes,omitempty"`
}

type ApplicationInstallationCondition struct {
	// Status of the condition, one of True, False, Unknown.
	Status corev1.ConditionStatus `json:"status"`
	// Last time we got an update on a given condition.
	// +optional
	LastHeartbeatTime metav1.Time `json:"lastHeartbeatTime,omitempty"`
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// (brief) reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// Human readable message indicating details about last transition.
	Message string `json:"message,omitempty"`

	// observedGeneration represents the .metadata.generation that the condition was set based upon.
	// For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date
	// with respect to the current state of the instance.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// +kubebuilder:validation:Enum=ManifestsRetrieved;Ready

// swagger:enum ApplicationInstallationConditionType
// All condition types must be registered within the `AllApplicationInstallationConditionTypes` variable.
type ApplicationInstallationConditionType string

const (
	// ManifestsRetrieved indicates all necessary manifests have been fetched from the external source.
	ManifestsRetrieved ApplicationInstallationConditionType = "ManifestsRetrieved"

	// Ready describes all components have been successfully rolled out and are ready.
	Ready ApplicationInstallationConditionType = "Ready"
)

var AllApplicationInstallationConditionTypes = []ApplicationInstallationConditionType{
	ManifestsRetrieved,
	Ready,
}

// SetCondition of the applicationInstallation. It take care of update LastHeartbeatTime and LastTransitionTime if needed.
func (appInstallation *ApplicationInstallation) SetCondition(conditionType ApplicationInstallationConditionType, status corev1.ConditionStatus, reason, message string) {
	now := metav1.Now()

	condition, exists := appInstallation.Status.Conditions[conditionType]
	if exists && condition.Status != status {
		condition.LastTransitionTime = now
	}

	condition.Status = status
	condition.LastHeartbeatTime = now
	condition.Reason = reason
	condition.Message = message
	condition.ObservedGeneration = appInstallation.Generation

	if appInstallation.Status.Conditions == nil {
		appInstallation.Status.Conditions = map[ApplicationInstallationConditionType]ApplicationInstallationCondition{}
	}
	appInstallation.Status.Conditions[conditionType] = condition
}

// SetReadyCondition sets the ReadyCondition and appInstallation.Status.Failures counter according to the installError.
func (appInstallation *ApplicationInstallation) SetReadyCondition(installErr error, hasLimitedRetries bool) {
	if installErr != nil {
		appInstallation.SetCondition(Ready, corev1.ConditionFalse, "InstallationFailed", installErr.Error())
		if hasLimitedRetries { // increment only if limited retries to avoid overflow otherwise
			appInstallation.Status.Failures++
		}
	} else {
		appInstallation.SetCondition(Ready, corev1.ConditionTrue, "InstallationSuccessful", "application successfully installed or upgraded")
		appInstallation.Status.Failures = 0
	}
}

// GetParsedValues parses the values either from the Values or ValuesBlock field.
// Will return an error if both fields are set.
func (ai *ApplicationInstallationSpec) GetParsedValues() (map[string]interface{}, error) {
	values := make(map[string]interface{})
	if len(ai.Values.Raw) > 0 && string(ai.Values.Raw) != "{}" && ai.ValuesBlock != "" {
		return nil, errors.New("the fields Values and ValuesBlock cannot be used simultaneously, please delete one of them")
	}
	if len(ai.Values.Raw) > 0 && string(ai.Values.Raw) != "{}" {
		err := json.Unmarshal(ai.Values.Raw, &values)
		return values, err
	}
	err := yaml.Unmarshal([]byte(ai.ValuesBlock), &values)
	return values, err
}
