/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/sap/go-generics/slices"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	fluxsourcev1 "github.com/fluxcd/source-controller/api/v1"
	fluxsourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
	"github.com/sap/component-operator/pkg/meta"
)

func MatchingDependency(component *operatorv1alpha1.Component) client.ListOption {
	return client.MatchingFields{dependenciesIndexKey: client.ObjectKeyFromObject(component).String()}
}

func MatchingBlueprint(blueprint *operatorv1alpha1.Blueprint) client.ListOption {
	return client.MatchingFields{blueprintIndexKey: client.ObjectKeyFromObject(blueprint).String()}
}

func MatchingBlueprintVersion(blueprintVersion *operatorv1alpha1.BlueprintVersion) client.ListOption {
	return client.MatchingFields{blueprintVersionIndexKey: client.ObjectKeyFromObject(blueprintVersion).String()}
}

func MatchingFluxSource(source meta.FluxSource) client.ListOption {
	indexKey := ""
	switch source.(type) {
	case *fluxsourcev1.GitRepository:
		indexKey = gitRepositoryIndexKey
	case *fluxsourcev1beta2.OCIRepository:
		indexKey = ociRepositoryIndexKey
	case *fluxsourcev1beta2.Bucket:
		indexKey = bucketIndexKey
	case *fluxsourcev1.HelmChart:
		indexKey = helmChartIndexKey
	default:
		panic("this cannot happen")
	}
	return client.MatchingFields{indexKey: client.ObjectKeyFromObject(source).String()}
}

func HasBlueprint() client.ListOption {
	return client.MatchingFields{sourceTypeIndexKey: sourceTypeBlueprint}
}

func HasHttpRepository() client.ListOption {
	return client.MatchingFields{sourceTypeIndexKey: sourceTypeHttpRepository}
}

func HasFluxGitRepository() client.ListOption {
	return client.MatchingFields{sourceTypeIndexKey: sourceTypeFluxGitRepository}
}

func HasFluxOciRepository() client.ListOption {
	return client.MatchingFields{sourceTypeIndexKey: sourceTypeFluxOciRepository}
}

func HasFluxBucket() client.ListOption {
	return client.MatchingFields{sourceTypeIndexKey: sourceTypeFluxBucket}
}

func HasFluxHelmChart() client.ListOption {
	return client.MatchingFields{sourceTypeIndexKey: sourceTypeFluxHelmChart}
}

func HasFluxSource() client.ListOption {
	return client.MatchingFields{sourceTypeIndexKey: sourceTypeFluxSource}
}

const (
	sourceTypeIndexKey string = ".metadata.sourceType"

	dependenciesIndexKey string = ".metadata.dependencies"

	blueprintIndexKey        string = ".metadata.cs.blueprint"
	blueprintVersionIndexKey string = ".metadata.cs.blueprintversion"

	gitRepositoryIndexKey string = ".metadata.flux.gitRepository"
	ociRepositoryIndexKey string = ".metadata.flux.ociRepository"
	bucketIndexKey        string = ".metadata.flux.bucket"
	helmChartIndexKey     string = ".metadata.flux.helmChart"
)

const (
	sourceTypeBlueprint         string = "blueprint"
	sourceTypeHttpRepository    string = "httpRepository"
	sourceTypeFluxGitRepository string = "fluxGitRepository"
	sourceTypeFluxOciRepository string = "fluxOciRepository"
	sourceTypeFluxBucket        string = "fluxBucket"
	sourceTypeFluxHelmChart     string = "fluxHelmChart"
	sourceTypeFluxSource        string = "fluxSource"
)

