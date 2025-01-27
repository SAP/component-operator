/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package generator

import (
	"context"
	"encoding/json"

	"github.com/drone/envsubst"
	"github.com/sap/go-generics/maps"

	"sigs.k8s.io/controller-runtime/pkg/client"
	kyaml "sigs.k8s.io/yaml"

	"github.com/sap/component-operator-runtime/pkg/component"
	"github.com/sap/component-operator-runtime/pkg/manifests"
	componentoperatorruntimetypes "github.com/sap/component-operator-runtime/pkg/types"

	operatorv1alpha1 "github.com/sap/component-operator/api/v1alpha1"
)

type Generator struct{}

var _ manifests.Generator = &Generator{}

func NewGenerator() (*Generator, error) {
	return &Generator{}, nil
}

func (g *Generator) Generate(ctx context.Context, namespace string, name string, parameters componentoperatorruntimetypes.Unstructurable) ([]client.Object, error) {
	reconcilerName, err := component.ReconcilerNameFromContext(ctx)
	if err != nil {
		return nil, err
	}

	spec := parameters.(*operatorv1alpha1.ComponentSpec)

	url := spec.SourceRef.Url()
	digest := spec.SourceRef.Digest()
	path := spec.Path

	var decryptionProvider string
	var decryptionKeys map[string][]byte
	if spec.Decryption != nil {
		decryptionProvider = spec.Decryption.Provider
		decryptionKeys = spec.Decryption.SecretRef.Data()
	}

	generator, err := GetGenerator(url, path, digest, decryptionProvider, decryptionKeys)
	if err != nil {
		return nil, err
	}

	values := make(map[string]any)
	for _, ref := range spec.ValuesFrom {
		var v map[string]any
		if err := kyaml.Unmarshal(ref.Value(), &v); err != nil {
			return nil, err
		}
		deepMerge(values, v)
	}
	if spec.Values != nil {
		var v map[string]any
		if err := json.Unmarshal(spec.Values.Raw, &v); err != nil {
			return nil, err
		}
		deepMerge(values, v)
	}

	objects, err := generator.Generate(ctx, namespace, name, componentoperatorruntimetypes.UnstructurableMap(values))
	if err != nil {
		return nil, err
	}

	substitutions := make(map[string]string)
	if spec.PostBuild != nil {
		for _, ref := range spec.PostBuild.SubstituteFrom {
			shallowMerge(substitutions, maps.Collect(ref.Data(), func(x []byte) string { return string(x) }))
		}
		shallowMerge(substitutions, spec.PostBuild.Substitute)
	}

	if len(substitutions) == 0 {
		return objects, nil
	}

	var substitutedObjects []client.Object
	for _, object := range objects {
		if object.GetAnnotations()[reconcilerName+"/disableSubstitution"] == "true" {
			continue
		}
		rawObject, err := kyaml.Marshal(object)
		if err != nil {
			return nil, err
		}
		stringObject := string(rawObject)
		stringObject, err = envsubst.Eval(stringObject, func(s string) string {
			return substitutions[s]
		})
		if err != nil {
			return nil, err
		}
		rawObject = []byte(stringObject)
		if err := kyaml.Unmarshal(rawObject, object); err != nil {
			return nil, err
		}
		substitutedObjects = append(substitutedObjects, object)
	}

	return substitutedObjects, nil
}
