/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"fmt"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	apitypes "k8s.io/apimachinery/pkg/types"

	"github.com/sap/component-operator-runtime/pkg/component"
	componentoperatorruntimetypes "github.com/sap/component-operator-runtime/pkg/types"
)

// ComponentSpec defines the desired state of Component.
type ComponentSpec struct {
	component.PlacementSpec `json:",inline"`
	component.ClientSpec    `json:",inline"`
	component.RequeueSpec   `json:",inline"`
	component.RetrySpec     `json:",inline"`
	component.TimeoutSpec   `json:",inline"`
	// +required
	SourceRef    SourceReference                `json:"sourceRef"`
	Revision     string                         `json:"revision,omitempty"`
	Path         string                         `json:"path,omitempty"`
	Values       *apiextensionsv1.JSON          `json:"values,omitempty"`
	ValuesFrom   []component.SecretKeyReference `json:"valuesFrom,omitempty" fallbackKeys:"values,values.yaml,values.yml" notFoundPolicy:"ignoreOnDeletion"`
	Decryption   *Decryption                    `json:"decryption,omitempty"`
	PostBuild    *PostBuild                     `json:"postBuild,omitempty"`
	Dependencies []Dependency                   `json:"dependencies,omitempty"`
}

// SourceReference models the source of the templates used to render the dependent resources.
// Exactly one of the options must be provided. Before accessing the Url() or Revision() methods,
// a SourceReference must be loaded by calling LoadSourceReference().
type SourceReference struct {
	FluxGitRepository *FluxGitRepository `json:"fluxGitRepository,omitempty"`
	FluxOciRepository *FluxOciRepository `json:"fluxOciRepository,omitempty"`
	FluxBucket        *FluxBucket        `json:"fluxBucket,omitempty"`
	FluxHelmChart     *FluxHelmChart     `json:"fluxHelmChart,omitempty"`
	url               string             `json:"-"`
	revision          string             `json:"-"`
	loaded            bool               `json:"-"`
}

// Initialize source reference. This is meant to be called by the reconciler.
// Other consumers should probably not (need to) call this.
func (r *SourceReference) Init(url string, revision string) {
	if r.loaded {
		// note: this panic indicates a programmatic error on the consumer side
		panic("reference already initialized")
	}
	r.url = url
	r.revision = revision
	r.loaded = true
}

// Get the URL of a loaded source reference. Calling Url() on a not-loaded source reference will panic.
// The returned URL can be used to download a gzipped tar archive containing the templates.
// Furthermore, the URL will be unique for the archive's content.
func (r *SourceReference) Url() string {
	if !r.loaded {
		// note: this panic indicates a programmatic error on the consumer side
		panic("access to unloaded reference")
	}
	return r.url
}

// Get the revision of a loaded source reference. Calling Revision() on a not-loaded source reference will panic.
// The returned revision is unique for the referenced archive (usually a Git SHA or hash or digest).
func (r *SourceReference) Revision() string {
	if !r.loaded {
		// note: this panic indicates a programmatic error on the consumer side
		panic("access to unloaded reference")
	}
	return r.revision
}

// Check if source reference equals other given source reference.
func (r *SourceReference) Equals(s *SourceReference) bool {
	return equal(r.FluxGitRepository, s.FluxGitRepository) &&
		equal(r.FluxOciRepository, s.FluxOciRepository) &&
		equal(r.FluxBucket, s.FluxBucket) &&
		equal(r.FluxHelmChart, s.FluxHelmChart)
}

// Reference to a flux GitRepository.
type FluxGitRepository struct {
	NamespacedName `json:",inline"`
}

// Reference to a flux OCIRepository.
type FluxOciRepository struct {
	NamespacedName `json:",inline"`
}

// Reference to a flux Bucket.
type FluxBucket struct {
	NamespacedName `json:",inline"`
}

// Reference to a flux HelmChart.
type FluxHelmChart struct {
	NamespacedName `json:",inline"`
}

// Decryption settings.
type Decryption struct {
	// Decryption provider. Currently, the only supported value is 'sops', which is the default if the
	// field is omitted.
	Provider string `json:"provider,omitempty"`
	// Reference to a secret containing the provider configuration. The structure of the secret is the same
	// as the one used in flux Kustomization.
	SecretRef component.SecretReference `json:"secretRef" notFoundPolicy:"ignoreOnDeletion"`
}

// Post-build settings. The rendered manifests may contain patterns as defined by https://github.com/drone/envsubst.
// The according variables can provided inline by Substitute or as secrets by SubstituteFrom.
// If a variable name appears in more than one secret, then later values have precedence,
// and inline values have precedence over those defined through secrets.
type PostBuild struct {
	// Variables to be substituted in the renderered manifests.
	Substitute map[string]string `json:"substitute,omitempty"`
	// Secrets containing variables to be used for substitution.
	SubstituteFrom []component.SecretReference `json:"substituteFrom,omitempty" notFoundPolicy:"ignoreOnDeletion"`
}

// Dependency models a dependency of the containing component to another Component (referenced by namespace and name).
type Dependency struct {
	NamespacedName `json:",inline"`
}

// A tuple of namespace and name.
type NamespacedName struct {
	Namespace string `json:"namespace,omitempty"`
	// +required
	Name string `json:"name"`
}

// Return a copy of the given NamespacedName, using the specified namespace if none is set.
// The retrieving NamespaceName remains unchanged.
func (n NamespacedName) WithDefaultNamespace(namespace string) NamespacedName {
	if n.Namespace == "" {
		n.Namespace = namespace
	}
	return n
}

// Return a beautified string representation of the NamespacedName.
func (n NamespacedName) String() string {
	if n.Namespace == "" {
		return n.Name
	} else {
		return fmt.Sprintf("%s/%s", n.Namespace, n.Name)
	}
}

// ComponentStatus defines the observed state of Component.
type ComponentStatus struct {
	component.Status      `json:",inline"`
	LastAttemptedRevision string `json:"lastAttemptedRevision,omitempty"`
	LastAppliedRevision   string `json:"lastAppliedRevision,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +genclient

// Component is the Schema for the components API.
type Component struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ComponentSpec `json:"spec,omitempty"`
	// +kubebuilder:default={"observedGeneration":-1}
	Status ComponentStatus `json:"status,omitempty"`
}

var _ component.Component = &Component{}

// Get the object key (namespace and name) of the component.
func (c *Component) NamespacedName() apitypes.NamespacedName {
	return apitypes.NamespacedName{
		Namespace: c.Namespace,
		Name:      c.Name,
	}
}

// Reports the readiness of the component.
func (c *Component) IsReady() bool {
	return c.Status.IsReady()
}

// +kubebuilder:object:root=true

// ComponentList contains a list of Component.
type ComponentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Component `json:"items"`
}

// Implement the component-operator-runtime Component interface.
func (s *ComponentSpec) ToUnstructured() map[string]any {
	result, err := runtime.DefaultUnstructuredConverter.ToUnstructured(s)
	if err != nil {
		panic(err)
	}
	return result
}

// Implement the component-operator-runtime Component interface.
func (c *Component) GetSpec() componentoperatorruntimetypes.Unstructurable {
	return &c.Spec
}

// Implement the component-operator-runtime Component interface.
func (c *Component) GetStatus() *component.Status {
	return &c.Status.Status
}

func equal[T comparable](x *T, y *T) bool {
	return x == nil && y == nil || x != nil && y != nil && *x == *y
}

func init() {
	SchemeBuilder.Register(&Component{}, &ComponentList{})
}
