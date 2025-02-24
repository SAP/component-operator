/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package operator

import (
	"flag"

	"github.com/pkg/errors"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	fluxevents "github.com/fluxcd/pkg/runtime/events"
	fluxsourcev1 "github.com/fluxcd/source-controller/api/v1"
	fluxsourcev1beta1 "github.com/fluxcd/source-controller/api/v1beta1"
	fluxsourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/sap/component-operator-runtime/pkg/cluster"
	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/operator"
	"github.com/sap/component-operator-runtime/pkg/reconciler"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
	"github.com/sap/component-operator/internal/generator"
	"github.com/sap/component-operator/internal/sources/flux"
	"github.com/sap/component-operator/internal/sources/httprepository"
)

// TODO: write some logs (e.g. in the hooks)
// TODO: use source digest instead of (resp. in parallel to) source revision

const (
	Name                    = "component-operator.cs.sap.com"
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
		operator.options.Name = Name
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
	return []client.Object{&operatorv1alpha1.Component{}}
}

func (o *Operator) Setup(mgr ctrl.Manager) error {
	blder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: o.options.MaxConcurrentReconciles})

	if err := setupCache(mgr, blder); err != nil {
		return errors.Wrap(err, "error registering component resource")
	}
	if err := flux.SetupCache(mgr, blder); err != nil {
		return errors.Wrap(err, "error registering flux resources")
	}

	resourceGenerator, err := generator.NewGenerator()
	if err != nil {
		return errors.Wrap(err, "error initializing resource generator")
	}

	newClient := func(clnt cluster.Client) (cluster.Client, error) {
		if o.options.EventsAddress == "" {
			return clnt, nil
		}
		eventRecorder, err := fluxevents.NewRecorderForScheme(clnt.Scheme(), clnt.EventRecorder(), mgr.GetLogger(), o.options.EventsAddress, o.options.Name)
		if err != nil {
			return nil, errors.Wrap(err, "error initializing wrapping event recorder")
		}
		return cluster.NewClient(clnt, clnt.DiscoveryClient(), eventRecorder, clnt.Config(), clnt.HttpClient()), nil
	}

	reconciler := component.NewReconciler[*operatorv1alpha1.Component](
		o.options.Name,
		resourceGenerator,
		component.ReconcilerOptions{
			DefaultServiceAccount: &o.options.DefaultServiceAccount,
			UpdatePolicy:          ref(reconciler.UpdatePolicySsaOverride),
			NewClient:             newClient,
		},
	).WithPostReadHook(
		makeFuncPostRead(),
	).WithPreReconcileHook(
		makeFuncPreReconcile(mgr.GetCache()),
	).WithPostReconcileHook(
		makeFuncPostReconcile(),
	).WithPreDeleteHook(
		makeFuncPreDelete(mgr.GetCache()),
	)

	if err := reconciler.SetupWithManagerAndBuilder(mgr, blder); err != nil {
		return errors.Wrapf(err, "unable to create controller")
	}

	mgr.Add(httprepository.NewChecker(mgr.GetCache(), reconciler, mgr.GetLogger()))

	return nil
}
