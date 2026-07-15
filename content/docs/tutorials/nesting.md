---
title: "Nesting Components"
linkTitle: "Nesting Components"
weight: 4
description: >
  Create a Component that manages another Component as a dependent object, with source-revision pinning
---

This tutorial demonstrates **nested components referencing the same source**: a pattern where one `Component` (the outer) creates and manages another `Component` (the inner) as one of its dependent objects. Combining this with source pinning ensures both components consistently reconcile against the same state of the referenced source.

## Prerequisites

You need a Kubernetes cluster with Flux source-controller and component-operator installed. If you don't have one yet, follow the [Cluster Setup](../../getting-started/setup) guide.

## What you will build

A single `Blueprint` (`nesting`) contains two subdirectories:

| Path | Content |
|---|---|
| `inner/` | A `Job` that sleeps for 30 seconds |
| `outer/` | A `Component` manifest that targets `inner/`, pinned to the outer component's current source state |

The user creates only the **outer** component. The outer component then creates the **inner** component as a dependent object. The inner component in turn creates the Job.

The resulting hierarchy in the cluster:

```
Component outer           (created by you)
  └── Component inner     (created by outer as a dependent object)
        └── Job sleep     (created by inner)
```

## 1. Create the Blueprint

Create a `Blueprint` source that holds the manifests of both components.

```yaml
# nesting-blueprint.yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Blueprint
metadata:
  name: nesting
  namespace: default
spec:
  files:
    dummy.version: |
      1
    inner/resources.yaml: |
      ---
      apiVersion: batch/v1
      kind: Job
      metadata:
        name: sleep-{{ componentRevision }}
      spec:
        template:
          spec:
            restartPolicy: Never
            containers:
              - name: sleep
                image: alpine
                command:
                  - /bin/sh
                  - -c
                  - sleep 30
    outer/resources.yaml: |
      {{- $outer := component }}
      ---
      apiVersion: core.cs.sap.com/v1alpha1
      kind: Component
      metadata:
        name: inner
      spec:
        sourceRef:
          blueprint:
            name: nesting
        path: inner
        revision: {{ $outer.Status.LastAttemptedRevision }}
        digest: {{ $outer.Status.LastAttemptedDigest }}
```

```bash
kubectl apply -f nesting-blueprint.yaml
```

The `component` template function returns the current state of the owning (outer) component. By writing `$outer.Status.LastAttemptedRevision` and `$outer.Status.LastAttemptedDigest` into the inner component's `spec`, we pin the inner component to exactly the same Blueprint snapshot the outer is currently reconciling. Note that in the example, we are pinning both the source digest and source revision. In many cases it is sufficient (or desired) to pin only one of them. 

If the source changes, two cases can happen.

**Case 1:** The inner component is reconciled before the outer component. In this case, the source version does not match the pinned version, which makes the inner component go into a `Pending` state. Once the outer component is reconciled, it updates the pinning of the inner component which can now proceed (because the pinning matches the source). While that happens the outer component is in a `Processing` state, waiting for the inner component to become ready. Once the inner component is ready, the outer component becomes ready as well.

**Case 2:** The outer component is reconciled before the inner component. It updates the inner component's pinning to the new source version. Because of the update, the inner component gets reconciled immediately. It observes the new source version, matching the pinning, and reconciles. While this happens, the outer component is in `Processing` state. Once the inner component becomes ready, the outer component does so too.

Without this pinning, there would be a risk that the outer component (reconciling first) observes the previous `Ready` state of the inner component, and therefore reaches a 'false' `Ready` state itself. A subsequent error in the inner component would affect the outer component's state only with the next periodic requeue.

## 2. Create the outer Component

```yaml
# outer-component.yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  name: outer
  namespace: default
spec:
  sourceRef:
    blueprint:
      name: nesting
  path: outer
```

```bash
kubectl apply -f outer-component.yaml
```

Watch the components in real time:

```bash
kubectl get component -w
```

You should observe the following sequence:

1. `outer` enters `Processing` and renders `outer/resources.yaml`.
2. A new `inner` component appears in the cluster, created by `outer` as a dependent object.
3. `inner` enters `Processing` and creates the `sleep` Job.
4. `inner` stays in `Processing` for ~30 seconds while the Job runs.
5. The Job completes. `inner` transitions to `Ready`.
6. `outer` sees that its dependent `inner` component is `Ready` and transitions to `Ready` itself.

## 3. Verify the pinning

Once both components are `Ready`, confirm that the inner component's `spec.revision` and `spec.digest` match the outer component's status:

```bash
kubectl get component outer \
  -o jsonpath='revision={.status.lastAttemptedRevision} digest={.status.lastAttemptedDigest}{"\n"}'

kubectl get component inner \
  -o jsonpath='revision={.spec.revision} digest={.spec.digest}{"\n"}'
```

Both outputs should show identical values, confirming that the inner component is locked to the same source snapshot as the outer.

## 4. Update the source

Now bump the source by making a dummy change:

```bash
kubectl patch blueprint nesting --type merge -p '{"spec":{"files":{"dummy.version":"2"}}}'
```

Observe that a new job gets created; this is because the job name includes the `componentRevision` template function (see the [templates reference](../../usage/manifests/#template-functions) for more information). Due to the pinning, the outer component always goes into a `Processing` state, waiting for the inner component to become ready again.

## 5. Cleanup

Deleting the outer component cascades: because `inner` is a dependent object of `outer`, it is deleted along with the Job it manages.

```bash
kubectl delete component outer
```

Verify both components are gone:

```bash
kubectl get component
```

Remove the Blueprint:

```bash
kubectl delete blueprint nesting
```
