/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package operator

import (
	"flag"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	fluxsourcev1 "github.com/fluxcd/source-controller/api/v1"
	fluxsourcev1beta1 "github.com/fluxcd/source-controller/api/v1beta1"
	fluxsourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/sap/component-operator-runtime/pkg/operator"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
	componentcache "github.com/sap/component-operator/internal/cache/component"
	blueprintcontroller "github.com/sap/component-operator/internal/controllers/blueprint"
	componentcontroller "github.com/sap/component-operator/internal/controllers/component"
	"github.com/sap/component-operator/internal/httprepository"
	"github.com/sap/component-operator/pkg/meta"
)

// TODO: write some logs (e.g. in the hooks)
// TODO: use source digest instead of (resp. in parallel to) source revision

const (
	MaxConcurrentReconciles = 5
)

type Options struct {
	Name                    string
	DefaultServiceAccount   string
	MaxConcurrentReconciles int
	EventsAddress           string
	FlagPrefix              string
}

type Operator struct {
	options Options
}

var defaultOperator operator.Operator = New()

func GetName() string {
	return defaultOperator.GetName()
}

func InitScheme(scheme *runtime.Scheme) {
	defaultOperator.InitScheme(scheme)
}

func InitFlags(flagset *flag.FlagSet) {
	defaultOperator.InitFlags(flagset)
}

func ValidateFlags() error {
	return defaultOperator.ValidateFlags()
}

func GetUncacheableTypes() []client.Object {
	return defaultOperator.GetUncacheableTypes()
}

func Setup(mgr ctrl.Manager) error {
	return defaultOperator.Setup(mgr)
}

func New() *Operator {
	return NewWithOptions(Options{})
}

func NewWithOptions(options Options) *Operator {
	operator := &Operator{options: options}
	if operator.options.Name == "" {
		operator.options.Name = meta.Name
	}
	if operator.options.MaxConcurrentReconciles == 0 {
		operator.options.MaxConcurrentReconciles = MaxConcurrentReconciles
	}
	return operator
}

func (o *Operator) GetName() string {
	return o.options.Name
}

func (o *Operator) InitScheme(scheme *runtime.Scheme) {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	utilruntime.Must(fluxsourcev1beta1.AddToScheme(scheme))
	utilruntime.Must(fluxsourcev1beta2.AddToScheme(scheme))
	utilruntime.Must(fluxsourcev1.AddToScheme(scheme))
}

func (o *Operator) InitFlags(flagset *flag.FlagSet) {
	flagset.StringVar(&o.options.DefaultServiceAccount, "default-service-account", o.options.DefaultServiceAccount, "Default service account name")
	flagset.IntVar(&o.options.MaxConcurrentReconciles, "max-concurrent-reconciles", o.options.MaxConcurrentReconciles, "Maximum number of concurrent reconciler workers")
	flagset.StringVar(&o.options.EventsAddress, "events-address", o.options.EventsAddress, "Address of the events receiver")
}

func (o *Operator) ValidateFlags() error {
	return nil
}

func (o *Operator) GetUncacheableTypes() []client.Object {
	return []client.Object{&operatorv1alpha1.Component{}, &operatorv1alpha1.Blueprint{}, &operatorv1alpha1.BlueprintVersion{}}
}

func (o *Operator) Setup(mgr ctrl.Manager) error {
	if err := componentcache.SetupWithManager(mgr); err != nil {
		return errors.Wrap(err, "error configuring component cache")
	}

	componentReconciler, err := componentcontroller.SetupWithManager(mgr, componentcontroller.ReconcilerOptions{
		Name:                    o.options.Name,
		DefaultServiceAccount:   o.options.DefaultServiceAccount,
		MaxConcurrentReconciles: o.options.MaxConcurrentReconciles,
		EventsAddress:           o.options.EventsAddress,
	})
	if err != nil {
		return errors.Wrapf(err, "error registering component controller")
	}

	if err := blueprintcontroller.SetupWithManager(mgr, blueprintcontroller.ReconcilerOptions{
		Name: o.options.Name,
	}); err != nil {
		return errors.Wrapf(err, "error registering blueprint controller")
	}

	if err := httprepository.SetupWithManager(mgr, componentReconciler); err != nil {
		return errors.Wrapf(err, "error registering http repository checker")
	}

	return nil
}
