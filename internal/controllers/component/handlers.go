/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"context"

	"github.com/go-logr/logr"

	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
	componentcache "github.com/sap/component-operator/internal/cache/component"
	"github.com/sap/component-operator/internal/object"
	"github.com/sap/component-operator/pkg/meta"
)

type componentHandler struct {
	cache cache.Cache
	log   logr.Logger
}

func newComponentHandler(cache cache.Cache, log logr.Logger) handler.TypedEventHandler[client.Object, reconcile.Request] {
	return &componentHandler{
		cache: cache,
		log:   log,
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
	if err := h.cache.List(ctx, componentList, componentcache.MatchingDependency(newComponent)); err != nil {
		h.log.Error(err, "failed to list components matching dependency")
		return
	}
	for _, c := range componentList.Items {
		// TODO: be more selective (i.e. queue only depending components that really need it)?
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

type blueprintHandler struct {
	cache cache.Cache
	log   logr.Logger
}

func newBlueprintHandler(cache cache.Cache, log logr.Logger) handler.TypedEventHandler[client.Object, reconcile.Request] {
	return &blueprintHandler{
		cache: cache,
		log:   log,
	}
}

func (h *blueprintHandler) Create(ctx context.Context, e event.TypedCreateEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	h.createOrUpdate(ctx, e.Object.(*operatorv1alpha1.Blueprint), q)
}

func (h *blueprintHandler) Update(ctx context.Context, e event.TypedUpdateEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	h.createOrUpdate(ctx, e.ObjectNew.(*operatorv1alpha1.Blueprint), q)
}

func (h *blueprintHandler) Delete(ctx context.Context, e event.TypedDeleteEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	// no need to queue components if blueprint is deleted (reconciliation of the component would anyway fail)
}

func (h *blueprintHandler) Generic(ctx context.Context, e event.TypedGenericEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	// generic events are not expected to arrive on the watch that uses this handler, so nothing to do here
}

func (h *blueprintHandler) createOrUpdate(ctx context.Context, blueprint *operatorv1alpha1.Blueprint, q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	componentList := &operatorv1alpha1.ComponentList{}
	if err := h.cache.List(ctx, componentList, componentcache.MatchingBlueprint(blueprint)); err != nil {
		h.log.Error(err, "failed to list components matching blueprint")
		return
	}
	for _, c := range componentList.Items {
		if c.IsReady() && c.Status.LastAttemptedDigest == blueprint.GetDigest() && c.Status.LastAttemptedRevision == blueprint.GetRevision() {
			continue
		}
		q.Add(reconcile.Request{NamespacedName: apitypes.NamespacedName{
			Namespace: c.Namespace,
			Name:      c.Name,
		}})
	}
}

type fluxSourceHandler struct {
	cache cache.Cache
	log   logr.Logger
}

func newFluxSourceHandler(cache cache.Cache, log logr.Logger) handler.TypedEventHandler[client.Object, reconcile.Request] {
	return &fluxSourceHandler{
		cache: cache,
		log:   log,
	}
}

func (h *fluxSourceHandler) Create(ctx context.Context, e event.TypedCreateEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	// new sources will never be immediately ready, so nothing has to be done here
}

func (h *fluxSourceHandler) Update(ctx context.Context, e event.TypedUpdateEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	newSource := e.ObjectNew.(meta.FluxSource)

	if !object.IsReady(newSource) {
		return
	}

	artifact := newSource.GetArtifact()
	if artifact == nil {
		return
	}
	newDigest := artifact.Digest
	if newDigest == "" {
		return
	}
	newRevision := artifact.Revision
	if newRevision == "" {
		return
	}

	componentList := &operatorv1alpha1.ComponentList{}
	if err := h.cache.List(ctx, componentList, componentcache.MatchingFluxSource(newSource)); err != nil {
		h.log.Error(err, "failed to list components matching flux source")
		return
	}
	for _, c := range componentList.Items {
		if c.IsReady() && c.Status.LastAttemptedDigest == newDigest && c.Status.LastAttemptedRevision == newRevision {
			continue
		}
		q.Add(reconcile.Request{NamespacedName: apitypes.NamespacedName{
			Namespace: c.Namespace,
			Name:      c.Name,
		}})
	}
}

func (h *fluxSourceHandler) Delete(ctx context.Context, e event.TypedDeleteEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	// no need to queue components if source is deleted (reconciliation of the component would anyway fail)
}

func (h *fluxSourceHandler) Generic(ctx context.Context, e event.TypedGenericEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	// generic events are not expected to arrive on the watch that uses this handler, so nothing to do here
}
