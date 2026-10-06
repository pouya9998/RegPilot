package registry

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client abstracts Docker Registry HTTP API V2 (and future providers).
type Client interface {
	Ping(ctx context.Context) error
	Catalog(ctx context.Context) ([]string, error)
	ListTags(ctx context.Context, repository string) ([]string, error)
	GetManifest(ctx context.Context, repository, reference string) (*Manifest, error)
	DeleteManifest(ctx context.Context, repository, digest string) error
	GetBlobSize(ctx context.Context, repository, digest string) (int64, error)
}

type Manifest struct {
	Digest       string         `json:"digest"`
	MediaType    string         `json:"media_type"`
	SchemaVersion int           `json:"schema_version"`
	Architecture string         `json:"architecture,omitempty"`
	OS           string         `json:"os,omitempty"`
	Created      string         `json:"created,omitempty"`
	Layers       []Layer        `json:"layers"`
	Config       json.RawMessage `json:"config,omitempty"`
	Raw          json.RawMessage `json:"-"`
	TotalSize    int64          `json:"total_size"`
}

type Layer struct {
	MediaType string `json:"media_type"`
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
}

type Config struct {
	BaseURL            string
	Username           string
	Password           string
	InsecureSkipVerify bool
	Timeout            time.Duration
}

type v2Client struct {
	baseURL    string
	username   string
	password   string
	httpClient *http.Client
	token      string
}

