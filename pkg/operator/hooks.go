/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package operator

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/pkg/component"
	componentoperatorruntimetypes "github.com/sap/component-operator-runtime/pkg/types"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
)

func makeFuncPostRead() component.HookFunc[*operatorv1alpha1.Component] {
	return func(ctx context.Context, clnt client.Client, component *operatorv1alpha1.Component) error {
		if !component.DeletionTimestamp.IsZero() {
			return nil
		}
		if component.Spec.Digest != "" && component.Spec.SourceRef.Artifact().Digest != component.Spec.Digest {
			return componentoperatorruntimetypes.NewRetriableError(fmt.Errorf("source digest (%s) does not match specified digest (%s)", component.Spec.SourceRef.Artifact().Digest, component.Spec.Digest), ref(10*time.Second))
		}
		if component.Spec.Revision != "" && component.Spec.SourceRef.Artifact().Revision != component.Spec.Revision {
			return componentoperatorruntimetypes.NewRetriableError(fmt.Errorf("source revision (%s) does not match specified revision (%s)", component.Spec.SourceRef.Artifact().Revision, component.Spec.Revision), ref(10*time.Second))
		}
		return nil
	}
}

func makeFuncPreReconcile(cache cache.Cache) component.HookFunc[*operatorv1alpha1.Component] {
	return func(ctx context.Context, clnt client.Client, component *operatorv1alpha1.Component) error {
		// note: it is crucial to set status.lastAttemptedDigest and status.lastAttemptedRevision here (in pre-reconcile), since generators
		// might fetch the component from their context, relying on the fields being already updated
		component.Status.LastAttemptedDigest = component.Spec.SourceRef.Artifact().Digest
		component.Status.LastAttemptedRevision = component.Spec.SourceRef.Artifact().Revision
		for _, dependency := range component.Spec.Dependencies {
			c := &operatorv1alpha1.Component{}
			if err := cache.Get(ctx, apitypes.NamespacedName(dependency.WithDefaultNamespace(component.Namespace)), c); err != nil {
				if apierrors.IsNotFound(err) {
					return componentoperatorruntimetypes.NewRetriableError(errors.Wrapf(err, "dependent component %s not found", dependency), nil)
				}
				return err
			}
			if c.Spec.SourceRef.Equals(&component.Spec.SourceRef) && (c.Status.LastAttemptedDigest == "" || c.Status.LastAttemptedDigest != component.Status.LastAttemptedDigest || c.Status.LastAttemptedRevision == "" || c.Status.LastAttemptedRevision != component.Status.LastAttemptedRevision) {
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
		component.Status.LastAppliedDigest = component.Status.LastAttemptedDigest
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
