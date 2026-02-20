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

package applicationcatalogmanager

const (
	// IncludeAnnotation is used to filter which default applications should be included
	// in the ApplicationCatalog. The value is a comma-separated list of application names.
	IncludeAnnotation = "defaultcatalog.k8c.io/include"

	// ManagedByLabelValue is the value for app.kubernetes.io/managed-by label.
	ManagedByLabelValue = "kubermatic-operator"

	// ComponentLabelValue is the value for app.kubernetes.io/component label.
	ComponentLabelValue = "application-catalog"

	// LabelManagedByApplicationCatalog is the label set by application-catalog-manager
	// to indicate that an ApplicationDefinition is managed by an ApplicationCatalog.
	LabelManagedByApplicationCatalog = "applicationcatalog.k8c.io/managed-by"

	// LabelApplicationCatalogName is the label set by application-catalog-manager
	// to indicate which ApplicationCatalog manages this ApplicationDefinition.
	LabelApplicationCatalogName = "applicationcatalog.k8c.io/catalog-name"
)
