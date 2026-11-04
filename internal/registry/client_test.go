package registry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestFetchCacheAndOffline(t *testing.T) {
	data, _ := fixture().Marshal()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write(data) }))
	client := Client{URL: server.URL, HTTP: server.Client(), CachePath: filepath.Join(t.TempDir(), "registry.json")}
	fresh, err := client.Fetch(context.Background())
	if err != nil || fresh.Stale || fresh.FetchedAt.IsZero() {
		t.Fatal(fresh, err)
	}
	server.Close()
	stale, err := client.Fetch(context.Background())
	if err != nil || !stale.Stale || stale.Warning == "" || !stale.FetchedAt.Equal(fresh.FetchedAt) {
		t.Fatal(stale, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Fetch(ctx); err == nil {
		t.Fatal("cancel ignored")
	}
	client.CachePath = ""
	if _, err := client.Fetch(context.Background()); err == nil {
		t.Fatal("network error ignored")
	}
}
func TestFetchInvalidAndRedirect(t *testing.T) {
	for _, body := range []string{`{"schemaVersion":2}`, `{"schemaVersion":1,"oops":1}`, strings.Repeat(" ", int(MaxIndexSize)+1)} {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
		client := Client{URL: server.URL, HTTP: server.Client()}
		if _, err := client.Fetch(context.Background()); err == nil {
			t.Fatal("invalid metadata accepted")
		}
		server.Close()
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "http://example.invalid", 302) }))
	defer server.Close()
	if _, err := (Client{URL: server.URL, HTTP: server.Client()}).Fetch(context.Background()); err == nil {
		t.Fatal("insecure redirect")
	}
	if _, err := (Client{URL: "http://example.invalid"}).Fetch(context.Background()); err == nil {
		t.Fatal("insecure source")
	}
}
