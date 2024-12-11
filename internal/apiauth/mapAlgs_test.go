// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package apiauth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustGenerateKey(what string) jwk.Key {
	list := strings.Split(what, ".")
	if len(list) < 3 {
		panic("invalid what.  Either 'rsa.public.kid123' or 'rsa.private.kid123' or 'ec.256.public.kid123' or 'ec.256.private.kid123'")
	}

	var generic any
	var kid string
	if len(list) == 3 {
		var kt, which string
		kt, which, kid = list[0], list[1], list[2]

		if kt != "rsa" {
			panic("invalid key type")
		}

		raw, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			panic(err)
		}

		generic = raw
		if which == "public" {
			generic = &raw.PublicKey
		}
	} else {
		var kt, bits, which string
		kt, bits, which, kid = list[0], list[1], list[2], list[3]

		if kt != "ec" {
			panic("invalid key type")
		}

		var ec elliptic.Curve
		switch bits {
		case "256":
			ec = elliptic.P256()
		case "384":
			ec = elliptic.P384()
		case "512", "521":
			ec = elliptic.P521()
		default:
			panic("invalid bits")
		}

		raw, err := ecdsa.GenerateKey(ec, rand.Reader)
		if err != nil {
			panic(err)
		}

		generic = raw
		if which == "public" {
			generic = &raw.PublicKey
		}
	}

	key, err := jwk.FromRaw(generic)
	if err != nil {
		panic(err)
	}

	err = key.Set("kid", kid)
	if err != nil {
		panic(err)
	}

	if key == nil {
		panic("key is nil")
	}
	return key
}

func TestExpandKeySetByAlgorithm(t *testing.T) {
	tests := []struct {
		key  jwk.Key
		want []jwa.SignatureAlgorithm
	}{
		{
			key: mustGenerateKey("rsa.private.kid123"),
			want: []jwa.SignatureAlgorithm{
				jwa.RS256,
				jwa.RS384,
				jwa.RS512,
				jwa.PS256,
				jwa.PS384,
				jwa.PS512,
			},
		}, {
			key:  mustGenerateKey("ec.256.private.kid123"),
			want: []jwa.SignatureAlgorithm{jwa.ES256},
		}, {
			key:  mustGenerateKey("ec.384.private.kid123"),
			want: []jwa.SignatureAlgorithm{jwa.ES384},
		}, {
			key:  mustGenerateKey("ec.512.private.kid123"),
			want: []jwa.SignatureAlgorithm{jwa.ES512},
		}, {
			key:  mustGenerateKey("ec.256.public.kid123"),
			want: []jwa.SignatureAlgorithm{jwa.ES256},
		}, {
			key:  mustGenerateKey("ec.384.public.kid123"),
			want: []jwa.SignatureAlgorithm{jwa.ES384},
		}, {
			key:  mustGenerateKey("ec.512.public.kid123"),
			want: []jwa.SignatureAlgorithm{jwa.ES512},
		},
	}

	for _, test := range tests {
		t.Run("", func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			ks := jwk.NewSet()
			require.NotNil(ks)

			err := ks.AddKey(test.key)
			require.NoError(err)

			ctx := context.Background()

			f := mapMissingAlgorithms(ctx)
			require.NotNil(f)

			newKS, err := f("localhost", ks)
			require.NoError(err)
			require.NotNil(newKS)

			keys := newKS.Keys(ctx)
			require.NotNil(keys)

			var got []jwa.SignatureAlgorithm
			for keys.Next(ctx) {
				key := keys.Pair().Value.(jwk.Key)
				alg, ok := key.Get(jwk.AlgorithmKey)
				require.True(ok)

				got = append(got, alg.(jwa.SignatureAlgorithm))
			}

			assert.ElementsMatch(test.want, got)

			// Now check to make sure if the algorithm is already set, we don't
			// overwrite it.
			again, err := f("localhost", newKS)
			require.NoError(err)
			require.NotNil(again)

			assert.ElementsMatch(test.want, got)
		})
	}
}

func TestGetKeys(t *testing.T) {
	// Create a test JWK set
	jwkSet := `{
        "keys": [
            {
                "kty": "EC",
                "crv": "P-256",
                "x": "f83OJ3D2xF4a8u6B3T1t5w",
                "y": "x_FEzRu9Q5R5t5w",
                "alg": "ES256",
                "kid": "1"
            }
        ]
    }`

	// Create a test HTTP server that returns the JWK set
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(jwkSet))
	}))
	defer server.Close()

	// Create a mock provider
	provider := Provider{
		URL:             server.URL,
		RefreshInterval: 15 * time.Minute,
	}

	// Call the getKeys function
	ctx := context.Background()
	keySet, err := provider.toKeySet(ctx)

	// Verify the results
	require.NoError(t, err)
	require.NotNil(t, keySet)

	keys := keySet.Keys(ctx)
	require.True(t, keys.Next(ctx))

	key := keys.Pair().Value.(jwk.Key)
	assert.Equal(t, "EC", key.KeyType().String())
	assert.Equal(t, "ES256", key.Algorithm().String())
	assert.Equal(t, "1", key.KeyID())
}
