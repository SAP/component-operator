/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package component

import (
	"github.com/pkg/errors"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	fluxevents "github.com/fluxcd/pkg/runtime/events"
	fluxsourcev1 "github.com/fluxcd/source-controller/api/v1"
	fluxsourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"

	"github.com/sap/component-operator-runtime/pkg/cluster"
	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/reconciler"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
	"github.com/sap/component-operator/internal/generator"
)

type ReconcilerOptions struct {
	Name                    string
	DefaultServiceAccount   string
	MaxConcurrentReconciles int
	EventsAddress           string
}

func SetupWithManager(mgr manager.Manager, options ReconcilerOptions) (*component.Reconciler[*operatorv1alpha1.Component], error) {
	blder := ctrl.NewControllerManagedBy(mgr).
		WithOptions(controller.Options{MaxConcurrentReconciles: options.MaxConcurrentReconciles}).
		Watches(
			&operatorv1alpha1.Component{},
			newComponentHandler(mgr.GetCache(), mgr.GetLogger())).
		Watches(
			&operatorv1alpha1.Blueprint{},
			newBlueprintHandler(mgr.GetCache(), mgr.GetLogger())).
		Watches(
			&fluxsourcev1.GitRepository{},
			newFluxSourceHandler(mgr.GetCache(), mgr.GetLogger())).
		Watches(
			&fluxsourcev1beta2.OCIRepository{},
			newFluxSourceHandler(mgr.GetCache(), mgr.GetLogger())).
		Watches(
			&fluxsourcev1beta2.Bucket{},
			newFluxSourceHandler(mgr.GetCache(), mgr.GetLogger())).
		Watches(
			&fluxsourcev1.HelmChart{},
			newFluxSourceHandler(mgr.GetCache(), mgr.GetLogger()))

	resourceGenerator, err := generator.NewGenerator(mgr.GetClient())
	if err != nil {
		return nil, errors.Wrap(err, "error initializing resource generator")
	}

	newClient := func(clnt cluster.Client) (cluster.Client, error) {
		if options.EventsAddress == "" {
			return clnt, nil
		}
		eventRecorder, err := fluxevents.NewRecorderForScheme(clnt.Scheme(), clnt.EventRecorder(), mgr.GetLogger(), options.EventsAddress, options.Name)
		if err != nil {
			return nil, errors.Wrap(err, "error initializing wrapping event recorder")
		}
		return cluster.NewClient(clnt, clnt.DiscoveryClient(), eventRecorder, clnt.Config(), clnt.HttpClient()), nil
	}

	reconciler := component.NewReconciler[*operatorv1alpha1.Component](
		options.Name,
		resourceGenerator,
		component.ReconcilerOptions{
			DefaultServiceAccount: &options.DefaultServiceAccount,
			UpdatePolicy:          new(reconciler.UpdatePolicySsaOverride),
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
		return nil, errors.Wrapf(err, "unable to create component controller")
	}

	return reconciler, nil
}
