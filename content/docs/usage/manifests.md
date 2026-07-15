---
title: "Manifests"
linkTitle: "Manifests"
weight: 2
description: >
  How manifests are structured, templated, and rendered
---

The files making up a component's manifests are provided by the configured [source](../sources). The field `spec.path` defines the subfolder within the source artifact to use as the entry point (default: the root of the artifact).

Component-operator automatically determines how to render the manifests based on the contents of that directory:

- If a `Chart.yaml` file exists in `spec.path`, the content is rendered as a **Helm chart** using the [HelmGenerator](https://sap.github.io/component-operator-runtime/docs/generators/helm/).
- Otherwise, the [KustomizeGenerator](https://sap.github.io/component-operator-runtime/docs/generators/kustomize/) logic is applied, which also supports plain YAML manifests without a `kustomization.yaml`.

## Values and ValuesFrom

Parameters for the manifests can be provided inline via `spec.values` or loaded from Kubernetes Secrets via `spec.valuesFrom`:

```yaml
spec:
  values:
    replicaCount: 3
    image:
      tag: v1.2.0
  valuesFrom:
    - name: my-values-secret
      # key: values.yaml  # optional; defaults to trying 'values', 'values.yaml', 'values.yml'
```

If multiple entries are listed in `spec.valuesFrom`, their contents are deeply merged in order of appearance. The inline `spec.values` is merged last and takes precedence over all secret-provided values.

The aggregated values are passed as parameters to either the HelmGenerator or the KustomizeGenerator.

## Helm Manifests

When `spec.path` contains a `Chart.yaml`, component-operator renders the source as a Helm chart. The HelmGenerator emulates Helm behavior without using the Helm SDK. A few differences apply:

- Not all Helm template functions are supported (`toToml` is unsupported; all others should work).
- The `.Release` builtin is supported. `Release.IsInstall` is `true` during the first reconcile iteration (when `status.revision` equals 1), and `Release.IsUpgrade` is its inverse. `Release.Revision` increments whenever the component manifest or any of its references changes.
- The `.Chart` builtin supports: `Name`, `Version`, `Type`, `AppVersion`, `Dependencies`.
- The `.Capabilities` builtin supports `KubeVersion` and `APIVersions`.
- The `.Files` builtin is supported but excludes Helm-reserved paths (`Chart.yaml`, `templates/`, etc.).
- `pre-delete` and `post-delete` hooks are not allowed. Test and rollback hooks are ignored. `pre-install`, `post-install`, `pre-upgrade`, and `post-upgrade` hooks are handled in a slightly adapted way.
- The `.helmignore` file is currently not evaluated.

For more details, see the [HelmGenerator documentation](https://sap.github.io/component-operator-runtime/docs/generators/helm/).

## Kustomize Manifests

When no `Chart.yaml` is present in `spec.path`, the KustomizeGenerator is used. It supports:

- Full kustomizations (with an explicit `kustomization.yaml`)
- Plain YAML manifests (auto-generates a `kustomization.yaml` from all `.yaml`/`.yml` files found)
- Go-templating on all YAML files (Helm-style templating with sprig functions, plus extras)

### Accessing values

Helm makes certain builtin variables available. For example, Helm uses `.Values` to allow access to the aggregated parameters, and has some additional builtin variables, such as `.Release` or `.Files`.

In KustomizeGenerator this is not the case, there are no builtin variables. The effective values (default values plus what is specified in `spec.values` and `spec.valuesFrom`) are accessible at `$` resp `.` inside templates. Functionality analogous to Helm's `.Release`, `.Chart`, `.Capabilities` or `.Files` variables is provided through template functions.

### Template Functions

In addition to all [sprig](http://masterminds.github.io/sprig) functions and the standard Helm-like set (`include`, `tpl`, `lookup`), the following important functions are available:

| Function | Description |
|----------|-------------|
| `namespace` | Return the deployment namespace (`spec.namespace`, if set) |
| `name` | Return the deployment name (`spec.name`, if set) |
| `required` | Fail with message if value is nil or empty |
| `lookup` | Lookup resource via target cluster client (return empty object on 404) |
| `mustLookup` | Lookup resource via target cluster client (fail on 404) |
| `localLookup` | Lookup resource via local cluster client (return empty object on 404) |
| `mustLocalLookup` | Lookup resource via local cluster client (fail on 404) |
| `lookupWithKubeConfig` | Lookup resource using a given kubeconfig (return empty object on 404) |
| `mustLookupWithKubeConfig` | Lookup resource using a given kubeconfig (fail on 404) |
| `lookupList` / `localLookupList` / `lookupListWithKubeConfig` | List resources |
| `listFiles` | List files matching a pattern relative to `spec.path` |
| `existsFile` | Check if a file exists relative to `spec.path` |
| `readFile` | Read a file relative to `spec.path` |
| `kubernetesVersion` | Return Kubernetes version details of the deployment target |
| `apiResources` | Return API discovery information of the deployment target |
| `componentDigest` | Returns the digest of the component, considering spec, annotations, and references, such as secrets |
| `componentRevision` | Returns the revision of the component; that is, a counter that is increased whenever the `componentDigest` changes |
| `component` | Returns the current component object as a whole, as golang struct |

The `lookup` / `mustLookup` functions use the **target** client (the cluster where dependent objects are deployed), while `localLookup` / `mustLocalLookup` use the **local** client (the cluster where the controller runs). See [Impersonation and Remote Clusters](../impersonation) for implications when a `kubeConfig` is provided.

The complete list of supported template functions can be found [here](https://sap.github.io/component-operator-runtime/docs/generators/kustomize/).

### Tuning the Kustomize Source

Behavior can be adjusted by placing special files in the `spec.path` directory:

- **`.component-ignore`**: Exclude files from templating (uses `.gitignore` syntax). Excluded files are still accessible via `readFile`.
- **`.component-config.yaml`**: Fine-grained configuration including custom template delimiters, additional included files, and included sub-kustomizations.

Example `.component-config.yaml`:

```yaml
leftTemplateDelimiter: "{%"
rightTemplateDelimiter: "%}"
values:
  image:
    repository: ghcr.io/acme/app
includedFiles:
  - ../shared/helpers.yaml
includedKustomizations:
  - subcomponent/
```

Note that remote references in kustomizations are not supported.

For full details, see the [KustomizeGenerator documentation](https://sap.github.io/component-operator-runtime/docs/generators/kustomize/).

## PostBuild

After the manifests are rendered, `spec.postBuild` allows for additional variable substitution and kustomize patches:

- **`spec.postBuild.substitute`**: Inline `KEY: VALUE` pairs used for bash-style variable substitution (using [envsubst](https://github.com/drone/envsubst) syntax).
- **`spec.postBuild.substituteFrom`**: Secrets whose keys are used as variable names and values as substitution values.
- **`spec.postBuild.patches`**: A list of kustomize strategic-merge or JSON patches applied after substitution.
- **`spec.postBuild.images`**: A list of kustomize image replacements applied after substitution.

If multiple `substituteFrom` secrets and inline `substitute` entries are provided, they are merged in order of appearance, with inline values taking final precedence.

```yaml
spec:
  postBuild:
    substitute:
      ENVIRONMENT: production
      REGION: eu-central-1
    substituteFrom:
      - name: env-secrets
    patches:
      - patch: |
          - op: replace
            path: /spec/replicas
            value: 5
        target:
          group: apps/v1
          kind: Deployment
          name: my-app
          namespace: my-ns
```
