/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package operator

import (
	"context"
	"flag"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	fluxsourcev1 "github.com/fluxcd/source-controller/api/v1"
	fluxsourcev1beta1 "github.com/fluxcd/source-controller/api/v1beta1"
	fluxsourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/operator"
	"github.com/sap/component-operator-runtime/pkg/reconciler"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
	"github.com/sap/component-operator/internal/generator"
)

// TODO: write some logs (e.g. in the hooks)

const Name = "component-operator.cs.sap.com"

const (
	dependenciesIndexKey  string = ".metadata.dependencies"
	gitRepositoryIndexKey string = ".metadata.gitRepository"
	ociRepositoryIndexKey string = ".metadata.ociRepository"
	bucketIndexKey        string = ".metadata.bucket"
	helmChartIndexKey     string = ".metadata.helmChart"
)

type Options struct {
	Name       string
	FlagPrefix string
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
}

func (o *Operator) ValidateFlags() error {
	return nil
}

func (o *Operator) GetUncacheableTypes() []client.Object {
	return []client.Object{&operatorv1alpha1.Component{}}
}

func (o *Operator) Setup(mgr ctrl.Manager) error {
	// TODO: should we pass a meaningful context?
	if err := mgr.GetCache().IndexField(context.TODO(), &operatorv1alpha1.Component{}, dependenciesIndexKey, indexByDependencies); err != nil {
		return errors.Wrapf(err, "failed setting index field %s", dependenciesIndexKey)
	}
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

	resourceGenerator, err := generator.NewGenerator(mgr.GetClient())
	if err != nil {
		return errors.Wrap(err, "error initializing resource generator")
	}

	blder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: 5}).
		Watches(
			&operatorv1alpha1.Component{},
			&componentHandler{cache: mgr.GetCache(), indexKey: dependenciesIndexKey}).
		Watches(
			&fluxsourcev1beta2.GitRepository{},
			&fluxSourceHandler{cache: mgr.GetCache(), indexKey: gitRepositoryIndexKey}).
		Watches(
			&fluxsourcev1beta2.OCIRepository{},
			&fluxSourceHandler{cache: mgr.GetCache(), indexKey: ociRepositoryIndexKey}).
		Watches(
			&fluxsourcev1beta2.Bucket{},
			&fluxSourceHandler{cache: mgr.GetCache(), indexKey: bucketIndexKey}).
		Watches(
			&fluxsourcev1beta2.HelmChart{},
			&fluxSourceHandler{cache: mgr.GetCache(), indexKey: helmChartIndexKey})

	if err := component.NewReconciler[*operatorv1alpha1.Component](
		o.options.Name,
		resourceGenerator,
		component.ReconcilerOptions{
			UpdatePolicy: &[]reconciler.UpdatePolicy{reconciler.UpdatePolicySsaOverride}[0],
		},
	).WithPostReadHook(
		operatorv1alpha1.LoadSourceReference,
	).WithPreReconcileHook(
		makeFuncPreReconcile(mgr.GetCache()),
	).WithPostReconcileHook(
		makeFuncPostReconcile(),
	).WithPreDeleteHook(
		makeFuncPreDelete(mgr.GetCache(), dependenciesIndexKey),
	).SetupWithManagerAndBuilder(mgr, blder); err != nil {
		return errors.Wrapf(err, "unable to create controller")
	}

	return nil
}
