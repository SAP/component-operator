/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// TODO: consolidate all the util files into an internal reuse package

func deepMerge(x map[string]any, y map[string]any) {
	for k := range y {
		if _, ok := x[k]; ok {
			if v, ok := x[k].(map[string]any); ok {
				if w, ok := y[k].(map[string]any); ok {
					deepMerge(v, w)
				} else {
					x[k] = w
				}
			} else {
				x[k] = y[k]
			}
		} else {
			x[k] = y[k]
		}
	}
}

func shallowMerge[T any](x map[string]T, y map[string]T) {
	for k := range y {
		x[k] = y[k]
	}
}

func sha256hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func calculateDigest(values ...any) string {
	raw, err := json.Marshal(values)
	if err != nil {
		panic(err)
	}
	return sha256hex(raw)
}
