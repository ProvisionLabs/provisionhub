package gitprovider

import (
	"context"
	"errors"
	"fmt"
	"github.com/ProvisionLabs/provisionhub/internal/credentials"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestDeviceFlowPendingSlowDownAndAccount(t *testing.T) {
	poll := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		if r.URL.Path == "/login/device/code" {
			if r.Form.Get("scope") != "" || r.Form.Get("client_secret") != "" {
				t.Error("excess permissions or secret")
			}
			fmt.Fprint(w, `{"device_code":"private-device","user_code":"USER-CODE","verification_uri":"https://github.com/login/device","expires_in":900,"interval":5}`)
			return
		}
		if r.URL.Path == "/user" {
			if r.Header.Get("Authorization") != "Bearer private-token" {
				t.Error("missing authentication")
			}
			fmt.Fprint(w, `{"id":123,"login":"sample-user"}`)
			return
		}
		poll++
		switch poll {
		case 1:
			fmt.Fprint(w, `{"error":"authorization_pending"}`)
		case 2:
			fmt.Fprint(w, `{"error":"slow_down","interval":10}`)
		default:
			fmt.Fprint(w, `{"access_token":"private-token","token_type":"bearer"}`)
		}
	}))
	defer server.Close()
	g := NewGitHub("registered-test-client")
	g.oauthURL = server.URL
	g.apiURL = server.URL
	intervals := []time.Duration{}
	g.wait = func(ctx context.Context, d time.Duration) error { intervals = append(intervals, d); return ctx.Err() }
	auth, err := g.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	secret, err := auth.Wait(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(intervals, []time.Duration{5 * time.Second, 5 * time.Second, 10 * time.Second}) {
		t.Fatal(intervals)
	}
	identity, err := g.Account(context.Background(), secret)
	if err != nil || identity.ID != "123" || identity.Username != "sample-user" {
		t.Fatalf("identity: %v %v", identity, err)
	}
	if strings.Contains(fmt.Sprintf("%+v %#v", secret, secret), "private-token") {
		t.Fatal("secret formatter leaked")
	}
}

func TestDeviceFailuresAndCancellation(t *testing.T) {
	for _, code := range []string{"expired_token", "token_expired", "access_denied", "incorrect_client_credentials", "device_flow_disabled"} {
		t.Run(code, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.HasSuffix(r.URL.Path, "/device/code") {
					fmt.Fprint(w, `{"device_code":"private","user_code":"USER-CODE","verification_uri":"https://github.com/login/device","expires_in":900,"interval":5}`)
				} else {
					fmt.Fprintf(w, `{"error":%q,"error_description":"sensitive response"}`, code)
				}
			}))
			defer server.Close()
			g := NewGitHub("test")
			g.oauthURL = server.URL
			g.wait = func(ctx context.Context, _ time.Duration) error { return ctx.Err() }
			auth, err := g.Begin(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			_, err = auth.Wait(context.Background())
			if err == nil || strings.Contains(err.Error(), "sensitive") {
				t.Fatal("unsafe failure", err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if _, err := auth.Wait(ctx); !errors.Is(err, context.Canceled) {
				t.Fatal(err)
			}
		})
	}
	g := NewGitHub("")
	if _, err := g.Begin(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(waitContext(ctx, time.Hour), context.Canceled) {
		t.Fatal("wait ignored cancellation")
	}
}

func TestAccountAndRefreshFailuresAreSanitized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/user" {
			w.WriteHeader(401)
			fmt.Fprint(w, "private-token")
			return
		}
		fmt.Fprint(w, `{"access_token":"new-private-token","refresh_token":"new-refresh","expires_in":3600,"refresh_token_expires_in":7200}`)
	}))
	g := NewGitHub("test")
	g.apiURL = server.URL
	g.oauthURL = server.URL
	if _, err := g.Account(context.Background(), credentials.Secret{AccessToken: "private-token"}); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}
	secret, err := g.Refresh(context.Background(), credentials.Secret{RefreshToken: "old-refresh"})
	if err != nil || secret.AccessToken != "new-private-token" || secret.RefreshToken != "new-refresh" || secret.ExpiresAt.IsZero() {
		t.Fatal("refresh failed", err)
	}
	server.Close()
	if _, err := g.Account(context.Background(), secret); !errors.Is(err, ErrUnavailable) {
		t.Fatal(err)
	}
}
