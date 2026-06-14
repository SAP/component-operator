/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"context"
	"fmt"
	"time"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	fluxeventv1beta1 "github.com/fluxcd/pkg/apis/event/v1beta1"
	fluxsourcev1 "github.com/fluxcd/source-controller/api/v1"
	fluxsourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/manifests"
	componentoperatorruntimetypes "github.com/sap/component-operator-runtime/pkg/types"

	httprepositoryutil "github.com/sap/component-operator/internal/httprepository/util"
	"github.com/sap/component-operator/internal/object"
	"github.com/sap/component-operator/pkg/meta"
)

// ComponentSpec defines the desired state of Component.
type ComponentSpec struct {
	component.PlacementSpec     `json:",inline"`
	component.ClientSpec        `json:",inline"`
	component.ImpersonationSpec `json:",inline"`
	component.SuspensionSpec    `json:",inline"`
	component.RequeueSpec       `json:",inline"`
	component.RetrySpec         `json:",inline"`
	component.TimeoutSpec       `json:",inline"`
	component.PolicySpec        `json:",inline"`
	component.TypeSpec          `json:",inline"`
	component.ReapplySpec       `json:",inline"`
	// +required
	SourceRef    SourceReference                `json:"sourceRef"`
	Digest       string                         `json:"digest,omitempty"`
	Revision     string                         `json:"revision,omitempty"`
	Sticky       bool                           `json:"sticky,omitempty"`
	Path         string                         `json:"path,omitempty"`
	Values       *apiextensionsv1.JSON          `json:"values,omitempty"`
	ValuesFrom   []component.SecretKeyReference `json:"valuesFrom,omitempty" fallbackKeys:"values,values.yaml,values.yml" notFoundPolicy:"ignoreOnDeletion"`
	Decryption   *Decryption                    `json:"decryption,omitempty"`
	PostBuild    *PostBuild                     `json:"postBuild,omitempty"`
	Dependencies []Dependency                   `json:"dependencies,omitempty"`
}

// +kubebuilder:validation:XValidation:rule="has(self.blueprint) && !has(self.httpRepository) && !has(self.fluxGitRepository) && !has(self.fluxOciRepository) && !has(self.fluxBucket) && !has(self.fluxHelmChart) || !has(self.blueprint) && has(self.httpRepository) && !has(self.fluxGitRepository) && !has(self.fluxOciRepository) && !has(self.fluxBucket) && !has(self.fluxHelmChart) || !has(self.blueprint) && !has(self.httpRepository) && has(self.fluxGitRepository) && !has(self.fluxOciRepository) && !has(self.fluxBucket) && !has(self.fluxHelmChart) || !has(self.blueprint) && !has(self.httpRepository) && !has(self.fluxGitRepository) && has(self.fluxOciRepository) && !has(self.fluxBucket) && !has(self.fluxHelmChart) || !has(self.blueprint) && !has(self.httpRepository) && !has(self.fluxGitRepository) && !has(self.fluxOciRepository) && has(self.fluxBucket) && !has(self.fluxHelmChart) || !has(self.blueprint) && !has(self.httpRepository) && !has(self.fluxGitRepository) && !has(self.fluxOciRepository) && !has(self.fluxBucket) && has(self.fluxHelmChart)",message="Exactly one of 'blueprint' or 'httpRepository' or 'fluxGitRepository' or 'fluxOciRepository' or 'fluxBucket' or 'fluxHelmChart' must be provided"

// SourceReference models the source of the templates used to render the dependent resources.
// Exactly one of the options must be provided. Before accessing the Artifact() method,
// the SourceReference must be loaded by calling Load().
type SourceReference struct {
	Blueprint         *BlueprintReference         `json:"blueprint,omitempty"`
	HttpRepository    *HttpRepository             `json:"httpRepository,omitempty"`
	FluxGitRepository *FluxGitRepositoryReference `json:"fluxGitRepository,omitempty"`
	FluxOciRepository *FluxOciRepositoryReference `json:"fluxOciRepository,omitempty"`
	FluxBucket        *FluxBucketReference        `json:"fluxBucket,omitempty"`
	FluxHelmChart     *FluxHelmChartReference     `json:"fluxHelmChart,omitempty"`
	artifact          Artifact                    `json:"-"`
	digest            string                      `json:"-"`
	loaded            bool                        `json:"-"`
}

