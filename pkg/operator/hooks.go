/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package operator

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	fluxsourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/sap/component-operator-runtime/pkg/component"
	componentoperatorruntimetypes "github.com/sap/component-operator-runtime/pkg/types"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
	"github.com/sap/component-operator/internal/object"
	"github.com/sap/component-operator/internal/sources/flux"
	"github.com/sap/component-operator/internal/sources/httprepository"
)

func makeFuncPostRead() component.HookFunc[*operatorv1alpha1.Component] {
	return func(ctx context.Context, clnt client.Client, component *operatorv1alpha1.Component) error {
		sourceRef := &component.Spec.SourceRef
		sourceRefUrl := ""
		sourceRefRevision := ""

		switch {
		case sourceRef.HttpRepository != nil:
			url, revision, err := httprepository.GetUrlAndRevision(component.Spec.SourceRef.HttpRepository.Url, component.Spec.SourceRef.HttpRepository.RevisionHeader)
			if err != nil {
				return err
			}

			sourceRefUrl = url
			sourceRefRevision = revision
		case sourceRef.FluxGitRepository != nil, sourceRef.FluxOciRepository != nil, sourceRef.FluxBucket != nil, sourceRef.FluxHelmChart != nil:
			var sourceName operatorv1alpha1.NamespacedName
			var source flux.Source

			switch {
			case sourceRef.FluxGitRepository != nil:
				sourceName = sourceRef.FluxGitRepository.WithDefaultNamespace(component.Namespace)
				source = &fluxsourcev1beta2.GitRepository{}
			case sourceRef.FluxOciRepository != nil:
				sourceName = sourceRef.FluxOciRepository.WithDefaultNamespace(component.Namespace)
				source = &fluxsourcev1beta2.OCIRepository{}
			case sourceRef.FluxBucket != nil:
				sourceName = sourceRef.FluxBucket.WithDefaultNamespace(component.Namespace)
				source = &fluxsourcev1beta2.Bucket{}
			case sourceRef.FluxHelmChart != nil:
				sourceName = sourceRef.FluxHelmChart.WithDefaultNamespace(component.Namespace)
				source = &fluxsourcev1beta2.HelmChart{}
			default:
				panic("this cannot happen")
			}

			if err := clnt.Get(ctx, apitypes.NamespacedName(sourceName), source); err != nil {
				if apimeta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
					return componentoperatorruntimetypes.NewRetriableError(err, ref(10*time.Second))
				}
				return err
			}
			if !object.IsReady(source) {
				return componentoperatorruntimetypes.NewRetriableError(fmt.Errorf("source not ready"), ref(10*time.Second))
			}

			artifact := source.GetArtifact()
			sourceRefUrl = artifact.URL
			sourceRefRevision = artifact.Revision
		default:
			return fmt.Errorf("unable to get source; one of httpRepository, fluxGitRepository, fluxOciRepository, fluxBucket, fluxHelmChart must be defined")
		}

		sourceRef.Init(sourceRefUrl, sourceRefRevision)

		if component.Spec.Revision != "" && sourceRefRevision != component.Spec.Revision {
			return componentoperatorruntimetypes.NewRetriableError(fmt.Errorf("source revision (%s) does not match specified revision (%s)", sourceRefRevision, component.Spec.Revision), ref(10*time.Second))
		}
		return nil
	}
}

func makeFuncPreReconcile(cache cache.Cache) component.HookFunc[*operatorv1alpha1.Component] {
	return func(ctx context.Context, clnt client.Client, component *operatorv1alpha1.Component) error {
		// note: it is crucial to set status.lastAttemptedRevision here (in pre-reconcile), since generators
		// might fetch the component from their context, relying on the field being already updated
		component.Status.LastAttemptedRevision = component.Spec.SourceRef.Revision()
		for _, dependency := range component.Spec.Dependencies {
			c := &operatorv1alpha1.Component{}
			if err := cache.Get(ctx, apitypes.NamespacedName(dependency.WithDefaultNamespace(component.Namespace)), c); err != nil {
				if apierrors.IsNotFound(err) {
					return componentoperatorruntimetypes.NewRetriableError(errors.Wrapf(err, "dependent component %s not found", dependency), nil)
				}
				return err
			}
			if c.Spec.SourceRef.Equals(&component.Spec.SourceRef) && (c.Status.LastAttemptedRevision == "" || c.Status.LastAttemptedRevision != component.Status.LastAttemptedRevision) {
				return componentoperatorruntimetypes.NewRetriableError(fmt.Errorf("dependent component %s not synced", dependency), nil)
			}
			if !c.IsReady() {
				return componentoperatorruntimetypes.NewRetriableError(fmt.Errorf("dependent component %s not ready", dependency), nil)
			}
		}
		return nil
	}
}

func makeFuncPostReconcile() component.HookFunc[*operatorv1alpha1.Component] {
	return func(ctx context.Context, clnt client.Client, component *operatorv1alpha1.Component) error {
		component.Status.LastAppliedRevision = component.Status.LastAttemptedRevision
		return nil
	}
}

func makeFuncPreDelete(cache cache.Cache) component.HookFunc[*operatorv1alpha1.Component] {
	return func(ctx context.Context, clnt client.Client, component *operatorv1alpha1.Component) error {
		componentList := &operatorv1alpha1.ComponentList{}
		if err := cache.List(ctx, componentList, client.MatchingFields{
			dependenciesIndexKey: client.ObjectKeyFromObject(component).String(),
		}); err != nil {
			return err
		}
		if len(componentList.Items) == 0 {
			return nil
		} else if len(componentList.Items) == 1 {
			return componentoperatorruntimetypes.NewRetriableError(fmt.Errorf("deletion blocked by depending component %s", componentList.Items[0].NamespacedName()), nil)
		} else {
			return componentoperatorruntimetypes.NewRetriableError(fmt.Errorf("deletion blocked by depending component %s (and %d others)", componentList.Items[0].NamespacedName(), len(componentList.Items)-1), nil)
		}
	}
}
