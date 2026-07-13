---
title: "Drift Detection"
linkTitle: "Drift Detection"
weight: 3
description: >
  How component-operator detects and handles drift
---

## Object Digest

Whenever a dependent object is to be applied to the cluster, a digest is calculated for the object. This digest value is persisted as annotation `component-operator.cs.sap.com/digest`. The object is considered as synced if the new digest value equals the currently persisted one, and as out of sync otherwise.

If the object is out of sync, component-operator reapplies it to the Kubernetes API, including the new digest value. If the object is in sync, it will normally not be reapplied, with one exception: after the effective reapply interval, the object is force-reapplied. This ensures that manual out-of-band changes are reverted, and other glitches get fixed.

## Reapply Interval

The default reapply interval is 60 minutes, which can be overridden on component level by setting `spec.reapplyInterval`, and on object level by specifying the annotation `component-operator.cs.sap.com/reapply-interval`.

## Reconcile Modes

By default, only the submitted manifest is considered to calculate the object digest. However, this behavior can be tweaked by setting a [reconcile policy](../reconcile-modes) on component or object level.