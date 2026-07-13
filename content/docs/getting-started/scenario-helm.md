---
title: "Scenario 1: Helm"
linkTitle: "Scenario 1: Helm"
weight: 2
description: >
  Deploy podinfo using its OCI-hosted Helm chart
---

Deploy [podinfo](https://github.com/stefanprodan/podinfo) using its OCI-hosted Helm chart. The equivalent plain Helm command would be:

```bash
helm upgrade -i podinfo oci://ghcr.io/stefanprodan/charts/podinfo
```

With component-operator, a Flux `HelmRepository` and `HelmChart` object serve the chart artifact, and a `Component` drives the deployment lifecycle.

## 1. Create the Flux source objects

```yaml
# podinfo-helm-source.yaml
---
apiVersion: source.toolkit.fluxcd.io/v1
kind: HelmRepository
metadata:
  name: podinfo
  namespace: default
spec:
  type: oci
  url: oci://ghcr.io/stefanprodan/charts
  interval: 5m
---
apiVersion: source.toolkit.fluxcd.io/v1
kind: HelmChart
metadata:
  name: podinfo
  namespace: default
spec:
  chart: podinfo
  sourceRef:
    kind: HelmRepository
    name: podinfo
  interval: 5m
```

```bash
kubectl apply -f podinfo-helm-source.yaml
```

## 2. Create the Component

```yaml
# podinfo-helm.yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  name: podinfo
  namespace: default
spec:
  sourceRef:
    fluxHelmChart:
      name: podinfo
  path: podinfo
```

```bash
kubectl apply -f podinfo-helm.yaml
```

## 3. Verify

```bash
kubectl get component podinfo
kubectl get pods -l app.kubernetes.io/name=podinfo
```

## 4. Cleanup

```bash
kubectl delete component podinfo
kubectl delete helmchart podinfo
kubectl delete helmrepository podinfo
```

---

Continue with [Scenario 2: Kustomize](../scenario-kustomize) to deploy the same application from a Git repository.
