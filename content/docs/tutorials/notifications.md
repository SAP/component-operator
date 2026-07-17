---
title: "Stream Events to the Flux Notification Controller"
linkTitle: "Stream Events to the Flux Notification Controller"
weight: 9
description: >
  Configure component-operator to emit reconciliation events to Flux and forward them to an external webhook
---

component-operator can emit reconciliation events — successful applies, errors, state transitions — to the [Flux notification controller](https://fluxcd.io/flux/components/notification/), which then forwards them to any configured alert provider (Slack, Microsoft Teams, PagerDuty, a generic webhook, etc.).

This tutorial wires component-operator events to a [smee.io](https://smee.io) channel so you can observe them in real time in your browser.

## Prerequisites

You need a Kubernetes cluster with Flux source-controller and component-operator installed. If you don't have one yet, follow the [Cluster Setup](../../getting-started/setup) guide.

### Install the Flux notification controller

The notification controller is not installed by the minimal `flux install --components source-controller` command used in the getting-started guide. Install it now:

```bash
flux install --components notification-controller
```

This adds the controller and its CRDs (`Alert`, `Provider`, `Receiver`) to the cluster.

## 1. Reconfigure component-operator to stream events

Pass the `--events-address` flag by reinstalling the Helm chart:

```bash
helm upgrade --install component-operator \
  oci://ghcr.io/sap/component-operator/charts/component-operator \
  --namespace flux-system \
  --set options.eventsAddress=http://notification-controller
```

The short hostname `notification-controller` resolves to the notification controller service inside the `flux-system` namespace. If component-operator runs in a **different** namespace, use the fully qualified name instead:

```
http://notification-controller.flux-system.svc.cluster.local/
```

In that case you may also need to extend the network policies in `flux-system` to allow inbound traffic from the component-operator namespace.

## 2. Patch the Alert CRD

Flux's `Alert` CRD does not list `Component` as an allowed event source kind out of the box. Add it with a JSON Patch:

```bash
kubectl patch crd alerts.notification.toolkit.fluxcd.io \
  --type json \
  -p '[{"op":"add","path":"/spec/versions/0/schema/openAPIV3Schema/properties/spec/properties/eventSources/items/properties/kind/enum/-","value":"Component"}]'
```

> **Note:** If your Flux installation has a CRD with multiple versions, repeat the patch for each version entry (changing `versions/0` to `versions/1`, etc.). See [Notifications](../../usage/notifications) for details.

## 3. Create a smee.io channel

[smee.io](https://smee.io) is a free webhook relay — events posted to your channel URL are displayed in the browser in real time.

1. Open [https://smee.io](https://smee.io) and click **Start a new channel**.
2. Copy the channel URL shown on the page (it looks like `https://smee.io/aBcDeFgHiJkLmNoP`).

Keep the smee.io page open in your browser — you will see events arrive here later.

## 4. Create the Flux Provider and Alert

Replace `<your-channel>` with the URL you copied.

```yaml
# smee-provider.yaml
---
apiVersion: notification.toolkit.fluxcd.io/v1beta3
kind: Provider
metadata:
  name: smee
  namespace: default
spec:
  type: generic
  address: https://smee.io/<your-channel>
```

```bash
kubectl apply -f smee-provider.yaml
```

```yaml
# component-alert.yaml
---
apiVersion: notification.toolkit.fluxcd.io/v1beta3
kind: Alert
metadata:
  name: component
  namespace: default
spec:
  providerRef:
    name: smee
  eventSources:
    - kind: Component
      name: '*'
      namespace: default
  eventSeverity: info
```

```bash
kubectl apply -f component-alert.yaml
```

## 5. Deploy podinfo and observe events

Follow [Scenario 2: Kustomize](../../getting-started/scenario-kustomize) to deploy podinfo from its Git repository. Create the source and component:

```yaml
# podinfo-git-source.yaml
---
apiVersion: source.toolkit.fluxcd.io/v1
kind: GitRepository
metadata:
  name: podinfo
  namespace: default
spec:
  interval: 5m
  url: https://github.com/stefanprodan/podinfo
  ref:
    branch: master
```

```bash
kubectl apply -f podinfo-git-source.yaml
```

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

Switch to your browser and watch the smee.io channel. As component-operator reconciles the `podinfo` component you should see a stream of events appear, each carrying:

- The component name and namespace
- The reconciliation state (`Progressing`, `Ready`, etc.)
- A human-readable message describing what happened

Once the component is `Ready`:

```bash
kubectl get component podinfo
```

Try triggering more events by suspending and resuming the component:

```bash
kubectl patch component podinfo --type merge -p '{"spec":{"suspend":true}}'
kubectl patch component podinfo --type merge -p '{"spec":{"suspend":false}}'
```

Each state change produces a new event in the smee.io channel.

## 6. Cleanup

Delete the podinfo resources:

```bash
kubectl delete component podinfo
kubectl delete gitrepository podinfo
```

Delete the notification objects:

```bash
kubectl delete alert component -n default
kubectl delete provider smee -n default
```

Restore component-operator to its default configuration (no event streaming):

```bash
helm upgrade --install component-operator \
  oci://ghcr.io/sap/component-operator/charts/component-operator \
  --namespace flux-system \
  --set options.eventsAddress=
```
