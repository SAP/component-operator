---
title: "Scenario 2: Kustomize"
linkTitle: "Scenario 2: Kustomize"
weight: 3
description: >
  Deploy podinfo using a Kustomize overlay from a Git repository
---

Deploy [podinfo](https://github.com/stefanprodan/podinfo) using the Kustomization at [`github.com/stefanprodan/podinfo//kustomize`](https://github.com/stefanprodan/podinfo/tree/master/kustomize). A Flux `GitRepository` serves the repository content, and the `Component` selects the `kustomize` subdirectory as its source path.

## 1. Create the Flux GitRepository

```yaml
# podinfo-git-source.yaml
---
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: podinfo
spec:
  interval: 5m
  url: https://github.com/stefanprodan/podinfo
  ref:
    branch: master
```

```bash
kubectl apply -f podinfo-git-source.yaml
```

## 2. Create the Component

```yaml
# podinfo-kustomize.yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  name: podinfo
  namespace: default
spec:
  sourceRef:
    fluxGitRepository:
      name: podinfo
  path: kustomize
```

```bash
kubectl apply -f podinfo-kustomize.yaml
```

## 3. Verify

```bash
kubectl get component podinfo
kubectl get pods -l app=podinfo
```

## 4. Cleanup

```bash
kubectl delete component podinfo
kubectl delete gitrepository podinfo
```



---

Continue to [Next Steps](../next-steps) to explore what component-operator has to offer beyond the basics.
