/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package operator

import (
	"github.com/sap/go-generics/slices"

	"sigs.k8s.io/controller-runtime/pkg/client"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
)

func indexByDependencies(object client.Object) []string {
	component := object.(*operatorv1alpha1.Component)
	return slices.Collect(component.Spec.Dependencies, func(dependency operatorv1alpha1.Dependency) string {
		return dependency.WithDefaultNamespace(component.Namespace).String()
	})
}

func indexByGitRepository(object client.Object) []string {
	component := object.(*operatorv1alpha1.Component)
	if component.Spec.SourceRef.FluxGitRepository == nil {
		return nil
	}
	return []string{component.Spec.SourceRef.FluxGitRepository.WithDefaultNamespace(component.Namespace).String()}
}

func indexByOciRepository(object client.Object) []string {
	component := object.(*operatorv1alpha1.Component)
	if component.Spec.SourceRef.FluxOciRepository == nil {
		return nil
	}
	return []string{component.Spec.SourceRef.FluxOciRepository.WithDefaultNamespace(component.Namespace).String()}
}

func indexByBucket(object client.Object) []string {
	component := object.(*operatorv1alpha1.Component)
	if component.Spec.SourceRef.FluxBucket == nil {
		return nil
	}
	return []string{component.Spec.SourceRef.FluxBucket.WithDefaultNamespace(component.Namespace).String()}
}

func indexByHelmChart(object client.Object) []string {
	component := object.(*operatorv1alpha1.Component)
	if component.Spec.SourceRef.FluxHelmChart == nil {
		return nil
	}
	return []string{component.Spec.SourceRef.FluxHelmChart.WithDefaultNamespace(component.Namespace).String()}
}