var _ component.Reference[*Component] = &SourceReference{}

// Implement the component.Reference interface.
func (r *SourceReference) Load(ctx context.Context, clnt client.Client, component *Component) error {
	if r.loaded {
		// note: this panic indicates a programmatic error on the consumer side
		panic("reference already initialized")
	}

	if !component.DeletionTimestamp.IsZero() {
		return nil
	}

	spec := &component.Spec
	status := &component.Status

	if spec.Sticky && isComponentProcessing(component) && status.SourceRef != nil {
		r.artifact = status.SourceRef.Artifact
		r.digest = status.SourceRef.Digest
		r.loaded = true
	} else {
		sourceRef := &spec.SourceRef
		sourceRefArtifact := Artifact{}
		var digestData []any

		switch {
		case sourceRef.Blueprint != nil:
			blueprint := &Blueprint{}
			if err := clnt.Get(ctx, apitypes.NamespacedName(sourceRef.Blueprint.WithDefaultNamespace(component.Namespace)), blueprint); err != nil {
				if apierrors.IsNotFound(err) {
					return componentoperatorruntimetypes.NewRetriableError(err, new(10*time.Second))
				}
				return err
			}

			blueprintDigest := blueprint.GetDigest()
			blueprintRevision := blueprint.GetRevision()
			blueprintVersion := &BlueprintVersion{
				TypeMeta: metav1.TypeMeta{
					Kind:       KindBlueprintVersion,
					APIVersion: GroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("%s--%s", blueprint.Name, blueprintDigest),
					Namespace: blueprint.Namespace,
				},
				Spec: BlueprintVersionSpec{
					Blueprint:     blueprint.Name,
					Digest:        blueprintDigest,
					Revision:      blueprintRevision,
					BlueprintSpec: blueprint.Spec,
				},
			}
			if err := clnt.Patch(ctx, blueprintVersion, client.Apply, client.FieldOwner(meta.Name), client.ForceOwnership); err != nil {
				return err
			}

			sourceRefArtifact.Url = fmt.Sprintf("blueprint://%s/%s/%s", blueprint.Namespace, blueprint.Name, blueprintDigest)
			sourceRefArtifact.Digest = blueprintDigest
			sourceRefArtifact.Revision = blueprintRevision
			digestData = []any{sourceRefArtifact.Url, sourceRefArtifact.Digest, sourceRefArtifact.Revision}
		case sourceRef.HttpRepository != nil:
			url, digest, revision, err := httprepositoryutil.GetArtifact(sourceRef.HttpRepository.Url, sourceRef.HttpRepository.DigestHeader, sourceRef.HttpRepository.RevisionHeader)
			if err != nil {
				return err
			}

			sourceRefArtifact.Url = url
			sourceRefArtifact.Digest = digest
			sourceRefArtifact.Revision = revision
			digestData = []any{sourceRefArtifact.Url, sourceRefArtifact.Digest, sourceRefArtifact.Revision}
		case sourceRef.FluxGitRepository != nil, sourceRef.FluxOciRepository != nil, sourceRef.FluxBucket != nil, sourceRef.FluxHelmChart != nil:
			var sourceName NamespacedName
			var source meta.FluxSource

			switch {
			case sourceRef.FluxGitRepository != nil:
				sourceName = sourceRef.FluxGitRepository.WithDefaultNamespace(component.Namespace)
				source = &fluxsourcev1.GitRepository{}
			case sourceRef.FluxOciRepository != nil:
				sourceName = sourceRef.FluxOciRepository.WithDefaultNamespace(component.Namespace)
				source = &fluxsourcev1beta2.OCIRepository{}
			case sourceRef.FluxBucket != nil:
				sourceName = sourceRef.FluxBucket.WithDefaultNamespace(component.Namespace)
				source = &fluxsourcev1beta2.Bucket{}
			case sourceRef.FluxHelmChart != nil:
				sourceName = sourceRef.FluxHelmChart.WithDefaultNamespace(component.Namespace)
				source = &fluxsourcev1.HelmChart{}
			default:
				panic("this cannot happen")
			}

			if err := clnt.Get(ctx, apitypes.NamespacedName(sourceName), source); err != nil {
				if apimeta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
					return componentoperatorruntimetypes.NewRetriableError(err, new(10*time.Second))
				}
				return err
			}
			if !object.IsReady(source) {
				return componentoperatorruntimetypes.NewRetriableError(fmt.Errorf("source not ready"), new(10*time.Second))
			}

			artifact := source.GetArtifact()
			if artifact == nil {
				return componentoperatorruntimetypes.NewRetriableError(fmt.Errorf("missing artifact on ready source"), new(10*time.Second))
			}

			if artifact.URL == "" {
				return componentoperatorruntimetypes.NewRetriableError(fmt.Errorf("source not ready (missing URL)"), new(10*time.Second))
			}
			if artifact.Digest == "" {
				return componentoperatorruntimetypes.NewRetriableError(fmt.Errorf("source not ready (missing digest)"), new(10*time.Second))
			}
			if artifact.Revision == "" {
				return componentoperatorruntimetypes.NewRetriableError(fmt.Errorf("source not ready (missing revision)"), new(10*time.Second))
			}

			sourceRefArtifact.Url = artifact.URL
			sourceRefArtifact.Digest = artifact.Digest
			sourceRefArtifact.Revision = artifact.Revision
			digestData = []any{source.GetUID(), source.GetGeneration(), source.GetAnnotations(), sourceRefArtifact.Url, sourceRefArtifact.Digest, sourceRefArtifact.Revision}
		default:
			return fmt.Errorf("unable to get source; one of httpRepository, fluxGitRepository, fluxOciRepository, fluxBucket, fluxHelmChart must be defined")
		}

		r.artifact = sourceRefArtifact
		r.digest = calculateDigest(digestData...)
		r.loaded = true

		status.SourceRef = &SourceReferenceStatus{
			Artifact: r.artifact,
			Digest:   r.digest,
		}
	}

	return nil
}

