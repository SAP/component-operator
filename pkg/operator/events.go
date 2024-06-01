/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package operator

import (
	"context"

	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
	"github.com/sap/component-operator/internal/flux"
)

type componentHandler struct {
	cache    cache.Cache
	indexKey string
}

func (h *componentHandler) Create(ctx context.Context, e event.CreateEvent, q workqueue.RateLimitingInterface) {
	// new components will never be immediately ready, so nothing has to be done here
}

func (h *componentHandler) Update(ctx context.Context, e event.UpdateEvent, q workqueue.RateLimitingInterface) {
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

func (h *componentHandler) Delete(ctx context.Context, e event.DeleteEvent, q workqueue.RateLimitingInterface) {
	component := e.Object.(*operatorv1alpha1.Component)
	for _, dependency := range component.Spec.Dependencies {
		q.Add(reconcile.Request{NamespacedName: apitypes.NamespacedName{
			Namespace: dependency.WithDefaultNamespace(component.Namespace).Namespace,
			Name:      dependency.Name,
		}})
	}
}

func (h *componentHandler) Generic(context.Context, event.GenericEvent, workqueue.RateLimitingInterface) {
	// generic events are not expected to arrive on the watch that uses this handler, so nothing to do here
}

type fluxSourceHandler struct {
	cache    cache.Cache
	indexKey string
}

func (h *fluxSourceHandler) Create(ctx context.Context, e event.CreateEvent, q workqueue.RateLimitingInterface) {
	// new sources will never be immediately ready, so nothing has to be done here
}

func (h *fluxSourceHandler) Update(ctx context.Context, e event.UpdateEvent, q workqueue.RateLimitingInterface) {
	newSource := e.ObjectNew.(flux.Source)

	if !flux.IsSourceReady(newSource) {
		return
	}

	if newSource.GetArtifact() == nil {
		return
	}

	componentList := &operatorv1alpha1.ComponentList{}
	if err := h.cache.List(ctx, componentList, client.MatchingFields{
		h.indexKey: client.ObjectKeyFromObject(e.ObjectNew).String(),
	}); err != nil {
		// TODO
		// log.Error(err, "failed to list objects for source revision change")
		return
	}
	for _, c := range componentList.Items {
		if c.IsReady() && c.Status.LastAttemptedRevision == newSource.GetArtifact().Revision {
			continue
		}
		q.Add(reconcile.Request{NamespacedName: apitypes.NamespacedName{
			Namespace: c.Namespace,
			Name:      c.Name,
		}})
	}
}

func (h *fluxSourceHandler) Delete(ctx context.Context, e event.DeleteEvent, q workqueue.RateLimitingInterface) {
	// no need to queue components if source is deleted (reconciliation of the component would anyway fail)
}

func (h *fluxSourceHandler) Generic(context.Context, event.GenericEvent, workqueue.RateLimitingInterface) {
	// generic events are not expected to arrive on the watch that uses this handler, so nothing to do here
}
