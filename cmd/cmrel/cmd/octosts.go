/*
Copyright 2026 The cert-manager Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"golang.org/x/oauth2"
)

// octoSTSExchange trades the OIDC token from ts for a short-lived GitHub
// installation token minted by octo-sts (https://github.com/octo-sts/app).
//
// scope is the "owner/repo" whose .github/chainguard/<identity>.sts.yaml trust
// policy decides whether the OIDC token is accepted and which permissions the
// GitHub token carries. baseURL is the octo-sts service, e.g. https://octo-sts.dev.
func octoSTSExchange(ctx context.Context, ts oauth2.TokenSource, baseURL, scope, identity string) (string, error) {
	q := url.Values{}
	q.Set("scope", scope)
	q.Set("identity", identity)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/sts/exchange?"+q.Encode(), nil)
	if err != nil {
		return "", err
	}

	resp, err := oauth2.NewClient(ctx, ts).Do(req)
	if err != nil {
		return "", fmt.Errorf("octo-sts exchange for %s/%s: %w", scope, identity, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<10))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("octo-sts exchange for %s/%s: %s: %s", scope, identity, resp.Status, body)
	}

	var out struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", fmt.Errorf("octo-sts exchange for %s/%s: decoding response: %w", scope, identity, err)
	}
	if out.Token == "" {
		return "", fmt.Errorf("octo-sts exchange for %s/%s: response contained no token", scope, identity)
	}
	return out.Token, nil
}