// Implement the component.Reference interface.
func (r *SourceReference) Digest() string {
	if !r.loaded {
		return ""
	}
	return r.digest
}

// Get the artifact of a loaded source reference. Calling Artifact() on a not-loaded source
// reference will panic.
func (r *SourceReference) Artifact() Artifact {
	if !r.loaded {
		// note: this panic indicates a programmatic error on the consumer side
		panic("access to unloaded reference")
	}
	return r.artifact
}

// Check if source reference equals other given source reference.
func (r *SourceReference) Equals(s *SourceReference) bool {
	return equal(r.Blueprint, s.Blueprint) &&
		equal(r.HttpRepository, s.HttpRepository) &&
		equal(r.FluxGitRepository, s.FluxGitRepository) &&
		equal(r.FluxOciRepository, s.FluxOciRepository) &&
		equal(r.FluxBucket, s.FluxBucket) &&
		equal(r.FluxHelmChart, s.FluxHelmChart)
}

// Reference to a Blueprint.
type BlueprintReference struct {
	NamespacedName `json:",inline"`
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
type FluxGitRepositoryReference struct {
	NamespacedName `json:",inline"`
}

// Reference to a flux OCIRepository.
type FluxOciRepositoryReference struct {
	NamespacedName `json:",inline"`
}

// Reference to a flux Bucket.
type FluxBucketReference struct {
	NamespacedName `json:",inline"`
}

// Reference to a flux HelmChart.
type FluxHelmChartReference struct {
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
// Furthermore, kustomize patches and image replacements can be defined, which are applied after the variable substitution.
type PostBuild struct {
	// Variables to be substituted in the renderered manifests.
	Substitute map[string]string `json:"substitute,omitempty"`
	// Secrets containing variables to be used for substitution.
	SubstituteFrom []component.SecretReference `json:"substituteFrom,omitempty" notFoundPolicy:"ignoreOnDeletion"`
	// A list of kustomize patches.
	Patches []manifests.KustomizePatch `json:"patches,omitempty"`
	// A list of kustomize image replacements.
	Images []manifests.KustomizeImage `json:"images,omitempty"`
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
	SourceRef             *SourceReferenceStatus `json:"sourceRef,omitempty"`
	LastAttemptedDigest   string                 `json:"lastAttemptedDigest,omitempty"`
	LastAttemptedRevision string                 `json:"lastAttemptedRevision,omitempty"`
	LastAppliedDigest     string                 `json:"lastAppliedDigest,omitempty"`
	LastAppliedRevision   string                 `json:"lastAppliedRevision,omitempty"`
}

type SourceReferenceStatus struct {
	Artifact Artifact `json:"artifact,omitempty"`
	Digest   string   `json:"digest,omitempty"`
}

// Artifact describes the underlying source artifact.
type Artifact struct {
	Url      string `json:"url"`
	Digest   string `json:"digest"`
	Revision string `json:"revision"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=='Ready')].reason`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=`.metadata.creationTimestamp`
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
func (c *Component) GetEventAnnotations(componentDigest string) map[string]string {
	annotations := make(map[string]string)
	// TODO: derive repositoryOwner, repositoryName from sourceRef (if GitRepository)
	// TODO: derive context from component annotation (or similar)
	annotations[fmt.Sprintf("%s/%s", GroupVersion.Group, fluxeventv1beta1.MetaRevisionKey)] = c.Status.LastAttemptedRevision
	annotations[fmt.Sprintf("%s/%s", GroupVersion.Group, fluxeventv1beta1.MetaTokenKey)] = fmt.Sprintf("%s:%s", c.UID, componentDigest)
	return annotations
}

func isComponentProcessing(c *Component) bool {
	// TODO: this is not good; it duplicates the defaulting logic for the timeout from component-operator-runtime,
	// which is error-prone; overall it would be good to have an exact timeout indicator on the status,
	// managed through component-operator-runtime itself, or if this method would be offered by component-operator-runtime
	timeout := 10 * time.Minute
	if c.Spec.RequeueInterval != nil {
		timeout = c.Spec.RequeueInterval.Duration
	}
	if c.Spec.Timeout != nil {
		timeout = c.Spec.Timeout.Duration
	}

	// TODO: is it wise to use c.Status.LastObservedAt here, or would it make things faster to use just time.Now()?
	return c.Status.ProcessingSince != nil && c.Status.LastObservedAt.Sub(c.Status.ProcessingSince.Time) < timeout
}

// BlueprintSpec defines the desired state of Blueprint.
type BlueprintSpec struct {
	Files map[string]string `json:"files,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +genclient

// Blueprint is the Schema for the blueprints API.
type Blueprint struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BlueprintSpec `json:"spec"`
}

func (b *Blueprint) GetDigest() string {
	return calculateDigest(b.Spec)
}

func (b *Blueprint) GetRevision() string {
	return fmt.Sprintf("generation:%d", b.Generation)
}

// +kubebuilder:object:root=true

// BlueprintList contains a list of Blueprint.
type BlueprintList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Blueprint `json:"items"`
}

// BlueprintVersionSpec defines the desired state of BlueprintVersion.
type BlueprintVersionSpec struct {
	Blueprint     string `json:"blueprint"`
	Digest        string `json:"digest"`
	Revision      string `json:"revision"`
	BlueprintSpec `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:selectablefield:JSONPath=".spec.blueprint"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +genclient

// BlueprintVersion is the Schema for the blueprint versions API.
type BlueprintVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec BlueprintVersionSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// BlueprintVersionList contains a list of BlueprintVersion.
type BlueprintVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []BlueprintVersion `json:"items"`
}

const (
	KindComponent        = "Component"
	KindBlueprint        = "Blueprint"
	KindBlueprintVersion = "BlueprintVersion"
)

func init() {
	SchemeBuilder.Register(&Component{}, &ComponentList{})
	SchemeBuilder.Register(&Blueprint{}, &BlueprintList{})
	SchemeBuilder.Register(&BlueprintVersion{}, &BlueprintVersionList{})
}
