/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package httprepository

import (
	"context"
	"time"

	"github.com/go-logr/logr"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/sap/component-operator-runtime/pkg/component"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
	componentcache "github.com/sap/component-operator/internal/cache/component"
	"github.com/sap/component-operator/internal/httprepository/util"
)

type checker struct {
	cache               cache.Cache
	componentReconciler *component.Reconciler[*operatorv1alpha1.Component]
	logger              logr.Logger
}

var _ manager.Runnable = &checker{}
var _ manager.LeaderElectionRunnable = &checker{}

func newChecker(cache cache.Cache, componentReconciler *component.Reconciler[*operatorv1alpha1.Component], logger logr.Logger) *checker {
	return &checker{
		cache:               cache,
		componentReconciler: componentReconciler,
		logger:              logger,
	}
}

func (c *checker) Start(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}

		componentList := &operatorv1alpha1.ComponentList{}
		if err := c.cache.List(context.TODO(), componentList, componentcache.HasHttpRepository()); err != nil {
			c.logger.Error(err, "error listing components")
			continue
		}
		for _, component := range componentList.Items {
			url := component.Spec.SourceRef.HttpRepository.Url
			digestHeader := component.Spec.SourceRef.HttpRepository.DigestHeader
			revisionHeader := component.Spec.SourceRef.HttpRepository.RevisionHeader
			_, digest, revision, err := util.GetArtifact(url, digestHeader, revisionHeader)
			if err == nil {
				if digest != component.Status.LastAttemptedDigest || revision != component.Status.LastAttemptedRevision {
					c.componentReconciler.Trigger(component.Namespace, component.Name)
				}
			} else {
				c.logger.Error(err, "error fetching revision from http repository", "url", url, "digestHeader", digestHeader, "revisionHeader", revisionHeader)
			}
		}
	}
}

func (c *checker) NeedLeaderElection() bool {
	return true
}

func SetupWithManager(mgr manager.Manager, componentReconciler *component.Reconciler[*operatorv1alpha1.Component]) error {
	mgr.Add(newChecker(mgr.GetCache(), componentReconciler, mgr.GetLogger()))
	return nil
}