func SetupWithManager(mgr manager.Manager) error {
	// TODO: should we pass a meaningful context?
	if err := mgr.GetCache().IndexField(context.TODO(), &operatorv1alpha1.Component{}, sourceTypeIndexKey, indexBySourceType); err != nil {
		return errors.Wrapf(err, "failed setting index field %s", sourceTypeIndexKey)
	}

	if err := mgr.GetCache().IndexField(context.TODO(), &operatorv1alpha1.Component{}, dependenciesIndexKey, indexByDependencies); err != nil {
		return errors.Wrapf(err, "failed setting index field %s", dependenciesIndexKey)
	}

	if err := mgr.GetCache().IndexField(context.TODO(), &operatorv1alpha1.Component{}, blueprintIndexKey, indexByBlueprint); err != nil {
		return errors.Wrapf(err, "failed setting index field %s", blueprintIndexKey)
	}
	if err := mgr.GetCache().IndexField(context.TODO(), &operatorv1alpha1.Component{}, blueprintVersionIndexKey, indexByBlueprintVersion); err != nil {
		return errors.Wrapf(err, "failed setting index field %s", blueprintVersionIndexKey)
	}

	if err := mgr.GetCache().IndexField(context.TODO(), &operatorv1alpha1.Component{}, gitRepositoryIndexKey, indexCompoonentByGitRepository); err != nil {
		return errors.Wrapf(err, "failed setting index field %s", gitRepositoryIndexKey)
	}
	if err := mgr.GetCache().IndexField(context.TODO(), &operatorv1alpha1.Component{}, ociRepositoryIndexKey, indexCompoonentByOciRepository); err != nil {
		return errors.Wrapf(err, "failed setting index field %s", ociRepositoryIndexKey)
	}
	if err := mgr.GetCache().IndexField(context.TODO(), &operatorv1alpha1.Component{}, bucketIndexKey, indexCompoonentByBucket); err != nil {
		return errors.Wrapf(err, "failed setting index field %s", bucketIndexKey)
	}
	if err := mgr.GetCache().IndexField(context.TODO(), &operatorv1alpha1.Component{}, helmChartIndexKey, indexCompoonentByHelmChart); err != nil {
		return errors.Wrapf(err, "failed setting index field %s", helmChartIndexKey)
	}

	return nil
}

func indexBySourceType(object client.Object) []string {
	component := object.(*operatorv1alpha1.Component)
	if component.Spec.SourceRef.Blueprint != nil {
		return []string{sourceTypeBlueprint}
	}
	if component.Spec.SourceRef.HttpRepository != nil {
		return []string{sourceTypeHttpRepository}
	}
	if component.Spec.SourceRef.FluxGitRepository != nil {
		return []string{sourceTypeFluxGitRepository, sourceTypeFluxSource}
	}
	if component.Spec.SourceRef.FluxOciRepository != nil {
		return []string{sourceTypeFluxOciRepository, sourceTypeFluxSource}
	}
	if component.Spec.SourceRef.FluxBucket != nil {
		return []string{sourceTypeFluxBucket, sourceTypeFluxSource}
	}
	if component.Spec.SourceRef.FluxHelmChart != nil {
		return []string{sourceTypeFluxHelmChart, sourceTypeFluxSource}
	}
	panic("this cannot happen")
}

func indexByDependencies(object client.Object) []string {
	component := object.(*operatorv1alpha1.Component)
	return slices.Collect(component.Spec.Dependencies, func(dependency operatorv1alpha1.Dependency) string {
		return dependency.WithDefaultNamespace(component.Namespace).String()
	})
}

func indexByBlueprint(object client.Object) []string {
	component := object.(*operatorv1alpha1.Component)
	if component.Spec.SourceRef.Blueprint == nil {
		return nil
	}
	return []string{component.Spec.SourceRef.Blueprint.WithDefaultNamespace(component.Namespace).String()}
}

func indexByBlueprintVersion(object client.Object) []string {
	component := object.(*operatorv1alpha1.Component)
	if component.Spec.SourceRef.Blueprint == nil {
		return nil
	}
	if component.Status.SourceRef == nil || component.Status.SourceRef.Artifact.Digest == "" {
		return nil
	}
	return []string{operatorv1alpha1.NamespacedName{
		Name:      fmt.Sprintf("%s--%s", component.Spec.SourceRef.Blueprint.Name, component.Status.SourceRef.Artifact.Digest),
		Namespace: component.Spec.SourceRef.Blueprint.Namespace,
	}.WithDefaultNamespace(component.Namespace).String()}
}

func indexCompoonentByGitRepository(object client.Object) []string {
	component := object.(*operatorv1alpha1.Component)
	if component.Spec.SourceRef.FluxGitRepository == nil {
		return nil
	}
	return []string{component.Spec.SourceRef.FluxGitRepository.WithDefaultNamespace(component.Namespace).String()}
}

func indexCompoonentByOciRepository(object client.Object) []string {
	component := object.(*operatorv1alpha1.Component)
	if component.Spec.SourceRef.FluxOciRepository == nil {
		return nil
	}
	return []string{component.Spec.SourceRef.FluxOciRepository.WithDefaultNamespace(component.Namespace).String()}
}

func indexCompoonentByBucket(object client.Object) []string {
	component := object.(*operatorv1alpha1.Component)
	if component.Spec.SourceRef.FluxBucket == nil {
		return nil
	}
	return []string{component.Spec.SourceRef.FluxBucket.WithDefaultNamespace(component.Namespace).String()}
}

func indexCompoonentByHelmChart(object client.Object) []string {
	component := object.(*operatorv1alpha1.Component)
	if component.Spec.SourceRef.FluxHelmChart == nil {
		return nil
	}
	return []string{component.Spec.SourceRef.FluxHelmChart.WithDefaultNamespace(component.Namespace).String()}
}