func NewV2Client(cfg Config) Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: cfg.InsecureSkipVerify, //nolint:gosec // explicit per-registry option
		},
	}
	base := strings.TrimRight(cfg.BaseURL, "/")
	return &v2Client{
		baseURL:  base,
		username: cfg.Username,
		password: cfg.Password,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

func (c *v2Client) Ping(ctx context.Context) error {
	resp, err := c.do(ctx, http.MethodGet, "/v2/", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("%w: status %d", ErrUnreachable, resp.StatusCode)
	}
	return nil
}

func (c *v2Client) Catalog(ctx context.Context) ([]string, error) {
	var all []string
	path := "/v2/_catalog?n=1000"
	for {
		resp, err := c.do(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}
		var body struct {
			Repositories []string `json:"repositories"`
		}
		if err := decodeJSON(resp, &body); err != nil {
			return nil, err
		}
		all = append(all, body.Repositories...)
		next := linkNext(resp.Header.Get("Link"))
		if next == "" {
			break
		}
		path = next
	}
	return all, nil
}

func (c *v2Client) ListTags(ctx context.Context, repository string) ([]string, error) {
	repo := strings.Trim(repository, "/")
	path := fmt.Sprintf("/v2/%s/tags/list", repo)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusNotFound {
		resp.Body.Close()
		return []string{}, nil
	}
	var body struct {
		Tags []string `json:"tags"`
	}
	if err := decodeJSON(resp, &body); err != nil {
		return nil, err
	}
	if body.Tags == nil {
		return []string{}, nil
	}
	return body.Tags, nil
}

func (c *v2Client) GetManifest(ctx context.Context, repository, reference string) (*Manifest, error) {
	repo := strings.Trim(repository, "/")
	path := fmt.Sprintf("/v2/%s/manifests/%s", repo, reference)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", strings.Join([]string{
		"application/vnd.docker.distribution.manifest.v2+json",
		"application/vnd.docker.distribution.manifest.list.v2+json",
		"application/vnd.oci.image.manifest.v1+json",
		"application/vnd.oci.image.index.v1+json",
		"application/vnd.docker.distribution.manifest.v1+prettyjws",
		"application/json",
	}, ", "))
	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	if resp.StatusCode == http.StatusUnauthorized {
		if err := c.handleAuthChallenge(ctx, resp); err != nil {
			resp.Body.Close()
			return nil, err
		}
		resp.Body.Close()
		req2, _ := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
		req2.Header = req.Header.Clone()
		c.applyAuth(req2)
		resp, err = c.httpClient.Do(req2)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrUnreachable, err)
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthorized
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf("manifest fetch failed: %d %s", resp.StatusCode, string(b))
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	digest := resp.Header.Get("Docker-Content-Digest")
	mediaType := resp.Header.Get("Content-Type")

	m := &Manifest{
		Digest:    digest,
		MediaType: mediaType,
		Raw:       raw,
	}

	var base struct {
		SchemaVersion int    `json:"schemaVersion"`
		MediaType     string `json:"mediaType"`
		Architecture  string `json:"architecture"`
		Config        *struct {
			MediaType string `json:"mediaType"`
			Digest    string `json:"digest"`
			Size      int64  `json:"size"`
		} `json:"config"`
		Layers []struct {
			MediaType string `json:"mediaType"`
			Digest    string `json:"digest"`
			Size      int64  `json:"size"`
		} `json:"layers"`
		History []struct {
			V1Compatibility string `json:"v1Compatibility"`
		} `json:"history"`
		Manifests []struct {
			Platform *struct {
				Architecture string `json:"architecture"`
				OS           string `json:"os"`
			} `json:"platform"`
			Digest string `json:"digest"`
			Size   int64  `json:"size"`
		} `json:"manifests"`
	}
	_ = json.Unmarshal(raw, &base)

	if m.MediaType == "" {
		m.MediaType = base.MediaType
	}
	m.SchemaVersion = base.SchemaVersion
	m.Architecture = base.Architecture

	for _, l := range base.Layers {
		m.Layers = append(m.Layers, Layer{MediaType: l.MediaType, Digest: l.Digest, Size: l.Size})
		m.TotalSize += l.Size
	}
	if base.Config != nil {
		m.TotalSize += base.Config.Size
	}

	// Multi-arch index: use first linux/amd64 or first entry for platform hint
	if len(base.Manifests) > 0 && m.Architecture == "" {
		for _, mm := range base.Manifests {
			if mm.Platform != nil {
				m.Architecture = mm.Platform.Architecture
				m.OS = mm.Platform.OS
				if mm.Platform.Architecture == "amd64" && (mm.Platform.OS == "linux" || mm.Platform.OS == "") {
					break
				}
			}
		}
	}

	// Schema v1 created time
	if len(base.History) > 0 && base.History[0].V1Compatibility != "" {
		var v1 struct {
			Created string `json:"created"`
			OS      string `json:"os"`
			Arch    string `json:"architecture"`
		}
		if json.Unmarshal([]byte(base.History[0].V1Compatibility), &v1) == nil {
			m.Created = v1.Created
			if m.OS == "" {
				m.OS = v1.OS
			}
			if m.Architecture == "" {
				m.Architecture = v1.Arch
			}
		}
	}

	// Fetch config blob for created/os when available
	if base.Config != nil && base.Config.Digest != "" && m.Created == "" {
		cfgBody, err := c.getBlob(ctx, repo, base.Config.Digest)
		if err == nil {
			var imgCfg struct {
				Created      string `json:"created"`
				Architecture string `json:"architecture"`
				OS           string `json:"os"`
			}
			if json.Unmarshal(cfgBody, &imgCfg) == nil {
				m.Created = imgCfg.Created
				if m.Architecture == "" {
					m.Architecture = imgCfg.Architecture
				}
				if m.OS == "" {
					m.OS = imgCfg.OS
				}
			}
		}
	}

	return m, nil
}

