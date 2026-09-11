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
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/oauth2"
)

func TestOctoSTSExchange(t *testing.T) {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "oidc-token"})

	t.Run("sends the OIDC token and returns the GitHub token", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/sts/exchange" {
				t.Errorf("path = %q", r.URL.Path)
			}
			if got := r.URL.Query().Get("scope"); got != "cert-manager/cert-manager" {
				t.Errorf("scope = %q", got)
			}
			if got := r.URL.Query().Get("identity"); got != "cmrel-publish" {
				t.Errorf("identity = %q", got)
			}
			if got := r.Header.Get("Authorization"); got != "Bearer oidc-token" {
				t.Errorf("Authorization = %q", got)
			}
			_, _ = w.Write([]byte(`{"token":"ghs_abc"}`))
		}))
		defer srv.Close()

		tok, err := octoSTSExchange(context.Background(), ts, srv.URL, "cert-manager/cert-manager", "cmrel-publish")
		if err != nil {
			t.Fatal(err)
		}
		if tok != "ghs_abc" {
			t.Errorf("token = %q", tok)
		}
	})

	t.Run("surfaces the octo-sts error body", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, `{"code":7,"message":"trust policy: subject did not match"}`, http.StatusForbidden)
		}))
		defer srv.Close()

		_, err := octoSTSExchange(context.Background(), ts, srv.URL, "cert-manager/cert-manager", "cmrel-publish")
		if err == nil || !strings.Contains(err.Error(), "subject did not match") {
			t.Fatalf("err = %v", err)
		}
	})

	t.Run("rejects a response without a token", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{}`))
		}))
		defer srv.Close()

		if _, err := octoSTSExchange(context.Background(), ts, srv.URL, "cert-manager/cert-manager", "cmrel-publish"); err == nil {
			t.Fatal("expected an error")
		}
	})
}
