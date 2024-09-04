---
title: "Introduction"
linkTitle: "Introduction"
weight: 10
type: "docs"
description: >
  Overview and Motivation
---

It is a common task to deploy a set of associated resource manifests into a Kubernetes cluster.
Here, 'deploying' means to do a fresh installation, to update the resources (potentially removing some of them or adding new ones),
or to finally remove the resources from the cluster.

This is where the operator and custom type `components.core.cs.sap.com` provided by this repository comes into play. A component may be deployed as

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
```

In this example, a Git repository (leveraging flux's source-controller) is used as a source providing the manifests of the resource manifests.
Other flux source types, such as OCI or Helm repositories are supported as well.