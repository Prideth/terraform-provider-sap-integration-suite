// Package auth provides OAuth 2.0 authentication for SAP Integration Suite
// APIs, with token caching, expiry handling, and on-demand invalidation
// built on golang.org/x/oauth2. Tokens come from the client credentials
// grant, or from the password grant when a user is configured.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/clientcredentials"
)

// tokenFetchTimeout bounds one request to the token URL.
const tokenFetchTimeout = 60 * time.Second

// Config holds the OAuth 2.0 client credentials needed to authenticate
// against one SAP Integration Suite API area.
type Config struct {
	TokenURL     string
	ClientID     string
	ClientSecret string

	// Scopes is optional; most Integration Suite OAuth clients derive their
	// scopes from the role collections assigned to the service key instead
	// of requesting explicit scopes.
	Scopes []string

	// Username and Password switch to the password grant through the same
	// client: the token then carries the user's roles instead of the
	// client's own authorities. API Composition's Configuration API needs
	// this, because a client-credentials token of its service key carried no
	// scope for the API on a tenant (see docs/guides/api-composition.md).
	Username string
	Password string

	// Origin is the origin key of the user's identity provider in the BTP
	// subaccount. It is sent as login_hint so that the token service checks
	// the password against that identity provider instead of the default
	// one. Only used with Username.
	Origin string
}

func (c Config) validate() error {
	if c.TokenURL == "" {
		return fmt.Errorf("oauth: token URL must not be empty")
	}
	if c.ClientID == "" {
		return fmt.Errorf("oauth: client ID must not be empty")
	}
	if c.ClientSecret == "" {
		return fmt.Errorf("oauth: client secret must not be empty")
	}
	if (c.Username == "") != (c.Password == "") {
		return fmt.Errorf("oauth: username and password must be set together")
	}
	if c.Origin != "" && c.Username == "" {
		return fmt.Errorf("oauth: an origin needs a username and password")
	}
	return nil
}

// invalidatableTokenSource caches a single token in memory and serves it to
// every caller until it expires or Invalidate is called. It is a thin cache
// around a raw, always-fetches-fresh source, deliberately not
// golang.org/x/oauth2's own built-in reuse cache: that cache has no exported
// way to force a refresh, and a caller that receives an HTTP 401 from a
// downstream API needs exactly that, since SAP can invalidate a token
// before its stated expiry.
type invalidatableTokenSource struct {
	mu     sync.Mutex
	fetch  func() (*oauth2.Token, error)
	cached *oauth2.Token
}

func (s *invalidatableTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cached != nil && s.cached.Valid() {
		return s.cached, nil
	}

	tok, err := s.fetch()
	if err != nil {
		return nil, err
	}
	s.cached = tok
	return tok, nil
}

// Invalidate discards the cached token, if any, so the next Token() call
// performs a fresh client-credentials fetch instead of reusing a token that
// a downstream API has just rejected with a 401.
func (s *invalidatableTokenSource) Invalidate() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cached = nil
}

// HTTPClient builds an OAuth2 authenticated *http.Client
// and returns an invalidate function alongside it: calling invalidate
// forces the next outgoing request to fetch a fresh token rather than reuse
// the cached one. Both the client and invalidate are safe for concurrent
// use.
//
// base is the underlying *http.Client used to reach the token endpoint (so
// that proxy settings, timeouts, and TLS configuration stay consistent with
// the rest of the provider); it must not be nil.
func (c Config) HTTPClient(ctx context.Context, base *http.Client) (client *http.Client, invalidate func(), err error) {
	if err := c.validate(); err != nil {
		return nil, nil, err
	}

	ccConfig := clientcredentials.Config{
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		TokenURL:     c.TokenURL,
		Scopes:       c.Scopes,
	}
	if c.Username != "" {
		ccConfig.EndpointParams = c.passwordGrantParams()
	}

	// The client outlives ctx: the provider builds it in ConfigureProvider,
	// and Terraform cancels that request's context as soon as the call
	// returns, long before the first resource fetches a token. Tokens are
	// therefore fetched with a context that keeps ctx's values but not its
	// cancellation, and each fetch gets its own time limit instead. The
	// first acceptance run against a tenant failed every request with
	// "context canceled" on the token URL before this.
	tokenCtx := context.WithValue(context.WithoutCancel(ctx), oauth2.HTTPClient, base)

	source := &invalidatableTokenSource{
		fetch: func() (*oauth2.Token, error) {
			fetchCtx, cancel := context.WithTimeout(tokenCtx, tokenFetchTimeout)
			defer cancel()
			return ccConfig.Token(fetchCtx)
		},
	}

	// Built directly rather than via oauth2.NewClient: that helper silently
	// wraps whatever TokenSource it is given in its own internal
	// oauth2.ReuseTokenSource, which has no exported way to invalidate it —
	// calling source.Invalidate would clear our cache while that outer,
	// still-valid-looking cache kept serving the same stale token forever.
	client = &http.Client{
		Transport: &oauth2.Transport{
			Source: source,
			Base:   transportOf(base),
		},
	}
	if base != nil {
		client.Timeout = base.Timeout
		client.CheckRedirect = base.CheckRedirect
		client.Jar = base.Jar
	}

	return client, source.Invalidate, nil
}

// passwordGrantParams turns the client-credentials request into a password
// grant. clientcredentials.Config lets EndpointParams override grant_type,
// so the token cache, the invalidation and the client authentication stay
// the same for both grants.
func (c Config) passwordGrantParams() url.Values {
	params := url.Values{
		"grant_type": {"password"},
		"username":   {c.Username},
		"password":   {c.Password},
	}
	if c.Origin != "" {
		hint, _ := json.Marshal(struct {
			Origin string `json:"origin"`
		}{c.Origin}) // a struct of one string cannot fail to encode
		params.Set("login_hint", string(hint))
	}
	return params
}

// transportOf returns base's RoundTripper, or nil if base is nil or has no
// Transport configured; oauth2.Transport falls back to http.DefaultTransport
// in either case.
func transportOf(base *http.Client) http.RoundTripper {
	if base == nil {
		return nil
	}
	return base.Transport
}
