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

	fluxeventv1beta1 "github.com/fluxcd/pkg/apis/event/v1beta1"

	"github.com/sap/component-operator-runtime/pkg/component"
	componentoperatorruntimetypes "github.com/sap/component-operator-runtime/pkg/types"
)

// ComponentSpec defines the desired state of Component.
type ComponentSpec struct {
	component.PlacementSpec     `json:",inline"`
	component.ClientSpec        `json:",inline"`
	component.ImpersonationSpec `json:",inline"`
	component.RequeueSpec       `json:",inline"`
	component.RetrySpec         `json:",inline"`
	component.TimeoutSpec       `json:",inline"`
	component.PolicySpec        `json:",inline"`
	component.TypeSpec          `json:",inline"`
	// +required
	SourceRef    SourceReference                `json:"sourceRef"`
	Digest       string                         `json:"digest,omitempty"`
	Revision     string                         `json:"revision,omitempty"`
	Path         string                         `json:"path,omitempty"`
	Values       *apiextensionsv1.JSON          `json:"values,omitempty"`
	ValuesFrom   []component.SecretKeyReference `json:"valuesFrom,omitempty" fallbackKeys:"values,values.yaml,values.yml" notFoundPolicy:"ignoreOnDeletion"`
	Decryption   *Decryption                    `json:"decryption,omitempty"`
	PostBuild    *PostBuild                     `json:"postBuild,omitempty"`
	Dependencies []Dependency                   `json:"dependencies,omitempty"`
}

// SourceReference models the source of the templates used to render the dependent resources.
// Exactly one of the options must be provided. Before accessing the Url(), Digest() or Revision() methods,
// a SourceReference must be loaded by calling Init().
type SourceReference struct {
	HttpRepository    *HttpRepository    `json:"httpRepository,omitempty"`
	FluxGitRepository *FluxGitRepository `json:"fluxGitRepository,omitempty"`
	FluxOciRepository *FluxOciRepository `json:"fluxOciRepository,omitempty"`
	FluxBucket        *FluxBucket        `json:"fluxBucket,omitempty"`
	FluxHelmChart     *FluxHelmChart     `json:"fluxHelmChart,omitempty"`
	url               string             `json:"-"`
	digest            string             `json:"-"`
	revision          string             `json:"-"`
	loaded            bool               `json:"-"`
}

// Initialize source reference. This is meant to be called by the reconciler.
// Other consumers should probably not (need to) call this.
func (r *SourceReference) Init(url string, digest string, revision string) {
	if r.loaded {
		// note: this panic indicates a programmatic error on the consumer side
		panic("reference already initialized")
	}
	r.url = url
	r.digest = digest
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

// Get the digest of a loaded source reference. Calling Digest() on a not-loaded source reference will panic.
// The returned digest uniquely identifies the content of the referenced archive.
func (r *SourceReference) Digest() string {
	if !r.loaded {
		// note: this panic indicates a programmatic error on the consumer side
		panic("access to unloaded reference")
	}
	return r.digest
}

// Get the revision of a loaded source reference. Calling Revision() on a not-loaded source reference will panic.
// The returned revision is often but not always unique for the referenced archive (usually a Git SHA or hash or digest).
func (r *SourceReference) Revision() string {
	if !r.loaded {
		// note: this panic indicates a programmatic error on the consumer side
		panic("access to unloaded reference")
	}
	return r.revision
}

// Check if source reference equals other given source reference.
func (r *SourceReference) Equals(s *SourceReference) bool {
	return equal(r.HttpRepository, s.HttpRepository) &&
		equal(r.FluxGitRepository, s.FluxGitRepository) &&
		equal(r.FluxOciRepository, s.FluxOciRepository) &&
		equal(r.FluxBucket, s.FluxBucket) &&
		equal(r.FluxHelmChart, s.FluxHelmChart)
}

// Reference to a generic http repository.
type HttpRepository struct {
	// URL of the source. Authentication is currently not supported. The operator will make HEAD requests to retrieve the digest/revision
	// and a potentially redirected actual location of the source artifact. Redirects will be followed as long as the response does not
	// contain the specified digest header.
	Url string `json:"url,omitempty"`
	// Name of the header containing the digest of the source artifact. The returned header value can be any format, but must uniquely identify the
	// content of the source artifact. Defaults to the ETag header.
	DigestHeader string `json:"digestHeader,omitempty"`
	// Name of the header containing the revision of the source artifact. The returned header value can be any format.
	// Defaults to the header specified in DigestHeader.
	RevisionHeader string `json:"revisionHeader,omitempty"`
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
	LastAttemptedDigest   string `json:"lastAttemptedDigest,omitempty"`
	LastAttemptedRevision string `json:"lastAttemptedRevision,omitempty"`
	LastAppliedDigest     string `json:"lastAppliedDigest,omitempty"`
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
	return c.Status.ObservedGeneration == c.Generation && c.Status.IsReady()
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

// Provide event metadata
// note: by implementing this method, component-operator-runtime will attach the returned annotations
// when calling EventRecorder.AnnotatedEventf(); also note that this controller wraps the standard
// event recorder with the flux notification recorder; this one expects annotation keys to be prefixed
// with the API group of the involved object ...
func (c *Component) GetEventAnnotations(previousState component.State, componentDigest string) map[string]string {
	annotations := make(map[string]string)
	annotations[fmt.Sprintf("%s/revision", GroupVersion.Group)] = c.Status.LastAttemptedRevision
	annotations[fmt.Sprintf("%s/%s", GroupVersion.Group, fluxeventv1beta1.MetaTokenKey)] = fmt.Sprintf("%s:%s", c.UID, componentDigest)
	if previousState != component.StateProcessing || c.Status.State != component.StateReady {
		annotations[fmt.Sprintf("%s/%s", GroupVersion.Group, fluxeventv1beta1.MetaCommitStatusKey)] = fluxeventv1beta1.MetaCommitStatusUpdateValue
	}
	return annotations
}

func equal[T comparable](x *T, y *T) bool {
	return x == nil && y == nil || x != nil && y != nil && *x == *y
}

func init() {
	SchemeBuilder.Register(&Component{}, &ComponentList{})
}
