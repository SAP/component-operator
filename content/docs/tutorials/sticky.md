---
title: "Sticky Deployments"
linkTitle: "Sticky Deployments"
weight: 10
description: >
  Use spec.sticky to ensure every source version reaches a final state before the next one is applied
---

By default, component-operator always reconciles towards the **latest** available state of the referenced source. If the source changes while a previous reconciliation is still in progress, the operator immediately abandons the previous source version and starts working on the newest one. The intermediate version is skipped without ever reaching a final `Ready` or `Error` state.

This is usually the right behaviour — you want the cluster to converge to the newest desired state as fast as possible. But it can be a problem when every reconciliation outcome matters. For example, if [event streaming to the Flux notification controller](../../usage/notifications) is enabled (see also the [notifications tutorial](../notifications)), a notification is emitted when a reconciliation completes. If versions are skipped, no final event is produced for them, and downstream systems that depend on those events (alerts, audit trails, CI/CD pipelines) will miss intermediate deployments.

Setting `spec.sticky: true` changes this behaviour. Once component-operator starts reconciling a version, it **sticks** to that version until the component reaches `Ready`, or until `spec.timeout` expires. Only then does it pick up the latest queued version — skipping any intermediate ones that arrived in the meantime. See [Timeout and Stickiness](../../usage/timeout) for the full reference.

## Prerequisites

You need a Kubernetes cluster with Flux source-controller and component-operator installed. If you don't have one yet, follow the [Cluster Setup](../../getting-started/setup) guide.

## 1. Create the Blueprint and Component

The Blueprint uses a `dummy.version` file as a trivial version bump trigger. It is listed in `.component-ignore` so it is excluded from template rendering, but its content still changes the Blueprint's digest and revision, which causes component-operator to start a new reconciliation. The actual workload is a `Job` whose name embeds the `componentRevision` counter such that a new job is created for each iteration; previous jobs are orphaned for better observability.

```yaml
# gizmo.yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Blueprint
metadata:
  name: gizmo
  namespace: default
spec:
  files:
    .component-ignore: |
      /dummy.version
    dummy.version: |
      v0.0.1
    resources.yaml: |
      {{- $dummyVersion := readFile "dummy.version" | toString | trim }}
      ---
      apiVersion: batch/v1
      kind: Job
      metadata:
        name: make-gizmo-{{ componentRevision }}
        annotations:
          component-operator.cs.sap.com/delete-policy: orphan-on-apply
        labels:
          dummy.version: {{ $dummyVersion }}
      spec:
        template:
          spec:
            restartPolicy: Never
            containers:
              - name: main
                image: alpine
                command:
                  - /bin/sh
                  - -c
                  - echo "Hello, Gizmo {{ $dummyVersion }}!"
                lifecycle:
                  postStart:
                    sleep:
                      seconds: 30
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  name: gizmo
  namespace: default
spec:
  sourceRef:
    blueprint:
      name: gizmo
  # sticky: true
```

```bash
kubectl apply -f gizmo.yaml
```

Wait for the component to reach `Ready`:

```bash
kubectl get component gizmo -w
```

## 2. Observe the default (non-sticky) behaviour

With `sticky` disabled (the default), bump the Blueprint version three times in quick succession — fast enough that the first reconciliation has not finished yet:

```bash
kubectl patch blueprint gizmo --type merge -p '{"spec":{"files":{"dummy.version":"v0.1.1"}}}'; sleep 3s
kubectl patch blueprint gizmo --type merge -p '{"spec":{"files":{"dummy.version":"v0.1.2"}}}'; sleep 3s
kubectl patch blueprint gizmo --type merge -p '{"spec":{"files":{"dummy.version":"v0.1.3"}}}'
```

Watch the Jobs being created:

```bash
kubectl get jobs -w
```

You will see a new Job appear for **each** bumped version, even while earlier Jobs are still running. component-operator does not wait for the current reconciliation to finish before picking up the next version — it races ahead to the latest available state. Note that every job has a label `dummy.version` which reveals its provenance.

Wait for all Jobs to complete and the component to become `Ready` again:

```bash
kubectl get component gizmo -w
```

## 3. Enable sticky mode

Edit `gizmo.yaml` (or apply the patch below) to enable `spec.sticky: true`:

```yaml
# gizmo.yaml (updated)
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  name: gizmo
  namespace: default
spec:
  sourceRef:
    blueprint:
      name: gizmo
  sticky: true
```

```bash
kubectl apply -f gizmo.yaml
```

Note that changing `spec.sticky` triggers another job execution (because this time, not the source, but the component itself has changed). Wait until things are settled:

```bash
kubectl get component gizmo -w
```

## 4. Observe the sticky behaviour

With sticky mode active, repeat the same rapid-fire version bumps:

```bash
kubectl patch blueprint gizmo --type merge -p '{"spec":{"files":{"dummy.version":"v0.2.1"}}}'; sleep 3
kubectl patch blueprint gizmo --type merge -p '{"spec":{"files":{"dummy.version":"v0.2.2"}}}'; sleep 3
kubectl patch blueprint gizmo --type merge -p '{"spec":{"files":{"dummy.version":"v0.2.3"}}}'
```

Watch the Jobs again:

```bash
kubectl get jobs -w
```

This time the behaviour is noticeably different:

1. A single Job for **version v0.2.1** appears and runs to completion.
2. Versions v0.2.2 and v0.2.3 arrived while version v0.2.1 was being reconciled. Both were queued, but only the **latest** of them (version v0.2.3) is ever acted on. Version v0.2.2 is silently skipped.
3. After the Job for version v0.2.1 completes, a Job for **version v0.2.3** appears.
4. No Job for version v0.2.2 is ever created.

```bash
kubectl get jobs
```

The component emits a final `Ready` event for every version it reconciles (v0.2.1 and v0.2.3), giving downstream notification consumers a complete and reliable delivery record. Intermediate versions (v0.2.2) are dropped in favour of the latest, but each version that does run reaches a definitive final state.

## 5. Cleanup

```bash
kubectl delete component gizmo
kubectl delete blueprint gizmo
kubectl delete jobs -l dummy.version
```
