package tests

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/regpilot/regpilot/internal/registry"
)

func TestAddRegistry(t *testing.T) {
	app := setupTestApp(t)
	tok := app.token(t, "admin", "change-me-admin")
	w := app.do("POST", "/api/registries", tok, map[string]interface{}{
		"name":     "Local",
		"url":      "http://127.0.0.1:5000",
		"username": "u",
		"password": "secret-pass",
	})
	if w.Code != 201 {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	data := resp["data"].(map[string]interface{})
	if _, ok := data["password"]; ok {
		t.Fatal("password must not be in response")
	}
	if data["has_password"] != true {
		t.Fatal("expected has_password true")
	}

	// list also hides password
	w = app.do("GET", "/api/registries", tok, nil)
	if strings.Contains(w.Body.String(), "secret-pass") {
		t.Fatal("password leaked in list response")
	}
}

func TestRegistryClientCatalogAndTags(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v2/" {
			w.WriteHeader(200)
			return
		}
		if r.URL.Path == "/v2/_catalog" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"repositories":["prime-core-api","prime-admin"]}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/tags/list") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"name":"prime-core-api","tags":["v1.0.0","v1.4.2"]}`))
			return
		}
		if strings.Contains(r.URL.Path, "/manifests/") {
			w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
			w.Header().Set("Docker-Content-Digest", "sha256:abc123")
			_, _ = w.Write([]byte(`{
				"schemaVersion":2,
				"mediaType":"application/vnd.docker.distribution.manifest.v2+json",
				"config":{"mediaType":"application/vnd.docker.container.image.v1+json","size":100,"digest":"sha256:cfg"},
				"layers":[{"mediaType":"application/vnd.docker.image.rootfs.diff.tar.gzip","size":420,"digest":"sha256:layer1"}]
			}`))
			return
		}
		if strings.Contains(r.URL.Path, "/blobs/") {
			if r.Method == http.MethodDelete {
				w.WriteHeader(202)
				return
			}
			w.WriteHeader(404)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/v2/prime-core-api/manifests/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(202)
			return
		}
		w.Header().Set("Docker-Content-Digest", "sha256:abc123")
		w.Header().Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
		_, _ = io.WriteString(w, `{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"x","size":100,"digest":"sha256:cfg"},"layers":[{"mediaType":"x","size":420,"digest":"sha256:layer1"}]}`)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client := registry.NewV2Client(registry.Config{BaseURL: srv.URL})
	ctx := context.Background()
	repos, err := client.Catalog(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 {
		t.Fatalf("repos: %v", repos)
	}
	tags, err := client.ListTags(ctx, "prime-core-api")
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 2 {
		t.Fatalf("tags: %v", tags)
	}
	m, err := client.GetManifest(ctx, "prime-core-api", "v1.4.2")
	if err != nil {
		t.Fatal(err)
	}
	if m.Digest != "sha256:abc123" {
		t.Fatalf("digest: %s", m.Digest)
	}
	if m.TotalSize != 520 {
		t.Fatalf("size: %d", m.TotalSize)
	}
	if err := client.DeleteManifest(ctx, "prime-core-api", "sha256:abc123"); err != nil {
		t.Fatal(err)
	}
}

func TestCryptoRoundTrip(t *testing.T) {
	enc, err := registryEncrypt("my-secret", "test-encryption-key-32b!!!!")
	if err != nil {
		t.Fatal(err)
	}
	if enc == "my-secret" {
		t.Fatal("not encrypted")
	}
	dec, err := registryDecrypt(enc, "test-encryption-key-32b!!!!")
	if err != nil {
		t.Fatal(err)
	}
	if dec != "my-secret" {
		t.Fatalf("got %q", dec)
	}
}
