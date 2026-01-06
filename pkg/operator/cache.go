/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package operator

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sap/go-generics/slices"

	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
)

const (
	dependenciesIndexKey string = ".metadata.dependencies"
)

func setupCache(mgr manager.Manager, blder *builder.Builder) error {
	// TODO: should we pass a meaningful context?
	if err := mgr.GetCache().IndexField(context.TODO(), &operatorv1alpha1.Component{}, dependenciesIndexKey, indexByDependencies); err != nil {
		return errors.Wrapf(err, "failed setting index field %s", dependenciesIndexKey)
	}

	blder.
		Watches(
			&operatorv1alpha1.Component{},
			newComponentHandler(mgr.GetCache(), dependenciesIndexKey))

	return nil
}

func indexByDependencies(object client.Object) []string {
	component := object.(*operatorv1alpha1.Component)
	return slices.Collect(component.Spec.Dependencies, func(dependency operatorv1alpha1.Dependency) string {
		return dependency.WithDefaultNamespace(component.Namespace).String()
	})
}

type componentHandler struct {
	cache    cache.Cache
	indexKey string
}

func newComponentHandler(cache cache.Cache, indexKey string) handler.TypedEventHandler[client.Object, reconcile.Request] {
	return &componentHandler{
		cache:    cache,
		indexKey: indexKey,
	}
}

func (h *componentHandler) Create(ctx context.Context, e event.TypedCreateEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	// new components will never be immediately ready, so nothing has to be done here
}

func (h *componentHandler) Update(ctx context.Context, e event.TypedUpdateEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	newComponent := e.ObjectNew.(*operatorv1alpha1.Component)

	if !newComponent.IsReady() {
		return
	}

	componentList := &operatorv1alpha1.ComponentList{}
	if err := h.cache.List(ctx, componentList, client.MatchingFields{
		h.indexKey: client.ObjectKeyFromObject(e.ObjectNew).String(),
	}); err != nil {
		// TODO
		// log.Error(err, "failed to list objects for component state change")
		return
	}
	for _, c := range componentList.Items {
		// TODO: be more selective
		q.Add(reconcile.Request{NamespacedName: apitypes.NamespacedName{
			Namespace: c.Namespace,
			Name:      c.Name,
		}})
	}
}

func (h *componentHandler) Delete(ctx context.Context, e event.TypedDeleteEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	component := e.Object.(*operatorv1alpha1.Component)
	for _, dependency := range component.Spec.Dependencies {
		q.Add(reconcile.Request{NamespacedName: apitypes.NamespacedName{
			Namespace: dependency.WithDefaultNamespace(component.Namespace).Namespace,
			Name:      dependency.Name,
		}})
	}
}

func (h *componentHandler) Generic(ctx context.Context, e event.TypedGenericEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	// generic events are not expected to arrive on the watch that uses this handler, so nothing to do here
}
