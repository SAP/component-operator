#!/usr/bin/env bash

set -eo pipefail

export GOROOT=$(go env GOROOT)

BASEDIR=$(realpath $(dirname "$0")/..)
TEMPDIR=$BASEDIR/tmp/gen
trap 'rm -rf "$TEMPDIR"' EXIT
mkdir -p "$TEMPDIR"

mkdir -p "$TEMPDIR"/apis/core.cs.sap.com
ln -s "$BASEDIR"/api/v1alpha1 "$TEMPDIR"/apis/core.cs.sap.com/v1alpha1

"$BASEDIR"/bin/client-gen \
  --clientset-name versioned \
  --input-base "" \
  --input github.com/sap/component-operator/tmp/gen/apis/core.cs.sap.com/v1alpha1 \
  --go-header-file "$BASEDIR"/hack/boilerplate.go.txt \
  --output-package github.com/sap/component-operator/pkg/client/clientset \
  --output-base "$TEMPDIR"/pkg/client \
  --plural-exceptions Component:components

"$BASEDIR"/bin/lister-gen \
  --input-dirs github.com/sap/component-operator/tmp/gen/apis/core.cs.sap.com/v1alpha1 \
  --go-header-file "$BASEDIR"/hack/boilerplate.go.txt \
  --output-package github.com/sap/component-operator/pkg/client/listers \
  --output-base "$TEMPDIR"/pkg/client \
  --plural-exceptions Component:components

"$BASEDIR"/bin/informer-gen \
  --input-dirs github.com/sap/component-operator/tmp/gen/apis/core.cs.sap.com/v1alpha1 \
  --versioned-clientset-package github.com/sap/component-operator/pkg/client/clientset/versioned \
  --listers-package github.com/sap/component-operator/pkg/client/listers \
  --go-header-file "$BASEDIR"/hack/boilerplate.go.txt \
  --output-package github.com/sap/component-operator/pkg/client/informers \
  --output-base "$TEMPDIR"/pkg/client \
  --plural-exceptions Component:components

find "$TEMPDIR"/pkg/client -name "*.go" -exec \
  perl -pi -e "s#github\.com/sap/component-operator/tmp/gen/apis/core\.cs\.sap\.com/v1alpha1#github.com/sap/component-operator/api/v1alpha1#g" \
  {} +

rm -rf "$BASEDIR"/pkg/client
mv "$TEMPDIR"/pkg/client/github.com/sap/component-operator/pkg/client "$BASEDIR"/pkg

cd "$BASEDIR"
go mod tidy
go fmt ./pkg/client/...
go vet ./pkg/client/...
