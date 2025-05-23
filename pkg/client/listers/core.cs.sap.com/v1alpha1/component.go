/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by lister-gen. DO NOT EDIT.

package v1alpha1

import (
	corecssapcomv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
	labels "k8s.io/apimachinery/pkg/labels"
	listers "k8s.io/client-go/listers"
	cache "k8s.io/client-go/tools/cache"
)

// ComponentLister helps list Components.
// All objects returned here must be treated as read-only.
type ComponentLister interface {
	// List lists all Components in the indexer.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*corecssapcomv1alpha1.Component, err error)
	// Components returns an object that can list and get Components.
	Components(namespace string) ComponentNamespaceLister
	ComponentListerExpansion
}

// componentLister implements the ComponentLister interface.
type componentLister struct {
	listers.ResourceIndexer[*corecssapcomv1alpha1.Component]
}

// NewComponentLister returns a new ComponentLister.
func NewComponentLister(indexer cache.Indexer) ComponentLister {
	return &componentLister{listers.New[*corecssapcomv1alpha1.Component](indexer, corecssapcomv1alpha1.Resource("component"))}
}

// Components returns an object that can list and get Components.
func (s *componentLister) Components(namespace string) ComponentNamespaceLister {
	return componentNamespaceLister{listers.NewNamespaced[*corecssapcomv1alpha1.Component](s.ResourceIndexer, namespace)}
}

// ComponentNamespaceLister helps list and get Components.
// All objects returned here must be treated as read-only.
type ComponentNamespaceLister interface {
	// List lists all Components in the indexer for a given namespace.
	// Objects returned here must be treated as read-only.
	List(selector labels.Selector) (ret []*corecssapcomv1alpha1.Component, err error)
	// Get retrieves the Component from the indexer for a given namespace and name.
	// Objects returned here must be treated as read-only.
	Get(name string) (*corecssapcomv1alpha1.Component, error)
	ComponentNamespaceListerExpansion
}

// componentNamespaceLister implements the ComponentNamespaceLister
// interface.
type componentNamespaceLister struct {
	listers.ResourceIndexer[*corecssapcomv1alpha1.Component]
}
