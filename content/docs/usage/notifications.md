---
title: "Notifications"
linkTitle: "Notifications"
weight: 16
description: >
  Streaming component events to the Flux notification controller
---

Component-operator can emit events to the [Flux notification controller](https://fluxcd.io/flux/components/notification/), enabling integration with Flux's alerting and notification infrastructure.

## Enabling Event Streaming

Start the component-operator controller with the `--events-address` flag pointing to the address of the Flux notification controller:

```
--events-address=http://notification-controller.flux-system.svc.cluster.local/
```

When set, component-operator will stream reconciliation events (such as successful applies, errors, and state transitions) to the notification controller, which can then forward them to configured alert providers (Slack, PagerDuty, etc.).

## Patching the Flux Alert CRD

To fully leverage Flux notifications — including the ability to create `Alert` objects that reference `Component` as an event source — the `alerts.notification.toolkit.fluxcd.io` CRD shipped with Flux must be patched to add `Component` as an allowed event source kind.

Apply the following JSON Patch to the Alert CRD:

```json
[
  {
    "op": "add",
    "path": "/spec/versions/0/schema/openAPIV3Schema/properties/spec/properties/eventSources/items/properties/kind/enum/-",
    "value": "Component"
  }
]
```

**Caveat:** in some older versions, the Alert CRD contained more versions. Adjust the patch according to your Flux version.

Once patched, you can create Flux `Alert` objects referencing `Component` resources as event sources, for example:

```yaml
apiVersion: notification.toolkit.fluxcd.io/v1beta3
kind: Alert
metadata:
  name: component-alerts
  namespace: flux-system
spec:
  providerRef:
    name: slack-provider
  eventSources:
    - kind: Component
      name: '*'
      namespace: production
  eventSeverity: info
```
