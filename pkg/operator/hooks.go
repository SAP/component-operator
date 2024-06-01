/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package operator

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apitypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/sap/component-operator-runtime/pkg/component"
	componentoperatorruntimetypes "github.com/sap/component-operator-runtime/pkg/types"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
)

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

func makeFuncPreDelete(cache cache.Cache, indexKey string) component.HookFunc[*operatorv1alpha1.Component] {
	return func(ctx context.Context, clnt client.Client, component *operatorv1alpha1.Component) error {
		componentList := &operatorv1alpha1.ComponentList{}
		if err := cache.List(ctx, componentList, client.MatchingFields{
			indexKey: client.ObjectKeyFromObject(component).String(),
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