func (c *v2Client) DeleteManifest(ctx context.Context, repository, digest string) error {
	if !strings.HasPrefix(digest, "sha256:") && !strings.HasPrefix(digest, "sha512:") {
		return fmt.Errorf("digest required for deletion, got %q", digest)
	}
	repo := strings.Trim(repository, "/")
	path := fmt.Sprintf("/v2/%s/manifests/%s", repo, digest)
	resp, err := c.do(ctx, http.MethodDelete, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return ErrNotFound
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusMethodNotAllowed {
		return fmt.Errorf("%w: registry may not have delete enabled (status %d)", ErrUnauthorized, resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("delete failed: %d %s", resp.StatusCode, string(b))
	}
	return nil
}

func (c *v2Client) GetBlobSize(ctx context.Context, repository, digest string) (int64, error) {
	repo := strings.Trim(repository, "/")
	path := fmt.Sprintf("/v2/%s/blobs/%s", repo, digest)
	resp, err := c.do(ctx, http.MethodHead, path, nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return 0, fmt.Errorf("blob head failed: %d", resp.StatusCode)
	}
	return resp.ContentLength, nil
}

func (c *v2Client) getBlob(ctx context.Context, repository, digest string) ([]byte, error) {
	path := fmt.Sprintf("/v2/%s/blobs/%s", repository, digest)
	resp, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("blob get failed: %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 8<<20))
}

func (c *v2Client) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	full := path
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		full = path
	} else if !strings.HasPrefix(path, "/") {
		full = c.baseURL + "/" + path
	} else {
		full = c.baseURL + path
	}

	req, err := http.NewRequestWithContext(ctx, method, full, body)
	if err != nil {
		return nil, err
	}
	c.applyAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}

	if resp.StatusCode == http.StatusUnauthorized {
		if err := c.handleAuthChallenge(ctx, resp); err == nil {
			resp.Body.Close()
			req2, _ := http.NewRequestWithContext(ctx, method, full, body)
			c.applyAuth(req2)
			return c.httpClient.Do(req2)
		}
	}
	return resp, nil
}

func (c *v2Client) applyAuth(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
		return
	}
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}
}

func (c *v2Client) handleAuthChallenge(ctx context.Context, resp *http.Response) error {
	io.Copy(io.Discard, resp.Body) //nolint:errcheck
	www := resp.Header.Get("Www-Authenticate")
	if www == "" {
		if c.username != "" {
			return ErrUnauthorized
		}
		return ErrUnauthorized
	}
	if strings.HasPrefix(strings.ToLower(www), "basic") {
		return nil // retry with basic via applyAuth
	}
	if !strings.HasPrefix(strings.ToLower(www), "bearer") {
		return ErrUnauthorized
	}

	realm, service, scope := parseBearerChallenge(www)
	if realm == "" {
		return ErrUnauthorized
	}
	u, err := url.Parse(realm)
	if err != nil {
		return err
	}
	q := u.Query()
	if service != "" {
		q.Set("service", service)
	}
	if scope != "" {
		q.Set("scope", scope)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}
	tokenResp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer tokenResp.Body.Close()
	if tokenResp.StatusCode >= 400 {
		return ErrUnauthorized
	}
	var tok struct {
		Token       string `json:"token"`
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tok); err != nil {
		return err
	}
	c.token = tok.Token
	if c.token == "" {
		c.token = tok.AccessToken
	}
	if c.token == "" {
		return ErrUnauthorized
	}
	return nil
}

func parseBearerChallenge(h string) (realm, service, scope string) {
	h = strings.TrimSpace(h)
	if i := strings.Index(strings.ToLower(h), "bearer"); i >= 0 {
		h = strings.TrimSpace(h[i+6:])
	}
	parts := splitAuthParams(h)
	return parts["realm"], parts["service"], parts["scope"]
}

func splitAuthParams(s string) map[string]string {
	out := map[string]string{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		kv := strings.SplitN(part, "=", 2)
		if len(kv) != 2 {
			continue
		}
		out[strings.ToLower(kv[0])] = strings.Trim(kv[1], `"`)
	}
	return out
}

func decodeJSON(resp *http.Response, dest interface{}) error {
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		return ErrUnauthorized
	}
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("request failed: %d %s", resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

func linkNext(link string) string {
	// <url>; rel="next"
	parts := strings.Split(link, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if !strings.Contains(p, `rel="next"`) && !strings.Contains(p, `rel=next`) {
			continue
		}
		start := strings.Index(p, "<")
		end := strings.Index(p, ">")
		if start >= 0 && end > start {
			return p[start+1 : end]
		}
	}
	return ""
}
