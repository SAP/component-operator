---
title: "Terminology"
linkTitle: "Terminology"
weight: 10
type: "docs"
description: >
  Terms, Glossary
---

In that context, a component means such a set of associated resources. The component is described through the Kubernetes manifests
(in yaml or json) of its resources.

### Component

The term component is overloaded. It would be more exact to distinguish between component definition and component instance, but in many places, just the short term is used.

A component definition describes a set of associated resources, to be applied to Kubernetes clusters. These resources are described through
Kubernetes manifests files, which can be templatized, using Go template syntax. As source of the manifests, a Git repository, an OCI package, or a Helm chart can be referenced. To be more precise, the component-operator currently supports all source types offered by the [flux source controller](https://fluxcd.io/flux/components/source/). In the case that the manifests leverage Go templating, probably certain variables are required to render the templates, which we refer to as component parameters or component values.

A component instance is the concrete instantiation of a component in a specific Kubernetes cluster. It is modelled by the custom resource type `components.core.cs.sap.com`, which is managed by the component-operator. A typical component may look like this:

```yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  namespace: default
  name: my-component
spec:
  sourceRef:
    fluxGitRepository:
      name: my-gitrepo
  path: ./my-component
  valuesFrom:
  - name: my-component-parameters
```

(using a Git repository to retrieve the manifests) or 

```yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  namespace: default
  name: my-component
spec:
  sourceRef:
    fluxHelmChart:
      name: my-helmchart
  path: my-chart
  valuesFrom:
  - name: my-component-parameters
```

(using a Helm chart as source for the manifests).

### Dependent object/resource

A dependent object (or dependent resource or simply a dependent) means one of the Kubernetes resources belonging to the component.
Each dependent object has an `owner-id` label/annotation referring to the component to which the object belongs to. Converseley, the list of dependent objects of a component can be seen in the `status.inventory` field of the `Component` resource.

### Component source

