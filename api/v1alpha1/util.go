/*
SPDX-FileCopyrightText: 2026 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package v1alpha1

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// TODO: consolidate all the util files into an internal reuse package

func ref[T any](x T) *T {
	return &x
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
