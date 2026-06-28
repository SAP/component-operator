/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package blueprint

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
)

type blueprintVersionHandler struct {
	cache cache.Cache
	log   logr.Logger
}

func newBlueprintVersionHandler(cache cache.Cache, log logr.Logger) handler.TypedEventHandler[client.Object, reconcile.Request] {
	return &blueprintVersionHandler{
		cache: cache,
		log:   log,
	}
}

func (h *blueprintVersionHandler) Create(ctx context.Context, e event.TypedCreateEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	blueprintVersion := e.Object.(*operatorv1alpha1.BlueprintVersion)
	q.Add(reconcile.Request{NamespacedName: apitypes.NamespacedName{
		Namespace: blueprintVersion.Namespace,
		Name:      blueprintVersion.Spec.Blueprint,
	}})
}

func (h *blueprintVersionHandler) Update(ctx context.Context, e event.TypedUpdateEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	// no need to queue components if blueprint version is updated
}

func (h *blueprintVersionHandler) Delete(ctx context.Context, e event.TypedDeleteEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	// no need to queue components if blueprint version is deleted
}

func (h *blueprintVersionHandler) Generic(ctx context.Context, e event.TypedGenericEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	// generic events are not expected to arrive on the watch that uses this handler, so nothing to do here
}
