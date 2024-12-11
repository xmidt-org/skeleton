// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package apiauth

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/xmidt-org/arrange/arrangehttp"
	"github.com/xmidt-org/bascule"
	"github.com/xmidt-org/bascule/basculehttp"
	"github.com/xmidt-org/bascule/basculejwt"
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

	// DisableAutoAddMissingAlgorithm is a flag that if set to true, will disable
	// the automatic addition of missing algorithms to keys that are missing them.
	// If this is not set, then the default is to add missing algorithms to keys
	// that are missing the "alg" field.  Sometimes the public key provider does
	// not provide the "alg" field, and this function can be used to add it.
	DisableAutoAddMissingAlgorithm bool
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
			return nil, errors.Join(err, fmt.Errorf("error creating basic auth middleware"))
		}
	}

	if auth.config.JWT.KeyProvider.URL != "" {
		auth.middleware, err = auth.config.JWT.middleware(ctx)
		if err != nil {
			return nil, errors.Join(err, fmt.Errorf("error creating jwt middleware"))
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
	keys, err := cfg.KeyProvider.toKeySet(ctx)
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("error getting public keys"))
	}

	jwtp, err := basculejwt.NewTokenParser(jwt.WithKeySet(keys))
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("error creating token parser"))
	}

	tp, err := basculehttp.NewAuthorizationParser(
		basculehttp.WithScheme(basculehttp.SchemeBearer, jwtp),
	)
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("error creating authorization parser"))
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
		for _, serviceCapability := range cfg.RequiredServiceCapabilities {
			if capability == serviceCapability {
				return nil
			}
		}
	}

	return bascule.ErrUnauthorized
}

func (cfg *Provider) toKeySet(ctx context.Context) (jwk.Set, error) {
	cache := jwk.NewCache(ctx)

	opts := []jwk.RegisterOption{
		jwk.WithRefreshInterval(cfg.RefreshInterval),
	}

	if !cfg.DisableAutoAddMissingAlgorithm {
		opts = append(opts, jwk.WithPostFetcher(mapMissingAlgorithms(ctx)))
	}

	client, err := cfg.HTTPClient.NewClient()
	if err != nil {
		return nil, err
	}

	opts = append(opts, jwk.WithHTTPClient(client))

	err = cache.Register(cfg.URL, opts...)
	if err != nil {
		return nil, err
	}

	return jwk.NewCachedSet(cache, cfg.URL), err
}
