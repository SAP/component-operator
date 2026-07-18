---
title: "CRD Deletion Safeguarding"
linkTitle: "CRD Deletion Safeguarding"
weight: 1
description: >
  Install cert-manager via component-operator, explore Helm hook annotation mappings, and observe extension-type deletion guards in action
---

This tutorial installs [cert-manager](https://cert-manager.io/) using component-operator. Along the way you will see how the HelmGenerator translates Helm hook annotations into component-operator lifecycle annotations, and observe how component-operator protects against premature deletion when foreign instances of managed CRDs still exist in the cluster.

## Prerequisites

You need a Kubernetes cluster with Flux source-controller and component-operator installed. If you don't have one yet, the [Cluster Setup](../../getting-started/setup) guide walks you through creating a local kind cluster with everything in place.

## 1. Create the Flux source objects

cert-manager publishes its Helm chart to an OCI registry. Create a `HelmRepository` pointing at the registry and a `HelmChart` selecting the `cert-manager` chart:

```yaml
# cert-manager-source.yaml
---
apiVersion: source.toolkit.fluxcd.io/v1
kind: HelmRepository
metadata:
  name: cert-manager
  namespace: default
spec:
  type: oci
  url: oci://quay.io/jetstack/charts
  interval: 5m
---
apiVersion: source.toolkit.fluxcd.io/v1
kind: HelmChart
metadata:
  name: cert-manager
  namespace: default
spec:
  chart: cert-manager
  sourceRef:
    kind: HelmRepository
    name: cert-manager
  interval: 5m
```

```bash
kubectl apply -f cert-manager-source.yaml
```

Wait until the chart artifact is fetched:

```bash
kubectl get helmchart cert-manager
```

The `READY` column should show `True` before continuing.

## 2. Create the Component

Create a `Component` that references the `HelmChart` source and deploys into the `cert-manager` namespace. Setting `crds.enabled: true` bundles the CRD manifests directly into the chart output so that component-operator manages them as regular dependent objects:

```yaml
# cert-manager-component.yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  name: cert-manager
  namespace: default
spec:
  sourceRef:
    fluxHelmChart:
      name: cert-manager
  path: cert-manager
  namespace: cert-manager
  values:
    crds:
      enabled: true
```

```bash
kubectl apply -f cert-manager-component.yaml
```

Watch the component converge:

```bash
kubectl wait component cert-manager --for condition=Ready
```

Observe that component-operator has automatically created the `cert-manager` namespace and applied all dependent objects in the right order.

## 3. Explore the dependent objects

### Inspecting the inventory

component-operator tracks every resource it has applied in the component's inventory. You can list the managed objects:

```bash
kubectl get component cert-manager -o json | jq .status.inventory
```

The list includes the CRD definitions, the `cert-manager`, `cert-manager-cainjector`, and `cert-manager-webhook` deployments, their service accounts and RBAC objects, and the services.

### Helm hook annotations and their mappings

The cert-manager chart ships a `startupapicheck` Job that carries standard Helm hook annotations:

```yaml
# original annotations on the Job in the chart template
helm.sh/hook: post-install
helm.sh/hook-weight: "1"
helm.sh/hook-delete-policy: before-hook-creation,hook-succeeded
```

When the HelmGenerator renders the chart, it translates these to component-operator annotations on the applied object:

- `helm.sh/hook-weight` in combination with `helm.sh/hook: post-install` maps to `component-operator.cs.sap.com/apply-order`, placing the Job in its own wave after the main cert-manager resources.
- `helm.sh/hook-delete-policy: before-hook-creation` changes the update policy of the job to `Recreate`, such that the job will in fact be deleted and created again with the next change being reconciled.
- `helm.sh/hook-delete-policy: hook-succeeded` maps to `component-operator.cs.sap.com/purge-order` with the same wave number. This tells component-operator to delete the Job from the cluster at the end of that apply wave, exactly mirroring the Helm hook behaviour.

You can inspect the annotations on the live object to verify (the Job is purged after the first successful run, so catch it while the component is first being applied, or look at what component-operator recorded):

```bash
kubectl get job -n cert-manager -l app.kubernetes.io/component=startupapicheck \
  -o jsonpath='{.items[0].metadata.annotations}' | jq .
```

### CRD deployment

Because the cert-manager CRDs are included in the chart output (`crds.enabled: true`), component-operator picks them up as regular dependent objects and applies its implicit ordering: CRD definitions deployed early to ensure that controllers reconciling them can properly start afterwards. Note that the annotation `helm.sh/resource-policy: keep` in the original Helm output maps to a deletion policy of `Orphan` for the CRD objects.

## 4. Create a Certificate using a self-signed Issuer

With cert-manager running, create a `ClusterIssuer` and a `Certificate` to verify the installation:

```yaml
# selfsigned-test.yaml
---
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: selfsigned
spec:
  selfSigned: {}
---
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: example
  namespace: default
spec:
  secretName: example-tls
  issuerRef:
    name: selfsigned
    kind: ClusterIssuer
  dnsNames:
    - example.local
```

```bash
kubectl apply -f selfsigned-test.yaml
```

Wait for the certificate to be issued:

```bash
kubectl wait certificate example --for condition=Ready
```

Once `READY` is `True`, cert-manager has signed the certificate and stored the result in the `example-tls` Secret in the `cert-manager` namespace.

## 5. Delete the Component — and observe the wait

Now delete the component:

```bash
kubectl delete component cert-manager
```

The component does **not** disappear immediately. Check its state:

```bash
kubectl get component cert-manager
```

```
NAME           STATE           REASON              ...
cert-manager   Deleting        DeletionBlocked     ...
```

### Why deletion is blocked

The cert-manager chart includes CRD definitions for types such as `Certificate`, `ClusterIssuer`, and `CertificateRequest`. component-operator recognises these as **extension types** it manages. The `ClusterIssuer` and `Certificate` you created in step 4 are **foreign instances** of those types — they exist in the cluster but are not part of the component's own inventory.

component-operator blocks deletion until all foreign instances of managed extension types are gone. This ensures that the cert-manager controller — which processes finalizers and admission webhooks for those resources — stays alive long enough to handle their removal cleanly. Without this guard, deleting the controller before its CRD instances are gone would leave orphaned resources with stuck finalizers.

Inspect the blocking message in the component status:

```bash
kubectl describe component cert-manager
```

## 6. Remove the foreign instances and complete deletion

Delete the `Certificate` and `ClusterIssuer`. cert-manager will also clean up the associated `CertificateRequest` and the `example-tls` Secret:

```bash
kubectl delete certificate example
kubectl delete clusterissuer selfsigned
```

Wait a moment for cert-manager to process the deletions, then check whether any instances of the managed types remain:

```bash
kubectl get certificates,clusterissuers,certificaterequests -A
```

Once no instances remain, component-operator detects the change and proceeds with the component deletion. Confirm that it is gone:

```bash
kubectl get component cert-manager
```

```
Error from server (NotFound): components.core.cs.sap.com "cert-manager" not found
```

The component, its deployments, RBAC resources, and all other dependent objects have been fully removed from the cluster.
Note that the CRD objects themselves are still there because the cert-manager maintainers decided to set the `helm.sh/resource-policy: keep` annotation on the CRDs, which makes component-operator orphan them on deletion. If they wouldn't have it done this way, and shipped the CRDs in the usual way via the chart's ./crds subdirectory, the CRDs would have been removed here as well.
