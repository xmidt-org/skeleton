// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package apiauth

import (
	"context"
	"errors"
	"net/http"
	"slices"
	"time"

	"github.com/lestrrat-go/jwx/v4/jwt"
	"github.com/xmidt-org/arrange/arrangehttp"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/bascule/basculehttp"
	"github.com/xmidt-org/bascule/basculejwt"
	"github.com/xmidt-org/clortho"
)

// Config is a struct that holds the configuration for the Auth middleware.
type Config struct {
	// Disable is a flag that if set to true, will disable all auth.  If this
	// is not set and no other auth is configured, then an error will be returned.
	Disable bool

	// Basic is a map of usernames to passwords.  If this is set, then basic auth
	// will be enabled.
	// If JWT is also set, then JWT will take precedence and basic auth will be ignored.
	Basic Basic

	// JWT holds the configuration for JWT based auth.
	JWT JWT
}

// JWT is a struct that holds the configuration for JWT based auth.
type JWT struct {
	// KeyProvider holds the configuration for the key provider.
	KeyProvider Provider

	// RequiredServiceCapabilities is a list of capabilities that are required
	// to be present to accept the token.  If any one of these capabilities are
	// present will allow the token to be accepted.
	RequiredServiceCapabilities []string
}

// Provider contains the configuration for accessing the public keys for JWT
// verification.
type Provider struct {
	// URL is the URL to get the public keys from.
	URL string

	// RefreshInterval is the interval to refresh the public keys.
	RefreshInterval time.Duration

	// HTTPClient is the configuration for the http client to use to get the public keys.
	HTTPClient arrangehttp.ClientConfig
}

// Basic is a map of usernames to passwords.
type Basic map[string]string

// Auth is a struct that holds the auth middleware.
type Auth struct {
	middleware *basculehttp.Middleware
	config     Config
}

// New creates a new Auth middleware.
func New(opts ...Option) (*Auth, error) {
	var err error
	var auth Auth

	opts = append(opts, validate())

	for _, opt := range opts {
		if opt != nil {
			if err := opt.apply(&auth); err != nil {
				return nil, err
			}
		}
	}

	ctx := context.Background()

	if auth.config.Basic != nil {
		auth.middleware, err = auth.config.Basic.middleware()
		if err != nil {
			return nil, errors.Join(err, errors.New("error creating basic auth middleware"))
		}
	}

	if auth.config.JWT.KeyProvider.URL != "" {
		auth.middleware, err = auth.config.JWT.middleware(ctx)
		if err != nil {
			return nil, errors.Join(err, errors.New("error creating jwt middleware"))
		}
	}

	return &auth, err
}

func (auth *Auth) Protected() bool {
	return auth.middleware != nil
}

func (auth *Auth) Then(h http.HandlerFunc) http.Handler {
	if auth.middleware == nil {
		return h
	}

	return auth.middleware.ThenFunc(h)
}

func (cfg *Basic) valid(token bascule.Token) error {
	if basic, ok := token.(basculehttp.BasicToken); ok {
		for u, p := range *cfg {
			if basic.UserName() == u && basic.Password() == p {
				return nil
			}
		}
	}

	return bascule.ErrBadCredentials
}

func (cfg *Basic) middleware() (*basculehttp.Middleware, error) {
	tp, err := basculehttp.NewAuthorizationParser(
		basculehttp.WithBasic(),
	)
	if err != nil {
		return nil, err
	}

	m, err := basculehttp.NewMiddleware(
		basculehttp.UseAuthenticator(

			basculehttp.NewAuthenticator(
				bascule.WithTokenParsers(tp),
				bascule.WithValidators(
					bascule.AsValidator[*http.Request](cfg.valid),
				),
			),
		),
	)

	return m, err
}

func (cfg *JWT) middleware(ctx context.Context) (*basculehttp.Middleware, error) {
	keys, err := cfg.KeyProvider.newKeyProvider(ctx)
	if err != nil {
		return nil, errors.Join(err, errors.New("error getting public keys"))
	}

	jwtp, err := basculejwt.NewTokenParser(jwt.WithKeyProvider(keys))
	if err != nil {
		return nil, errors.Join(err, errors.New("error creating token parser"))
	}

	tp, err := basculehttp.NewAuthorizationParser(
		basculehttp.WithScheme(basculehttp.SchemeBearer, jwtp),
	)
	if err != nil {
		return nil, errors.Join(err, errors.New("error creating authorization parser"))
	}

	m, err := basculehttp.NewMiddleware(
		basculehttp.UseAuthenticator(
			basculehttp.NewAuthenticator(
				bascule.WithTokenParsers(tp),
				bascule.WithValidators(
					bascule.AsValidator[*http.Request](cfg.valid),
				),
			),
		),
	)

	return m, err
}

// just checking for at least one capabiilty
func (cfg *JWT) valid(token bascule.Token) error {
	_, ok := token.(basculejwt.Claims)
	if !ok {
		return bascule.ErrBadCredentials
	}

	if len(cfg.RequiredServiceCapabilities) == 0 {
		return nil
	}

	capabilities, _ := bascule.GetCapabilities(token)
	for _, capability := range capabilities {
		if slices.Contains(cfg.RequiredServiceCapabilities, capability) {
			return nil
		}
	}

	return bascule.ErrUnauthorized
}

// keyReadyTimeout is how long to wait for the first set of public keys.
const keyReadyTimeout = 30 * time.Second

// newKeyProvider builds and starts a clortho key provider and waits for the
// first set of keys to be retrieved, so a misconfigured provider fails at
// startup instead of rejecting every token.
func (cfg *Provider) newKeyProvider(ctx context.Context) (*clortho.KeySetProvider, error) {
	client, err := cfg.HTTPClient.NewClient()
	if err != nil {
		return nil, err
	}

	kp, err := clortho.NewKeySetProvider(clortho.KeySetConfig{
		Sources: []clortho.RefreshSource{{
			URI:             cfg.URL,
			RefreshInterval: cfg.RefreshInterval,
			Client:          client,
		}},
	})
	if err != nil {
		return nil, err
	}

	err = kp.Start(ctx)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, keyReadyTimeout)
	defer cancel()

	err = waitForKeys(ctx, kp)
	if err != nil {
		return nil, errors.Join(err, kp.Stop(context.Background()))
	}

	return kp, nil
}

// waitForKeys blocks until every source has been retrieved at least once.
func waitForKeys(ctx context.Context, kp *clortho.KeySetProvider) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		ready := true
		for _, st := range kp.Status() {
			if st.LastRetrieved.IsZero() {
				ready = false
			}
		}

		if ready {
			return nil
		}

		select {
		case <-ctx.Done():
			return errors.Join(ctx.Err(), lastError(kp))
		case <-ticker.C:
		}
	}
}

func lastError(kp *clortho.KeySetProvider) error {
	var errs []error
	for _, st := range kp.Status() {
		if st.LastErr != nil {
			errs = append(errs, st.LastErr)
		}
	}

	return errors.Join(errs...)
}
