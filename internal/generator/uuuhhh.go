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

// for internal reasons, we need kyaml's filesys implementation to allow '#' in filenames;
// unfortunately, there is no way to configure this, so we have to use go:linkname ...

func init() {
	legalFileNamePattern = regexp.MustCompile("^[a-zA-Z0-9-_.:#]+$")
}
