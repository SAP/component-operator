---
title: "Cluster Setup"
linkTitle: "Cluster Setup"
weight: 1
description: >
  Create a local cluster and install the required components
---

## Prerequisites

Install the following tools before you begin:

- [`kind`](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
- [`kubectl`](https://kubernetes.io/docs/tasks/tools/)
- [`flux`](https://fluxcd.io/flux/installation/)
- [`helm`](https://helm.sh/docs/intro/install/)

## Step 1 – Create a kind cluster

```bash
kind create cluster
```

## Step 2 – Install the Flux source controller

Component-operator relies on the [Flux source controller](https://fluxcd.io/flux/components/source/) to fetch and serve artifact content. Install only the source controller component:

```bash
flux install --components source-controller
```

## Step 3 – Install component-operator

```bash
helm upgrade --install component-operator \
  oci://ghcr.io/sap/component-operator/charts/component-operator \
  --namespace flux-system
```

Wait for the operator to become ready:

```bash
kubectl -n flux-system rollout status deployment/component-operator
```

---

Continue with [Scenario 1: Helm](../scenario-helm) to deploy your first application.
