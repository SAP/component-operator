---
title: "Deploying to a Remote Cluster"
linkTitle: "Deploying to a Remote Cluster"
weight: 3
description: >
  Use spec.kubeConfig and mustLocalLookup to forward a local secret to a remote cluster
---

This tutorial deploys a resource to a second, **remote** Kubernetes cluster. It demonstrates `spec.kubeConfig` for targeting a remote cluster, and the `mustLocalLookup` template function, which reads objects from the **local** cluster even when the component is deploying to a remote one.

## Prerequisites

You need two clusters.

**Local cluster** (`kind`) — hosts Flux source-controller and component-operator. If you don't have one yet, the [Cluster Setup](../../getting-started/setup) guide walks you through creating it.

**Remote cluster** (`kind-target`) — the deployment target. No Flux or component-operator installation is needed here. Create it now:

```bash
kind create cluster --name kind-target
```

Switch back the context to the local cluster:

```bash
kubectl config use-context kind-kind
```

## 1. Create the source secret in the local cluster

Switch to the local cluster context and create a secret that the component will mirror to the remote cluster:

```bash
kubectl create secret generic original --from-literal foo=bar
```

## 2. Make the remote kubeconfig available as a secret

component-operator reads a kubeconfig from a Kubernetes Secret to authenticate against the remote cluster. The default kubeconfig produced by `kind get kubeconfig` uses `server: https://127.0.0.1:<port>` — this is reachable from your laptop, but not from inside a pod. From a pod in the `kind` cluster, the `kind-target` API server is reachable at `kind-target-control-plane:6443`, the hostname of the `kind-target` control plane container on the shared Docker `kind` network.

Fetch the kubeconfig, patch the server address, and store it as a secret in one step:

```bash
kind get kubeconfig --name kind-target \
  | sed 's|server: https://127.0.0.1:[0-9]*|server: https://kind-target-control-plane:6443|' \
  | kubectl create secret generic kind-target-kubeconfig \
    --from-file=value=/dev/stdin
```

The key name `value` is the convention component-operator tries first when `spec.kubeConfig.secretRef.key` is not set.

## 3. Create the Blueprint

The Blueprint contains a single template that reads the `original` secret from the **local** cluster and renders it as a manifest to be applied to the **remote** cluster.

```yaml
# secret-copy-blueprint.yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Blueprint
metadata:
  name: secret-copy
  namespace: default
spec:
  files:
    secret.yaml: |
      {{- $src := mustLocalLookup "v1" "Secret" "default" "original" }}
      apiVersion: v1
      kind: Secret
      metadata:
        name: original-copy
        namespace: default
      type: Opaque
      data: {{ $src.data | toJson }}
```

```bash
kubectl apply -f secret-copy-blueprint.yaml
```

**Why `mustLocalLookup` and not `lookup`?**

When `spec.kubeConfig` is set, the regular `lookup` and `mustLookup` functions query the **remote** (target) cluster. The `original` secret lives in the local cluster (where the component lives), so using `lookup` would make no sense. `mustLocalLookup` always queries the local cluster regardless of where the component deploys. The `must` variant causes the reconciliation to fail immediately if the source secret is absent, rather than silently propagating empty data.

See [Impersonation and Remote Clusters](../../usage/impersonation) for the full reference on local vs. target lookup.

## 4. Create the Component

```yaml
# secret-copy-component.yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  name: secret-copy
  namespace: default
spec:
  sourceRef:
    blueprint:
      name: secret-copy
  kubeConfig:
    secretRef:
      name: kind-target-kubeconfig
```

```bash
kubectl apply -f secret-copy-component.yaml
```

Watch the component converge:

```bash
kubectl get component secret-copy -w
```

Once `STATE` shows `Ready`, the secret has been written to the remote cluster.

## 5. Verify

Check that the secret was created in the remote cluster:

```bash
kubectl --context kind-kind-target get secret original-copy -n default
```

Decode the value to confirm the content matches:

```bash
kubectl --context kind-kind-target \
  get secret original-copy -o jsonpath='{.data.foo}' | base64 -d
# bar
```

## 6. Observe live propagation

Update the source secret in the local cluster:

```bash
kubectl patch secret original -p '{"stringData":{"foo":"updated"}}'
```

`mustLocalLookup` is evaluated on every reconcile cycle, so the next time component-operator reconciles the component (within the default `requeueInterval` of 10 minutes) the remote secret will reflect the new value. To trigger a reconcile immediately, touch the component:

```bash
kubectl annotate component secret-copy reconcile=now --overwrite
```

Then verify the remote secret was updated:

```bash
kubectl --context kind-kind-target \
  get secret original-copy -o jsonpath='{.data.foo}' | base64 -d
# updated
```

## 7. Cleanup

Delete the component first — this removes the `original` secret from the **remote** cluster:

```bash
kubectl delete component secret-copy
```

Then remove the local resources:

```bash
kubectl delete blueprint secret-copy
kubectl delete secret original kind-target-kubeconfig
```

Optionally delete the remote cluster:

```bash
kind delete cluster --name kind-target
```
