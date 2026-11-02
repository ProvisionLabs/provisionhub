package gitprovider

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ProvisionLabs/provisionhub/internal/credentials"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type GitHub struct {
	clientID         string
	http             *http.Client
	oauthURL, apiURL string
	wait             func(context.Context, time.Duration) error
}

func NewGitHub(clientID string) *GitHub {
	return &GitHub{clientID: strings.TrimSpace(clientID), http: &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirect refused") }}, oauthURL: "https://github.com", apiURL: "https://api.github.com", wait: waitContext}
}
func (g *GitHub) Configured() bool { return g.clientID != "" }
func (g *GitHub) Describe() Descriptor {
	return Descriptor{ID: "github", InstanceURL: "https://github.com", Methods: []string{"device_code"}}
}

func waitContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// request intentionally never returns provider bodies, request URLs, or transport errors.
// Those may contain tokens, user-controlled content, or OAuth codes.
func (g *GitHub) request(ctx context.Context, endpoint string, form url.Values, token string, out any) error {
	method := http.MethodGet
	var body io.Reader
	if form != nil {
		method = http.MethodPost
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return ErrUnavailable
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "ProvisionHub")
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}
	response, err := g.http.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return ErrUnavailable
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusUnauthorized {
		return ErrInvalid
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ErrUnavailable
	}
	if json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(out) != nil {
		return ErrUnavailable
	}
	return nil
}

func (g *GitHub) Begin(ctx context.Context) (Authorization, error) {
	if !g.Configured() {
		return Authorization{}, ErrNotConfigured
	}
	var result struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Interval        int    `json:"interval"`
	}
	// Empty scope requests public profile identification only, no repository permissions.
	if err := g.request(ctx, g.oauthURL+"/login/device/code", url.Values{"client_id": {g.clientID}}, "", &result); err != nil {
		return Authorization{}, err
	}
	if result.DeviceCode == "" || result.UserCode == "" || result.VerificationURI != "https://github.com/login/device" || result.ExpiresIn <= 0 || result.ExpiresIn > 3600 || result.Interval < 1 || result.Interval > 3600 {
		return Authorization{}, ErrUnavailable
	}
	expires := time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	return Authorization{UserCode: result.UserCode, VerificationURL: result.VerificationURI, ExpiresAt: expires, complete: func(ctx context.Context) (credentials.Secret, error) {
		pollCtx, cancel := context.WithDeadline(ctx, expires)
		defer cancel()
		interval := time.Duration(result.Interval) * time.Second
		for {
			if err := g.wait(pollCtx, interval); err != nil {
				if ctx.Err() != nil {
					return credentials.Secret{}, ctx.Err()
				}
				return credentials.Secret{}, ErrExpired
			}
			secret, code, newInterval, err := g.token(pollCtx, url.Values{"client_id": {g.clientID}, "device_code": {result.DeviceCode}, "grant_type": {"urn:ietf:params:oauth:grant-type:device_code"}})
			if err != nil {
				if ctx.Err() != nil {
					return credentials.Secret{}, ctx.Err()
				}
				if pollCtx.Err() != nil {
					return credentials.Secret{}, ErrExpired
				}
				return credentials.Secret{}, err
			}
			switch code {
			case "":
				return secret, nil
			case "authorization_pending":
				continue
			case "slow_down":
				interval += 5 * time.Second
				if time.Duration(newInterval)*time.Second > interval {
					interval = time.Duration(newInterval) * time.Second
				}
			case "expired_token", "token_expired":
				return credentials.Secret{}, ErrExpired
			case "access_denied":
				return credentials.Secret{}, ErrDenied
			default:
				return credentials.Secret{}, ErrUnavailable
			}
		}
	}}, nil
}

func (g *GitHub) token(ctx context.Context, form url.Values) (credentials.Secret, string, int, error) {
	var result struct {
		AccessToken      string `json:"access_token"`
		RefreshToken     string `json:"refresh_token"`
		ExpiresIn        int    `json:"expires_in"`
		RefreshExpiresIn int    `json:"refresh_token_expires_in"`
		Error            string `json:"error"`
		Interval         int    `json:"interval"`
	}
	if err := g.request(ctx, g.oauthURL+"/login/oauth/access_token", form, "", &result); err != nil {
		return credentials.Secret{}, "", 0, err
	}
	if result.Error != "" {
		return credentials.Secret{}, result.Error, result.Interval, nil
	}
	if result.AccessToken == "" {
		return credentials.Secret{}, "", 0, ErrInvalid
	}
	secret := credentials.Secret{AccessToken: result.AccessToken, RefreshToken: result.RefreshToken}
	if result.ExpiresIn > 0 {
		secret.ExpiresAt = time.Now().Add(time.Duration(result.ExpiresIn) * time.Second)
	}
	if result.RefreshExpiresIn > 0 {
		secret.RefreshExpiresAt = time.Now().Add(time.Duration(result.RefreshExpiresIn) * time.Second)
	}
	return secret, "", 0, nil
}

func (g *GitHub) Account(ctx context.Context, secret credentials.Secret) (Identity, error) {
	var result struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
	}
	if secret.AccessToken == "" {
		return Identity{}, ErrInvalid
	}
	if err := g.request(ctx, g.apiURL+"/user", nil, secret.AccessToken, &result); err != nil {
		return Identity{}, err
	}
	if result.ID <= 0 || result.Login == "" || len(result.Login) > 100 || strings.ContainsAny(result.Login, "\x00\r\n\x1b") {
		return Identity{}, ErrInvalid
	}
	return Identity{ID: strconv.FormatInt(result.ID, 10), Username: result.Login}, nil
}

func (g *GitHub) Refresh(ctx context.Context, old credentials.Secret) (credentials.Secret, error) {
	if !g.Configured() {
		return credentials.Secret{}, ErrNotConfigured
	}
	if old.RefreshToken == "" || (!old.RefreshExpiresAt.IsZero() && time.Now().After(old.RefreshExpiresAt)) {
		return credentials.Secret{}, ErrInvalid
	}
	secret, code, _, err := g.token(ctx, url.Values{"client_id": {g.clientID}, "grant_type": {"refresh_token"}, "refresh_token": {old.RefreshToken}})
	if err != nil {
		return credentials.Secret{}, err
	}
	if code != "" {
		return credentials.Secret{}, ErrInvalid
	}
	return secret, nil
}

// Local disconnect does not require client secrets or revoke other app sessions.
// Users can revoke the OAuth grant in GitHub's application settings.
func (g *GitHub) Disconnect(ctx context.Context, _ credentials.Secret) error { return ctx.Err() }
