---
title: "Deep Dive into Kustomize Sources"
linkTitle: "Deep Dive into Kustomize Sources"
weight: 5
description: >
  Explore .component-config.yaml, _helpers.tpl, includedFiles, includedKustomizations
---

This tutorial demonstrates advanced Kustomize source features: custom template delimiters, shared helper templates, reading external files with `readFile`, and including a base kustomization as a reusable sub-component. Everything lives inside a single self-contained `Blueprint`.

## Prerequisites

You need a Kubernetes cluster with Flux source-controller and component-operator installed. If you don't have one yet, follow the [Cluster Setup](../../getting-started/setup) guide.

## Source structure

The Blueprint organises its files like this:

```
kustomize-demo/
├── version                             ← version string, read via readFile
├── config/
│   ├── dev.yaml                        ← per-stage log configuration
│   └── prod.yaml
├── bases/
│   └── default/
│       ├── .component-config.yaml      ← makes version file available to the base
│       ├── _helpers.tpl                ← defines the "version" named template
│       └── configmap.yaml              ← ConfigMap that exposes the version
└── component/
    ├── .component-config.yaml          ← custom delimiters, includedFiles, includedKustomizations
    ├── _helpers.tpl                    ← defines selectorLabels, commonLabels, image, logLevel
    ├── kustomization.yaml              ← merges own resources with the base, injects labels
    └── deployment.yaml                 ← Deployment rendered with helper templates
```

The `component/` path is the entrypoint used by the component as `spec.path`. It references `bases/default` as a kustomize base.
The `config/` directory, and the `version` file contain configuration and other metadata which are dynamically read by the component while it is rendered.

## 1. Create the Blueprint

```yaml
# kustomize-demo-blueprint.yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Blueprint
metadata:
  name: kustomize-demo
  namespace: default
spec:
  files:

    # ── shared files at the root ──────────────────────────────────────────────

    version: |
      1.2.3

    config/dev.yaml: |
      logLevel: debug

    config/prod.yaml: |
      logLevel: warn

    # ── base kustomization ────────────────────────────────────────────────────

    bases/default/.component-config.yaml: |
      includedFiles:
        - ../../version

    bases/default/_helpers.tpl: |
      {{- define "version" -}}
      {{- readFile "../../version" | toString | trim -}}
      {{- end -}}

    bases/default/configmap.yaml: |
      ---
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: {{ name }}-version
      data:
        version: {{ include "version" . }}

    # ── component overlay ─────────────────────────────────────────────────────

    component/.component-config.yaml: |
      leftTemplateDelimiter: "{%"
      rightTemplateDelimiter: "%}"
      includedFiles:
        - ../config
      includedKustomizations:
        - ../bases/default

    component/_helpers.tpl: |
      {%- define "selectorLabels" -%}
      app: {% name %}
      {%- end -%}

      {%- define "commonLabels" -%}
      {% include "selectorLabels" . %}
      managed-by: component-operator
      {%- end -%}

      {%- define "image" -%}
      {%- with .image -%}
      {%- .repository | default "nginx" -%}:{%- .tag | default "latest" -%}
      {%- else -%}
      nginx:latest
      {%- end -%}
      {%- end -%}

      {%- define "logLevel" -%}
      {%- (.stage | default "prod" | printf "../config/%s.yaml" | readFile | toString | fromYaml).logLevel -%}
      {%- end -%}

    component/kustomization.yaml: |
      apiVersion: kustomize.config.k8s.io/v1beta1
      kind: Kustomization
      resources:
        - deployment.yaml
        - ../bases/default
      labels:
        - pairs:
            {%- include "commonLabels" . | nindent 12 %}

    component/deployment.yaml: |
      ---
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: {% name %}
        labels:
          {%- include "selectorLabels" . | nindent 10 %}
      spec:
        selector:
          matchLabels:
            {%- include "selectorLabels" . | nindent 12 %}
        template:
          metadata:
            labels:
              {%- include "selectorLabels" . | nindent 14 %}
          spec:
            containers:
              - name: app
                image: {% include "image" . %}
                env:
                  - name: LOG_LEVEL
                    value: {% include "logLevel" . %}
```

```bash
kubectl apply -f kustomize-demo-blueprint.yaml
```

## 2. Walk through the features

### Custom template delimiters

`component/.component-config.yaml` changes the template markers from the default `{{` / `}}` to `{%` / `%}`:

