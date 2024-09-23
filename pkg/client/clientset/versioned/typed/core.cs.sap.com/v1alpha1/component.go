/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

// Code generated by client-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"

	v1alpha1 "github.com/sap/component-operator/api/v1alpha1"
	scheme "github.com/sap/component-operator/pkg/client/clientset/versioned/scheme"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	gentype "k8s.io/client-go/gentype"
)

// ComponentsGetter has a method to return a ComponentInterface.
// A group's client should implement this interface.
type ComponentsGetter interface {
	Components(namespace string) ComponentInterface
}

// ComponentInterface has methods to work with Component resources.
type ComponentInterface interface {
	Create(ctx context.Context, component *v1alpha1.Component, opts v1.CreateOptions) (*v1alpha1.Component, error)
	Update(ctx context.Context, component *v1alpha1.Component, opts v1.UpdateOptions) (*v1alpha1.Component, error)
	// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
	UpdateStatus(ctx context.Context, component *v1alpha1.Component, opts v1.UpdateOptions) (*v1alpha1.Component, error)
	Delete(ctx context.Context, name string, opts v1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error
	Get(ctx context.Context, name string, opts v1.GetOptions) (*v1alpha1.Component, error)
	List(ctx context.Context, opts v1.ListOptions) (*v1alpha1.ComponentList, error)
	Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.Component, err error)
	ComponentExpansion
}

// components implements ComponentInterface
type components struct {
	*gentype.ClientWithList[*v1alpha1.Component, *v1alpha1.ComponentList]
}

// newComponents returns a Components
func newComponents(c *CoreV1alpha1Client, namespace string) *components {
	return &components{
		gentype.NewClientWithList[*v1alpha1.Component, *v1alpha1.ComponentList](
			"components",
			c.RESTClient(),
			scheme.ParameterCodec,
			namespace,
			func() *v1alpha1.Component { return &v1alpha1.Component{} },
			func() *v1alpha1.ComponentList { return &v1alpha1.ComponentList{} }),
	}
}
