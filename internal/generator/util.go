/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package generator

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