```yaml
leftTemplateDelimiter: "{%"
rightTemplateDelimiter: "%}"
```

Every template expression in `component/` therefore uses `{% name %}`, `{% include "image" . %}`, and so on. Using custom delimiters is a useful practice when the templates contain other templates (using the standard delimiters) that should be treated here as opaque strings (instead of being rendered).

The base kustomization (`bases/default/`) uses the default delimiters.

### includedFiles

```yaml
includedFiles:
  - ../../version
```

```yaml
includedFiles:
  - ../config
```

`includedFiles` makes files and directories **outside** the component's root path accessible to `readFile` and related functions. Without this declaration, a call like `readFile "../../version"` would fail because the path escapes the component's directory.

The `logLevel` helper demonstrates how to access configuration files with a dynamically calculated per-stage path at runtime:

```
{%- (.stage | default "prod" | printf "../config/%s.yaml" | readFile | toString | fromYaml).logLevel -%}
```

Step by step:
1. `.stage | default "prod"` — resolves the stage from values (falls back to `"prod"`)
2. `| printf "../config/%s.yaml"` — builds `../config/dev.yaml` (or `prod.yaml`)
3. `| readFile` — reads the file content (permitted because `../config` is in `includedFiles`)
4. `| toString | fromYaml` — parses the YAML into a map
5. `.logLevel` — extracts the `logLevel` field

### includedKustomizations

```yaml
includedKustomizations:
  - ../bases/default
```

This allows the component to reference `bases/default` as a base resource in `component/kustomization.yaml`, although being located outside `component/`.

```yaml
resources:
  - deployment.yaml
  - ../bases/default        ← merges the base's resources
```

Both entries are necessary: `includedKustomizations` handles **template scope**, `resources` handles **kustomize resource merging**.

### _helpers.tpl and named templates

Files whose names end in `.tpl` are rendered as templates, like all other files. However, it is a common practice to use `.tpl` files to define named reuse templates. In addition, `.tpl` files should not produce any output when rendered. 

`component/_helpers.tpl` defines four helpers:

| Template | What it does |
|---|---|
| `selectorLabels` | Returns a single label pair used in `matchLabels` |
| `commonLabels` | Extends `selectorLabels` with an additional `managed-by` label |
| `image` | Builds the container image string from `.image.repository` and `.image.tag` values |
| `logLevel` | Reads the stage-appropriate config file and extracts the `logLevel` key |

`bases/default/_helpers.tpl` defines one helper:

| Template | What it does |
|---|---|
| `version` | Reads `../../version` and returns the trimmed string |

## 3. Create the Component

```yaml
# kustomize-demo-component.yaml
---
apiVersion: core.cs.sap.com/v1alpha1
kind: Component
metadata:
  name: kustomize-demo
  namespace: default
spec:
  sourceRef:
    blueprint:
      name: kustomize-demo
  path: component
  values:
    stage: dev
    image:
      repository: nginx
      tag: alpine
```

```bash
kubectl apply -f kustomize-demo-component.yaml
```

Watch the component reach `Ready`:

```bash
kubectl get component kustomize-demo -w
```

## 4. Verify

Check that both resources were created and carry the injected labels:

```bash
kubectl get deployment,configmap -l app=kustomize-demo
```

The `kustomize-demo` Deployment (from the component overlay) and the `kustomize-demo-version` ConfigMap (from the base) should both appear. The `app=kustomize-demo` label was injected by the `labels` section in `component/kustomization.yaml`.

Verify the version ConfigMap — its value comes from `bases/default/_helpers.tpl` calling `readFile "../../version"`:

```bash
kubectl get configmap kustomize-demo-version \
  -o jsonpath='{.data.version}'
# 1.2.3
```

Verify the `LOG_LEVEL` environment variable — set to `debug` because `spec.values.stage` is `dev` and `config/dev.yaml` contains `logLevel: debug`:

```bash
kubectl get deployment kustomize-demo \
  -o jsonpath='{.spec.template.spec.containers[0].env[0].value}'
# debug
```

Switch the stage to `prod` and observe the variable change on the next reconcile:

```bash
kubectl patch component kustomize-demo \
  --type merge -p '{"spec":{"values":{"stage":"prod"}}}'

kubectl get deployment kustomize-demo \
  -o jsonpath='{.spec.template.spec.containers[0].env[0].value}'
# warn
```

## 5. Cleanup

```bash
kubectl delete component kustomize-demo
kubectl delete blueprint kustomize-demo
```
