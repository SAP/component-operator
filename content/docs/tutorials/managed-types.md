---
title: "Declaring Additional Managed Types"
linkTitle: "Declaring Additional Managed Types"
weight: 7
description: >
  Use spec.additionalManagedTypes to protect against stuck finalizers when an operator registers CRDs implicitly at startup
---

This tutorial demonstrates `spec.additionalManagedTypes` — a mechanism that lets you tell component-operator about CRD types that a component introduces **implicitly** (by a controller registering them at startup) rather than by including the CRD manifests directly. Without this declaration, deleting the component while instances of those types still exist leads to the well-known **stuck finalizer problem**. With it, component-operator blocks the deletion until all foreign instances are gone first.

[Crossplane](https://www.crossplane.io/) providers are a textbook example: the provider itself is a small `Provider` object, but it registers a whole set of CRDs when it starts up. Those CRDs never appear in the component's manifest set, so component-operator cannot infer them automatically.

## Prerequisites

You need a Kubernetes cluster with Flux source-controller and component-operator installed. If you don't have one yet, follow the [Cluster Setup](../../getting-started/setup) guide to create a local kind cluster with everything in place.

## 1. Install Crossplane

Create a dedicated namespace and apply the Flux source objects and Component for Crossplane:

```bash
kubectl create namespace crossplane
```

```yaml
# crossplane.yaml
---
apiVersion: source.toolkit.fluxcd.io/v1
kind: HelmRepository
metadata:
  name: crossplane
  namespace: crossplane
spec:
  url: https://charts.crossplane.io/stable
  interval: 5m
---
apiVersion: source.toolkit.fluxcd.io/v1
kind: HelmChart
metadata:
  name: crossplane
  namespace: crossplane
spec:
  chart: crossplane
  sourceRef:
    kind: HelmRepository
    name: crossplane
  interval: 5m
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  name: crossplane
  namespace: crossplane
spec:
  sourceRef:
    fluxHelmChart:
      name: crossplane
  path: crossplane
```

```bash
kubectl apply -f crossplane.yaml
```

Wait for Crossplane to become ready:

```bash
kubectl get component crossplane -n crossplane -w
```

## 2. Install the dummy provider (without additionalManagedTypes)

The [provider-dummy](https://github.com/upbound/provider-dummy) is a minimal Crossplane provider designed for testing. It registers a handful of CRDs — including `Robot` in the `iam.dummy.upbound.io` group — when it starts up. None of those CRDs are part of the `Provider` manifest itself.

Create the Blueprint and Component:

```yaml
# crossplane-provider-dummy.yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Blueprint
metadata:
  name: crossplane-provider-dummy
  namespace: crossplane
spec:
  files:
    resources.yaml: |
      ---
      apiVersion: pkg.crossplane.io/v1
      kind: Provider
      metadata:
        name: provider-dummy
      spec:
        package: xpkg.upbound.io/upbound/provider-dummy:v0.3.0
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  name: crossplane-provider-dummy
  namespace: crossplane
spec:
  sourceRef:
    blueprint:
      name: crossplane-provider-dummy
```

```bash
kubectl apply -f crossplane-provider-dummy.yaml
```

Wait for the provider to become ready (Crossplane pulls the package and installs the CRDs):

```bash
kubectl get component crossplane-provider-dummy -n crossplane -w
```

## 3. Create an instance of a provider-managed type

Once the provider is running, create a `Robot` custom resource from the group it registered:

```yaml
# robot.yaml
---
apiVersion: dummy.upbound.io/v1alpha1
kind: ProviderConfig
metadata:
  name: default
spec:
  endpoint: http://127.0.0.1:9090
---
apiVersion: iam.dummy.upbound.io/v1alpha1
kind: Robot
metadata:
  name: example
spec:
  forProvider:
    color: yellow
```

```bash
kubectl apply -f robot.yaml
kubectl get robot example
```

## 4. Witness the stuck-finalizer problem

### Delete the provider component

Delete the `crossplane-provider-dummy` component without first removing the `Robot`:

```bash
kubectl delete component crossplane-provider-dummy -n crossplane
```

The deletion succeeds immediately. component-operator removes the `Provider` object from the cluster. The provider pod shuts down and its control loop stops.

### Try to delete the Robot

```bash
kubectl delete robot example
```

Kubernetes accepts the request and sets a deletion timestamp on the object, but the `Robot` never disappears:

```bash
kubectl get robot example
```

The object is stuck. The provider has placed a finalizer on it, and the only controller that knows how to remove that finalizer — the provider — is no longer running. This is the stuck finalizer problem.

### Manual recovery

To get out of this situation you have to forcibly remove the finalizer by hand:

```bash
kubectl patch robot example \
  --type json \
  -p '[{"op":"remove","path":"/metadata/finalizers"}]'
```

The object is now deleted. Finally the same has to be done for the `ProviderConfig``: 

```bash
kubectl patch providerconfigs.dummy.upbound.io default \
  --type json \
  -p '[{"op":"remove","path":"/metadata/finalizers"}]'
```

## 5. Reinstall the provider with additionalManagedTypes

The fix is to declare the provider's implicitly managed types in the Component so that component-operator knows about them before any deletion is attempted.

```yaml
# crossplane-provider-dummy-v2.yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  name: crossplane-provider-dummy
  namespace: crossplane
spec:
  sourceRef:
    blueprint:
      name: crossplane-provider-dummy
  additionalManagedTypes:
    - group: dummy.upbound.io
      kind: "*"
    - group: "*.dummy.upbound.io"
      kind: "*"
```

```bash
kubectl apply -f crossplane-provider-dummy-v2.yaml
```

Wait for the provider to be ready again:

```bash
kubectl get component crossplane-provider-dummy -n crossplane -w
```

### Recreate the Robot

```bash
kubectl apply -f robot.yaml
```

## 6. Delete safely this time

Attempt to delete the provider component again:

```bash
kubectl delete component crossplane-provider-dummy -n crossplane
```

This time the component does **not** disappear immediately. Instead it enters `Deleting/DeletionBlocked` state:

```bash
kubectl get component crossplane-provider-dummy -n crossplane
```

```
NAMESPACE    NAME                        STATE            REASON
crossplane   crossplane-provider-dummy   Deleting         DeletionBlocked
```

component-operator has recognised `Robot` as a foreign instance of a type in the `*.dummy.upbound.io` groups — an additional managed type — and is holding back the deletion until no instances remain.

You can read the blocking message in the component status:

```bash
kubectl describe component crossplane-provider-dummy -n crossplane
```

### Delete the Robot and watch the cascade

Now delete the `Robot` — the provider is still running, so the finalizer is processed cleanly:

```bash
kubectl delete robot example
kubectl get robot example
# Error from server (NotFound): ...
```

Similarly, delete the `ProviderConfig`: 

```bash
kubectl delete providerconfigs.dummy.upbound.io default
kubectl get providerconfigs.dummy.upbound.io default
# Error from server (NotFound): ...
```

Once all foreign instances of the managed types are gone, component-operator detects the change and completes the component deletion automatically:

```bash
kubectl get component crossplane-provider-dummy -n crossplane
# Error from server (NotFound): ...
```

The provider, its `Provider` object, and all associated resources are removed in the correct order, with no stuck finalizers.

## 7. Cleanup

Delete the Crossplane installation and the namespace:

```bash
kubectl delete namespace crossplane
```
