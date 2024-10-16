/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package flux

import (
	"context"

	"github.com/pkg/errors"

	apitypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	fluxsourcev1 "github.com/fluxcd/source-controller/api/v1"
	fluxsourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
	"github.com/sap/component-operator/internal/object"
)

type Source interface {
	object.Object
	fluxsourcev1.Source
}

const (
	gitRepositoryIndexKey string = ".metadata.flux.gitRepository"
	ociRepositoryIndexKey string = ".metadata.flux.ociRepository"
	bucketIndexKey        string = ".metadata.flux.bucket"
	helmChartIndexKey     string = ".metadata.flux.helmChart"
)

func SetupCache(mgr manager.Manager, blder *builder.Builder) error {
	// TODO: should we pass a meaningful context?
	if err := mgr.GetCache().IndexField(context.TODO(), &operatorv1alpha1.Component{}, gitRepositoryIndexKey, indexByGitRepository); err != nil {
		return errors.Wrapf(err, "failed setting index field %s", gitRepositoryIndexKey)
	}
	if err := mgr.GetCache().IndexField(context.TODO(), &operatorv1alpha1.Component{}, ociRepositoryIndexKey, indexByOciRepository); err != nil {
		return errors.Wrapf(err, "failed setting index field %s", ociRepositoryIndexKey)
	}
	if err := mgr.GetCache().IndexField(context.TODO(), &operatorv1alpha1.Component{}, bucketIndexKey, indexByBucket); err != nil {
		return errors.Wrapf(err, "failed setting index field %s", bucketIndexKey)
	}
	if err := mgr.GetCache().IndexField(context.TODO(), &operatorv1alpha1.Component{}, helmChartIndexKey, indexByHelmChart); err != nil {
		return errors.Wrapf(err, "failed setting index field %s", helmChartIndexKey)
	}

	blder.
		Watches(
			&fluxsourcev1beta2.GitRepository{},
			newSourceHandler(mgr.GetCache(), gitRepositoryIndexKey)).
		Watches(
			&fluxsourcev1beta2.OCIRepository{},
			newSourceHandler(mgr.GetCache(), ociRepositoryIndexKey)).
		Watches(
			&fluxsourcev1beta2.Bucket{},
			newSourceHandler(mgr.GetCache(), bucketIndexKey)).
		Watches(
			&fluxsourcev1beta2.HelmChart{},
			newSourceHandler(mgr.GetCache(), helmChartIndexKey))

	return nil
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

type sourceHandler struct {
	cache    cache.Cache
	indexKey string
}

func newSourceHandler(cache cache.Cache, indexKey string) handler.TypedEventHandler[client.Object, reconcile.Request] {
	return &sourceHandler{
		cache:    cache,
		indexKey: indexKey,
	}
}

func (h *sourceHandler) Create(ctx context.Context, e event.TypedCreateEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	// new sources will never be immediately ready, so nothing has to be done here
}

func (h *sourceHandler) Update(ctx context.Context, e event.TypedUpdateEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	newSource := e.ObjectNew.(Source)

	if !object.IsReady(newSource) {
		return
	}

	artifact := newSource.GetArtifact()
	if artifact == nil {
		return
	}
	newRevision := artifact.Revision
	if newRevision == "" {
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
		if c.IsReady() && c.Status.LastAttemptedRevision == newRevision {
			continue
		}
		q.Add(reconcile.Request{NamespacedName: apitypes.NamespacedName{
			Namespace: c.Namespace,
			Name:      c.Name,
		}})
	}
}

func (h *sourceHandler) Delete(ctx context.Context, e event.TypedDeleteEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	// no need to queue components if source is deleted (reconciliation of the component would anyway fail)
}

func (h *sourceHandler) Generic(ctx context.Context, e event.TypedGenericEvent[client.Object], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
	// generic events are not expected to arrive on the watch that uses this handler, so nothing to do here
}
