/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
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
)

type Checker struct {
	cache      cache.Cache
	reconciler *component.Reconciler[*operatorv1alpha1.Component]
	logger     logr.Logger
}

var _ manager.Runnable = &Checker{}
var _ manager.LeaderElectionRunnable = &Checker{}

func NewChecker(cache cache.Cache, reconciler *component.Reconciler[*operatorv1alpha1.Component], logger logr.Logger) *Checker {
	return &Checker{
		cache:      cache,
		reconciler: reconciler,
		logger:     logger,
	}
}

func (c *Checker) Start(ctx context.Context) error {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}

		componentList := &operatorv1alpha1.ComponentList{}
		if err := c.cache.List(context.TODO(), componentList); err != nil {
			c.logger.Error(err, "error listing components")
			continue
		}
		for _, component := range componentList.Items {
			if component.Spec.SourceRef.HttpRepository != nil {
				_, revision, err := GetUrlAndRevision(component.Spec.SourceRef.HttpRepository.Url, component.Spec.SourceRef.HttpRepository.RevisionHeader)
				if err == nil {
					if revision != component.Status.LastAttemptedRevision {
						c.reconciler.Trigger(component.Namespace, component.Name)
					}
				} else {
					c.logger.Error(err, "error fetching revision from http repository", "url", component.Spec.SourceRef.HttpRepository.Url, "revisionHeader", component.Spec.SourceRef.HttpRepository.RevisionHeader)
				}
			}
		}
	}
}

func (c *Checker) NeedLeaderElection() bool {
	return true
}
