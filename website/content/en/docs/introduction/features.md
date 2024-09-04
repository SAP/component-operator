---
title: "Features"
linkTitle: "Features"
weight: 20
type: "docs"
description: >
  List of key features
---

### Enhanced readiness checks
By default, a resource's readiness will be checked by means of the [kstatus](https://github.com/kubernetes-sigs/cli-utils/blob/master/pkg/kstatus/README.md) framework. That means, standard builtin resources will be evaluated canonically, and other resources will be judged according to their 'Ready' condition. In addition, the checks can be tweaked by setting certain annotations in the manifests, which allows, for example, to look at other conditions.

### Apply and delete waves
By setting according annotations in the manifests, the resources of a component can be grouped into ordered apply and delete waves. When applying the component to the cluster, the component operator proceeds from wave to wave, that is, resources belonging to a given apply wave will only be processed once all resources of all previous apply waves are applied **and ready** according to the readiness criteria explained before. Analogously, resources belonging to a certain delete wave will only be processed after all resources of all previous delete waves are delete **and gone**.

### Consistent handling of operators and extension types
Whenever a component containing an extension type definition (such as a `CustomResourceDefinition`, or an `APIService` object) is to be deleted, the deletion *all* the component's resources is deleayed until there are no more (foreign) instances of the contained types. This ensures that the operator controlling the extension type (probably also being part of the component) is not prematurely removed, and has a chance to properly reconcile the deletion of all instances which it is responsible for. In other words, the usual problems around orphaned custom resources with stuck finalizers are much more unlikely to occur.

### Improved templating
Source manifests can be provided as Helm charts or kustomizations. Other than usual, not only Helm charts, but also the kustomization's files can be go templates, which will be rendered before passing the output to the kustomize machinery. In these templates, a similar, actually more powerful compared to Helm, set of template functions is availbel. The variable bindings for the templates are provided through the component instance, inline or as secret references.

### Enhanced dependency management between components
Many deployers (such as the [flux Kustomization controller](https://fluxcd.io/flux/components/kustomize/)) allow to declare dependencies between components, in the sense that the reconciliation of a component happens only if all components it depends on are successfully reconciled. But this usually affects only the process of applying the component (i.e. creating or updating it). In contrast, the component operator honors declared dependencies as well during deletion, in reverse order.

### Deterministic nesting of related components
It is a common use case to nest components, in the sense that one component includes another one as a dependent object. Often, both the outer and inner components are referencing the same source. In that case, if the content of the source changes, there is a risk of a race condition; if the outer component reconciles before the inner one, it may observe the previous state of the inner component and therefore report its own status incorrectly. Of course, the situation would eventually resolve, but nevertheless, the temporarily incorrect status may cause problems. To overcome this race condition, the component operator allows to pin a component (the inner one in this situation) to a specific revision of the source.