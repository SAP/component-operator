/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package blueprint

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
	componentcache "github.com/sap/component-operator/internal/cache/component"
	"github.com/sap/component-operator/pkg/meta"
)

const (
	reasonDeletionBlocked = "DeletionBlocked"
)

type ReconcilerOptions struct {
	Name string
}

type reconciler struct {
	client        client.Client
	cache         cache.Cache
	eventRecorder record.EventRecorder
}

func newReconciler(clnt client.Client, cache cache.Cache, eventRecorder record.EventRecorder) *reconciler {
	return &reconciler{
		client:        clnt,
		cache:         cache,
		eventRecorder: eventRecorder,
	}
}

func (r *reconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	blueprint := &operatorv1alpha1.Blueprint{}
	if err := r.client.Get(ctx, req.NamespacedName, blueprint); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if blueprint.DeletionTimestamp.IsZero() {
		if controllerutil.AddFinalizer(blueprint, meta.Name) {
			if err := r.client.Update(ctx, blueprint); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	blueprintVersionList := &operatorv1alpha1.BlueprintVersionList{}
	if err := r.client.List(ctx, blueprintVersionList, client.InNamespace(blueprint.Namespace), client.MatchingFields{
		"spec.blueprint": blueprint.Name,
	}); err != nil {
		return ctrl.Result{}, err
	}

	numDeleted := 0
	for _, blueprintVersion := range blueprintVersionList.Items {
		comoponentList := &operatorv1alpha1.ComponentList{}
		if err := r.cache.List(ctx, comoponentList, componentcache.MatchingBlueprintVersion(&blueprintVersion), client.Limit(1)); err != nil {
			return ctrl.Result{}, err
		}
		if len(comoponentList.Items) == 0 {
			// TODO: rule out caching race conditions
			if blueprintVersion.DeletionTimestamp.IsZero() {
				if err := r.client.Delete(ctx, &blueprintVersion); err != nil {
					return ctrl.Result{}, err
				}
				numDeleted++
			} else {
				return ctrl.Result{}, fmt.Errorf("blueprintversion %s/%s is expected to be deleted but is not yet deleted", blueprintVersion.Namespace, blueprintVersion.Name)
			}
		}
	}

	if numDeleted > 0 {
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}

	if blueprint.DeletionTimestamp.IsZero() {
		return ctrl.Result{RequeueAfter: 10 * time.Minute}, nil
	} else {
		if len(blueprintVersionList.Items) == 0 {
			if controllerutil.RemoveFinalizer(blueprint, meta.Name) {
				if err := r.client.Update(ctx, blueprint); err != nil {
					return ctrl.Result{}, err
				}
			}
			return ctrl.Result{}, nil
		} else {
			r.eventRecorder.Eventf(blueprint, corev1.EventTypeNormal, reasonDeletionBlocked, "Blueprint cannot be deleted because there are still %d BlueprintVersions referencing it", len(blueprintVersionList.Items))
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
	}
}

func SetupWithManager(mgr ctrl.Manager, options ReconcilerOptions) error {
	reconciler := newReconciler(mgr.GetClient(), mgr.GetCache(), mgr.GetEventRecorderFor(options.Name))

	return ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1alpha1.Blueprint{}, builder.WithPredicates(predicate.Or(predicate.GenerationChangedPredicate{}, predicate.AnnotationChangedPredicate{}))).
		Watches(&operatorv1alpha1.BlueprintVersion{}, newBlueprintVersionHandler(mgr.GetCache(), mgr.GetLogger())).
		Complete(reconciler)
}
