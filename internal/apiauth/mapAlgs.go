// SPDX-FileCopyrightText: 2024 Comcast Cable Communications Management, LLC
// SPDX-License-Identifier: Apache-2.0

package apiauth

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// mapMissingAlgorithms is a jwk.PostFetchFunc that will add missing
// algorithms to keys that are missing them.  This is useful when
// you have a key that is missing the "alg" field, but you know what
// the algorithm should be.  Sometimes the public key provider does not
// provide the "alg" field, and this function can be used to add it.
func mapMissingAlgorithms(ctx context.Context) jwk.PostFetchFunc {
	return func(url string, keySet jwk.Set) (jwk.Set, error) {
		newKeys := jwk.NewSet()
		keys := keySet.Keys(ctx)
		for keys.Next(ctx) {
			key := keys.Pair().Value.(jwk.Key)
			if key.Algorithm().String() != "" {
				err := newKeys.AddKey(key)
				if err != nil {
					return keySet, err
				}
				continue
			}

			algs, err := keyToAlgs(key)
			if err != nil {
				return keySet, err
			}

			for _, alg := range algs {
				keyDup, err := key.Clone()
				if err != nil {
					return keySet, err
				}

				err = keyDup.Set(jwk.AlgorithmKey, alg)
				if err != nil {
					return keySet, err
				}

				err = newKeys.AddKey(keyDup)
				if err != nil {
					return keySet, err
				}
			}
		}
		return newKeys, nil
	}
}

// keyToAlgs is a helper function that will return the algorithms that are
// supported by the key.
func keyToAlgs(key jwk.Key) ([]jwa.SignatureAlgorithm, error) {
	kt := key.KeyType()

	switch kt {
	case jwa.RSA:
		return []jwa.SignatureAlgorithm{jwa.RS256, jwa.RS384, jwa.RS512, jwa.PS256, jwa.PS384, jwa.PS512}, nil
	case jwa.OKP:
		return []jwa.SignatureAlgorithm{jwa.EdDSA}, nil
	case jwa.EC:
		var raw any
		err := key.Raw(&raw)
		if err != nil {
			return nil, err
		}
		switch k := raw.(type) {
		case *ecdsa.PublicKey:
			switch k.Curve {
			case elliptic.P256():
				return []jwa.SignatureAlgorithm{jwa.ES256}, nil
			case elliptic.P384():
				return []jwa.SignatureAlgorithm{jwa.ES384}, nil
			case elliptic.P521():
				return []jwa.SignatureAlgorithm{jwa.ES512}, nil
			}
		case *ecdsa.PrivateKey:
			switch k.Curve {
			case elliptic.P256():
				return []jwa.SignatureAlgorithm{jwa.ES256}, nil
			case elliptic.P384():
				return []jwa.SignatureAlgorithm{jwa.ES384}, nil
			case elliptic.P521():
				return []jwa.SignatureAlgorithm{jwa.ES512}, nil
			}
		default:
		}
	default:
	}

	return nil, fmt.Errorf("unsupported key type %s", kt)
}
