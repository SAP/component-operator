/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package generator

import (
	"regexp"
	_ "unsafe"
)

//go:linkname legalFileNamePattern sigs.k8s.io/kustomize/kyaml/filesys.legalFileNamePattern
var legalFileNamePattern *regexp.Regexp

func init() {
	legalFileNamePattern = regexp.MustCompile("^[a-zA-Z0-9-_.:#]+$")
}
