/*
SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and component-operator contributors
SPDX-License-Identifier: Apache-2.0
*/

package httprepository

import (
	"fmt"
	"net/http"
)

func GetArtifact(url string, digestHeader string, revisionHeader string) (string, string, string, error) {
	if digestHeader == "" {
		digestHeader = "etag"
	}
	if revisionHeader == "" {
		revisionHeader = digestHeader
	}

	httpClient := http.Client{
		CheckRedirect: func(req *http.Request, _ []*http.Request) error {
			if req.Response.Header.Get(digestHeader) != "" {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}
	resp, err := httpClient.Head(url)
	if err != nil {
		return "", "", "", err
	}
	switch {
	case resp.StatusCode >= 400:
		return "", "", "", fmt.Errorf("error calling source reference URL: %d (%s)", resp.StatusCode, http.StatusText(resp.StatusCode))
	case resp.StatusCode >= 300:
		location, err := resp.Location()
		if err != nil {
			return "", "", "", err
		}
		url = location.String()
	case resp.StatusCode >= 200:
		url = resp.Request.URL.String()
	default:
		return "", "", "", fmt.Errorf("referenced source not ready")
	}
	digest := resp.Header.Get(digestHeader)
	if digest == "" {
		return "", "", "", fmt.Errorf("missing digest on source reference")
	}
	revision := resp.Header.Get(revisionHeader)
	if revision == "" {
		return "", "", "", fmt.Errorf("missing revision on source reference")
	}
	return url, digest, revision, nil
}
