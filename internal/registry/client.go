package registry

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

const MaxIndexSize int64 = 2 << 20

// Source allows future registry adapters without coupling services to HTTP.
type Source interface {
	Fetch(context.Context) (Snapshot, error)
}
type Snapshot struct {
	Index     Index     `json:"index"`
	FetchedAt time.Time `json:"fetchedAt"`
	Stale     bool      `json:"-"`
	Warning   string    `json:"-"`
	URL       string    `json:"url"`
}
type Client struct {
	URL       string
	CachePath string
	HTTP      *http.Client
}

func HTTPClient() *http.Client {
	return &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		if req.URL.Scheme != "https" || req.URL.User != nil {
			return errors.New("registry redirects require HTTPS")
		}
		return nil
	}}
}
func (c Client) endpoint() string {
	if c.URL != "" {
		return c.URL
	}
	return OfficialURL
}
func (c Client) Fetch(ctx context.Context) (Snapshot, error) {
	endpoint := c.endpoint()
	if !secureURL(endpoint) {
		return Snapshot{}, errors.New("registry requires an HTTPS URL")
	}
	client := c.HTTP
	if client == nil {
		client = HTTPClient()
	}
	// Keep redirect security even for an injected transport.
	secured := *client
	secured.CheckRedirect = HTTPClient().CheckRedirect
	if secured.Timeout == 0 {
		secured.Timeout = 30 * time.Second
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Snapshot{}, err
	}
	response, err := secured.Do(req)
	if err != nil {
		return c.fallback(ctx, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return c.fallback(ctx, fmt.Errorf("registry returned HTTP %d", response.StatusCode))
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, MaxIndexSize+1))
	if err != nil {
		return c.fallback(ctx, err)
	}
	if int64(len(data)) > MaxIndexSize {
		return Snapshot{}, errors.New("registry exceeds 2 MiB limit")
	}
	index, err := Parse(data)
	if err != nil {
		return Snapshot{}, err
	}
	result := Snapshot{Index: index, FetchedAt: time.Now().UTC(), URL: endpoint}
	if c.CachePath != "" {
		if err := c.save(result); err != nil {
			result.Warning = "Registry fetched, but cache could not be saved: " + err.Error()
		}
	}
	return result, nil
}
func (c Client) Cached() (Snapshot, error) {
	info, err := os.Lstat(c.CachePath)
	if err != nil {
		return Snapshot{}, err
	}
	if !info.Mode().IsRegular() || info.Size() > MaxIndexSize+4096 {
		return Snapshot{}, errors.New("invalid registry cache file")
	}
	data, err := os.ReadFile(c.CachePath)
	if err != nil {
		return Snapshot{}, err
	}
	var result Snapshot
	if err := json.Unmarshal(data, &result); err != nil {
		return result, err
	}
	if result.URL != c.endpoint() || result.FetchedAt.IsZero() {
		return Snapshot{}, errors.New("registry cache source mismatch")
	}
	if err := result.Index.Validate(); err != nil {
		return Snapshot{}, err
	}
	result.Stale = true
	result.Warning = "Cached registry; not freshly verified"
	return result, nil
}
func (c Client) fallback(ctx context.Context, cause error) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	if c.CachePath != "" {
		if result, err := c.Cached(); err == nil {
			result.Warning = "Cached registry; refresh failed: " + cause.Error()
			return result, nil
		}
	}
	return Snapshot{}, fmt.Errorf("fetch registry (no usable cache): %w", cause)
}
func (c Client) save(snapshot Snapshot) error {
	if err := os.MkdirAll(filepath.Dir(c.CachePath), 0o700); err != nil {
		return err
	}
	data, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(c.CachePath), ".registry-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), c.CachePath)
}
